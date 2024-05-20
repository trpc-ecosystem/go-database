package localcache

import (
	"sync"

	"github.com/cespare/xxhash"
)

// store is a storage for storing key-value data concurrently and safely.
// This file temporarily uses the fragmented map implementation.
type store interface {
	// get returns the value corresponding to key
	get(string) (interface{}, bool)
	// set adds a new key-value to the storage
	set(string, interface{})
	// del delete key-value
	del(string)
	// clear Clear all contents in storage
	clear()
	// len returns the size of the storage
	len() int
}

// newStore returns the default implementation of storage
func newStore() store {
	return newShardedMap()
}

const numShards uint64 = 256

// shardedMap storage sharding
type shardedMap struct {
	shards []*lockedMap
}

func newShardedMap() *shardedMap {
	sm := &shardedMap{
		shards: make([]*lockedMap, int(numShards)),
	}
	for i := range sm.shards {
		sm.shards[i] = newLockedMap()
	}
	return sm
}

func (sm *shardedMap) get(key string) (interface{}, bool) {
	return sm.shards[xxhash.Sum64String(key)&(numShards-1)].get(key)
}

func (sm *shardedMap) set(key string, value interface{}) {
	sm.shards[xxhash.Sum64String(key)&(numShards-1)].set(key, value)
}

func (sm *shardedMap) del(key string) {
	sm.shards[xxhash.Sum64String(key)&(numShards-1)].del(key)
}

func (sm *shardedMap) clear() {
	for i := uint64(0); i < numShards; i++ {
		sm.shards[i].clear()
	}
}

func (sm *shardedMap) len() int {
	length := 0
	for i := uint64(0); i < numShards; i++ {
		length += sm.shards[i].len()
	}
	return length
}

// lockedMap concurrently safe map
type lockedMap struct {
	sync.RWMutex
	data map[string]interface{}
}

func newLockedMap() *lockedMap {
	return &lockedMap{
		data: make(map[string]interface{}),
	}
}

func (m *lockedMap) get(key string) (interface{}, bool) {
	m.RLock()
	val, ok := m.data[key]
	m.RUnlock()
	return val, ok
}

func (m *lockedMap) set(key string, value interface{}) {
	m.Lock()
	m.data[key] = value
	m.Unlock()
}

func (m *lockedMap) del(key string) {
	m.Lock()
	delete(m.data, key)
	m.Unlock()
}

func (m *lockedMap) clear() {
	m.Lock()
	m.data = make(map[string]interface{})
	m.Unlock()
}

func (m *lockedMap) len() int {
	m.RLock()
	defer m.RUnlock()
	return len(m.data)
}
