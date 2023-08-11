package redcas

import (
	redis "github.com/redis/go-redis/v9"
	goredis "trpc.group/trpc-go/trpc-database/goredis"
)

// SetCmd is set result parser.
type SetCmd struct {
	redis.Cmder
}

// Check if SetCmd fully implements redis.Cmder.
var _ redis.Cmder = (*SetCmd)(nil)

// Err is error message.
func (cmd *SetCmd) Err() error {
	return goredis.TRPCErr(cmd.Cmder.Err())
}

// Val returns results.
func (cmd *SetCmd) Val() string {
	switch v := cmd.Cmder.(type) {
	case *redis.StatusCmd:
		return v.Val()
	case *redis.Cmd:
		return ""
	default:
		return ""
	}
}

// Result returns results.
func (cmd *SetCmd) Result() (string, error) {
	return cmd.Val(), cmd.Err()
}

// HSetCmd HSet is result class.
type HSetCmd struct {
	redis.Cmder
	cas int64
}

// Check if HSetCmd fully implements redis.Cmder.
var _ redis.Cmder = (*HSetCmd)(nil)

// Err is error message.
func (cmd *HSetCmd) Err() error {
	return goredis.TRPCErr(cmd.Cmder.Err())
}

// Val is hset value.
func (cmd *HSetCmd) Val() int64 {
	switch v := cmd.Cmder.(type) {
	case *redis.IntCmd:
		return v.Val()
	case *redis.Cmd:
		var createCount int64 = 1
		if cmd.cas > 0 {
			createCount = 0
		}
		return createCount
	default:
		return 0
	}
}

// Result is hset result.
func (cmd *HSetCmd) Result() (int64, error) {
	return cmd.Val(), cmd.Err()
}

// GetCmd get command.
type GetCmd struct {
	*redis.StringCmd
}

var _ redis.Cmder = (*GetCmd)(nil)

// Val get value.
func (cmd *GetCmd) Val() (string, int64) {
	bs, cas, _ := cmd.Bytes()
	return BytesToString(bs), cas
}

// Result gets results.
func (cmd *GetCmd) Result() (string, int64, error) {
	bs, cas, err := cmd.Bytes()
	if err != nil {
		return "", 0, err
	}
	s := BytesToString(bs)
	return s, cas, err
}

// Bytes gets the result converted to []byte.
func (cmd *GetCmd) Bytes() ([]byte, int64, error) {
	bs, err := cmd.StringCmd.Bytes()
	if err != nil {
		return nil, 0, err
	}
	return Decode(bs)
}

// Unmarshal is type parsing for commonly used types.
func (cmd *GetCmd) Unmarshal(message interface{}) (int64, error) {
	bs, cas, err := cmd.Bytes()
	if err != nil {
		return cas, err
	}
	return cas, Unmarshal(bs, message)
}

// MSetCmd is mset result parser.
type MSetCmd struct {
	*redis.Cmd
	commands []*SetCmd
}

// Check if SetCmd fully implements redis.Cmder.
var _ redis.Cmder = (*MSetCmd)(nil)

// Err is error massage.
func (cmd *MSetCmd) Err() error {
	return goredis.TRPCErr(cmd.Cmd.Err())
}

// Result returns results.
func (cmd *MSetCmd) Result() ([]error, error) {
	errs := make([]error, len(cmd.commands))
	for i, command := range cmd.commands {
		errs[i] = goredis.TRPCErr(command.Err())
	}
	return errs, cmd.Err()
}

// MGetCmd MGet is result class.
type MGetCmd struct {
	*redis.SliceCmd
}

// Check if MGetCmd fully implements redis.Cmder.
var _ redis.Cmder = (*MGetCmd)(nil)

// Unmarshal is type parsing for commonly used types.
func (cmd *MGetCmd) Unmarshal(i int64, message interface{}) (int64, error) {
	results, err := cmd.SliceCmd.Result()
	if err != nil {
		return 0, err
	}
	if i >= int64(len(results)) {
		return 0, goredis.ErrParamInvalid
	}
	interfaceResult := results[i]
	if interfaceResult == nil {
		return 0, redis.Nil
	}
	stringResult, ok := interfaceResult.(string)
	if !ok {
		return 0, goredis.ErrTypeMismatch
	}
	bs, cas, err := Decode([]byte(stringResult))
	if err != nil {
		return 0, err
	}
	return cas, Unmarshal(bs, message)
}
