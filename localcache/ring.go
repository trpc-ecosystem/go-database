package localcache

import (
	"container/list"
	"sync"
)

// ringConsumer accept and consume data
type ringConsumer interface {
	push([]*list.Element) bool
}

// ringStrip ring buffer caches the metadata of Get requests to batch update the location of elements in the LRU
type ringStripe struct {
	consumer ringConsumer
	data     []*list.Element
	capacity int
}

func newRingStripe(consumer ringConsumer, capacity int) *ringStripe {
	return &ringStripe{
		consumer: consumer,
		data:     make([]*list.Element, 0, capacity),
		capacity: capacity,
	}
}

// push records an accessed element to the ring buffer
func (r *ringStripe) push(ele *list.Element) {
	r.data = append(r.data, ele)
	if len(r.data) >= r.capacity {
		if r.consumer.push(r.data) {
			r.data = make([]*list.Element, 0, r.capacity)
		} else {
			r.data = r.data[:0]
		}
	}
}

// RingBuffer pools ringStripe, allowing multiple goroutines to write elements to Buff without locking,
// which is more efficient than writing to the same channel concurrently. At the same time, objects in
// the pool will be automatically removed without any notification. This random loss of access metadata
// can reduce the cache's operation on LRU and reduce concurrency competition with Set/Expire/Delete
// operations. The cache does not need to be completely strict LRU. It is necessary to actively discard
// some access metadata to reduce concurrency competition and improve write efficiency.
type ringBuffer struct {
	pool *sync.Pool
}

func newRingBuffer(consumer ringConsumer, capacity int) *ringBuffer {
	return &ringBuffer{
		pool: &sync.Pool{
			New: func() interface{} { return newRingStripe(consumer, capacity) },
		},
	}
}

// push records an accessed element
func (b *ringBuffer) push(ele *list.Element) {
	ringStripe := b.pool.Get().(*ringStripe)
	ringStripe.push(ele)
	b.pool.Put(ringStripe)
}
