package localcache

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestFuncGetAndSet tests Get and Set
func TestFuncGetAndSet(t *testing.T) {
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
		Set(mock.key, mock.val, mock.ttl)
	}

	time.Sleep(wait)

	for _, mock := range mocks {
		val, found := Get(mock.key)
		if !found || val.(string) != mock.val {
			t.Fatalf("Unexpected value: %v (%v) to key: %v", val, found, mock.key)
		}
	}
}

// TestFuncExpireSet tests Set with Expire
func TestFuncExpireSet(t *testing.T) {
	Set("Foo", "Bar", 1)
	time.Sleep(1 * time.Second)
	if val, found := Get("Foo"); found {
		t.Fatalf("unexpected expired value: %v to key: %v", val, "Foo")
	}
}

// TestFuncGetWithLoad tests Get in the case of WithLoad
func TestFuncGetWithLoad(t *testing.T) {
	m := map[string]interface{}{
		"A1": "a",
		"B":  "b",
		"":   "null",
	}
	loadFunc := func(ctx context.Context, key string) (interface{}, error) {
		if v, exist := m[key]; exist {
			return v, nil
		}
		return nil, errors.New("key not exist")
	}
	value, err := GetWithLoad(context.TODO(), "A1", loadFunc, 2)
	if err != nil || value.(string) != "a" {
		t.Fatalf("unexpected GetWithLoad value: %v, want:a, err:%v", value, err)
	}

	time.Sleep(wait)

	got2, found := Get("A1")
	if !found || got2.(string) != "a" {
		t.Fatalf("unexpected Get value: %v, want:a, found:%v", got2, found)
	}
}

func TestDelAndClear(t *testing.T) {
	Set("Foo", "bar", 10)
	Set("Foo1", "bar", 10)
	time.Sleep(time.Millisecond * 10)
	_, ok := Get("Foo")
	assert.True(t, ok)
	Del("Foo")
	time.Sleep(time.Millisecond * 10)
	_, ok = Get("Foo")
	assert.False(t, ok)
	Clear()
	time.Sleep(time.Millisecond * 10)
	_, ok = Get("Foo1")
	assert.False(t, ok)
}
