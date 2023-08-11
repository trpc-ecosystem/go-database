package redlock

import (
	"testing"
	"time"
)

func Test_mutex_Extend(t *testing.T) {
	c := newMiniClient(t)
	lock, nErr := New(c)
	if nErr != nil {
		t.Fatal(nErr)
	}
	const key = "k7"
	t.Run("extend", func(t *testing.T) {
		mu, err := lock.Lock(testCtx, key)
		if err != nil {
			t.Fatalf("%+v", err)
		}
		ttl, err := mu.TTL(testCtx)
		if err != nil {
			t.Fatalf("%+v", err)
		}
		t.Logf("ttl %v", ttl)
		if err = mu.Extend(testCtx, WithExtendInterval(20*time.Second)); err != nil {
			t.Fatalf("%+v", err)
		}
		if ttl, err = mu.TTL(testCtx); err != nil {
			t.Fatalf("%+v", err)
		}
		t.Logf("ttl %v", ttl)
	})
}
