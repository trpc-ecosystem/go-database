package localcache

import (
	"container/list"
	"sync"
	"testing"
)

// testConsumer test Consumer structure
type testConsumer struct {
	pf   func([]*list.Element)
	save bool
}

func (c *testConsumer) push(elements []*list.Element) bool {
	if c.save {
		c.pf(elements)
		return true
	}
	return false
}

// TestRingBufferPush tests RingBuffer's push method
func TestRingBufferPush(t *testing.T) {
	drains := 0
	r := newRingBuffer(&testConsumer{
		pf: func(elements []*list.Element) {
			drains++
		},
		save: true,
	}, 1)

	for i := 0; i < 100; i++ {
		r.push(&list.Element{})
	}

	if drains != 100 {
		t.Fatal("elements shouldn't be dropped with capacity == 1")
	}
}

// TestRingReset tests RingBuffer's Reset method
func TestRingReset(t *testing.T) {
	drains := 0
	r := newRingBuffer(&testConsumer{
		pf: func(elements []*list.Element) {
			drains++
		},
		save: false,
	}, 4)
	for i := 0; i < 100; i++ {
		r.push(&list.Element{})
	}
	if drains != 0 {
		t.Fatal("elements shouldn't be drained")
	}
}

// TestRingConsumer tests RingBuffer's Consumer
func TestRingConsumer(t *testing.T) {
	mu := &sync.Mutex{}
	drainElements := make(map[*list.Element]struct{})

	r := newRingBuffer(&testConsumer{
		pf: func(elements []*list.Element) {
			mu.Lock()
			defer mu.Unlock()
			for i := range elements {
				drainElements[elements[i]] = struct{}{}
			}
		},
		save: true,
	}, 4)
	for i := 0; i < 100; i++ {
		r.push(&list.Element{})
	}
	l := len(drainElements)
	if l == 0 || l > 100 {
		t.Fatal("drains not being process correctly")
	}
}
