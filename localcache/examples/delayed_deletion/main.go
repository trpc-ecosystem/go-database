package main

import (
	"fmt"
	"time"

	"trpc.group/trpc-go/trpc-database/localcache"
)

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
