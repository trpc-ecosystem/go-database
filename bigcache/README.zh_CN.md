[English](README.md) | 中文

# tRPC-Go bigcache 插件

[![Go Reference](https://pkg.go.dev/badge/trpc.group/trpc-go/trpc-database/bigcache.svg)](https://pkg.go.dev/trpc.group/trpc-go/trpc-database/bigcache)
[![Go Report Card](https://goreportcard.com/badge/trpc.group/trpc-go/trpc-database/bigcache)](https://goreportcard.com/report/trpc.group/trpc-go/trpc-database/bigcache)
[![Tests](https://github.com/trpc-ecosystem/go-database/actions/workflows/bigcache.yml/badge.svg)](https://github.com/trpc-ecosystem/go-database/actions/workflows/bigcache.yml)
[![Coverage](https://codecov.io/gh/trpc-ecosystem/go-database/branch/coverage/graph/badge.svg?flag=bigcache&precision=2)](https://app.codecov.io/gh/trpc-ecosystem/go-database/tree/coverage/bigcache)

封装社区的 [bigcache](https://github.com/allegro/bigcache) ，配合 trpc 使用。

> bigcache 封装了 golang 下高效的本地缓存第三方库，封装增加了 value 支持的类型，以方便使用。

## 使用样例
```go
package main

import (
    "fmt"
    "strconv"

    "trpc.group/trpc-go/trpc-database/bigcache"
)

func main() {
    cache := bigcache.New(
        //设置分区数，默认 256
        bigcache.WithShards(1024),
        //设置生命周期，默认 7 天
        bigcache.WithLifeWindow(time.Hour),
        //设置清理周期，默认 1 分钟
        bigcache.WithCleanWindow(-1*time.Second),
        //设置最大数据条数，默认 200000，小值可降低初始化使用内存，仅初始化使用
        bigcache.WithMaxEntriesInWindow(50000),
        //设置最大数据长度（单位字节），默认 30，小值可降低初始化使用内存，仅初始化使用
        bigcache.WithMaxEntrySize(20),
    )
    var err error
    var v string
    for j := 0; j < 1000000; j++ {
        key := fmt.Sprintf("k%010d", j)
        v = "v" + strconv.Itoa(j)
        // 写入数据
        err = cache.Set(key, v)
        if err != nil {
            fmt.Println(err)
        }
        // 查找数据
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

## 简介：
1. 封装接口，添加 []byte 外的其它类型支持，
value 目前支持 []byte,string,int,int8,int16,int32,int64,uint,uint8,uint16,uint32,uint64,float32,float64,bool,error 格式，
2. GC 消耗可忽略，在百万量级缓存数据降低 CPU 消耗明显，详见原 [README.md](https://github.com/allegro/bigcache/blob/master/README.md)```How it works```
3. 特点：支持 Iterator 访问、Len 查询数据条数、Capacity 查询 cache 中的字节数、Status 查询缓存命中状态
4. log 日志使用 trpc-go 框架提供
5. 新增 compare_test 目录做与 Sync.Map、[localcache](https://trpc.group/trpc-go/trpc-database/localcache) 的性能对比，详见目录下 README.md
