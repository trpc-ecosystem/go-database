// Package redlock distributed lock。
package redlock

import (
	"context"
	"time"

	redis "github.com/redis/go-redis/v9"
	goredis "trpc.group/trpc-go/trpc-database/goredis"
	trpc "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/errs"
)

//go:generate mockgen -source=redlock.go -destination=mockredlock/redlocker_mock.go -package=mockredlock

// RedLocker is distributed lock interface.
type RedLocker interface {
	// TryLock tries to unlock, return immediately and report an error if you don't get the lock,
	// and make sure that all operations are done on the master node.
	TryLock(ctx context.Context, key string, opts ...Option) (Mutex, error)
	// Lock will sleep and wait if the lock is not acquired until it grabs the lock or times out.
	Lock(ctx context.Context, key string, opts ...Option) (Mutex, error)
}

var _ RedLocker = &redLock{}

// redLock is redis distributed lock.
type redLock struct {
	cmdable redis.Cmdable
	opts    []Option
}

// New creates a new distributed lock object.
func New(c redis.Cmdable, opts ...Option) (RedLocker, error) {
	l := &redLock{
		cmdable: c,
		opts:    opts,
	}
	return l, nil
}

// Unlock releases lock function type.
type Unlock func(ctx context.Context) error

// TryLock tries to unlock the lock, return immediately and report an error if the lock is not obtained,
// and ensure that all operations are completed on the master node.
func (l *redLock) TryLock(ctx context.Context, key string, opts ...Option) (Mutex, error) {
	options := l.newOptions(opts...)
	return l.tryLock(ctx, key, options)
}

// Lock will sleep and wait if the lock is not acquired until it grabs the lock or times out.
func (l *redLock) Lock(ctx context.Context, key string, opts ...Option) (Mutex, error) {
	options := l.newOptions(opts...)
	ctx, cancel := context.WithTimeout(ctx, options.lockTimeout)
	defer cancel()
	var ticker *time.Ticker
	for {
		mu, err := l.tryLock(ctx, key, options)
		if err != goredis.ErrLockOccupied {
			return mu, err
		}
		// Performance optimization: Start the timer without acquiring the lock.
		if ticker == nil {
			ticker = time.NewTicker(options.lockInterval)
			defer ticker.Stop()
		}
		// The lock is already taken and needs to be retried.
		select {
		case <-ctx.Done():
			return nil, errs.New(errs.RetClientTimeout, context.DeadlineExceeded.Error())
		case <-ticker.C:
			// This will only jump out of select, not for.
		}
	}
}

// TryLock tries to unlock the lock, returns immediately and reports an error if the lock is not obtained,
// and ensures that all operations are completed on the master node.
func (l *redLock) tryLock(ctx context.Context, key string, options *Options) (Mutex, error) {
	mu := newMutex(l.cmdable, options, key)
	// Modification operations will only be done on the master node.
	ok, err := l.cmdable.SetNX(ctx, key, mu.value, options.keyExpiration).Result()
	if err != nil {
		// In any case, if it times out, you need to clear the lock you created.
		// ctx may have timed out, so don't use.
		_ = mu.Unlock(trpc.CloneContext(ctx))
		return nil, err
	}
	if !ok {
		return nil, goredis.ErrLockOccupied
	}
	return mu, nil
}

// Option is lock Option callback function type.
type Option func(options *Options)
