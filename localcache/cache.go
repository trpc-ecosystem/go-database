// Package localcache is a stand-alone local K-V cache component that allows concurrent access by multiple goroutines.
// The cache eliminates elements based on LRU and expiration time strategies.
package localcache

import (
	"container/list"
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

//go:generate mockgen -source=cache.go -destination=./mocklocalcache/localcache_mock.go -package=mocklocalcache

const (
	// The maximum size of the cache
	maxCapacity = 1 << 30

	ringBufSize = 16

	// Asynchronous metadata for cache operations
	setBufSize    = 1 << 15
	updateBufSize = 1 << 15
	delBufSize    = 1 << 15
	expireBufSize = 1 << 15

	// The default expiration time is 60s
	ttl = 60
)

// CachedStatus cache status
type CachedStatus int

const (
	// CacheNotExist cache data does not exist
	CacheNotExist CachedStatus = 1
	// CacheExist cache data exists
	CacheExist CachedStatus = 2
	// CacheExpire cache data exists but has expired. You can choose whether to use it
	CacheExpire CachedStatus = 3
)

// ErrCacheExpire the cache exists but has expired
var ErrCacheExpire = errors.New("cache exist, but expired")

// currentTime is an alias of time.Now, which is convenient for specifying the current time during testing
var currentTime = time.Now

// Cache is a local K-V memory store that supports expiration time
type Cache interface {
	// Get returns the value corresponding to key, bool returns true.
	// If the key does not exist or expires, bool returns false
	Get(key string) (interface{}, bool)
	// GetWithStatus returns the value corresponding to the key and returns the cache status
	GetWithStatus(key string) (interface{}, CachedStatus)
	// GetWithLoad returns the value corresponding to the key.
	// If the key does not exist, use the user-defined loading function to obtain the data and cache it.
	GetWithLoad(ctx context.Context, key string) (interface{}, error)
	// MGetWithLoad returns values corresponding to multiple keys.
	// When some keys do not exist, use a custom batch loading function to obtain data and cache it
	// For a key that does not exist in the cache and does not exist in the calling result of the mLoad function,
	// the return result of MGetWithLoad includes the key and the corresponding value is nil.
	MGetWithLoad(ctx context.Context, keys []string) (map[string]interface{}, error)
	// GetWithCustomLoad returns the value corresponding to the key.
	// If the key does not exist, the passed in load function is used to load and cache the ttl time.
	// If the load function does not exist, err will be returned.
	// If you do not need to pass in the load function every time you get it, please use the option method
	// to set the load in the new cache, and use the GetWithLoad method to obtain the cache value.
	GetWithCustomLoad(ctx context.Context, key string, customLoad LoadFunc, ttl int64) (interface{}, error)
	// MGetWithCustomLoad returns values corresponding to multiple keys.
	// When some keys do not exist, the passed in load function is used to load and cache the ttl time.
	// For a key that does not exist in the cache and does not exist in the calling result of the mLoad function,
	// the return result of MGetWithLoad includes the key and the corresponding value is nil.
	// If the load function does not exist, err will be returned.
	// If you do not need to pass in the load function every time you get it, please use the option method to set
	// the load in the new cache, and use the MGetWithLoad method to obtain the cache value.
	MGetWithCustomLoad(ctx context.Context, keys []string, customLoad MLoadFunc, ttl int64) (map[string]interface{}, error)
	// Set key and value
	Set(key string, value interface{}) bool
	// SetWithExpire sets key, value, and sets different ttl (expiration time in seconds) for different keys
	SetWithExpire(key string, value interface{}, ttl int64) bool
	// Del delete key
	Del(key string)
	// Len key quantity
	Len() int
	// Clear clears all queues and caches
	Clear()
	// Close Close cache
	Close()
}

type eleWithFinish struct {
	ele    *list.Element
	finish func()
}

type entWithFinish struct {
	ent    *entry
	finish func()
}

type keyWithFinish struct {
	key    string
	finish func()
}

// cache K-V memory storage
type cache struct {
	capacity int
	store    store

	// key entry and elimination strategies
	policy policy

	getBuf     *ringBuffer
	elementsCh chan []*list.Element

	setBuf    chan *entWithFinish
	updateBuf chan *eleWithFinish
	delBuf    chan *keyWithFinish
	expireBuf chan string

	g     group
	load  LoadFunc
	mLoad MLoadFunc

	// Element expiration time (seconds)
	ttl int64
	// Delay time for deleting expired keys
	delay int64
	// Delete the task queue of expired key
	expireQueue *expireQueue

	stop chan struct{}

	// Triggered when deleted: deletion triggered by element expiration, active deletion, deletion triggered by lru
	onDel ItemCallBackFunc
	// Triggered on expiration
	onExpire ItemCallBackFunc

	// syncUpdateFlag data setting and updating method.
	// When it is true, the synchronous method is used to set or update the data.
	// Otherwise, the asynchronous method is used to set the data by default.
	syncUpdateFlag bool
	// settingTimeout timeout for synchronizing setting data or updating data
	settingTimeout time.Duration
	// syncDelFlag is the data deletion method.
	// If it is true, the data will be deleted synchronously.
	// Otherwise, the data will be deleted asynchronously by default.
	syncDelFlag bool
}

// LoadFunc loads the value data corresponding to the key and is used to fill the cache
type LoadFunc func(ctx context.Context, key string) (interface{}, error)

// MLoadFunc loads the value data of multiple keys in batches to fill the cache
type MLoadFunc func(ctx context.Context, keys []string) (map[string]interface{}, error)

// ItemCallBackFunc callback function triggered when the element expires/deletes
type ItemCallBackFunc func(*Item)

// ItemFlag The event type that triggers the callback
type ItemFlag int

const (
	// ItemDelete Triggered when actively deleted/expired
	ItemDelete ItemFlag = iota
	// ItemLruDel LRU triggered deletion
	ItemLruDel
)

// Item The element that triggered the callback event
type Item struct {
	Flag  ItemFlag
	Key   string
	Value interface{}
}

// call refers to the implementation of singleflight, used for loading user-defined data
type call struct {
	wg  sync.WaitGroup
	val interface{}
	err error
}

// group refers to the implementation of singleflight and is used for user-defined data loading
type group struct {
	mu sync.Mutex       // protects m
	m  map[string]*call // lazily initialized
}

// Option parameter tool function
type Option func(*cache)

// WithCapacity sets the maximum number of keys
func WithCapacity(capacity int) Option {
	if capacity > maxCapacity {
		capacity = maxCapacity
	}
	if capacity <= 0 {
		capacity = 1
	}
	return func(c *cache) {
		c.capacity = capacity
	}
}

// WithExpiration sets the expiration time of the element (in seconds).
// If the expiration time is not set, the default is 60 seconds.
func WithExpiration(ttl int64) Option {
	if ttl <= 0 {
		ttl = 1
	}
	return func(c *cache) {
		c.ttl = ttl
	}
}

// WithLoad sets a custom data loading function
func WithLoad(f LoadFunc) Option {
	return func(c *cache) {
		c.load = f
	}
}

// WithMLoad sets a custom data batch loading function
func WithMLoad(f MLoadFunc) Option {
	return func(c *cache) {
		c.mLoad = f
	}
}

// WithDelay delay deletion interval of expired keys (unit seconds)
func WithDelay(duration int64) Option {
	return func(c *cache) {
		c.delay = duration
	}
}

// WithOnDel sets the callback function when the element is deleted
func WithOnDel(delCallBack ItemCallBackFunc) Option {
	return func(c *cache) {
		c.onDel = delCallBack
	}
}

// WithOnExpire sets the callback function triggered when the element expires
func WithOnExpire(expireCallback ItemCallBackFunc) Option {
	return func(c *cache) {
		c.onExpire = expireCallback
	}
}

// WithSettingTimeout causes elements to be written to the store synchronously, and a timeout needs to be set.
func WithSettingTimeout(t time.Duration) Option {
	return func(c *cache) {
		c.syncUpdateFlag = true
		c.settingTimeout = t
	}
}

// WithSyncDelFlag sets the synchronous method to delete elements in the store when deleting elements.
func WithSyncDelFlag(flag bool) Option {
	return func(c *cache) {
		c.syncDelFlag = flag
	}
}

// New generate cache object
func New(opts ...Option) Cache {
	// Initialize cache with default values
	cache := &cache{
		capacity: maxCapacity,
		store:    newStore(),

		elementsCh: make(chan []*list.Element, 3),
		setBuf:     make(chan *entWithFinish, setBufSize),
		updateBuf:  make(chan *eleWithFinish, updateBufSize),
		delBuf:     make(chan *keyWithFinish, delBufSize),
		expireBuf:  make(chan string, expireBufSize),

		ttl:         ttl,
		expireQueue: newExpireQueue(time.Second, 60),
		stop:        make(chan struct{}),
	}
	// Set cache using passed parameters
	for _, opt := range opts {
		opt(cache)
	}

	cache.policy = newPolicy(cache.capacity, cache.store)
	cache.getBuf = newRingBuffer(cache, ringBufSize)

	go cache.processEntries()

	return cache
}

// processEntries asynchronously processes cache operations
func (c *cache) processEntries() {
	for {
		select {
		case elements := <-c.elementsCh:
			c.access(elements)
		case buf := <-c.setBuf:
			c.add(buf.ent)
			if buf.finish != nil {
				buf.finish()
			}
		case buf := <-c.updateBuf:
			c.update(buf.ele)
			if buf.finish != nil {
				buf.finish()
			}
		case buf := <-c.delBuf:
			c.del(buf.key)
			if buf.finish != nil {
				buf.finish()
			}
		case key := <-c.expireBuf:
			c.expire(key)
		case <-c.stop:
			return
		}
	}
}

// Get returns the value corresponding to key, bool returns true.
// If the key does not exist or expires, bool returns false
func (c *cache) Get(key string) (interface{}, bool) {
	value, status := c.GetWithStatus(key)
	if status == CacheExist {
		return value, true
	}
	return value, false
}

// GetWithStatus returns the value corresponding to the key.
// Since the user may cache nil and cannot distinguish the data, CachedStatus is used to represent the return status.
func (c *cache) GetWithStatus(key string) (interface{}, CachedStatus) {
	if c == nil {
		return nil, CacheNotExist
	}

	value, hit := c.store.get(key)
	if hit {
		ele, _ := value.(*list.Element)
		ent := getEntry(ele)
		ent.mux.RLock()
		defer ent.mux.RUnlock()
		if ent.expireTime.Before(currentTime()) {
			return ent.value, CacheExpire
		}
		c.getBuf.push(ele)
		return ent.value, CacheExist
	}

	return nil, CacheNotExist
}

// GetWithLoad returns the value corresponding to the key.
// If the key does not exist, use the user-defined filling function to load the data and return it, and cache it.
func (c *cache) GetWithLoad(ctx context.Context, key string) (interface{}, error) {
	return c.GetWithCustomLoad(ctx, key, c.load, c.ttl)
}

// MGetWithLoad returns values corresponding to multiple keys.
// When some keys do not exist, use a custom batch loading function to obtain data and cache it
// For a key that does not exist in the cache and does not exist in the calling result of the mLoad function,
// the return result of MGetWithLoad includes the key and the corresponding value is nil.
func (c *cache) MGetWithLoad(ctx context.Context, keys []string) (map[string]interface{}, error) {
	return c.MGetWithCustomLoad(ctx, keys, c.mLoad, c.ttl)
}

// GetWithCustomLoad returns the value corresponding to the key.
// If the key does not exist, the passed in load function is used to load and cache the ttl time.
// If the load function does not exist, err will be returned.
// If you do not need to pass in the load function every time you get it, please use the option method to set
// the load in the new cache, and use the GetWithLoad method to obtain the cache value.
func (c *cache) GetWithCustomLoad(ctx context.Context, key string, customLoad LoadFunc, ttl int64) (
	interface{}, error) {
	if customLoad == nil {
		return nil, errors.New("undefined LoadFunc in cache")
	}

	val, status := c.GetWithStatus(key)
	if status == CacheExist {
		return val, nil
	}
	latest, err := c.loadData(ctx, key, customLoad, ttl)
	if err != nil {
		if status == CacheExpire {
			return val, fmt.Errorf("load key %s err %v, %w", key, err, ErrCacheExpire)
		}
		return nil, err
	}
	return latest, nil
}

// MGetWithCustomLoad returns values corresponding to multiple keys.
// When some keys do not exist, the passed in load function is used to load and cache the ttl time.
// For a key that does not exist in the cache and does not exist in the calling result of the mLoad function,
// the return result of MGetWithLoad includes the key and the corresponding value is nil.
// If the load function does not exist, err will be returned.
// If you do not need to pass in the load function every time you get it, please use the option method to set the
// load in the new cache, and use the MGetWithLoad method to obtain the cache value.
func (c *cache) MGetWithCustomLoad(ctx context.Context, keys []string, customMLoad MLoadFunc, ttl int64) (
	map[string]interface{}, error) {
	if customMLoad == nil {
		return nil, errors.New("undefined MLoadFunc in cache")
	}
	values := make(map[string]interface{}, len(keys))
	var noCacheKeys []string
	for _, key := range keys {
		value, ok := c.Get(key)
		if !ok {
			noCacheKeys = append(noCacheKeys, key)
		}
		values[key] = value
	}
	if len(noCacheKeys) == 0 {
		return values, nil
	}
	return c.loadNoCacheKeys(ctx, customMLoad, noCacheKeys, values, ttl)
}

func (c *cache) loadNoCacheKeys(ctx context.Context,
	mLoad MLoadFunc,
	noCacheKeys []string,
	values map[string]interface{},
	ttl int64) (map[string]interface{}, error) {
	latest, err := mLoad(ctx, noCacheKeys)
	if err != nil {
		return values, err
	}
	for key, value := range latest {
		values[key] = value
		c.SetWithExpire(key, value, ttl)
	}

	return values, nil
}

// Set key, value
func (c *cache) Set(key string, value interface{}) bool {
	return c.SetWithExpire(key, value, c.ttl)
}

// SetWithExpire sets key, value, time to live (seconds), and sets different expiration times for different elements
func (c *cache) SetWithExpire(key string, value interface{}, ttl int64) bool {
	if c == nil {
		return false
	}
	expireTime := currentTime().Add(time.Second * time.Duration(ttl))

	val, hit := c.store.get(key)
	if hit {
		// If the key exists, immediately update the latest value in the storage to prevent Get from obtaining dirty data.
		ele, _ := val.(*list.Element)
		oldEnt := getEntry(ele)

		oldEnt.mux.Lock()
		oldEnt.value = value
		oldEnt.expireTime = expireTime
		oldEnt.mux.Unlock()
		if c.syncUpdateFlag {
			waitFinish := make(chan struct{}, 1)
			select {
			case c.updateBuf <- &eleWithFinish{
				ele,
				func() {
					close(waitFinish)
				},
			}:
				<-waitFinish
				return true
			case <-time.After(c.settingTimeout):
				return false
			}
		} else {
			select {
			case c.updateBuf <- &eleWithFinish{ele, nil}:
			default:
			}
		}
		c.expireQueue.update(key, expireTime.Add(time.Duration(c.delay)*time.Second), c.afterExpire(key))
		return true
	}

	ent := &entry{
		key:        key,
		value:      value,
		expireTime: expireTime,
	}
	// Add new key and value. In the extreme case where syncSet is false, the Set operation is not guaranteed to
	// be successful.
	// Because of asynchronous processing and heavy load, there may be a ms-level delay in the Set result.
	if c.syncUpdateFlag {
		waitFinish := make(chan struct{}, 1)
		select {
		case c.setBuf <- &entWithFinish{
			ent,
			func() {
				close(waitFinish)
			},
		}:
			<-waitFinish
			return true
		case <-time.After(c.settingTimeout):
			return false
		}
	}
	select {
	case c.setBuf <- &entWithFinish{ent, nil}:
		return true
	default:
		return false
	}
}

// Del deletes key, supports synchronous deletion and asynchronous deletion
func (c *cache) Del(key string) {
	if c == nil {
		return
	}

	// Enable synchronous deletion, block and wait for deletion to complete before returning
	if c.syncDelFlag {
		waitFinish := make(chan struct{}, 1)
		c.delBuf <- &keyWithFinish{
			key: key,
			finish: func() {
				close(waitFinish)
			},
		}
		<-waitFinish
		return
	}
	c.delBuf <- &keyWithFinish{key: key}
}

// Clear clears all queues and caches.
// It is a non-atomic operation and should be called after there are no Get and Set operations.
func (c *cache) Clear() {
	// Block until processEntries goroutine ends
	c.stop <- struct{}{}

	c.elementsCh = make(chan []*list.Element, 3)
	c.setBuf = make(chan *entWithFinish, setBufSize)
	c.updateBuf = make(chan *eleWithFinish, updateBufSize)
	c.delBuf = make(chan *keyWithFinish, delBufSize)
	c.expireBuf = make(chan string, expireBufSize)

	c.store.clear()
	c.policy.clear()
	c.expireQueue.clear()

	// Restart processEntries goroutine
	go c.processEntries()
}

// Close cache
func (c *cache) Close() {
	// Block until processEntries goroutine ends
	c.stop <- struct{}{}
	close(c.stop)

	close(c.elementsCh)
	close(c.setBuf)
	close(c.updateBuf)
	close(c.delBuf)
	close(c.expireBuf)

	c.expireQueue.stop()
}

// access is called asynchronously to handle access operations
func (c *cache) access(elements []*list.Element) {
	c.policy.push(elements)
}

// add is called asynchronously to handle new operations
func (c *cache) add(ent *entry) {
	// Store new key-value.
	// After reaching the upper limit of capacity, return the eliminated entry.
	key := ent.key
	victimEnt := c.policy.add(ent)

	expireTime := ent.expireTime.Add(time.Second * time.Duration(c.delay))
	c.expireQueue.add(key, expireTime, c.afterExpire(key))

	// Remove eliminated entries from the expiration queue
	if victimEnt != nil {
		c.expireQueue.remove(victimEnt.key)
		if c.onDel != nil {
			c.onDel(&Item{ItemLruDel, victimEnt.key, victimEnt.value})
		}
	}
}

// update is called asynchronously to handle update operations
func (c *cache) update(ele *list.Element) {
	c.policy.hit(ele)
}

// del is called asynchronously to handle the deletion operation
func (c *cache) del(key string) {
	delEnt := c.policy.del(key)
	c.expireQueue.remove(key)
	if delEnt != nil && c.onDel != nil {
		c.onDel(&Item{ItemDelete, delEnt.key, delEnt.value})
	}
}

// expire is called asynchronously to process expired data
func (c *cache) expire(key string) {
	delEnt := c.policy.del(key)

	if delEnt != nil && c.onExpire != nil {
		c.onExpire(&Item{ItemDelete, delEnt.key, delEnt.value})
	}

	if delEnt != nil && c.onDel != nil {
		c.onDel(&Item{ItemDelete, delEnt.key, delEnt.value})
	}
}

// loadData uses user-defined functions to load and cache data
func (c *cache) loadData(ctx context.Context, key string, load LoadFunc, ttl int64) (interface{}, error) {
	// Refer to singleflight implementation to prevent cache breakdown
	c.g.mu.Lock()
	if c.g.m == nil {
		c.g.m = make(map[string]*call)
	}
	if call, ok := c.g.m[key]; ok {
		c.g.mu.Unlock()
		call.wg.Wait()
		return call.val, call.err
	}
	call := new(call)
	call.wg.Add(1)
	c.g.m[key] = call
	c.g.mu.Unlock()
	if c.syncUpdateFlag {
		// Try to read the cache and return if the cache is read. Otherwise load load
		value, hit := c.store.get(key)
		if hit {
			ele, _ := value.(*list.Element)
			ent := getEntry(ele)
			ent.mux.RLock()
			if ent.expireTime.After(currentTime()) {
				ent.mux.RUnlock()
				return ent.value, nil
			}
			ent.mux.RUnlock()
		}
	}
	call.val, call.err = load(ctx, key)
	if call.err == nil {
		if ok := c.SetWithExpire(key, call.val, ttl); !ok {
			call.err = fmt.Errorf("set key [%s] fail", key)
		}
	}

	c.g.mu.Lock()
	call.wg.Done()
	delete(c.g.m, key)
	c.g.mu.Unlock()

	return call.val, call.err
}

// afterExpire returns the callback task after the key expires
func (c *cache) afterExpire(key string) func() {
	return func() {
		select {
		case c.expireBuf <- key:
		default:
		}
	}
}

// push writes read requests into the channel in batches
func (c *cache) push(elements []*list.Element) bool {
	if len(elements) == 0 {
		return true
	}
	select {
	case c.elementsCh <- elements:
		return true
	default:
		return false
	}
}

// Len key quantity
func (c *cache) Len() int {
	return c.store.len()
}
