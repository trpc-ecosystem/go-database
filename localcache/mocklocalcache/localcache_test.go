package mocklocalcache

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"trpc.group/trpc-go/trpc-database/localcache"
)

func TestLocalCache(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	// 1. Generate mock cache
	cache := NewMockCache(ctrl)
	// 2. Assign a value to the cache variable at the code call or pass it as a parameter
	// gLocalcache = cache
	// 3. Mock corresponding function
	cache.EXPECT().Get(gomock.Any()).DoAndReturn(func(key interface{}) (interface{}, bool) {
		t.Logf("key:%v", key)
		return nil, true
	})
	cache.EXPECT().SetWithExpire(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(key, value interface{}, ttl int64) bool {
			t.Logf("key:%v", key)
			return true
		})
	cache.EXPECT().GetWithLoad(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, key string) (interface{}, error) {
			t.Logf("GetWithLoad()")
			return nil, nil
		})
	cache.EXPECT().MGetWithLoad(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, keys []string) (map[string]interface{}, error) {
			t.Logf("MGetWithLoad()")
			return nil, nil
		})
	cache.EXPECT().Set(gomock.Any(), gomock.Any()).
		DoAndReturn(func(key string, value interface{}) bool {
			t.Logf("Set()")
			return true
		})
	cache.EXPECT().Del(gomock.Any()).
		DoAndReturn(func(key string) {
			t.Logf("Del()")
		})
	cache.EXPECT().GetWithCustomLoad(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, key string, customLoad localcache.LoadFunc, ttl int64) (
			interface{}, error) {
			t.Logf("GetWithCustomLoad()")
			return nil, nil
		})
	cache.EXPECT().MGetWithCustomLoad(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, keys []string, customLoad localcache.MLoadFunc, ttl int64) (
			map[string]interface{}, error) {
			t.Logf("MGetWithCustomLoad()")
			return nil, nil
		})
	cache.EXPECT().Len().Return(4)
	cache.EXPECT().Clear().DoAndReturn(func() {})
	cache.EXPECT().Close().DoAndReturn(func() {})

	// 4. Call
	ctx := context.Background()
	ok := cache.SetWithExpire("Foo", "bar", 1)
	assert.Equal(t, true, ok)
	v, ok := cache.Get("time")
	assert.Nil(t, v)
	assert.Equal(t, true, ok)
	_, err := cache.GetWithLoad(ctx, "Foo")
	assert.Nil(t, err)
	_, err = cache.MGetWithLoad(ctx, []string{"Foo"})
	assert.Nil(t, err)
	ok = cache.Set("abs", "abs")
	assert.Equal(t, true, ok)
	cache.Del("abs")
	_, err = cache.GetWithCustomLoad(ctx, "Foo",
		func(ctx context.Context, key string) (interface{}, error) {
			return nil, nil
		}, 12,
	)
	assert.Nil(t, err)
	_, err = cache.MGetWithCustomLoad(ctx, []string{"Foo"},
		func(ctx context.Context, key []string) (map[string]interface{}, error) {
			return nil, nil
		}, 12,
	)
	cacheLen := cache.Len()
	assert.Equal(t, 4, cacheLen)
	assert.Nil(t, err)
	cache.Clear()
	cache.Close()
}

func TestLocalCache2(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	// 1. Generate mock cache
	cache := NewMockCache(ctrl)
	// 2. Assign a value to the cache variable at the code call or pass it as a parameter
	// gLocalcache = cache
	// 3. Mock corresponding function
	cache.EXPECT().GetWithStatus(gomock.Any()).DoAndReturn(func(key interface{}) (interface{}, localcache.CachedStatus) {
		t.Logf("key:%v", key)
		return "bar", localcache.CacheExpire
	})
	// 4. Call
	val, status := cache.GetWithStatus("foo")
	assert.Equal(t, val, "bar")
	assert.Equal(t, status, localcache.CacheExpire)
}
