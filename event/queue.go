package event

import (
	"encoding/binary"
	"github.com/elliotmr/gdl/ticker"
	"github.com/pkg/errors"
	"sync"
	"sync/atomic"
	"time"
)

const MaxQueued = 65535

const (
	Add = iota
	Peek
	Get
)

var WaitTimeoutExceeded error = eventFilteredError{}

type eventFilteredError struct{}

func (eventFilteredError) Error() string   { return "wait timeout exceeded" }
func (eventFilteredError) Timeout() bool   { return true }
func (eventFilteredError) Temporary() bool { return true }

type Filter func(userdata interface{}, event Event) bool

//
type Pumper interface {
	Pump(q *Queue)
}

type Watcher struct {
	Callback Filter
	Userdata interface{}
}

type Entry struct {
	ev   Data
	prev *Entry
	next *Entry
}

// TODO(mde): implement disabled events

type Queue struct {
	// Sync helpers
	lock *sync.Mutex

	// Atomics
	active int32
	count  int32

	// pointers
	head *Entry
	tail *Entry
	free *Entry

	// other
	maxEventsSeen int32
	// TODO(mde): implement MWMsg

	// i/o sources sources
	sources  []Pumper
	watchers []*Watcher
	wmu      *sync.Mutex

	// event filter
	ok     Filter
	okdata interface{}

	disabled [256][8]uint32
}

func (q *Queue) Start() error {
	if q == nil {
		q = &Queue{}
	}
	if q.lock == nil {
		q.lock = &sync.Mutex{}
		q.wmu = &sync.Mutex{}
	}
	q.Disable(TextInput)
	q.Disable(TextEditing)
	q.Disable(SysWMEvent)

	atomic.StoreInt32(&q.active, 1)
	return nil
}

func (q *Queue) Stop() {
	if q.lock != nil {
		q.lock.Lock()
		defer q.lock.Unlock()
	}

	atomic.StoreInt32(&q.active, 0)
	atomic.StoreInt32(&q.count, 0)
	q.maxEventsSeen = 0
	q.head = nil
	q.tail = nil
	q.free = nil

	q.watchers = q.watchers[:0]
}

func (q *Queue) Add(ev Event) error {
	initialCount := atomic.LoadInt32(&q.count)
	if initialCount >= MaxQueued {
		return errors.New("ev queue is full")
	}

	var entry *Entry
	if q.free == nil {
		entry = &Entry{}
	} else {
		entry = q.free
		q.free = q.free.next
	}
	entry.ev = ev.Raw()

	if q.tail != nil {
		q.tail.next = entry
		entry.prev = q.tail
		q.tail = entry
		entry.next = nil
	} else {
		if q.head != nil {
			panic("invalid queue state, tail exists without head")
		}
		q.head = entry
		q.tail = entry
		entry.prev = nil
		entry.next = nil
	}

	finalCount := atomic.AddInt32(&q.count, 1) + 1
	if finalCount > q.maxEventsSeen {
		q.maxEventsSeen = finalCount
	}

	return nil
}

func (q *Queue) Cut(entry *Entry) {
	if entry.prev != nil {
		entry.prev.next = entry.next
	}
	if entry.next != nil {
		entry.next.prev = entry.prev
	}
	if entry == q.head {
		if entry.prev != nil {
			panic("invalid ev queue state, queue head is not beginning")
		}
		q.head = entry.next
	}
	if entry == q.tail {
		if entry.next != nil {
			panic("invalid ev queue state, queue tail is not the end")
		}
		q.tail = entry.prev
	}
	entry.next = q.free
	q.free = entry
	atomic.AddInt32(&q.count, -1)
}

// Note: removed numevents to just use the slice length
func (q *Queue) Peep(events []Event, action int, minType, maxType uint32) (int, error) {
	if atomic.LoadInt32(&q.active) == 0 {
		return 0, errors.New("the ev queue is not active")
	}
	if q.lock == nil {
		return 0, errors.New("ev queue lock doesn't exist")
	}
	q.lock.Lock()
	defer q.lock.Unlock()

	used := 0
	switch action {
	case Add:
		for _, ev := range events {
			if err := q.Add(ev); err != nil {
				return used, errors.Wrap(err, "unable to add event")
			}
			used++
		}
	case Get:
		// TODO(mde): deal with wmmsg types
		fallthrough
	case Peek:
		for entry := q.head; entry != nil && (events == nil || used < len(events)); entry = entry.next {
			if !(minType <= entry.ev.Type() && entry.ev.Type() <= maxType) {
				continue
			}
			if events != nil {
				events[used] = entry.ev
				if entry.ev.Type() == SysWMEvent {
					// TODO(mde): deal with wmmsg types
				}

				if action == Get {
					q.Cut(entry)
				}
			}
			used++
		}
	default:
		return 0, errors.New("invalid action type")
	}

	return used, nil
}

func (q *Queue) HasType(evType uint32) (bool, error) {
	cnt, err := q.Peep(nil, Peek, evType, evType)
	return cnt > 0, errors.Wrap(err, "unable to peep")
}

func (q *Queue) HasTypes(minType, maxType uint32) (bool, error) {
	cnt, err := q.Peep(nil, Peek, minType, maxType)
	return cnt > 0, errors.Wrap(err, "unable to peep")
}

func (q *Queue) FlushType(evType uint32) error {
	return q.FlushTypes(evType, evType)
}

func (q *Queue) FlushTypes(minType, maxType uint32) error {
	if atomic.LoadInt32(&q.active) == 0 {
		return nil
	}
	if q.lock == nil {
		return errors.New("ev queue lock doesn't exist")
	}
	q.lock.Lock()
	defer q.lock.Unlock()

	for entry := q.head; entry != nil; entry = entry.next {
		if minType <= entry.ev.Type() && entry.ev.Type() <= maxType {
			q.Cut(entry)
		}
	}
	return nil
}

func (q *Queue) Pump() {
	for _, p := range q.sources {
		p.Pump(q)
	}

	// TODO(mde) : Pending Quit?
}

func (q *Queue) Poll() (Event, error) {
	return q.WaitTimeout(0)
}

func (q *Queue) Wait() (Event, error) {
	return q.WaitTimeout(-1)
}

func (q *Queue) WaitTimeout(timeout time.Duration) (Event, error) {
	var expiration time.Time
	if timeout > 0 {
		expiration = time.Now().Add(timeout)
	}

	for {
		q.Pump()
		buf := make([]Event, 1)
		n, err := q.Peep(buf, Get, FirstEvent, LastEvent)
		switch {
		case err != nil:
			return nil, errors.Wrap(err, "queue peep error")
		case n == 1:
			return buf[0], nil
		case n == 0 && timeout != -1 && (timeout == 0 || time.Now().After(expiration)):
			return nil, WaitTimeoutExceeded
		default:
			// I don't really like this, but they do the same in SDL2
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func (q *Queue) Push(ev Event) (bool, error) {
	binary.LittleEndian.PutUint32(ev.Raw()[:4], ticker.GetAsMS())
	if !q.ok(q.okdata, ev) {
		return false, nil
	}

	q.wmu.Lock()
	for _, w := range q.watchers {
		w.Callback(w.Userdata, ev)
	}
	q.wmu.Unlock()

	buf := make([]Event, 1)
	_, err := q.Peep(buf, Add, 0, 0)
	if err != nil {
		return true, errors.Wrap(err, "unable to add event to queue")
	}

	// TODO(mde): Add gesture event processing

	return true, nil
}

func (q *Queue) SetFilter(f Filter, userdata interface{}) error {
	err := q.FlushTypes(FirstEvent, LastEvent)
	if err != nil {
		errors.Wrap(err, "unable to flush queue")
	}
	q.ok = f
	q.okdata = userdata
	return nil
}

func (q *Queue) GetFilterr() (Filter, interface{}) {
	return q.ok, q.okdata
}

func (q *Queue) AddWatch(watcher *Watcher) {
	q.wmu.Lock()
	defer q.wmu.Unlock()
	q.watchers = append(q.watchers, watcher)
}

func (q *Queue) DelWatch(watcher *Watcher) {
	q.wmu.Lock()
	defer q.wmu.Unlock()
	updatedWatchers := q.watchers[:0]
	for _, w := range q.watchers {
		if !(w == watcher) {
			updatedWatchers = append(updatedWatchers, w)
		}
	}
	q.watchers = updatedWatchers
}

func (q *Queue) Filter(f Filter, userdata interface{}) error {
	if atomic.LoadInt32(&q.active) == 0 {
		return nil
	}
	if q.lock == nil {
		return errors.New("ev queue lock doesn't exist")
	}
	q.lock.Lock()
	defer q.lock.Unlock()

	for entry := q.head; entry != nil; entry = entry.next {
		if !f(userdata, entry.ev) {
			q.Cut(entry)
		}
	}
	return nil
}

func (q *Queue) Disable(ev Event) {
	hi := uint8((ev.Type() >> 8) & 0xFF)
	lo := uint8(ev.Type() & 0xFF)
	q.disabled[hi][lo/32] |= 1 << (lo & 31)
	q.FlushType(ev.Type())
}

func (q *Queue) Enable(ev Event) {
	hi := uint8((ev.Type() >> 8) & 0xFF)
	lo := uint8(ev.Type() & 0xFF)
	q.disabled[hi][lo/32] &^= 1 << (lo & 31)
}

func (q *Queue) Enabled(ev Event) bool {
	hi := uint8((ev.Type() >> 8) & 0xFF)
	lo := uint8(ev.Type() & 0xFF)
	return q.disabled[hi][lo/32]&1<<(lo&31) == 0
}
