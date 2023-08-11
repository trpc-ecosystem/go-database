// Package redcas realizes CAS extension function.
package redcas

import (
	"context"
	"fmt"
	"math"
	"time"

	redis "github.com/redis/go-redis/v9"
)

const (
	ForceSetCAS = -1            // Mandatory modification cas.
	MaxCAS      = math.MaxInt32 // Lua only supports float64,
	// not uint64, to prevent overflow.
)

// Set redis set function, cas function description: ForceSetCAS(-1) force modification,
// 0 means set if it does not exist, duration 0 means no expiration time.
// EvalSha has compatibility: when executing the SCRIPT LOAD command on a single node,
// it is not guaranteed to save the Lua script to other nodes,
// See：https://help.aliyun.com/document_detail/92942.html#section-8f7-qgv-dlv
func Set(ctx context.Context, c redis.Cmdable, key string, value interface{}, cas int64,
	duration time.Duration) *SetCmd {
	cmd := &SetCmd{Cmder: &redis.StatusCmd{}}
	newCAS := NextCAS(cas)
	valueBytes, err := Marshal(value)
	if err != nil {
		cmd.SetErr(err)
		return cmd
	}
	raw := Encode(valueBytes, newCAS)
	if ForceSetCAS == cas {
		cmd.Cmder = c.Set(ctx, key, raw, duration)
	} else {
		cmd.Cmder = SetScript.RunEx(ctx, c, []string{key}, raw, cas, duration.Milliseconds())
	}
	return cmd
}

// Get redis Get, redis.Nil indicates that the key does not exist.
func Get(ctx context.Context, c redis.Cmdable, key string) *GetCmd {
	cmd := &GetCmd{StringCmd: c.Get(ctx, key)}
	return cmd
}

// HSet redis hSet function, CAS ForceSetCAS(-1) force reset, 0 means set without.
func HSet(ctx context.Context, c redis.Cmdable, key, field string, value interface{}, cas int64) *HSetCmd {
	cmd := &HSetCmd{Cmder: &redis.StatusCmd{}, cas: cas}
	newCAS := NextCAS(cas)
	valueBytes, err := Marshal(value)
	if err != nil {
		cmd.SetErr(err)
		return cmd
	}
	raw := Encode(valueBytes, newCAS)
	if ForceSetCAS == cas {
		cmd.Cmder = c.HSet(ctx, key, field, raw)
	} else {
		cmd.Cmder = HSetScript.RunEx(ctx, c, []string{key}, field, raw, cas)
	}
	return cmd
}

// MSet redis, MSet functions are implemented through pipelines,
// compatible with all versions of redis, it is recommended to use the cluster version.
// Lua script version reference MSetWithEval.
func MSet(ctx context.Context, c redis.Cmdable, values ...*S) *MSetCmd {
	cmd := &MSetCmd{
		Cmd:      &redis.Cmd{},
		commands: make([]*SetCmd, len(values)),
	}
	if len(values) == 0 {
		return cmd
	}
	_, err := c.Pipelined(ctx, func(pipeliner redis.Pipeliner) error {
		for i, v := range values {
			cmd.commands[i] = Set(ctx, pipeliner, v.Key, v.Value, v.CAS, v.Duration)
		}
		return nil
	})
	cmd.SetErr(err)
	return cmd
}

// MSetWithEval redis MSet lua script version, strong performance,
// not compatible with the cluster version key across slots, etc., it is recommended to use the redis standard version.
// Key cross-slot can also be adjusted by adding hash tags,
// specifically: http://www.redis.cn/commands/cluster-keyslot.html,
func MSetWithEval(ctx context.Context, c redis.Cmdable, values ...*S) *MSetCmd {
	cmd := &MSetCmd{Cmd: &redis.Cmd{}}
	if len(values) == 0 {
		return cmd
	}
	keys := make([]string, len(values))
	args := make([]interface{}, len(values)*2)
	for i, v := range values {
		newCAS := NextCAS(v.CAS)
		mv, err := Marshal(v.Value)
		if err != nil {
			cmd.SetErr(fmt.Errorf("key %s marshal fail %w", v.Key, err))
			return cmd
		}
		newValue := Encode(mv, newCAS)
		keys[i] = v.Key
		args[i*2] = newValue
		args[i*2+1] = v.CAS
	}
	cmd.Cmd = MSetScript.RunEx(ctx, c, keys, args...)
	return cmd
}

// HGet redis hget, redis.Nil indicates that the key does not exist.
func HGet(ctx context.Context, c redis.Cmdable, key, field string) *GetCmd {
	cmd := &GetCmd{StringCmd: c.HGet(ctx, key, field)}
	return cmd
}

// S is Set/HSet value structure.
type S struct {
	Key      string        // MSet is the key field, HMSet is the field.
	Value    interface{}   // Setting value.
	CAS      int64         // cas ForceSetCAS(-1) mandatory modification, 0 means set if it does not exist.
	Duration time.Duration // Validity period, only valid for MSetWithPipelined.
}

// MGet redis MGet
func MGet(ctx context.Context, c redis.Cmdable, keys ...string) *MGetCmd {
	cmd := &MGetCmd{SliceCmd: c.MGet(ctx, keys...)}
	return cmd
}

// HMSet is redis set function, cas function description:
// ForceSetCAS(-1) force modification, 0 means set if it does not exist, duration 0 means no expiration time.
func HMSet(ctx context.Context, c redis.Cmdable, key string, values ...*S) *MSetCmd {
	cmd := &MSetCmd{Cmd: &redis.Cmd{}}
	args := make([]interface{}, len(values)*3)
	for i, v := range values {
		newCAS := NextCAS(v.CAS)
		mv, err := Marshal(v.Value)
		if err != nil {
			cmd.SetErr(fmt.Errorf("field %s marshal fail %w", v.Key, err))
			return cmd
		}
		newValue := Encode(mv, newCAS)
		args[i*3] = v.Key
		args[i*3+1] = newValue
		args[i*3+2] = v.CAS
	}
	cmd.Cmd = HMSetScript.RunEx(ctx, c, []string{key}, args...)
	return cmd
}

// HMGet redis HMGet
func HMGet(ctx context.Context, c redis.Cmdable, key string, fields ...string) *MGetCmd {
	cmd := &MGetCmd{SliceCmd: c.HMGet(ctx, key, fields...)}
	return cmd
}

// NextCAS calculates the next cas.
func NextCAS(cas int64) int64 {
	cas++
	if cas <= 0 || cas >= MaxCAS {
		cas = 1
	}
	return cas
}
