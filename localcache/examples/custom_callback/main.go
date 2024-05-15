package main

import (
	"fmt"
	"time"

	"trpc.group/trpc-go/trpc-database/localcache"
)

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
