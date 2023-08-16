package redcas

import (
	"context"
	"fmt"
	"testing"

	miniredis "github.com/alicebob/miniredis/v2"
	redis "github.com/redis/go-redis/v9"
	goredis "trpc.group/trpc-go/trpc-database/goredis"
	pb "trpc.group/trpc-go/trpc-database/goredis/internal/proto"
	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/errs"
)

var (
	testCtx context.Context
)

func init() {
	trpc.ServerConfigPath = "../trpc_go.yaml"
	trpc.NewServer()
	testCtx = trpc.BackgroundContext()
}

func TestSet(t *testing.T) {
	c := newMiniClient(t)
	key := "goredis_TestSet_key"
	outString := ""
	type args struct {
		in  interface{}
		out interface{}
	}
	tests := []struct {
		name string
		args args
		want *SetCmd
	}{
		{
			name: "bytes",
			args: args{
				in:  []byte("bytes"),
				out: &[]byte{},
			},
		},
		{
			name: "string",
			args: args{
				in:  "string",
				out: &outString,
			},
		},
		{
			name: "proto",
			args: args{
				in: &pb.QueryOptions{
					IsProxy:      true,
					PoolSize:     100,
					MinIdleConns: 100,
				},
				out: &pb.QueryOptions{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, err := Set(testCtx, c, key, tt.args.in, ForceSetCAS, 0).Result()
			if err != nil {
				t.Fatalf("Set -1 fail Err=[%v]", err)
			}
			t.Logf("Set -1 ok ok=[%v]", ok)
			cas, err := Get(testCtx, c, key).Unmarshal(tt.args.out)
			if err != nil {
				t.Fatalf("Get -1 fail Err=[%v]", err)
			}
			t.Logf("Get -1 ok Value=[%v] CAS=[%v]", MarshalJSONString(tt.args.out), cas)
			ok, err = Set(testCtx, c, key, tt.args.in, cas, 0).Result()
			if err != nil {
				t.Fatalf("Set -1 fail name=[%v] Err=[%v]", tt.name, err)
			}
			t.Logf("Set 0 ok name=[%v] ok=[%v]", tt.name, ok)
			cas, err = Get(testCtx, c, key).Unmarshal(tt.args.out)
			if err != nil {
				t.Fatalf("Get 0 fail name=[%v] Err=[%v]", tt.name, err)
			}
			t.Logf("Get 0 ok name=[%v] out1=[%v] CAS=[%v]", tt.name, MarshalJSONString(tt.args.out), cas)
		})
	}
}

func TestHSet(t *testing.T) {
	c := newMiniClient(t)
	key := "goredis_TestHSet_key"
	field := "field"
	outString := ""
	type args struct {
		in  interface{}
		out interface{}
	}
	tests := []struct {
		name string
		args args
		want *SetCmd
	}{
		{
			name: "bytes",
			args: args{
				in:  []byte("bytes"),
				out: &[]byte{},
			},
		},
		{
			name: "string",
			args: args{
				in:  "string",
				out: &outString,
			},
		},
		{
			name: "proto",
			args: args{
				in: &pb.QueryOptions{
					IsProxy:      true,
					PoolSize:     100,
					MinIdleConns: 100,
				},
				out: &pb.QueryOptions{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, err := HSet(testCtx, c, key, field, tt.args.in, -1).Result()
			if err != nil {
				t.Fatalf("HSet -1 fail Err=[%v]", err)
			}
			t.Logf("HSet -1 ok ok=[%v]", ok)
			cas, err := HGet(testCtx, c, key, field).Unmarshal(tt.args.out)
			if err != nil {
				t.Fatalf("HGet -1 fail Err=[%v]", err)
			}
			t.Logf("HGet -1 ok Value=[%v] CAS=[%v]", MarshalJSONString(tt.args.out), cas)
			ok, err = HSet(testCtx, c, key, field, tt.args.in, cas).Result()
			if err != nil {
				t.Fatalf("HSet -1 fail name=[%v] Err=[%v]", tt.name, err)
			}
			t.Logf("HSet 0 ok name=[%v] ok=[%v]", tt.name, ok)
			cas, err = HGet(testCtx, c, key, field).Unmarshal(tt.args.out)
			if err != nil {
				t.Fatalf("HGet 0 fail name=[%v] Err=[%v]", tt.name, err)
			}
			t.Logf("HGet 0 ok name=[%v] out1=[%v] CAS=[%v]", tt.name, MarshalJSONString(tt.args.out), cas)
		})
	}
}

func TestGetNil(t *testing.T) {
	c := newMiniClient(t)
	key := "goredis_TestSet_nil_key"
	c.Del(testCtx, key)
	value, cas, err := Get(testCtx, c, key).Result()
	t.Logf("Get fail Err=[%v] Value=[%v] CAS=[%v]", err, value, cas)
}

func TestHMGet(t *testing.T) {
	key := "h1"
	field1 := "f1"
	field2 := "f2"
	fields := []string{field1, field2}
	c := newMiniClient(t)
	t.Run("Del", func(t *testing.T) {
		if _, err := c.Del(testCtx, key).Result(); err != nil {
			t.Fatalf("del fail %v", err)
		}
	})
	values := []*S{
		{Key: field1, Value: "v1", CAS: ForceSetCAS},
		{Key: field2, Value: "v2", CAS: 0},
	}
	fMSet := func(t *testing.T) {
		if _, err := HMSet(testCtx, c, key, values...).Result(); err != nil {
			t.Fatalf("MSet fail %v", err)
		}
	}
	fMGet := func(t *testing.T) {
		cmd := HMGet(testCtx, c, key, fields...)
		for i, field := range fields {
			v := ""
			cas, err := cmd.Unmarshal(int64(i), &v)
			values[i].CAS = cas
			if err != nil {
				if err == redis.Nil {
					t.Logf("HMGet %s redis.Nil %d", field, cas)
					continue
				}
				t.Fatalf("HMGet %s fail %v", field, err)
			}
			t.Logf("HMGet %s %s %d", field, v, cas)
		}
	}
	t.Run("HMSet init", fMSet)
	t.Run("HMGet init", fMGet)
	values[0].Value = "v11"
	values[1].Value = "v21"
	t.Run("HMSet modify", fMSet)
	t.Run("HMGet modify", fMGet)
}

func TestMSetEval(t *testing.T) {
	c := newMiniClient(t)
	key1 := "k1"
	key2 := "k2{k1}" // Evacuate two keys in the same slot through hash tags.
	keys := []string{key1, key2}
	t.Run("Del", func(t *testing.T) {
		if _, err := c.Del(testCtx, keys...).Result(); err != nil {
			t.Fatalf("del fail %v", err)
		}
	})
	values := []*S{
		{Key: key1, Value: "v1", CAS: ForceSetCAS},
		{Key: key2, Value: "v2", CAS: 0},
	}
	fMSet := func(t *testing.T) {
		if _, err := MSetWithEval(testCtx, c, values...).Result(); err != nil {
			t.Fatalf("MSet fail %v", err)
		}
	}
	fMGet := func(t *testing.T) {
		cmd := MGet(testCtx, c, keys...)
		for i, key := range keys {
			v := ""
			cas, err := cmd.Unmarshal(int64(i), &v)
			values[i].CAS = cas
			if err != nil {
				if err == redis.Nil {
					t.Logf("MGet %s redis.Nil %d", key, cas)
					continue
				}
				t.Fatalf("MGet %s fail %v", keys, err)
			}
			t.Logf("MGet %s %s %d", key, v, cas)
		}
	}
	t.Run("MSet init", fMSet)
	t.Run("MGet init", fMGet)
	values[0].Value = "v12"
	values[1].Value = "v22"
	t.Run("MSet modify", fMSet)
	t.Run("MGet modify", fMGet)
}

func TestMSetPipelined(t *testing.T) {
	key1 := "pipelined_set_k1"
	key2 := "pipelined_set_k2"
	keys := []string{key1, key2}
	c := newMiniClient(t)
	t.Run("Del", func(t *testing.T) {
		if _, err := c.Del(testCtx, keys...).Result(); err != nil {
			t.Fatalf("del fail %v", err)
		}
	})
	values := []*S{
		{Key: key1, Value: "v1", CAS: ForceSetCAS},
		{Key: key2, Value: "v2", CAS: 0},
	}
	fMSet := func(t *testing.T) {
		if _, err := MSet(testCtx, c, values...).Result(); err != nil {
			t.Fatalf("MSet fail %v", err)
		}
	}
	fMGet := func(t *testing.T) {
		cmd := MGet(testCtx, c, keys...)
		for i, key := range keys {
			v := ""
			cas, err := cmd.Unmarshal(int64(i), &v)
			values[i].CAS = cas
			if err != nil {
				if err == redis.Nil {
					t.Logf("MGet %s redis.Nil %d", key, cas)
					continue
				}
				t.Fatalf("MGet %s fail %v", keys, err)
			}
			t.Logf("MGet %s %s %d", key, v, cas)
		}
	}
	t.Run("MSet init", fMSet)
	t.Run("MGet init", fMGet)
	values[0].Value = "v13"
	values[1].Value = "v23"
	t.Run("MSet modify", fMSet)
	t.Run("MGet modify", fMGet)
}

func TestCasMismatch(t *testing.T) {
	c := newMiniClient(t)
	t.Run("set", func(t *testing.T) {
		key := "cas_set_key1"
		value := "cas_set_value1"
		if _, err := Set(testCtx, c, key, value, ForceSetCAS, 0).Result(); err != nil {
			t.Fatalf("Set fail %v", err)
		}
		_, err := Set(testCtx, c, key, value, 100, 0).Result()
		if errs.Code(err) != goredis.RetCASMismatch {
			t.Fatalf("Set fail %v", err)
		}
	})
	t.Run("hset", func(t *testing.T) {
		key := "cas_hset_key1"
		field := "cas_hset_field1"
		value := "cas_hset_value1"
		if _, err := HSet(testCtx, c, key, field, value, ForceSetCAS).Result(); err != nil {
			t.Fatalf("HSet fail %v", err)
		}
		_, err := HSet(testCtx, c, key, field, value, 100).Result()
		if errs.Code(err) != goredis.RetCASMismatch {
			t.Fatalf("HSet fail %v", err)
		}
	})
	t.Run("mset", func(t *testing.T) {
		value := &S{
			Key:   "cas_mset_key1",
			Value: "cas_mset_value1",
			CAS:   ForceSetCAS,
		}
		if _, err := MSet(testCtx, c, value).Result(); err != nil {
			t.Fatalf("MSet fail %v", err)
		}
		value.CAS = 100
		valueErrs, err := MSet(testCtx, c, value).Result()
		if errs.Code(err) != goredis.RetCASMismatch {
			t.Fatalf("MSet fail %v", err)
		}
		for _, err := range valueErrs {
			if errs.Code(err) != goredis.RetCASMismatch {
				t.Fatalf("MSet fail %v", err)
			}
		}
	})
	t.Run("HMSet", func(t *testing.T) {
		key := "cas_hmset_key1"
		value := &S{
			Key:   "cas_hmset_field1",
			Value: "cas_hmset_value1",
			CAS:   ForceSetCAS,
		}
		if _, err := HMSet(testCtx, c, key, value).Result(); err != nil {
			t.Fatalf("MSet fail %v", err)
		}
		value.CAS = 100
		valueErrs, err := HMSet(testCtx, c, key, value).Result()
		if errs.Code(err) != goredis.RetCASMismatch {
			t.Fatalf("MSet fail %v", err)
		}
		for _, err := range valueErrs {
			if errs.Code(err) != goredis.RetCASMismatch {
				t.Fatalf("MSet fail %v", err)
			}
		}
	})
}

// newMiniClient creates a new memory version of redis.
func newMiniClient(t *testing.T) redis.UniversalClient {
	s := miniredis.RunT(t)
	target := fmt.Sprintf("redis://%s/0", s.Addr())
	c, err := goredis.New("trpc.gamecenter.test.redis", client.WithTarget(target))
	if err != nil {
		t.Fatalf("%+v", err)
	}
	return c
}
