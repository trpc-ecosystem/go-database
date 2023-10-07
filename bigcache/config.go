package bigcache

import (
	"time"

	bigcache "github.com/allegro/bigcache/v3"
)

// Config is a configuration parameter that can be used to initialize BigCache.
type Config struct {
	// Shards is the number of shards, must be a power of 2, such as 2, 4, 8, 16, 32, etc.
	Shards int
	// LifeWindow is the life window of data in the cache.
	LifeWindow time.Duration
	// CleanWindow is the window between each cleanup of expired data, set to a value less than 0 to disable cleanup.
	// Default value is 1 second.
	CleanWindow time.Duration
	// MaxEntriesInWindow is the maximum number of data entries, only used when initializing cache shards.
	MaxEntriesInWindow int
	// MaxEntrySize is the maximum size of a value in bytes, only used when initializing cache shards.
	MaxEntrySize int
	// StatsEnabled enables tracking of hit count for individual keys when set to true.
	StatsEnabled bool
	// Verbose enables output of memory allocation information when set to true.
	Verbose bool
	// HardMaxCacheSize is the maximum cache size in MB.
	// It's set to prevent excessive memory allocation and avoid OOM errors,
	// and it only takes effect when set to a value greater than 0.
	HardMaxCacheSize int
	Logger           bigcache.Logger
	// OnRemove is a callback fired when the oldest entry is removed because of its expiration time or no space left
	// for the new entry, or because delete was called.
	// Default value is nil which means no callback and it prevents from unwrapping the oldest entry.
	// ignored if OnRemoveWithMetadata is specified.
	OnRemove func(key string, entry []byte)
	// OnRemoveWithMetadata is a callback fired when the oldest entry is removed because of its expiration time
	// or no space left for the new entry, or because delete was called.
	// A structure representing details about that specific entry.
	// Default value is nil which means no callback and it prevents from unwrapping the oldest entry.
	OnRemoveWithMetadata func(key string, entry []byte, keyMetadata bigcache.Metadata)
	// OnRemoveWithReason is a callback fired when the oldest entry is removed because of its expiration time
	// or no space left for the new entry, or because delete was called.
	// A constant representing the reason will be passed through.
	// Default value is nil which means no callback and it prevents from unwrapping the oldest entry.
	// Ignored if OnRemove is specified.
	OnRemoveWithReason func(key string, entry []byte, reason bigcache.RemoveReason)
	// Hasher used to map between string keys and unsigned 64bit integers, by default fnv64 hashing is used.
	Hasher bigcache.Hasher
}

// DefaultConfig  initializes the default values for the main config parameters.
// The default life window is set to 7 days, and the default cleaning window is set to 1 minute.
func DefaultConfig() Config {
	return Config{
		Shards:             1024,
		LifeWindow:         7 * 24 * time.Hour,
		CleanWindow:        time.Minute,
		MaxEntriesInWindow: 1000 * 10 * 60,
		MaxEntrySize:       500,
		StatsEnabled:       false,
		Verbose:            true,
		HardMaxCacheSize:   0,
	}
}

// InitDefaultConfig initializes the config used by the third-party library BigCache.
// You can directly modify the bigcache.Config if necessary.
func InitDefaultConfig(cfg Config) bigcache.Config {
	return bigcache.Config{
		Shards:               cfg.Shards,
		LifeWindow:           cfg.LifeWindow,
		CleanWindow:          cfg.CleanWindow,
		MaxEntriesInWindow:   cfg.MaxEntriesInWindow,
		MaxEntrySize:         cfg.MaxEntrySize,
		StatsEnabled:         cfg.StatsEnabled,
		Verbose:              cfg.Verbose,
		Hasher:               cfg.Hasher,
		HardMaxCacheSize:     cfg.HardMaxCacheSize,
		Logger:               cfg.Logger,
		OnRemove:             cfg.OnRemove,
		OnRemoveWithReason:   cfg.OnRemoveWithReason,
		OnRemoveWithMetadata: cfg.OnRemoveWithMetadata,
	}
}

// Option declares the option functions of the cache
type Option func(*Config)

// WithShards sets the number of shards
func WithShards(i int) Option {
	return func(opts *Config) {
		opts.Shards = i
	}
}

// WithLifeWindow sets the life window
func WithLifeWindow(t time.Duration) Option {
	return func(opts *Config) {
		opts.LifeWindow = t
	}
}

// WithCleanWindow sets the clean window
func WithCleanWindow(t time.Duration) Option {
	return func(opts *Config) {
		opts.CleanWindow = t
	}
}

// WithMaxEntriesInWindow sets the maximum number of data entries, used only during initialization,
// and affects the minimum memory usage.
func WithMaxEntriesInWindow(i int) Option {
	return func(opts *Config) {
		opts.MaxEntriesInWindow = i
	}
}

// WithMaxEntrySize sets the maximum size of a value in bytes, used only during initialization,
// and affects the minimum memory usage.
func WithMaxEntrySize(i int) Option {
	return func(opts *Config) {
		opts.MaxEntrySize = i
	}
}

// WithLogger sets the logger
func WithLogger(l bigcache.Logger) Option {
	return func(opts *Config) {
		opts.Logger = l
	}
}

// WithHardMaxCacheSize sets the maximum cache size in MB.
// It's set to prevent excessive memory allocation and avoid OOM errors,
// and it only takes effect when set to a value greater than 0.
func WithHardMaxCacheSize(m int) Option {
	return func(opts *Config) {
		opts.HardMaxCacheSize = m
	}
}

// WithVerbose sets whether to enable output of memory allocation information.
func WithVerbose(v bool) Option {
	return func(opts *Config) {
		opts.Verbose = v
	}
}

// WithStatsEnabled sets whether to enable tracking of hit count for individual keys.
func WithStatsEnabled(s bool) Option {
	return func(opts *Config) {
		opts.StatsEnabled = s
	}
}

// WithHasher sets hasher interface
func WithHasher(h bigcache.Hasher) Option {
	return func(config *Config) {
		config.Hasher = h
	}
}

// WithOnRemoveCallback sets OnRemove callback function
func WithOnRemoveCallback(f func(key string, entry []byte)) Option {
	return func(opts *Config) {
		opts.OnRemove = f
	}
}

// WithOnRemoveWithMetadataCallback sets OnRemoveWithMetadata callback function
func WithOnRemoveWithMetadataCallback(f func(key string, entry []byte, keyMetadata bigcache.Metadata)) Option {
	return func(opts *Config) {
		opts.OnRemoveWithMetadata = f
	}
}

// WithOnRemoveWithReasonCallback sets OnRemoveWithReason callback function
func WithOnRemoveWithReasonCallback(f func(key string, entry []byte, reason bigcache.RemoveReason)) Option {
	return func(opts *Config) {
		opts.OnRemoveWithReason = f
	}
}
