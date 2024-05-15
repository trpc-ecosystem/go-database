package localcache

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

var wait = time.Millisecond

// TestCacheSetRace tests concurrency competition of Cache Set
func TestCacheSetRace(t *testing.T) {
	cache := New(WithExpiration(100))
	n := 8128
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			cache.Set("foo", "bar")
			cache.Get("foo")
			wg.Done()
		}()
	}
	wg.Wait()
}

// TestCacheSetGet tests Cache Set first and then Get
func TestCacheSetGet(t *testing.T) {
	mocks := []struct {
		key string
		val string
		ttl int64
	}{
		{"A", "a", 1000},
		{"B", "b", 1000},
		{"", "null", 1000},
	}
	C := New()
	c := C.(*cache)
	defer c.Close()

	for _, mock := range mocks {
		c.SetWithExpire(mock.key, mock.val, mock.ttl)
	}

	time.Sleep(wait)
	for _, mock := range mocks {
		val, found := c.Get(mock.key)
		if !found || val.(string) != mock.val {
			t.Fatalf("Unexpected value: %v (%v) to key: %v", val, found, mock.key)
		}
	}

	// update
	c.SetWithExpire(mocks[0].key, mocks[0].key+"foobar", 1000)
	val, found := c.Get(mocks[0].key)
	if !found || val.(string) == mocks[0].key {
		t.Fatalf("Unexpected value: %v (%v) to key: %v, want: %v", val, found, mocks[0].key, mocks[0].val)
	}

	// set struct
	type Foo struct {
		Name string
		Age  int
	}
	valStruct := Foo{
		"Bob",
		18,
	}
	valPtr := &valStruct
	c.SetWithExpire("foo", valStruct, 1000)
	c.SetWithExpire("bar", valPtr, 1000)
	time.Sleep(wait)
	if val, found := c.Get("foo"); !found || val.(Foo) != valStruct {
		t.Fatalf("Unexpected value: %v (%v) to key: %v, want: %v", val, found, "foo", valStruct)
	}
	if val, found := c.Get("bar"); !found || val.(*Foo) != valPtr {
		t.Fatalf("Unexpected value: %v (%v) to key: %v, want: %v", val, found, "foo", valPtr)
	}
}

// TestCacheMaxCap tests whether MaxCap meets expectations after Cache WithCapacity
func TestCacheMaxCap(t *testing.T) {
	C := New(WithCapacity(2))
	c := C.(*cache)
	defer c.Close()

	mocks := []struct {
		key string
		val string
		ttl int64
	}{
		{"A", "a", 1000},
		{"B", "b", 1000},
		{"", "null", 1000},
	}

	for _, mock := range mocks {
		c.SetWithExpire(mock.key, mock.val, mock.ttl)
	}

	time.Sleep(wait)
	if c.store.len() != 2 {
		t.Fatalf("unexpected Length:%d, want:%d", c.store.len(), 2)
	}
}

// TestCacheExpire tests the expiration function of Cache
func TestCacheExpire(t *testing.T) {
	C := New(WithCapacity(2), WithExpiration(1))
	c := C.(*cache)
	defer c.Close()

	c.Set("A", "a")
	c.Set("B", "b")

	time.Sleep(1 * time.Second)
	if c.store.len() != 0 {
		t.Fatalf("unexpected Length:%d, want:%d", c.store.len(), 0)
	}
	if val, found := c.Get("A"); found {
		t.Fatalf("unexpected expired value: %v to key: %v", val, "A")
	}
	if val, found := c.Get("B"); found {
		t.Fatalf("unexpected expired value: %v to key: %v", val, "B")
	}
}

// TestCacheSpecExpire tests the expiration function of Cache for the specified key
func TestCacheSpecExpire(t *testing.T) {
	C := New(WithCapacity(2))
	c := C.(*cache)
	defer c.Close()

	mocks := []struct {
		key string
		val string
		ttl int64
	}{
		{"A", "a", 1},
		{"B", "b", 1},
		{"", "null", 1},
	}

	for _, mock := range mocks {
		c.SetWithExpire(mock.key, mock.val, mock.ttl)
	}

	time.Sleep(time.Second)
	if c.store.len() != 0 {
		t.Fatalf("unexpected Length:%d, want:%d", c.store.len(), 0)
	}
	for _, mock := range mocks {
		val, found := c.Get(mock.key)
		if found {
			t.Fatalf("unexpected expired value: %v to key: %v", val, mock.key)
		}
	}
}

// TestCacheGetWithDelay tests the delayed deletion function of expired keys
func TestCacheGetWithDelay(t *testing.T) {
	C := New(WithDelay(2))
	defer C.Close()

	C.SetWithExpire("A", "B", 1)
	time.Sleep(time.Second)
	value, ok := C.Get("A")
	if ok {
		t.Fatalf("unexpected found: %v", ok)
	}
	if s, ok := value.(string); !ok || s != "B" {
		t.Fatalf("unexpected value: %v, want: %s", s, "B")
	}
}

// TestCacheGetWithStatus tests the delayed deletion status of expired keys
func TestCacheGetWithStatus(t *testing.T) {
	C := New(WithDelay(2))
	defer C.Close()

	C.SetWithExpire("A", "B", 1)
	time.Sleep(time.Second)
	value, status := C.GetWithStatus("A")
	if status != CacheExpire {
		t.Fatalf("unexpected status: %v", status)
	}
	if s, ok := value.(string); !ok || s != "B" {
		t.Fatalf("unexpected value: %v, want: %s", s, "B")
	}
}

// TestCacheMGetWithLoad tests the WithMLoad function of Cache
func TestCacheMGetWithLoad(t *testing.T) {
	m := map[string]interface{}{
		"A": "a",
		"B": "b",
		"":  "null",
	}
	mLoadFunc := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
		ret := make(map[string]interface{}, 0)
		for _, key := range keys {
			if v, ok := m[key]; ok {
				ret[key] = v
			}
		}
		return ret, nil
	}

	C := New(WithMLoad(mLoadFunc))
	c := C.(*cache)
	defer c.Close()

	type args struct {
		ctx  context.Context
		keys []string
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]interface{}
		wantErr bool
	}{
		{"A", args{context.Background(), []string{"A", "B"}}, map[string]interface{}{"A": "a", "B": "b"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := c.MGetWithLoad(tt.args.ctx, tt.args.keys)
			time.Sleep(wait)
			if (err != nil) != tt.wantErr {
				t.Errorf("cache.GetWithLoad() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("cache.GetWithLoad() = %v, want %v", got, tt.want)
			}
			// Get
			for _, key := range tt.args.keys {
				value, found := c.Get(key)
				if !found || !reflect.DeepEqual(value, tt.want[key]) {
					t.Errorf("cache.Get() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

// TestCacheMGetWithLoadError Load error when testing batch acquisition of Cache
func TestCacheMGetWithLoadError(t *testing.T) {
	C := New()
	defer C.Close()
	if _, err := C.MGetWithLoad(context.Background(), []string{"a"}); !strings.Contains(err.Error(), "undefined MLoadFunc in cache") {
		t.Errorf("got unexpected:%s, want contains [undefined MLoadFunc]", err.Error())
	}

	mLoadFunc := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
		return nil, errors.New("unknown keys")
	}
	C = New(WithMLoad(mLoadFunc))
	if _, err := C.MGetWithLoad(context.Background(), []string{"a"}); !strings.Contains(err.Error(), "unknown keys") {
		t.Errorf("got unexpected:%s, want contains [unknown keys]", err.Error())
	}
}

// TestCacheGetWithLoad tests the WithLoad function of Cache
func TestCacheGetWithLoad(t *testing.T) {
	m := map[string]interface{}{
		"A": "a",
		"B": "b",
		"C": nil,
		"":  "null",
	}
	loadFunc := func(ctx context.Context, key string) (interface{}, error) {
		if v, exist := m[key]; exist {
			return v, nil
		}
		return nil, errors.New("key not exist")
	}

	C := New(WithLoad(loadFunc))
	c := C.(*cache)
	defer c.Close()

	type args struct {
		ctx context.Context
		key string
	}
	tests := []struct {
		name    string
		args    args
		want    interface{}
		wantErr bool
	}{
		{"A", args{context.Background(), "A"}, "a", false},
		{"B", args{context.Background(), "B"}, "b", false},
		{"C", args{context.Background(), "C"}, nil, false},
		{"", args{context.Background(), ""}, "null", false},
		{"unrecognized-key", args{context.Background(), "unregonizedKey"}, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := c.GetWithLoad(tt.args.ctx, tt.args.key)
			time.Sleep(wait)
			got2, _ := c.Get(tt.args.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("cache.GetWithLoad() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("cache.GetWithLoad() = %v, want %v", got, tt.want)
			}
			// Get
			if !reflect.DeepEqual(got2, tt.want) {
				t.Errorf("cache.Get() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestCacheGetWithLoadError tests Cache Load error
func TestCacheGetWithLoadError(t *testing.T) {
	C := New()
	defer C.Close()
	if _, err := C.GetWithLoad(context.Background(), "A"); !strings.Contains(err.Error(), "undefined LoadFunc in cache") {
		t.Errorf("got unexpected:%s, want contains [undefined LoadFunc]", err.Error())
	}

	loadFunc := func(ctx context.Context, keys string) (interface{}, error) {
		return nil, errors.New("load fail")
	}
	C = New(WithLoad(loadFunc), WithDelay(2))
	C.SetWithExpire("A", "B", 1)
	time.Sleep(time.Second)
	if value, err := C.GetWithLoad(context.Background(), "A"); !errors.Is(err, ErrCacheExpire) {
		t.Errorf("got unexpected:%v, want contains [ErrCacheExpire]", err)
	} else {
		if s, ok := value.(string); !ok || s != "B" {
			t.Fatalf("unexpected value: %v, want: %s", s, "B")
		}
	}
	time.Sleep(2 * time.Second)
	if _, err := C.GetWithLoad(context.Background(), "A"); !strings.Contains(err.Error(), "load fail") {
		t.Errorf("got unexpected:%s, want contains [load fail]", err.Error())
	}
}

// TestCacheDel tests the Cache deletion function
func TestCacheDel(t *testing.T) {
	C := New(WithCapacity(3))
	c := C.(*cache)
	defer c.Close()

	mocks := []struct {
		key string
		val string
		ttl int64
	}{
		{"A", "a", 10},
		{"B", "b", 10},
		{"", "null", 10},
	}

	for _, mock := range mocks {
		c.SetWithExpire(mock.key, mock.val, mock.ttl)
	}
	time.Sleep(wait)
	c.Del(mocks[0].key)
	time.Sleep(wait)

	if val, found := c.Get(mocks[0].key); found {
		t.Fatalf("unexpected deleted value: %v to key: %v", val, mocks[0].key)
	}

	for i := 1; i < len(mocks); i++ {
		if val, found := c.Get(mocks[i].key); !found || val.(string) != mocks[i].val {
			t.Fatalf("unexpected deleted value: %v (%v) to key: %v, want: %v", val, found, mocks[i].key, mocks[i].val)
		}
	}
}

// TestCacheClear tests the Cache clearing function
func TestCacheClear(t *testing.T) {
	C := New()
	c := C.(*cache)
	defer c.Close()
	for i := 0; i < 10; i++ {
		k := fmt.Sprint(i)
		v := fmt.Sprint(i)
		c.SetWithExpire(k, v, 10)
	}
	time.Sleep(wait)

	c.Clear()

	for i := 0; i < 10; i++ {
		k := fmt.Sprint(i)
		if _, found := c.Get(k); found {
			t.Fatalf("Shouldn't found value from clear cache")
		}
	}
	if c.store.len() != 0 {
		t.Fatalf("Length(%d) is not equal to 0, after clear", c.store.len())
	}
}

// BenchmarkCacheGet Get function of Benchmark Cache
func BenchmarkCacheGet(b *testing.B) {
	k := "A"
	v := "a"

	C := New()
	c := C.(*cache)
	c.SetWithExpire(k, v, 100000)

	b.SetBytes(1)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c.Get(k)
		}
	})
}

// BenchmarkCacheGet Set function of Benchmark Cache
func BenchmarkCacheSet(b *testing.B) {
	C := New()
	c := C.(*cache)
	rand.New(rand.NewSource(currentTime().Unix()))

	b.SetBytes(1)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			k := fmt.Sprint(rand.Int())
			v := k
			c.SetWithExpire(k, v, 10000)
		}
	})
}

func TestCacheGetWithCustomLoad(t *testing.T) {
	m := map[string]interface{}{
		"A": "a",
		"B": "b",
		"":  "null",
	}
	loadFunc := func(ctx context.Context, key string) (interface{}, error) {
		if v, exist := m[key]; exist {
			return v, nil
		}
		return nil, errors.New("key not exist")
	}

	C := New()
	c := C.(*cache)
	defer c.Close()

	type args struct {
		ctx context.Context
		key string
	}
	tests := []struct {
		name    string
		args    args
		want    interface{}
		wantErr bool
	}{
		{"A", args{context.Background(), "A"}, "a", false},
		{"B", args{context.Background(), "B"}, "b", false},
		{"", args{context.Background(), ""}, "null", false},
		{"unrecognized-key", args{context.Background(), "unregonizedKey"}, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := c.GetWithCustomLoad(tt.args.ctx, tt.args.key, loadFunc, 10)
			if (err != nil) != tt.wantErr {
				t.Errorf("cache.GetWithCustomLoad() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("cache.GetWithCustomLoad() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCacheMGetWithCustomLoad(t *testing.T) {
	m := map[string]interface{}{
		"A": "a",
		"B": "b",
		"":  "null",
	}
	mLoadFunc := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
		ret := make(map[string]interface{}, 4)
		for _, key := range keys {
			if v, ok := m[key]; ok {
				ret[key] = v
			}
		}
		return ret, nil
	}

	C := New(WithExpiration(10))
	c := C.(*cache)
	defer c.Close()

	type args struct {
		ctx   context.Context
		keys  []string
		ttl   int64
		sleep time.Duration
		found map[string]bool
		setup func()
		after func()
	}
	tests := []struct {
		name    string
		args    args
		want1   map[string]interface{}
		want2   map[string]interface{}
		wantErr bool
	}{
		{
			name: "A",
			args: args{
				ctx:   context.Background(),
				keys:  []string{"A", "B"},
				ttl:   60,
				sleep: time.Millisecond,
				found: map[string]bool{
					"A": true, "B": true,
				},
				setup: func() {
					c.Clear()
				},
				after: func() {
					c.Clear()
				},
			},
			want1:   map[string]interface{}{"A": "a", "B": "b"},
			want2:   map[string]interface{}{"A": "a", "B": "b"},
			wantErr: false,
		},
		{
			name: "B",
			args: args{
				ctx:   context.Background(),
				keys:  []string{"A", "B"},
				ttl:   1,
				sleep: time.Second * 2,
				found: map[string]bool{
					"A": false, "B": false,
				},
				setup: func() {
					c.Clear()
				},
				after: func() {
					c.Clear()
				},
			},
			want1:   map[string]interface{}{"A": "a", "B": "b"},
			want2:   map[string]interface{}{"A": "", "B": ""},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.args.setup()
			got, err := c.MGetWithCustomLoad(tt.args.ctx, tt.args.keys, mLoadFunc, tt.args.ttl)
			if (err != nil) != tt.wantErr {
				t.Errorf("cache.GetWithLoad() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want1) {
				t.Errorf("cache.GetWithLoad() = %v, want1 %v", got, tt.want1)
			}
			time.Sleep(tt.args.sleep)
			// Get
			for _, key := range tt.args.keys {
				value, found := c.Get(key)
				if found != tt.args.found[key] {
					t.Errorf("found miss match, cache.Get() = %v, want2 %v", got, tt.want2)
				}
				if found && !reflect.DeepEqual(value, tt.want2[key]) {
					t.Errorf("value not equal, cache.Get() = %v, want2 %v", got, tt.want2)
				}
			}
			tt.args.after()
		})
	}
}

func TestKeysExceedCapacity(t *testing.T) {
	rand.New(rand.NewSource(time.Now().UnixNano()))
	var testKeys []string
	for i := 0; i < 10; i++ {
		testKeys = append(testKeys, fmt.Sprintf("test%d", i))
	}

	type args struct {
		capacity   int
		ttl        int64
		concurrent int
		count      int
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "cap=1, ttl=30, concurrent=1, count=100000",
			args: args{
				capacity:   1,
				ttl:        30,
				concurrent: 1,
				count:      100000,
			},
		},
		{
			name: "cap=9, ttl=30, concurrent=10, count=100000",
			args: args{
				capacity:   9,
				ttl:        30,
				concurrent: 10,
				count:      100000,
			},
		},
	}
	type testObj struct {
		key string
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New(WithCapacity(tt.args.capacity), WithExpiration(tt.args.ttl))
			var wg sync.WaitGroup
			for i := 0; i < tt.args.concurrent; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for i := 0; i < tt.args.count; i++ {
						key := testKeys[rand.Intn(len(testKeys))]
						val, ok := c.Get(key)
						if !ok {
							c.Set(key, &testObj{key: key})
							continue
						}
						if val.(*testObj).key != key {
							t.Errorf("cache.Get() = %v, want %v", val, key)
						}
					}
				}()
			}
			wg.Wait()
		})
	}
}

// TestCacheLen tests the length query function of Cache
func TestCacheLen(t *testing.T) {
	C := New(WithCapacity(4))
	c := C.(*cache)
	defer c.Close()
	mocks := []*struct {
		key string
		val string
		ttl int64
	}{
		{"A", "a", 1},
		{"B", "b", 2},
		{"C", "c", 3},
		{"D", "d", 3},
		{"E", "e", 3},
	}
	for _, mock := range mocks {
		c.SetWithExpire(mock.key, mock.val, mock.ttl)
	}
	testSheet := []*struct {
		expire time.Duration
		len    int
	}{
		{
			expire: 1 * time.Millisecond,
			len:    4,
		},
		{
			expire: 1010 * time.Millisecond,
			len:    4,
		},
		{
			expire: 1010 * time.Millisecond,
			len:    3,
		},
		{
			expire: 1010 * time.Millisecond,
			len:    0,
		},
	}
	for i, v := range testSheet {
		time.Sleep(v.expire)
		if s := c.Len(); s != v.len {
			t.Errorf("#%d cache.Len() = %v, want %v", i, s, v.len)
		}
	}
}

// BenchmarkCacheLen Len function of Benchmark Cache
func BenchmarkCacheLen(b *testing.B) {
	C := New()
	c := C.(*cache)
	defer c.Close()
	mocks := []*struct {
		key string
		val string
		ttl int64
	}{
		{"A", "a", 1},
		{"B", "b", 2},
		{"C", "c", 3},
		{"D", "d", 4},
		{"E", "e", 5},
	}
	for _, mock := range mocks {
		c.SetWithExpire(mock.key, mock.val, mock.ttl)
	}
	b.SetBytes(1)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c.Len()
		}
	})
}

// ====================== Callback function test =========================

// TestCacheOnExpire tests callback when cache expires
func TestCacheOnExpire(t *testing.T) {
	delCount := map[string]int{"A": 0, "B": 0, "": 0}
	expireCount := map[string]int{"A": 0, "B": 0, "": 0}
	C := New(
		WithCapacity(4),
		WithOnExpire(func(item *Item) {
			if item.Flag != ItemDelete {
				return
			}
			if _, ok := expireCount[item.Key]; ok {
				expireCount[item.Key]++
			}
		}),
		WithOnDel(func(item *Item) {
			if item.Flag != ItemDelete {
				return
			}
			if _, ok := delCount[item.Key]; ok {
				delCount[item.Key]++
			}
		}))

	c := C.(*cache)
	defer c.Close()

	mocks := []struct {
		key string
		val string
		ttl int64
	}{
		{"A", "a", 1},
		{"B", "b", 300},
		{"", "null", 300},
	}

	for _, mock := range mocks {
		c.SetWithExpire(mock.key, mock.val, mock.ttl)
	}
	time.Sleep(1001 * wait)

	for _, mock := range mocks {
		key := mock.key
		if key == "A" && delCount[key] != 1 {
			t.Fatalf("unexpected delCount value: %d  to key: %v, want: %d", delCount[key], key, 1)
		}
		if key != "A" && delCount[key] != 0 {
			t.Fatalf("unexpected delCount value: %d  to key: %v, want: %d", delCount[key], key, 0)
		}
		if key == "A" && expireCount[key] != 1 {
			t.Fatalf("unexpected expireCount value: %d  to key: %v, want: %d", expireCount[key], key, 1)
			continue
		}
		if key != "A" && expireCount[key] != 0 {
			t.Fatalf("unexpected expireCount value: %d  to key: %v, want: %d", expireCount[key], key, 0)
		}
	}
}

// TestCacheOnLruDel tests callback when cache LRU is deleted
func TestCacheOnLruDel(t *testing.T) {
	delCount := map[string]int{"A": 0, "B": 0, "": 0}
	C := New(WithCapacity(2), WithOnDel(func(item *Item) {
		if item.Flag != ItemLruDel {
			return
		}
		if _, ok := delCount[item.Key]; ok {
			delCount[item.Key]++
		}
	}))
	c := C.(*cache)
	defer c.Close()

	mocks := []struct {
		key string
		val string
		ttl int64
	}{
		{"A", "a", 300},
		{"B", "b", 300},
		{"", "null", 300},
	}

	for _, mock := range mocks {
		c.SetWithExpire(mock.key, mock.val, mock.ttl)
	}
	time.Sleep(2 * wait)

	for key, delCount := range delCount {
		if key == mocks[0].key && delCount != 1 {
			t.Fatalf("unexpected delCount value: %d  to key: %v, want: %d", delCount, key, 1)
		}
		if key != mocks[0].key && delCount != 0 {
			t.Fatalf("unexpected delCount value: %d  to key: %v, want: %d", delCount, key, 0)
		}
	}
}

// TestCacheOnDel tests callback when cache is deleted
func TestCacheOnDel(t *testing.T) {
	mocks := []struct {
		key string
		val string
		ttl int64
	}{
		{"A", "a", 300},
		{"B", "b", 300},
		{"", "null", 300},
	}
	delCount := map[string]int{"A": 0, "B": 0, "": 0}
	C := New(WithCapacity(4), WithOnDel(func(item *Item) {
		if item.Flag != ItemDelete {
			return
		}
		if _, ok := delCount[item.Key]; ok {
			delCount[item.Key]++
		}
	}))
	c := C.(*cache)
	defer c.Close()

	for _, mock := range mocks {
		c.SetWithExpire(mock.key, mock.val, mock.ttl)
	}

	time.Sleep(2 * wait)
	c.Del(mocks[0].key)
	time.Sleep(2 * wait)

	for key, delCount := range delCount {
		if key == mocks[0].key && delCount != 1 {
			t.Fatalf("unexpected delCount value: %d  to key: %v, want: %d", delCount, key, 1)
		}
		if key != mocks[0].key && delCount != 0 {
			t.Fatalf("unexpected delCount value: %d  to key: %v, want: %d", delCount, key, 0)
		}
	}
}

// generateFunction is executed only once
func generateFunction() func(ctx context.Context, key string) (interface{}, error) {
	alreadyExecuted := false

	return func(ctx context.Context, key string) (interface{}, error) {
		if alreadyExecuted {
			return "", errors.New("function can only be executed once")
		}

		alreadyExecuted = true
		return "Function executed successfully", nil
	}
}

// Test_GetAndSetMultipleKey concurrently reads and writes, observe whether the results are affected
func Test_GetAndSetMultipleKey(t *testing.T) {
	mocks := []struct {
		key string
		val string
		ttl int64
	}{
		{"A", "a", 300},
		{"B", "b", 300},
		{"", "null", 300},
	}
	loadFunc := generateFunction()
	// Asynchronous scenarios will have multiple loads
	dataCache := New(WithCapacity(10000001),
		WithExpiration(int64(1)), WithLoad(loadFunc), WithSettingTimeout(wait))
	dataCache.SetWithExpire(mocks[0].key, mocks[0].val, 1)
	time.Sleep(time.Second)
	// Whether the reading data observation value can be loaded multiple times
	if v, err := dataCache.GetWithLoad(context.TODO(), mocks[0].key); err == nil {
		if v != "Function executed successfully" {
			t.Fatalf("unexpected value: %v  to key: %v, want: %v", v, mocks[0].key, "Function executed successfully")
		}
	} else {
		t.Fatal("error multiple load 1")
	}
	if v, err := dataCache.GetWithLoad(context.TODO(), mocks[0].key); err == nil {
		if v != "Function executed successfully" {
			t.Fatalf("unexpected value: %v  to key: %v, want: %v", v, mocks[0].key, "Function executed successfully")
		}
	} else {
		t.Fatal("error multiple load 2")
	}
	if v, err := dataCache.GetWithLoad(context.TODO(), mocks[0].key); err == nil {
		if v != "Function executed successfully" {
			t.Fatalf("unexpected value: %v  to key: %v, want: %v", v, mocks[0].key, "Function executed successfully")
		}
	} else {
		t.Fatal("error multiple load 3")
	}
}

// Test_GetAndSetMultipleKeyAsync concurrent asynchronous reading and writing, observe whether the results are affected
func Test_GetAndSetMultipleKeyAsync(t *testing.T) {
	mocks := []struct {
		key string
		val string
		ttl int64
	}{
		{"A", "a", 300},
		{"B", "b", 300},
		{"", "null", 300},
	}
	loadFunc := generateFunction()
	// Asynchronous scenarios will have multiple loads
	dataCache := New(WithCapacity(10000001),
		WithExpiration(int64(1)), WithLoad(loadFunc))
	dataCache.SetWithExpire(mocks[0].key, mocks[0].val, 1)
	time.Sleep(time.Second)
	// Read the data and observe how many times it is loaded
	// From the second time, err will be expected
	if v, err := dataCache.GetWithLoad(context.TODO(), mocks[0].key); err == nil {
		if v != "Function executed successfully" {
			t.Fatalf("unexpected value: %v  to key: %v, want: %v", v, mocks[0].key, "Function executed successfully")
		}
	} else {
		// Can be loaded successfully for the first time
		t.Fatal("error multiple load 1")
	}
	if _, err := dataCache.GetWithLoad(context.TODO(), mocks[0].key); err == nil {
		// Unable to load successfully the second time
		t.Fatal("error multiple load 2")
	}
}

// Test_UpdateKeySync concurrent asynchronous reading and writing, writing is synchronous,
// observe whether the results are affected
func Test_UpdateKeySync(t *testing.T) {
	mocks := []struct {
		key string
		val string
		ttl int64
	}{
		{"A", "a", 300},
		{"B", "b", 300},
		{"", "null", 300},
	}
	loadFunc := func(ctx context.Context, key string) (interface{}, error) {
		return "success", nil
	}
	// Asynchronous scenarios will have multiple loads
	c := New(WithCapacity(4), WithLoad(loadFunc), WithSettingTimeout(time.Duration(0*time.Microsecond)), WithExpiration(1))
	for _, mock := range mocks {
		c.SetWithExpire(mock.key, mock.val, 0)
	}
	n := 10000000
	// Test data update
	for i := 0; i < n; i++ {
		go func() {
			time.Sleep(time.Second)
			for _, mock := range mocks {
				// At this time, you will go to update to check whether the update is successful or times out
				if val, err := c.GetWithLoad(context.TODO(), mock.key); err == nil {
					// success
					if val != "success" {
						t.Error("update fail")
						return
					}
				} else {
					// timeout
					if err.Error() != fmt.Sprintf("set key [%s] fail", mock.key) {
						t.Error("unexpected error")
						return
					}
				}

			}
		}()
	}
}

func Test_DelKeyAsync(t *testing.T) {
	mocks := []struct {
		key string
		val string
		ttl int64
	}{
		{"A", "a", 300},
		{"B", "b", 300},
		{"", "null", 300},
	}
	delCount := map[string]int{"A": 0, "B": 0, "": 0}
	C := New(WithCapacity(4), WithOnDel(func(item *Item) {
		if item.Flag != ItemDelete {
			return
		}
		if _, ok := delCount[item.Key]; ok {
			delCount[item.Key]++
		}
	}), WithSettingTimeout(1))
	c := C.(*cache)
	defer c.Close()

	for _, mock := range mocks {
		c.SetWithExpire(mock.key, mock.val, mock.ttl)
	}

	time.Sleep(2 * wait)
	c.Del(mocks[0].key)
	time.Sleep(2 * wait)

	for key, delCount := range delCount {
		if key == mocks[0].key && delCount != 1 {
			t.Fatalf("unexpected delCount value: %d  to key: %v, want: %d", delCount, key, 1)
		}
		if key != mocks[0].key && delCount != 0 {
			t.Fatalf("unexpected delCount value: %d  to key: %v, want: %d", delCount, key, 0)
		}
	}
}

func Test_DelKeySync(t *testing.T) {
	mocks := []struct {
		key string
		val string
		ttl int64
	}{
		{"A", "a", 300},
		{"B", "b", 300},
		{"", "null", 300},
	}
	delCount := map[string]int{"A": 0, "B": 0, "": 0}
	C := New(WithCapacity(4), WithOnDel(func(item *Item) {
		if item.Flag != ItemDelete {
			return
		}
		if _, ok := delCount[item.Key]; ok {
			delCount[item.Key]++
		}
	}), WithSettingTimeout(1), WithSyncDelFlag(true))
	c := C.(*cache)
	defer c.Close()

	for _, mock := range mocks {
		c.SetWithExpire(mock.key, mock.val, mock.ttl)
	}

	c.Del(mocks[0].key)

	for key, delCount := range delCount {
		if key == mocks[0].key && delCount != 1 {
			t.Fatalf("unexpected delCount value: %d  to key: %v, want: %d", delCount, key, 1)
		}
		if key != mocks[0].key && delCount != 0 {
			t.Fatalf("unexpected delCount value: %d  to key: %v, want: %d", delCount, key, 0)
		}
	}
}
