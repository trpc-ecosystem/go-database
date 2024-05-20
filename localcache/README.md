English | [中文](README.zh_CN.md)

# tRPC-Go localcache plugin

[![Go Reference](https://pkg.go.dev/badge/trpc.group/trpc-go/trpc-database/localcache.svg)](https://pkg.go.dev/trpc.group/trpc-go/trpc-database/localcache)
[![Go Report Card](https://goreportcard.com/badge/trpc.group/trpc-go/trpc-database/localcache)](https://goreportcard.com/report/trpc.group/trpc-go/trpc-database/localcache)
[![Tests](https://github.com/trpc-ecosystem/go-database/actions/workflows/localcache.yml/badge.svg)](https://github.com/trpc-ecosystem/go-database/actions/workflows/localcache.yml)
[![Coverage](https://codecov.io/gh/trpc-ecosystem/go-database/branch/main/graph/badge.svg?flag=localcache&precision=2)](https://app.codecov.io/gh/trpc-ecosystem/go-database/tree/main/localcache)

localcache is a standalone local K-V cache component that allows multiple goroutines to access it concurrently and supports LRU and expiration time based elimination policy.
After the capacity of localcache reaches the upper limit, it will carry out data elimination based on LRU, and the deletion of expired key-value is realized based on time wheel.

**applies to readcache scenarios, not to writecache scenarios.**


## Quick Start
Use the functions directly from the localcache package.
```go
package main

import (
    "trpc.group/trpc-go/trpc-database/localcache"
)

func LoadData(ctx context.Context, key string) (interface{}, error) {
    return "cat", nil
}

func main() {
    // Cache the key-value, and set the expiration time to 5 seconds.
    localcache.Set("foo", "bar", 5)

    // Get the value corresponding to the key
    value, found := localcache.Get("foo")

    // Get the value corresponding to the key.
    // If the key does not exist in the cache, it is loaded from the data source using the custom LoadData function
    // and cached in the cache.
    // And Set an expiration time of 5 seconds.
    value, err := localcache.GetWithLoad(context.TODO(), "tom", LoadData, 5)

    // Delete key
    localcache.Del("foo")

    // Clear cache
    localcache.Clear()
}
```

## Configuring Usage
New() generates a Cache instance and calls the functional functions of that instance
### Optional parameters
#### **WithCapacity(capacity int)**
Sets the maximum size of the cache, with a minimum value of 1 and a maximum value of 1e30. When the cache is full, the last element of the queue is eliminated based on LRU. Default value is 1e30.
#### **WithExpiration(ttl int64)**
Sets the expiration time of the element in seconds. The default value is 60 seconds.
#### **WithLoad(f LoadFunc)**
```go
type LoadFunc func(ctx context.Context, key string) (interface{}, error)
```
Set data load function, when the key does not exist in cache, use this function to load the corresponding value, and cache it in cache. Use with GetWithLoad().
#### **WithMLoad(f MLoadFunc)**
```go
type MLoadFunc func(ctx context.Context, keys []string) (map[string]interface{}, error)
```
Set **bulk** data loading function, use this function to bulk load keys that don't exist in cache. Use with MGetWithLoad().
#### **WithDelay(duration int64)**
Sets the interval in seconds for delayed deletion of expired keys in cache. The default value is 0, the key is deleted immediately after it expires.

Usage Scenario: When the key expires, at the same time the data downstream service exception of the business, I hope to be able to get the expired value from the cache in the business as the backing data.

#### **WithOnDel(delCallBack ItemCallBackFunc)**

```go
type ItemCallBackFunc func(item *Item)
```

Setting the callback function when an element is deleted: expired deletions, active deletions, and LRU deletions all trigger this callback function.

#### **WithOnExpire(expireCallback ItemCallbackFunc)**

```go
type ItemCallBackFunc func(item *Item)
```

Set the callback function when an element expires. Two callback functions are triggered when an element expires: the expiration callback and the deletion callback.

#### Cache Interface

```go
type Cache interface {
    // Get returns the value of the key, bool returns true. bool returns false if the key does not exist or is expired.
    Get(key string) (interface{}, bool)

    // GetWithLoad returns the value corresponding to the key, if the key does not exist, use the customized load function
    // to get the data and cache it.
    GetWithLoad(ctx context.Context, key string) (interface{}, error)

    // MGetWithLoad returns values corresponding to multiple keys. when some keys don't exist, use a customized bulk load
    // function to fetch the data and cache it.
    // For a key that does not exist in the cache, and does not exist in the result of the mLoad function call, the return 
    // result of MGetWithLoad will contain the key, and the corresponding value will be nil.
    MGetWithLoad(ctx context.Context, keys []string) (map[string]interface{}, error)

    // GetWithCustomLoad returns the value corresponding to the key. If the key does not exist, it is loaded using the load
    // function passed in and caches the ttl time.
    // load function does not exist will return err, if you do not need to pass in the load function every time you get
    // please use the option to set load when new cache, and use the GetWithLoad method to get the cache value.
    GetWithCustomLoad(ctx context.Context, key string, customLoad LoadFunc, ttl int64) (interface{}, error)

    // MGetWithCustomLoad returns values corresponding to multiple keys. when some keys don't exist, they are loaded using the 
    // load function passed in and cached ttl time. For a key that does not exist in the cache, and does not exist in the result
    // of the mLoad function call, the result of MGetWithLoad contains the key, and the corresponding value is nil. oad function
    // does not exist will return err, if you do not need to pass in the load function every time you get please use the option
    // to set load when new cache, and use the MGetWithLoad method to get the cache value.
    MGetWithCustomLoad(ctx context.Context, keys []string, customLoad MLoadFunc, ttl int64) (map[string]interface{}, error)

    // Set cache key-value
    Set(key string, value interface{}) bool

    // SetWithExpire caches key-values, and sets a specific ttl (expiration time in seconds) for a key.
    SetWithExpire(key string, value interface{}, ttl int64) bool

    // Delete key
    Del(key string)

    // Clear all queues and caches
    Clear()

    // Close cache
    Close()
}
```

## Example
#### Setting capacity and expiration time

```go
func main() {
    var lc localcache.Cache

    // Create a cache with a size of 100 and an element expiration time of 5 seconds.
    lc = localcache.New(localcache.WithCapacity(100), localcache.WithExpiration(5))

    // Set key-value, expiration time 5 seconds
    lc.Set("foo", "bar")

    // Set a specific expiration time (10 seconds) for the key, without using the expiration parameter in the New() method
    lc.SetWithExpire("tom", "cat", 10)

    // Short wait for asynchronous processing to complete from cache
    time.Sleep(time.Millisecond)

    // Get value
    val, found := lc.Get("foo")
    
    // Delete key: "foo"
    lc.Del("foo")
}
```

### Delayed deletion
```go
func main() {
	// Set the delayed deletion interval to 3 seconds
	lc := localcache.New(localcache.WithDelay(3))
	// Set key expiration time to 1 second: key expires after 1 second and is removed from cache after 4 seconds
	lc.SetWithExpire("tom", "cat", 1)
	// sleep 2s
	time.Sleep(time.Second * 2)

	value, ok := lc.Get("tom")
	if !ok {
		// key has expired after 2 seconds of sleep.
		fmt.Printf("key:%s is expired or empty\n", "tom")
	}

	if s, ok := value.(string); ok && s == "cat" {
		// Expired values are still returned, and the business side can decide whether or not to use them.
		fmt.Printf("get expired value: %s\n", "cat")
	}
}
```

#### Custom Load Functions
Set up a custom data loading function and use GetWithLoad(key) to get the value
```go
func main() {
    loadFunc := func(ctx context.Context, key string) (interface{}, error) {
        return "cat", nil
    }

    lc := localcache.New(localcache.WithLoad(loadFunc), localcache.WithExpiration(5))

    // err is the error message returned directly by the loadFunc function.
    val, err := lc.GetWithLoad(context.TODO(), "tom")

    // Or you can pass in the load function at get time
    otherLoadFunc := func(ctx context.Context, key string) (interface{}, error) {
        return "dog", nil
    }

    val,err := lc.GetWithCustomLoad(context.TODO(),"tom",otherLoadFunc,10)
}
```
Set the bulk data load function and use MGetWithLoad(keys) to get values
```go
func main() {
    mLoadFunc := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
        return map[string]interface{} {
            "foo": "bar",
            "tom": "cat",
        }, nil
    }

    lc := localcache.New(localcache.WithMLoad(mLoadFunc), localcache.WithExpiration(5))

    // err is the error message returned directly by the mLoadFunc function.
    val, err := lc.MGetWithLoad(context.TODO(), []string{"foo", "tom"})

    // Or you can pass in the load function at get time
    val,err := lc.MGetWithCustomLoad(context.TODO(),"tom",mLoadFunc,10)
}
```

#### Customizing the callback on expiration/deletion of deleted elements

```go
func main() {
	delCount := map[string]int{"A": 0, "B": 0, "": 0}
	expireCount := map[string]int{"A": 0, "B": 0, "": 0}
	c := localcache.New(
		localcache.WithCapacity(4),
		localcache.WithOnExpire(func(item *localcache.Item) {
			if item.Flag != localcache.ItemDelete {
				return
			}
			if _, ok := expireCount[item.Key]; ok {
				expireCount[item.Key]++
			}
		}),
		localcache.WithOnDel(func(item *localcache.Item) {
			if item.Flag != localcache.ItemDelete {
				return
			}
			if _, ok := delCount[item.Key]; ok {
				delCount[item.Key]++
			}
		}))

	defer c.Close()

	elems := []struct {
		key string
		val string
		ttl int64
	}{
		{"A", "a", 1},
		{"B", "b", 300},
		{"C", "c", 300},
	}

	for _, elem := range elems {
		c.SetWithExpire(elem.key, elem.val, elem.ttl)
	}
	time.Sleep(1001 * time.Millisecond)
	fmt.Printf("del info:%v\n", delCount)
	fmt.Printf("expire info:%v\n", expireCount)
}
```



### TODO

1. Add Metrics statistics
2. increase the control of memory usage
3. Introduce an upgraded version of LRU: W-tinyLRU, which controls key writing and elimination more efficiently.