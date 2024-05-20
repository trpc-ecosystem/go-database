package localcache

import "context"

var defaultLocalCache = New()

// Get returns the value corresponding to key, bool returns true.
// If the key does not exist or expires, bool returns false
func Get(key string) (interface{}, bool) {
	return defaultLocalCache.Get(key)
}

// GetWithLoad returns the value corresponding to the key.
// If the key does not exist, use the load function to load the return and cache it for ttl seconds.
func GetWithLoad(ctx context.Context, key string, load LoadFunc, ttl int64) (interface{}, error) {
	value, found := defaultLocalCache.Get(key)
	if found {
		return value, nil
	}
	c, _ := defaultLocalCache.(*cache)
	return c.loadData(ctx, key, load, ttl)
}

// Set key, value, expiration time ttl (seconds)
func Set(key string, value interface{}, ttl int64) bool {
	return defaultLocalCache.SetWithExpire(key, value, ttl)
}

// Del Delete key
func Del(key string) {
	defaultLocalCache.Del(key)
}

// Clear all queues and caches
func Clear() {
	defaultLocalCache.Clear()
}
