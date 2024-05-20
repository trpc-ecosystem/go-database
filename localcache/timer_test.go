package localcache

import (
	"testing"
	"time"
)

// TestExpireQueue_Add tests the Add method of the expireQueue
func TestExpireQueue_Add(t *testing.T) {
	q := newExpireQueue(time.Second, 60)
	mocks := []struct {
		k   string
		ttl time.Duration
	}{
		{"A", time.Second},
		{"B", 1 * time.Second},
		{"", 2 * time.Second},
	}
	for _, mock := range mocks {
		t.Run("", func(t *testing.T) {
			exitC := make(chan time.Time)

			start := currentTime()
			q.add(mock.k, start.Add(mock.ttl), func() {
				exitC <- currentTime()
			})

			got := (<-exitC).Truncate(time.Second)
			min := start.Add(mock.ttl).Truncate(time.Second)

			err := time.Second
			if got.Before(min) || got.After(min.Add(err)) {
				t.Fatalf("Timer(%s) expiration: want [%s, %s], got %s", mock.ttl, min, min.Add(err), got)
			}
		})
	}
	if len(q.timers) != 0 {
		t.Fatalf("Length(%d) of timers is not equal to 0", len(q.timers))
	}
}

// TestExpireQueue_Remove tests the Remove method of the expireQueue
func TestExpireQueue_Remove(t *testing.T) {
	q := newExpireQueue(time.Second, 60)
	mocks := []struct {
		k   string
		ttl time.Duration
	}{
		{"B", 1 * time.Second},
		{"", 1 * time.Second},
	}
	for _, mock := range mocks {
		t.Run("", func(t *testing.T) {
			exitC := make(chan time.Time)

			start := currentTime()
			q.add(mock.k, start.Add(mock.ttl), func() {
				exitC <- currentTime()
			})
			q.remove(mock.k)

			timerC := time.NewTimer(time.Second * 2).C
			select {
			case <-exitC:
				t.Fatalf("Failed to remove timer(%s, %v)", mock.k, mock.ttl)
			case <-timerC:
				return
			}
		})
	}
	if len(q.timers) != 0 {
		t.Fatalf("Length(%d) of timers is not equal to 0", len(q.timers))
	}
}

// TestExpireQueue_Update tests the Update method of the expireQueue
func TestExpireQueue_Update(t *testing.T) {
	q := newExpireQueue(time.Second, 60)

	exitC := make(chan time.Time)

	start := currentTime()
	q.add("A", start.Add(2*time.Second), func() {
		exitC <- currentTime()
	})

	q.update("A", start.Add(2*time.Second), func() {
		exitC <- currentTime()
	})

	got := (<-exitC).Truncate(time.Second)
	min := start.Add(2 * time.Second).Truncate(time.Second)

	err := time.Second
	if got.Before(min) || got.After(min.Add(err)) {
		t.Fatalf("Timer(%s) expiration: want [%s, %s], got %s", "5", min, min.Add(err), got)
	}

	if len(q.timers) != 0 {
		t.Fatalf("Length(%d) of timers is not equal to 0", len(q.timers))
	}

}

// TestExpireQueue_Clear tests the Clear method of the expireQueue
func TestExpireQueue_Clear(t *testing.T) {
	q := newExpireQueue(time.Second, 60)

	exitC := make(chan time.Time)

	start := currentTime()
	q.add("A", start.Add(time.Second), func() {
		exitC <- currentTime()
	})
	q.clear()

	// After clearing, determine whether it will be cleared
	timerC := time.NewTimer(2 * time.Second).C
	select {
	case <-exitC:
		t.Fatalf("Failed to remove timer(%s, %v)", "A", "2s")
	case <-timerC:
	}

	// re-add
	start = currentTime()
	q.add("B", start.Add(2*time.Second), func() {
		exitC <- currentTime()
	})

	got := (<-exitC).Truncate(time.Second)
	min := start.Add(2 * time.Second).Truncate(time.Second)

	err := time.Second
	if got.Before(min) || got.After(min.Add(err)) {
		t.Fatalf("Timer(%s) expiration: want [%s, %s], got %s", "5", min, min.Add(err), got)
	}

	if len(q.timers) != 0 {
		t.Fatalf("Length(%d) of timers is not equal to 0", len(q.timers))
	}
}

// TestExpireQueue_Stop tests the stop method of the expireQueue
func TestExpireQueue_Stop(t *testing.T) {
	q := newExpireQueue(time.Second, 60)

	exitC := make(chan time.Time)

	start := currentTime()
	q.add("A", start.Add(2*time.Second), func() {
		exitC <- currentTime()
	})
	q.stop()

	// After clearing, determine whether it will be cleared
	timerC := time.NewTimer(3 * time.Second).C
	select {
	case <-exitC:
		t.Fatalf("Failed to remove timer(%s, %v)", "A", "2s")
	case <-timerC:
	}
}
