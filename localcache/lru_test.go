package localcache

import (
	"container/list"
	"fmt"
	"runtime"
	"testing"
)

func assertLRULen(t *testing.T, l *lru, n int) {
	if l.store.len() != n || l.len() != n {
		_, file, line, _ := runtime.Caller(1)
		t.Fatalf("%s:%d unexpected store length (s-%d l-%d), want: %d",
			file, line, l.store.len(), l.len(), n)
	}
}

func assertLRUEntry(t *testing.T, ent *entry, k string, v string) {
	if ent.key != k || ent.value.(string) != v {
		_, file, line, _ := runtime.Caller(1)
		t.Fatalf("%s:%d unexpected entry:%+v, want: {key: %s, value:%s}",
			file, line, ent, k, v)
	}
}

func TestLRU(t *testing.T) {
	store := newStore()
	lru := newLRU(3, store)
	ents := make([]*entry, 4)
	for i := 0; i < len(ents); i++ {
		k := fmt.Sprintf("%d", i)
		v := k
		ents[i] = &entry{key: k, value: v}
	}

	// set 0, lru order: 0
	victim := lru.add(ents[0])
	if victim != nil {
		t.Fatalf("unexpected entry removed: %v", victim)
	}
	assertLRULen(t, lru, 1)

	val, _ := lru.store.get(ents[0].key)
	ele0 := val.(*list.Element)
	ent0 := getEntry(ele0)
	assertLRUEntry(t, ent0, "0", "0")

	// set 1, lru order: 1-0
	victim = lru.add(ents[1])
	if victim != nil {
		t.Fatalf("unexpected entry removed: %v", victim)
	}
	assertLRULen(t, lru, 2)

	val, _ = lru.store.get(ents[1].key)
	ele1 := val.(*list.Element)
	ent1 := getEntry(ele1)
	assertLRUEntry(t, ent1, "1", "1")

	// lru order: 0-1
	lru.hit(ele0)

	// set 2, lru order: 2-0-1
	victim = lru.add(ents[2])
	if victim != nil {
		t.Fatalf("unexpected entry removed: %v", victim)
	}
	assertLRULen(t, lru, 3)

	// set 3, lru order: 3-2-0,  evict 1
	victim = lru.add(ents[3])
	if victim == nil {
		t.Fatal("1 entry should be removed")
	} else {
		assertLRUEntry(t, victim, "1", "1")
		assertLRULen(t, lru, 3)
	}

	val, _ = lru.store.get(ents[3].key)
	ele3 := val.(*list.Element)
	ent3 := getEntry(ele3)
	assertLRUEntry(t, ent3, "3", "3")

	// remove 2, lru order 3-0
	lru.del(ents[2].key)
	assertLRULen(t, lru, 2)

	// again add 3, lru order 3-0
	victim = lru.add(ents[3])
	if victim != nil {
		t.Fatalf("unexpected entry removed: %v", victim)
	}
	assertLRULen(t, lru, 2)

	// remove not exist key
	lru.del("None")
	assertLRULen(t, lru, 2)

	lru.clear()
	lru.store.clear()
	assertLRULen(t, lru, 0)
}
