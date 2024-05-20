package localcache

import (
	"testing"
)

// TestStoreSetGet tests the Set and Get methods of Store
func TestStoreSetGet(t *testing.T) {
	store := newStore()
	mocks := []struct {
		key string
		val string
	}{
		{"A", "a"},
		{"B", "b"},
		{"", "null"},
	}

	for _, mock := range mocks {
		store.set(mock.key, mock.val)
	}

	for _, mock := range mocks {
		val, ok := store.get(mock.key)
		if !ok || val.(string) != mock.val {
			t.Fatalf("unexpected value: %v (%v) to key: %v", val, ok, mock.key)
		}
	}
}

// TestStoreSetNil tests the situation when Store Set nil
func TestStoreSetNil(t *testing.T) {
	store := newStore()
	store.set("no", nil)
	val, ok := store.get("no")
	if !ok || val != nil {
		t.Fatalf("unexpected value: %v (%v)", val, ok)
	}
}

// TestStoreDel tests Store's Del method
func TestStoreDel(t *testing.T) {
	store := newStore()
	mocks := []struct {
		key string
		val interface{}
	}{
		{"A", "a"},
		{"B", "b"},
		{"C", nil},
		{"", "null"},
	}

	for _, mock := range mocks {
		store.set(mock.key, mock.val)
	}

	for _, mock := range mocks {
		store.del(mock.key)
		val, ok := store.get(mock.key)
		if ok || val != nil {
			t.Fatalf("del error, key: %v value: %v (%v)", mock.key, val, ok)
		}
	}
}

// TestStoreClear tests Store's Clear method
func TestStoreClear(t *testing.T) {
	store := newStore()
	mocks := []struct {
		key string
		val interface{}
	}{
		{"A", "a"},
		{"B", "b"},
		{"C", nil},
		{"", "null"},
	}
	for _, mock := range mocks {
		store.set(mock.key, mock.val)
	}
	store.clear()
	for _, mock := range mocks {
		val, ok := store.get(mock.key)
		if ok || val != nil {
			t.Fatalf("clear error, key: %v value: %v (%v)", mock.key, val, ok)
		}
	}
}

// BenchmarkStoreGet Benchmark Store's Get method
func BenchmarkStoreGet(b *testing.B) {
	k := "A"
	v := "a"

	s := newStore()
	s.set(k, v)
	b.SetBytes(1)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			s.get(k)
		}
	})
}

// BenchmarkStoreGet Benchmark Store's Set method
func BenchmarkStoreSet(b *testing.B) {
	k := "A"
	v := "a"

	s := newStore()
	b.SetBytes(1)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			s.set(k, v)
		}
	})
}
