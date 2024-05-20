[English](README.md) | 中文

# tRPC-Go localcache 插件

[![Go Reference](https://pkg.go.dev/badge/trpc.group/trpc-go/trpc-database/localcache.svg)](https://pkg.go.dev/trpc.group/trpc-go/trpc-database/localcache)
[![Go Report Card](https://goreportcard.com/badge/trpc.group/trpc-go/trpc-database/localcache)](https://goreportcard.com/report/trpc.group/trpc-go/trpc-database/localcache)
[![Tests](https://github.com/trpc-ecosystem/go-database/actions/workflows/localcache.yml/badge.svg)](https://github.com/trpc-ecosystem/go-database/actions/workflows/localcache.yml)
[![Coverage](https://codecov.io/gh/trpc-ecosystem/go-database/branch/main/graph/badge.svg?flag=localcache&precision=2)](https://app.codecov.io/gh/trpc-ecosystem/go-database/tree/main/localcache)

localcache是一个单机的本地K-V缓存组件，允许多个goroutine并发访问，支持基于LRU和过期时间的淘汰策略。
localcache容量达到上限后，会基于LRU进行数据淘汰，过期key-value的删除基于time wheel实现。

**适用于读cache场景，而不适用于写cache场景。**


## 快速使用
直接使用localcache包下的功能函数
```go
package main

import (
    "trpc.group/trpc-go/trpc-database/localcache"
)

func LoadData(ctx context.Context, key string) (interface{}, error) {
    return "cat", nil
}

func main() {
    // 缓存key-value, 并设置5秒的过期时间
    localcache.Set("foo", "bar", 5)

    // 获取key对应的value
    value, found := localcache.Get("foo")

    // 获取key对应的value 
    // 如果key在cache中不存在，则使用自定义的LoadData函数从数据源加载，并缓存在cache中
    // 同时设置5秒的过期时间
    value, err := localcache.GetWithLoad(context.TODO(), "tom", LoadData, 5)

    // 删除key
    localcache.Del("foo")

    // 清空缓存
    localcache.Clear()
}
```

## 配置使用
New()生成Cache实例，调用该实例的功能函数
### 可选参数
#### **WithCapacity(capacity int)**
设置cache的最大容量， 最小值1，最大值1e30。缓存满后，则基于LRU淘汰队尾元素。默认值1e30
#### **WithExpiration(ttl int64)**
设置元素的过期时间，单位秒。默认值60秒。
#### **WithLoad(f LoadFunc)**
```go
type LoadFunc func(ctx context.Context, key string) (interface{}, error)
```
设置数据加载函数, key在cache中不存在时，使用该函数加载对应的value, 并缓存在cache中。和GetWithLoad()搭配使用。
#### **WithMLoad(f MLoadFunc)**
```go
type MLoadFunc func(ctx context.Context, keys []string) (map[string]interface{}, error)
```
设置**批量**数据加载函数, 在cache中不存在的keys，使用该函数进行批量加载。和MGetWithLoad()搭配使用。
#### **WithDelay(duration int64)**
设置cache中过期key的延迟删除的时间间隔，单位秒。默认值0，key过期后立即删除。

使用场景：当key过期时，同时业务的数据下游服务异常，希望可以从cache中拿到过期的value在业务上做为兜底数据。

#### **WithOnDel(delCallBack ItemCallBackFunc)**

```go
type ItemCallBackFunc func(item *Item)
```

设置元素删除时的回调函数：过期删除、主动删除、LRU 删除都会触发该回调函数

#### **WithOnExpire(expireCallback ItemCallbackFunc)**

```go
type ItemCallBackFunc func(item *Item)
```

设置元素过期时的回调函数，元素过期时会触发两个回调函数：过期回调，删除回调

#### Cache 接口

```go
type Cache interface {
    // Get 返回key对应的value值，bool返回true。如果key不存在或过期，bool返回false
    Get(key string) (interface{}, bool)

    // GetWithLoad 返回key对应的value, 如果key不存在，使用自定义的加载函数获取数据，并缓存
    GetWithLoad(ctx context.Context, key string) (interface{}, error)

    // MGetWithLoad 返回多个key对应的values。当某些keys不存在时，使用自定义的批量加载函数获取数据，并缓存
    // 对于cache中不存在, 且在mLoad函数的调用结果中不存在的key，MGetWithLoad的返回结果中，会包含该key，且对应的value为nil
    MGetWithLoad(ctx context.Context, keys []string) (map[string]interface{}, error)

    // GetWithCustomLoad 返回key对应的value， 如果key不存在，则使用传入的load函数加载并缓存ttl时间
    // load函数不存在会返回err，如果不需要每次get时都传入load函数请在new cache时使用option方式设置load，并使用GetWithLoad方法获取缓存值
    GetWithCustomLoad(ctx context.Context, key string, customLoad LoadFunc, ttl int64) (interface{}, error)

    // MGetWithCustomLoad 返回多个key对应的values。当某些keys不存在时，则使用传入的load函数加载并缓存ttl时间
    // 对于cache中不存在, 且在mLoad函数的调用结果中不存在的key，MGetWithLoad的返回结果中，包含该key，且对应的value为nil
    // load函数不存在会返回err，如果不需要每次get时都传入load函数请在new cache时使用option方式设置load，并使用MGetWithLoad方法获取缓存值
    MGetWithCustomLoad(ctx context.Context, keys []string, customLoad MLoadFunc, ttl int64) (map[string]interface{}, error)

    // Set 缓存key-value
    Set(key string, value interface{}) bool

    // SetWithExpire 缓存key-value, 并为某个key设置特定的ttl(过期时间，单位秒)
    SetWithExpire(key string, value interface{}, ttl int64) bool

    // Del 删除key
    Del(key string)

    // Clear 清空所有队列和缓存
    Clear()

    // Close 关闭cache
    Close()
}
```

## 使用示例
#### 设置容量和过期时间

```go
func main() {
    var lc localcache.Cache

    // 创建一个容量大小100, 元素过期时间5秒的缓存
    lc = localcache.New(localcache.WithCapacity(100), localcache.WithExpiration(5))

    // 设置key-value, 过期时间5秒
    lc.Set("foo", "bar")

    // 为key设置特定的过期时间(10秒)，不使用New()方法中的过期参数
    lc.SetWithExpire("tom", "cat", 10)

    // 短暂地等待，从缓存中异步处理完成
    time.Sleep(time.Millisecond)

    // 获取value
    val, found := lc.Get("foo")
	fmt.Println(val, found)
    
    // 删除key: "foo"
    lc.Del("foo")
}
```

### 延迟删除
```go
func main() {
    // 设置延迟删除的时间间隔为3秒
    lc := localcache.New(localcache.WithDelay(3))
    // 设置key的过期时间为1秒： key在1秒后过期，4秒后从cache中删除
    lc.SetWithExpire("tom", "cat", 1)
    // sleep 2秒
    time.Sleep(time.Second * 2)

    value, ok := lc.Get("tom")
    if !ok {
        // key 在sleep 2 秒后已过期
        fmt.Printf("key:%s is expired or empty", "tom")
    }
    
    if s, ok := value.(string); ok && s == "cat" {
        // 过期的value依旧返回了，业务侧可以决定是否使用过期的value
        fmt.Printf("get expired value: %s\n", "cat")
    }
}
```

#### 自定义加载函数
设置自定义数据加载函数，并使用GetWithLoad(key)获取value
```go
func main() {
    loadFunc := func(ctx context.Context, key string) (interface{}, error) {
        return "cat", nil
    }

    lc := localcache.New(localcache.WithLoad(loadFunc), localcache.WithExpiration(5))

    // err 为loadFunc函数直接返回的error信息
    val, err := lc.GetWithLoad(context.TODO(), "tom")

    // 或者可以在get时传入load函数
    otherLoadFunc := func(ctx context.Context, key string) (interface{}, error) {
        return "dog", nil
    }

    val,err := lc.GetWithCustomLoad(context.TODO(),"tom",otherLoadFunc,10)
}
```
设置批量数据加载函数，并使用MGetWithLoad(keys)获取values
```go
func main() {
    mLoadFunc := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
        return map[string]interface{} {
            "foo": "bar",
            "tom": "cat",
        }, nil
    }

    lc := localcache.New(localcache.WithMLoad(mLoadFunc), localcache.WithExpiration(5))

    // err 为mLoadFunc函数直接返回的error信息
    val, err := lc.MGetWithLoad(context.TODO(), []string{"foo", "tom"})

    // 或者可以在get时传入load函数
    val,err := lc.MGetWithCustomLoad(context.TODO(),"tom",mLoadFunc,10)
}
```

#### 自定义删除元素过期/删除时的回调

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

1. 增加Metrics数据统计
2. 增加对内存使用量的控制
3. 引入升级版的LRU：W-tinyLRU，更高效地控制key的写入和淘汰