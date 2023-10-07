// Package main is the main package.
package main

import (
	"fmt"
	"log"
	"runtime"
	"runtime/debug"
	"strconv"
	"sync"
	"time"

	bigcache "github.com/allegro/bigcache/v3"
	bc "trpc.group/trpc-go/trpc-database/bigcache"
	"trpc.group/trpc-go/trpc-database/localcache"
)

const (
	vPre      = "v"
	kPre      = "1234567890k"
	keys      = 1000000
	readTimes = 80
)

var ss debug.GCStats

func gcPause(caller string) {
	runtime.GC()
	debug.ReadGCStats(&ss)
}

func gcPauseX(caller string) {
	runtime.GC()
	var ss1 debug.GCStats
	debug.ReadGCStats(&ss1)
	log.Printf(" %s Pause:%d %d\n", caller, ss1.PauseTotal-ss.PauseTotal, ss1.NumGC-ss.NumGC)
}

func test0() {
	cache, _ := bc.NewBigCache(
		bc.WithBigcacheShards(256),
		bc.WithBigcacheLifeWindow(time.Hour),
		bc.WithBigcacheCleanWindow(-1*time.Second),
		bc.WithBigcacheMaxEntriesInWindow(500000),
		bc.WithBigcacheMaxEntrySize(20),
	)
	{
		startT := time.Now()
		for i := 0; i < keys; i++ {
			cache.Set(fmt.Sprintf("%s%010d", kPre, i), []byte(vPre+strconv.Itoa(i)))
		}
		fmt.Printf("write bigCache cost = %v\n", time.Since(startT))
	}
	// read performance test
	startT := time.Now()
	var wg sync.WaitGroup
	wg.Add(readTimes)
	var v []byte
	for i := 0; i < readTimes; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < keys; j++ {
				err := cache.Get(fmt.Sprintf("keyprefix%010d", i, j), v)
				if v == nil || err != nil {
					fmt.Sprintf("%d", 1)
				}
			}

		}()
	}
	wg.Wait()
	fmt.Printf("read BigCache cost = %v\n", time.Since(startT))

	gcPause("bigCache nilStart")
	gcPauseX("bigCache nil")
	// del performance test
	{
		cache, _ := bigcache.NewBigCache(bigcache.DefaultConfig(5 * time.Second))
		for i := 0; i < keys; i++ {
			cache.Set(fmt.Sprintf("%s%d", kPre, i), []byte(vPre+strconv.Itoa(i)))
		}
		startT1 := time.Now()
		for j := 0; j < keys; j++ {
			cache.Delete(fmt.Sprintf("%s%d", kPre, j))
		}
		fmt.Printf("del BigCache cost = %v\n", time.Since(startT1))
	}
}

func test1() {
	cache := localcache.New()
	{
		startT := time.Now()
		for i := 0; i < keys; i++ {
			cache.Set(fmt.Sprintf("%s%010d", kPre, i), []byte(vPre+strconv.Itoa(i)))
		}
		fmt.Printf("write local cost = %v\n", time.Since(startT))
	}
	// read performance test
	startT := time.Now()
	var wg sync.WaitGroup
	wg.Add(readTimes)
	for i := 0; i < readTimes; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < keys; j++ {
				v, err := cache.Get(fmt.Sprintf("%s%010d", kPre, j))
				if v == nil || err {
					fmt.Sprintf("%d", 1)
				}
			}

		}()
	}
	wg.Wait()
	fmt.Printf("read local cost = %v\n", time.Since(startT))
	gcPause("localcache nil start")
	gcPauseX("localcache nil")
	{
		startT1 := time.Now()
		for j := 0; j < keys; j++ {
			cache.Del(fmt.Sprintf("%s%d", kPre, j))
		}
		fmt.Printf("delete local cost = %v\n", time.Since(startT1))
	}
}

func test2() {
	var cache sync.Map
	{
		startT := time.Now()
		for i := 0; i < keys; i++ {
			cache.Store(fmt.Sprintf("%s%010d", kPre, i), []byte(vPre+strconv.Itoa(i)))
		}
		fmt.Printf("write sync.Map cost = %v\n", time.Since(startT))
	}
	// read performance test
	startT := time.Now()
	var wg sync.WaitGroup
	wg.Add(readTimes)
	for i := 0; i < readTimes; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < keys; j++ {
				v, err := cache.Load(fmt.Sprintf("keyprefix%010d", i, j))
				if v == nil || !err {
					fmt.Sprintf("%d", 1)
				}
			}

		}()
	}
	wg.Wait()
	fmt.Printf("read sync.Map cost = %v\n", time.Since(startT))

	gcPause("sync.Map nilStart")
	gcPauseX("sync.Map nil")
	// del performance test
	{
		startT1 := time.Now()
		for j := 0; j < keys; j++ {
			cache.Delete(fmt.Sprintf("%s%d", kPre, j))
		}
		fmt.Printf("del sync.Map cost = %v\n", time.Since(startT1))
	}
}

func main() {
	fmt.Printf("readRoutines:%d, totalItems:%d, kpre:%s vpre:%s\n", readTimes, keys, kPre, vPre)
	test0()
	test1()
	test2()
}
