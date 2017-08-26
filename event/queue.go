package event

import (
	"sync"
	"sync/atomic"
	"github.com/pkg/errors"
	"time"
)

const MaxQueued = 65535

const (
	Add = iota
	Peek
	Get
)

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
	count int32

	// pointers
	head *Entry
	tail *Entry
	free *Entry

	// other
	maxEventsSeen int32
	// TODO(mde): implement MWMsg

	// input sources
	sources []Pumper
}

var watchers []*Watcher

func (q *Queue) StartQueue() error {
	if q.lock == nil {
		q.lock = &sync.Mutex{}
	}
	atomic.StoreInt32(&q.active, 1)
	return nil
}

func (q *Queue) StopQueue()  {
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

	watchers = watchers[:0]
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
}

func (q *Queue) Pump() {
	for _, p := range q.sources {
		p.Pump(q)
	}

	// TODO(mde) : Pending Quit?
}

func (q *Queue) WaitTimeout(timeout time.Duration) (*Event, error) {
	expiration := time.Now().Add(timeout)

	for {
		q.Pump()

	}

}