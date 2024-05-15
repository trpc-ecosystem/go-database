package main

import (
	"fmt"
	"time"

	"trpc.group/trpc-go/trpc-database/localcache"
)

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
	fmt.Println(val, found)

	// Delete key: "foo"
	lc.Del("foo")
}
