// Package bigcache encapsulates a low garbage collection (GC) consuming local cache library named BigCache.
package bigcache

import (
	"fmt"
	"reflect"

	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"

	bigcache "github.com/allegro/bigcache/v3"
)

const (
	// ErrEntryNotFound entry not found
	ErrEntryNotFound = -1
)

// BigCache is the local cache struct, encapsulating the corresponding struct of third-party library.
type BigCache struct {
	bc *bigcache.BigCache
}

// New initializes a BigCache with default parameters. If the initialization fails, it returns an error,
// which is then handled by the business layer.
func New(opt ...Option) (BigCache, error) {
	cfg := DefaultConfig()
	// write configuration
	for _, o := range opt {
		o(&cfg)
	}
	bc, err := bigcache.NewBigCache(InitDefaultConfig(cfg))
	if err != nil {
		return BigCache{}, err
	}

	return BigCache{bc: bc}, nil
}

// Close sends a shutdown signal, exits the clean coroutine, and ensures that it can be garbage collected.
func (c *BigCache) Close() error {
	return c.bc.Close()
}

// GetBytes gets the raw bytes by the given key. If the key does not exist, it returns ErrEntryNotFound.
func (c *BigCache) GetBytes(key string) ([]byte, error) {
	bd, err := c.bc.Get(key)
	if bd == nil || err != nil {
		return nil, errs.New(ErrEntryNotFound, err.Error())
	}
	return bd, nil
}

// GetBytesWithInfo gets the raw bytes by the given key. If the key does not exist,
// it returns ErrEntryNotFound, along with additional information indicating whether the entry has expired.
func (c *BigCache) GetBytesWithInfo(key string) ([]byte, bigcache.Response, error) {
	b, rsp, err := c.bc.GetWithInfo(key)
	if err != nil {
		return nil, rsp, errs.New(ErrEntryNotFound, err.Error())
	}
	if b == nil {
		return nil, rsp, errs.New(ErrEntryNotFound, "val is nil")
	}
	return b, rsp, nil
}

// Get gets the value by the given key. If the key does not exist, it returns ErrEntryNotFound.
// The value is automatically parsed using the specified serialization type.
func (c *BigCache) Get(key string, val interface{}, serializationType int) (err error) {
	bd, err := c.bc.Get(key)
	if bd == nil || err != nil {
		return errs.New(ErrEntryNotFound, err.Error())
	}
	return codec.Unmarshal(serializationType, bd, val)
}

// GetWithInfo gets the value by the key. If the entry has been removed or expired, it returns ErrEntryNotFound,
// along with additional information indicating whether the entry has expired.
// The value is automatically parsed using the specified serialization type.
func (c *BigCache) GetWithInfo(key string, val interface{}, serializationType int) (bigcache.Response, error) {
	b, rsp, err := c.bc.GetWithInfo(key)
	if err != nil {
		return rsp, errs.New(ErrEntryNotFound, err.Error())
	}
	if b == nil {
		return rsp, errs.New(ErrEntryNotFound, "val is nil")
	}
	return rsp, codec.Unmarshal(serializationType, b, val)
}

// Set saves a key-value pair, which may fail if the value format is not supported.
// The data is automatically packaged using the input serialization type.
func (c *BigCache) Set(key string, val interface{}, serializationType ...int) error {
	if len(serializationType) > 0 {
		data, err := codec.Marshal(serializationType[0], val)
		if err != nil {
			return err
		}
		return c.bc.Set(key, data)
	}

	switch val := val.(type) {
	case []byte:
		return c.bc.Set(key, val)
	case string:
		return c.bc.Set(key, []byte(val))
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, bool, error:
		b, e := ToString(val)
		if e != nil {
			return fmt.Errorf("value type:%s convert to string error:%s", reflect.TypeOf(val), e.Error())
		}
		return c.bc.Set(key, []byte(b))
	default:
		return fmt.Errorf("bigcache value not supoort type:%s", reflect.TypeOf(val))
	}
}

// Delete deletes the key
func (c *BigCache) Delete(key string) error {
	return c.bc.Delete(key)
}

// Reset clears all shards of the cache.
func (c *BigCache) Reset() error {
	return c.bc.Reset()
}

// Len returns the number of data entries in the cache
func (c *BigCache) Len() int {
	return c.bc.Len()
}

// Capacity returns the number of bytes used in the cache.
func (c *BigCache) Capacity() int {
	return c.bc.Capacity()
}

// Stats returns the statistical data for cache hits.
func (c *BigCache) Stats() bigcache.Stats {
	return c.bc.Stats()
}

// Iterator returns an iterator that can traverse the entire cache.
func (c *BigCache) Iterator() *bigcache.EntryInfoIterator {
	return c.bc.Iterator()
}

// KeyMetadata returns the hit count for the key.
func (c *BigCache) KeyMetadata(key string) bigcache.Metadata {
	return c.bc.KeyMetadata(key)
}
