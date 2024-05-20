package localcache

import (
	"container/list"
)

// policy store policy
type policy interface {
	// add adds an element
	add(ent *entry) *entry
	// hit handles accessing an element hit
	hit(elements *list.Element)
	// push processes accessed elements in batches
	push(elements []*list.Element)
	// del deletes an element based on key
	del(key string) *entry
	// clear space
	clear()
}

func newPolicy(capacity int, store store) policy {
	return newLRU(capacity, store)
}
