// Package script optimizes lua script execution.
package script

import (
	"context"

	redis "github.com/redis/go-redis/v9"
)

// Script is script optimization class.
type Script struct {
	*redis.Script
}

// New creates a new redis script optimizer object.
func New(src string) *Script {
	return &Script{
		Script: redis.NewScript(src),
	}
}

// RunEx executes the lua script. If the script does not exist on the server,
// it will be executed and uploaded through Eval, and if it exists, it will be executed through EvalSha
// However: in pipeline mode, if redis.Script.Run executes EvalSha and reports that the script does not exist,
// it will not be able to call Eval to retry, so optimization is prohibited in pipeline mode.
func (s *Script) RunEx(ctx context.Context, c redis.Scripter, keys []string, args ...interface{}) *redis.Cmd {
	// redis.Script.Run If the script does not exist after executing EvalSha,
	// it will not be able to call Eval to retry,
	// There are compatibility issues in pipeline mode, so disable it.
	if _, ok := c.(*redis.Pipeline); ok {
		return s.Eval(ctx, c, keys, args...)
	}
	return s.Run(ctx, c, keys, args...)
}
