package localcache

import (
	"container/list"
	"sync"
	"time"
)

// entry store entity
type entry struct {
	mux        sync.RWMutex
	key        string
	value      interface{}
	expireTime time.Time
}

func getEntry(ele *list.Element) *entry {
	return ele.Value.(*entry)
}

func setEntry(ele *list.Element, ent *entry) {
	ele.Value = ent
}

// lru non-concurrency-safe lru queue
type lru struct {
	ll       *list.List
	store    store
	capacity int
}

func newLRU(capacity int, store store) *lru {
	return &lru{
		ll:       list.New(),
		store:    store,
		capacity: capacity,
	}
}

func (l *lru) add(ent *entry) *entry {
	val, ok := l.store.get(ent.key)
	ele, _ := val.(*list.Element)
	if ok {
		setEntry(ele, ent)
		l.ll.MoveToFront(ele)
		return nil
	}
	if l.capacity <= 0 || l.ll.Len() < l.capacity {
		ele := l.ll.PushFront(ent)
		l.store.set(ent.key, ele)
		return nil
	}
	// When lru is full, the last element is deleted and the new element is added to the head of the list.
	ele = l.ll.Back()
	if ele == nil {
		return ent
	}
	l.ll.Remove(ele)
	victimEnt := getEntry(ele)
	l.store.del(victimEnt.key)

	ele = l.ll.PushFront(ent)
	l.store.set(ent.key, ele)
	return victimEnt
}

func (l *lru) hit(ele *list.Element) {
	l.ll.MoveToFront(ele)
}

func (l *lru) push(elements []*list.Element) {
	for _, ele := range elements {
		l.ll.MoveToFront(ele)
	}
}

func (l *lru) del(key string) *entry {
	value, ok := l.store.get(key)
	if !ok {
		return nil
	}
	ele, _ := value.(*list.Element)
	delEnt := getEntry(ele)
	l.ll.Remove(ele)
	l.store.del(key)
	return delEnt
}

func (l *lru) len() int {
	return l.ll.Len()
}

func (l *lru) clear() {
	l.ll = list.New()
}
