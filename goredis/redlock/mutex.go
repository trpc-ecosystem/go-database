package redlock

import (
	"context"
	"strconv"
	"time"

	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
	"trpc.group/trpc-go/trpc-database/goredis"
	"trpc.group/trpc-go/trpc-database/goredis/internal/script"
	"trpc.group/trpc-go/trpc-go/errs"
)

//go:generate mockgen -source=mutex.go -destination=mockredlock/mutex_mock.go -package=mockredlock

// redis lua script.
const (
	// checkLua Check if it is the same person who locks and releases.
	checkLua = `
local key=KEYS[1];
local value=ARGV[1];
local oldValue=redis.call('get', key);
if oldValue == false then
	return redis.error_reply(string.format("lock expired, key %q value %q", key, value));	
end
if value ~= oldValue then
	return redis.error_reply(string.format("lock robbed, key %q check %q old %q", key, value, oldValue));	
end
`
	// UnlockLua unlock lua script
	UnlockLua = checkLua + `
return redis.call('del', key);
`
	// ExtendLua renewals lua script，
	// Note: The cut-off time (pexpireat) is not used,
	// and the main concern is that the local time is inconsistent with the server.
	ExtendLua = checkLua + `
local at=ARGV[2];
return redis.call('pexpire', key, at);
`
	TTLLua = checkLua + `
return redis.call('pttl', key);
`
)

var (
	// UnlockScript consistent unlock script optimizer,
	// Execute and upload the script through Eval if the server does not exist,
	// and execute it through EvalSha if it exists
	UnlockScript = script.New(UnlockLua)
	// ExtendScript extends the script optimizer,
	// Execute and upload the script through Eval if the server does not exist,
	// and execute it through EvalSha if it exists
	ExtendScript = script.New(ExtendLua)
	// TTLScript lock remaining time to live script optimizer,
	// Execute and upload the script through Eval if the server does not exist,
	// and execute it through EvalSha if it exists
	TTLScript = script.New(TTLLua)
)

// Mutex is locked object.
type Mutex interface {
	// Unlock provides unlock function.
	Unlock(ctx context.Context) error
	// Extend is renewal.
	Extend(ctx context.Context, opts ...Option) error
	// TTL is the remaining validity period of the lock.
	TTL(ctx context.Context) (time.Duration, error)
}

type mutex struct {
	cmdable redis.Cmdable
	key     string
	value   string
	options *Options
}

func newMutex(cmdable redis.Cmdable, options *Options, key string) *mutex {
	return &mutex{
		cmdable: cmdable,
		key:     key,
		value:   uuid.New().String(),
		options: options,
	}
}

// Unlock provides unlock function.
func (m *mutex) Unlock(ctx context.Context) error {
	// Lua scripts will only be completed on the master node.
	_, err := UnlockScript.RunEx(ctx, m.cmdable, []string{m.key}, m.value).Result()
	return goredis.TRPCErr(err)
}

// Extend is renewal, note: the lock must exist and not expired to allow renewal.
func (m *mutex) Extend(ctx context.Context, opts ...Option) error {
	options := m.options.clone()
	for _, o := range opts {
		o(options)
	}
	d := strconv.FormatInt(int64(options.extendInterval/time.Millisecond), 10)
	v, err := ExtendScript.RunEx(ctx, m.cmdable, []string{m.key}, m.value, d).Int64()
	if err != nil {
		return goredis.TRPCErr(err)
	}
	if v != 1 {
		return errs.Newf(goredis.RetLockExtend, "pexpire fail %d", v)
	}
	return nil
}

// TTL locks remaining validity period.
func (m *mutex) TTL(ctx context.Context) (time.Duration, error) {
	v, err := TTLScript.RunEx(ctx, m.cmdable, []string{m.key}, m.value).Int64()
	if err != nil {
		return 0, goredis.TRPCErr(err)
	}
	d := time.Duration(v) * time.Millisecond
	return d, nil
}
