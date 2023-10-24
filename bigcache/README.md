English | [中文](README.zh_CN.md)

# tRPC-Go bigcache plugin

[![Coverage](https://codecov.io/gh/trpc-ecosystem/go-database/branch/coverage/graph/badge.svg?flag=bigcache&precision=2)](https://app.codecov.io/gh/trpc-ecosystem/go-database/tree/coverage/bigcache)

Encapsulate the [bigcache](https://github.com/allegro/bigcache) library from the GitHub community to use with the tRPC-Go framework.

> bigcache encapsulates a high-performance third-party local cache library in Golang. We enhances its functionality by adding support for additional value types, making it more convenient to use.

## Example usage
```
package main

import (
    "fmt"
    "strconv"

    "trpc.group/trpc-go/trpc-database/bigcache"
)

func main() {
    cache := bigcache.New(
        // Set the number of shards, default is 256.
        bigcache.WithShards(1024),
        // Set the life window, default is 7 days.
        bigcache.WithLifeWindow(time.Hour),
        // Set the clean window, default is 1 minute.
        bigcache.WithCleanWindow(-1*time.Second),
        // Set the maximum number of entries in the window, default is 200,000.
		    // A smaller value can reduce memory usage during initialization, and it's only used during initialization.
        bigcache.WithMaxEntriesInWindow(50000),
        // Set the maximum entry size in bytes, default is 30.
		    // A smaller value can reduce memory usage during initialization, and it's only used during initialization.
        bigcache.WithMaxEntrySize(20),
    )
    var err error
    var v string
    for j := 0; j < 1000000; j++ {
        key := fmt.Sprintf("k%010d", j)
        v = "v" + strconv.Itoa(j)
        // write data
        err = cache.Set(key, v)
        if err != nil {
            fmt.Println(err)
        }
        // get data
        val, err := cache.GetBytes(key)
        if err != nil {
            fmt.Println(err)
        }
        fmt.Printf("got value: %+v", val)
        if j%10000 == 0 {
            fmt.Println(v)
        }
    }
}
```

## Overview
1. Encapsulate the interface to add support for types other than []byte. The current supported value types include []byte, string, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, bool, and error formats. 
2. The GC overhead is negligible and CPU consumption is significantly reduced when dealing with cache data in the order of millions. For more details, please refer to the original [README.md](https://github.com/allegro/bigcache/blob/master/README.md) under the section ```How it works```.
3. Key features: support for Iterator to access cache, Len to query the number of data entries, Capacity to query the size of the cache in bytes, and Status to check cache hit status. 
4. Logging utilizes the capabilities provided by the tRPC-Go framework.
5. Add the compare_test directory for performance comparison with Sync.Map and [localcache](https://trpc.group/trpc-go/trpc-database/localcache). For more information, please refer to the README.md file in that directory.
