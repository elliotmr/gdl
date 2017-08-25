package event

import (
	"sync"
	"sync/atomic"
	"github.com/pkg/errors"
)

const MaxQueued = 65535

type Filter func(userdata interface{}, event Event) bool

type Watcher struct {
	Callback Filter
	Userdata interface{}
}

type Entry struct {
	event Data
	prev  *Entry
	next  *Entry
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
}

var watchers []*Watcher
var q Queue

func StartQueue() error {
	if q.lock == nil {
		q.lock = &sync.Mutex{}
	}
	atomic.StoreInt32(&q.active, 1)
	return nil
}

func StopQueue()  {
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


func Add(event Event) error {
	initialCount := atomic.LoadInt32(&q.count)
	if initialCount >= MaxQueued {
		return errors.New("event queue is full")
	}

	var entry *Entry
	if q.free == nil {
		entry = &Entry{}
	} else {
		entry = q.free
		q.free = q.free.next
	}
	e   ntry.event = event.Raw()

	if q.tail != nil {
		q.tail.next = entry
		entry.prev = q.tail
		q.tail = entry
		entry.next = nil
	} else {
		if q.head != nil {
			panic("invalid queue state")
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