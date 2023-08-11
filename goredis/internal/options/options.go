// Package options is redis configuration parameters.
package options

import (
	"net/url"
	"runtime"
	"strconv"
	"strings"
	"time"

	redis "github.com/redis/go-redis/v9"
	pb "trpc.group/trpc-go/trpc-database/goredis/internal/proto"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/naming/selector"
	"trpc.group/trpc-go/trpc-go/transport"

	// register http protocol.
	_ "trpc.group/trpc-go/trpc-go/http"
	// Calculate the number of CPU cores in the container, which is needed to determine the pool_size.
	_ "go.uber.org/automaxprocs"
)

// Constant.
const (
	RetParseConfigFail   = 30012 // Configuration parsing failed.
	MaxConnectionsPreCPU = 250   // The maximum number of connections to a single cpu core.

	RedisSelectorName  = "redis"  // redis
	RedissSelectorName = "rediss" // redis security.
)

// Options is redis configuration parameters.
type Options struct {
	Namespace string // Namespaces.
	Service   string // Called service name.

	trpcOption  *client.Options
	RedisOption *redis.UniversalOptions
	QueryOption *pb.QueryOptions
}

// New handles configuration parameter parsing.
// redis.ParseURL: https://github.com/redis/go-redis/blob/v9.0.2/options.go#L222
// redis.ParseClusterURL: https://github.com/redis/go-redis/blob/v9.0.2/cluster.go#L137
func New(trpcOption *client.Options) (*Options, error) {
	o := &Options{
		Namespace:  parseNamespace(trpcOption),
		trpcOption: trpcOption,
	}
	u, err := url.Parse(trpcOption.Target)
	if err != nil {
		return nil, errs.Wrapf(err, RetParseConfigFail, "url.Parse fail %v", err)
	}
	o.RedisOption, o.QueryOption, err = newOptions(u.RawQuery)
	if err != nil {
		return nil, err
	}
	if o.Service, err = parseURI(o.RedisOption, u, o.Namespace, o.QueryOption.IsProxy); err != nil {
		return nil, err
	}
	// Fix default field values.
	if err = o.fixRedisOptions(u.RawQuery); err != nil {
		return nil, err
	}
	return o, nil
}

// fixRedisOptions fixes redis.Options defaults.
func (o *Options) fixRedisOptions(rawQuery string) error {
	q, err := url.ParseQuery(rawQuery)
	if err != nil {
		return errs.Wrapf(err, RetParseConfigFail, "url.ParseQuery fail %v", err)
	}
	r := o.RedisOption
	if !q.Has("context_timeout_enabled") {
		r.ContextTimeoutEnabled = true
	}
	if !q.Has("pool_size") {
		r.PoolSize = int(calcPoolSize())
	}
	if !q.Has("dial_timeout") {
		r.DialTimeout = o.trpcOption.Timeout
	}
	if !q.Has("read_timeout") {
		r.ReadTimeout = o.trpcOption.Timeout
	}
	if !q.Has("write_timeout") {
		r.WriteTimeout = o.trpcOption.Timeout
	}
	// use trpc password instead.
	if r.Password == "" {
		r.Password = parseTRPCPassword(o.trpcOption.CallOptions)
	}
	return nil
}

func newOptions(rawQuery string) (*redis.UniversalOptions, *pb.QueryOptions, error) {
	queryOptions, err := newQueryOptions(rawQuery)
	if err != nil {
		return nil, nil, err
	}
	// Create RedisOption from queryOptions.
	redisOption := newRedisOptions(queryOptions)
	return redisOption, queryOptions, nil
}

// newQueryOptions creates a new QueryOptions。
func newQueryOptions(rawQuery string) (*pb.QueryOptions, error) {
	// Parse the url query field.
	queryOptions := &pb.QueryOptions{}
	if err := codec.Unmarshal(codec.SerializationTypeGet, []byte(rawQuery), queryOptions); err != nil {
		return nil, err
	}
	return queryOptions, nil

}

// newRedisOptions creates a new redis.Options。
func newRedisOptions(q *pb.QueryOptions) *redis.UniversalOptions {
	return &redis.UniversalOptions{
		// redis.Options field：https://github.com/redis/go-redis/blob/v9.0.2/options.go#L425
		ClientName:      q.ClientName,
		MaxRetries:      int(q.MaxRetries),
		MinRetryBackoff: fixDuration(q.MinRetryBackoff),
		MaxRetryBackoff: fixDuration(q.MaxRetryBackoff),
		DialTimeout:     fixDuration(q.DialTimeout),
		ReadTimeout:     fixDuration(q.ReadTimeout),
		WriteTimeout:    fixDuration(q.WriteTimeout),
		PoolFIFO:        q.PoolFifo,
		PoolSize:        int(q.PoolSize),
		PoolTimeout:     fixDuration(q.PoolTimeout),
		MinIdleConns:    int(q.MinIdleConns),
		MaxIdleConns:    int(q.MaxIdleConns),
		ConnMaxIdleTime: fixDuration(q.ConnMaxIdleTime),
		ConnMaxLifetime: fixDuration(q.ConnMaxLifetime),
		// redis.ClusterOptions field：https://github.com/redis/go-redis/blob/v9.0.2/cluster.go#L215
		MaxRedirects:   int(q.MaxRedirects),
		ReadOnly:       q.ReadOnly,
		RouteByLatency: q.RouteByLatency,
		RouteRandomly:  q.RouteRandomly,
		// Expanded fields.
		MasterName:            q.MasterName,
		ContextTimeoutEnabled: q.ContextTimeoutEnabled,
	}
}

// parseURI is the filled uri field.
func parseURI(redisOptions *redis.UniversalOptions, u *url.URL, namespace string, isProxy bool) (string, error) {
	if u.User != nil {
		if redisOptions.Username == "" {
			redisOptions.Username = u.User.Username()
		}
		if redisOptions.Password == "" {
			redisOptions.Password, _ = u.User.Password()
		}
	}
	if redisOptions.DB == 0 {
		if strings.Index(u.Path, "/") == 0 {
			db, err := strconv.ParseInt(u.Path[1:], 10, 64)
			if err != nil {
				return "", errs.Wrapf(err, RetParseConfigFail, "u.Path '%s' invalid %v", u.Path, err)
			}
			redisOptions.DB = int(db)
		}
	}
	var err error
	if redisOptions.Addrs, err = parseScheme(u.Scheme, namespace, u.Host, isProxy); err != nil {
		return "", err
	}
	return u.Host, nil
}

// parseNamespace parses command space.
func parseNamespace(trpcOption *client.Options) string {
	selectorOptions := &selector.Options{}
	for _, o := range trpcOption.SelectOptions {
		o(selectorOptions)
	}
	return selectorOptions.Namespace
}

// parseTRPCPassword parses the trpc password field.
func parseTRPCPassword(callOptions []transport.RoundTripOption) string {
	roundTripOptions := &transport.RoundTripOptions{}
	for _, option := range callOptions {
		option(roundTripOptions)
	}
	// Add url decode to the password,
	// if the decoding fails, use the old url directly.
	password, _ := urlUnescape(roundTripOptions.Password)
	return password
}

// fixDuration fix time interval.
func fixDuration(ms int64) time.Duration {
	// special meaning
	if ms <= 0 {
		return time.Duration(ms)
	}
	return time.Duration(ms) * time.Millisecond
}

// calcPoolSize recalculates the pool size.
// 1 The default number is 10, and the v4 machine only has 40 connections,
// which does not conform to the actual production environment,
// https://github.com/redis/go-redis/blob/v9.0.2/options.go#L98
// 2 Numbers are based on the definition of online stress testing: v4 machine 1000, v8 machine 2000;
// 3 No adjustment phenomenon: the time consumption of a single request will expand (for example: 10ms -> 100ms),
// and a large number of requests are blocked in places waiting to obtain connections;
func calcPoolSize() int64 {
	cpuCore := runtime.GOMAXPROCS(0)
	if cpuCore <= 0 {
		return 0
	}
	return int64(cpuCore) * MaxConnectionsPreCPU
}

// urlUnescape is url decode。
func urlUnescape(rawURL string) (string, error) {
	decodeURL, err := url.QueryUnescape(rawURL)
	if err != nil {
		return rawURL, err
	}
	return decodeURL, nil
}
