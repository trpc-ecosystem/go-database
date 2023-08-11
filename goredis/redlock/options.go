package redlock

import "time"

const (
	defaultLockTimeout = 1 * time.Second // The default is the maximum waiting time for a single lock grab,
	// and the Lock function needs to be used.
	defaultKeyExpiration = 10 * time.Second       // Default lock key expiration time.
	defaultLockInterval  = 100 * time.Millisecond // The default sleep time for a single lock grab,
	// the Lock function needs to be used.
	defaultExtendInterval = 10 * time.Second // Default lock key renewal time.
)

// Options is lock parameters.
type Options struct {
	lockTimeout    time.Duration // The longest waiting time for a single lock, the Lock function needs to be used.
	keyExpiration  time.Duration // Redis lock key expiration time.
	lockInterval   time.Duration // For the sleep time of a single lock grab, the Lock function needs to be used.
	extendInterval time.Duration // Renewal interval.
}

// WithLockTimeout is the longest waiting time for a single lock grab,
// the Lock function needs to be used.
func WithLockTimeout(d time.Duration) Option {
	return func(options *Options) {
		options.lockTimeout = d
	}
}

// WithLockInterval is the sleep time for a single lock grab, and the Lock function needs to be used.
func WithLockInterval(d time.Duration) Option {
	return func(options *Options) {
		options.lockInterval = d
	}
}

// WithKeyExpiration is redis lock key expiration time.
func WithKeyExpiration(d time.Duration) Option {
	return func(options *Options) {
		options.keyExpiration = d
	}
}

// WithExtendInterval is redis lock key renewal interval.
func WithExtendInterval(d time.Duration) Option {
	return func(options *Options) {
		options.extendInterval = d
	}
}

// clone is parameter copy.
func (o *Options) clone() *Options {
	n := *o
	return &n
}

func (l *redLock) newOptions(opts ...Option) *Options {
	options := &Options{
		lockTimeout:    defaultLockTimeout,
		keyExpiration:  defaultKeyExpiration,
		lockInterval:   defaultLockInterval,
		extendInterval: defaultExtendInterval,
	}
	for _, o := range l.opts {
		o(options)
	}
	for _, o := range opts {
		o(options)
	}
	return options
}
