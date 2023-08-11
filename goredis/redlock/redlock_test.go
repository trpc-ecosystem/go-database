package redlock

import (
	"context"
	"fmt"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	redis "github.com/redis/go-redis/v9"
	goredis "trpc.group/trpc-go/trpc-database/goredis"
	trpc "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
)

var (
	testCtx context.Context
)

func init() {
	trpc.ServerConfigPath = "../trpc_go.yaml"
	trpc.NewServer()
	testCtx = trpc.BackgroundContext()
}

func TestRedLock_TryLock(t *testing.T) {
	c := newMiniClient(t)
	redLock, err := New(c)
	if err != nil {
		t.Fatal(err)
	}
	key := "l1"
	opts := []Option{WithKeyExpiration(3 * time.Second)}
	t.Run("ok", func(t *testing.T) {
		mu, err := redLock.TryLock(testCtx, key, opts...)
		if err != nil {
			t.Fatalf("try lock fail %v", err)
		}
		_ = mu.Unlock(testCtx)
		mu2, err := redLock.TryLock(testCtx, key, opts...)
		if err != nil {
			t.Fatalf("try lock fail %v", err)
		}
		_ = mu2.Unlock(testCtx)
	})
	t.Run("occupied", func(t *testing.T) {
		mu, err := redLock.TryLock(testCtx, key, opts...)
		if err != nil {
			t.Fatalf("try lock fail %v", err)
		}
		defer mu.Unlock(testCtx) //nolint:errcheck
		if _, err = redLock.TryLock(testCtx, key, opts...); err != nil {
			t.Logf("try lock fail %v", err)
			return
		}
	})
}

func TestRedLock_Lock(t *testing.T) {
	c := newMiniClient(t)
	redLock, nErr := New(c)
	if nErr != nil {
		t.Fatal(nErr)
	}
	t.Run("ok", func(t *testing.T) {
		key := "l2"
		c.Del(testCtx, key)
		opts := []Option{
			WithKeyExpiration(1 * time.Second),
			WithLockTimeout(500 * time.Millisecond),
			WithLockInterval(5 * time.Millisecond),
		}
		start := time.Now()
		mu, err := redLock.Lock(testCtx, key, opts...)
		if err != nil {
			t.Fatalf("lock fail %v %v", err, time.Since(start))
		}
		t.Logf("lock success %v", time.Since(start))
		if err = mu.Unlock(testCtx); err != nil {
			t.Fatalf("unlock fail %v %v", err, time.Since(start))
		}
		mu2, err := redLock.Lock(testCtx, key, opts...)
		if err != nil {
			t.Fatalf("lock fail %v %v", err, time.Since(start))
		}
		defer mu2.Unlock(testCtx) //nolint:errcheck
	})
	t.Run("timeout", func(t *testing.T) {
		key := "l3"
		start := time.Now()
		c.Del(testCtx, key)
		_, err := redLock.Lock(testCtx, key, WithLockTimeout(1*time.Millisecond))
		if err != nil {
			t.Logf("lock fail %v %v", err, time.Since(start))
			time.Sleep(1 * time.Second)
			return
		}
	})
	t.Run("expired lock", func(t *testing.T) {
		key := "l4"
		c.Del(testCtx, key)
		mu, err := redLock.Lock(testCtx, key, WithLockTimeout(1*time.Second))
		if err != nil {
			t.Fatalf("lock fail %v", err)
		}
		t.Logf("%v lock success", time.Now())
		time.Sleep(3100 * time.Millisecond)
		if err = mu.Unlock(testCtx); err != nil {
			t.Logf("unlock fail %v", err)
			return
		}
	})
	t.Run("been robbed", func(t *testing.T) {
		key := "l6"
		c.Del(testCtx, key)
		opts := []Option{
			WithKeyExpiration(3 * time.Second),
			WithLockTimeout(1 * time.Second),
		}
		mu, err := redLock.Lock(testCtx, key, opts...)
		if err != nil {
			t.Fatalf("lock fail %v", err)
		}
		t.Logf("%v lock success", time.Now())
		time.Sleep(3100 * time.Millisecond)
		mu2, err := redLock.Lock(testCtx, key, opts...)
		if err != nil {
			t.Logf("lock fail %v", err)
			return
		}
		if err = mu.Unlock(testCtx); err != nil {
			t.Logf("unlock fail %v", err)
		}
		if err = mu2.Unlock(testCtx); err != nil {
			t.Fatalf("unlock2 fail %v", err)
		}
	})
}

// newMiniClient creates a new memory version of redis.
func newMiniClient(t *testing.T) redis.UniversalClient {
	s := miniredis.RunT(t)
	target := fmt.Sprintf("redis://%s/0", s.Addr())
	c, err := goredis.New("trpc.gamecenter.test.redis", client.WithTarget(target))
	if err != nil {
		t.Fatalf("%+v", err)
	}
	return c
}
