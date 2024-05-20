package localcache

import (
	"sync"
	"time"

	"github.com/RussellLuo/timingwheel"
)

// expireQueue stores tasks that are automatically deleted after the key expires
type expireQueue struct {
	tick      time.Duration
	wheelSize int64
	// The time wheel stores tasks that are scheduled to expire and be deleted.
	tw *timingwheel.TimingWheel

	mu     sync.Mutex
	timers map[string]*timingwheel.Timer
}

// newExpireQueue generates an expireQueue object.
// The queue is implemented through a time wheel.
// The elements in the queue are deleted regularly according to the expiration time.
func newExpireQueue(tick time.Duration, wheelSize int64) *expireQueue {
	queue := &expireQueue{
		tick:      tick,
		wheelSize: wheelSize,

		tw:     timingwheel.NewTimingWheel(tick, wheelSize),
		timers: make(map[string]*timingwheel.Timer),
	}

	// Start a goroutine to handle expired entries
	queue.tw.Start()
	return queue
}

// add scheduled expired tasks.
// When each scheduled task expires, it will be executed as an independent goroutine.
func (q *expireQueue) add(key string, expireTime time.Time, f func()) {
	q.mu.Lock()
	defer q.mu.Unlock()

	d := expireTime.Sub(currentTime())
	timer := q.tw.AfterFunc(d, q.task(key, f))
	q.timers[key] = timer

	return
}

// update the expiration time of the key element
func (q *expireQueue) update(key string, expireTime time.Time, f func()) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if timer, ok := q.timers[key]; ok {
		timer.Stop()
	}

	d := expireTime.Sub(currentTime())
	timer := q.tw.AfterFunc(d, q.task(key, f))
	q.timers[key] = timer
}

// remove element key
func (q *expireQueue) remove(key string) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if timer, ok := q.timers[key]; ok {
		timer.Stop()
		delete(q.timers, key)
	}
}

// clear the queue
func (q *expireQueue) clear() {
	q.tw.Stop()
	q.tw = timingwheel.NewTimingWheel(q.tick, q.wheelSize)
	q.timers = make(map[string]*timingwheel.Timer)

	// Restart a goroutine to process expired entries
	q.tw.Start()
}

// stop the running of the time wheel queue
func (q *expireQueue) stop() {
	q.tw.Stop()
}

func (q *expireQueue) task(key string, f func()) func() {
	return func() {
		f()
		q.mu.Lock()
		delete(q.timers, key)
		q.mu.Unlock()
	}
}
