package goredis

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	redis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	trpc "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
)

var (
	testCtx context.Context
)

func init() {
	testing.Init()
	trpc.NewServer()
	testCtx = trpc.BackgroundContext()
}

func TestSet(t *testing.T) {
	t.Run("string", func(t *testing.T) {
		key := "k1"
		value := "v1"
		doGetSetDel(t, key, value)
	})
	t.Run("bytes", func(t *testing.T) {
		key := "k2"
		value := "1\xaa1"
		doGetSetDel(t, key, value)
	})
	t.Run("text", func(t *testing.T) {
		key := "k3"
		value := "12345678901234567890123456789012345678901234567890" +
			"1234567890123456789012345678901234567890123456789012345678901234567890"
		doGetSetDel(t, key, value)
	})
}

func TestPipeline(t *testing.T) {
	c := newMiniClient(t)
	key3 := "k4"
	c.Del(testCtx, key3)
	cmdList := make([]*redis.StatusCmd, 2)
	getCmd := &redis.StringCmd{}
	_, err := c.Pipelined(testCtx,
		func(pipe redis.Pipeliner) error {
			cmdList[0] = pipe.Set(testCtx, "k1", "v1", 0)
			cmdList[1] = pipe.Set(testCtx, "k2", "v2", 0)
			getCmd = pipe.Get(testCtx, "k2")
			return nil
		})
	if err != nil {
		t.Fatalf("Pipelined fail err=[%v]", err)
	} else {
		t.Logf("Pipelined success")
	}
	for _, cmd := range cmdList {
		if ok, cmdErr := cmd.Result(); err != nil {
			t.Fatalf("cmd fail err=[%v] cmd=[%v]", cmdErr, cmd)
		} else {
			t.Logf("cmd success ok=[%v] cmd=[%v]", ok, cmd)
		}
	}
	v3, err := getCmd.Result()
	t.Logf("Get fail err=[%v] v1=[%v]", err, v3)
}

func Test_pipelineRspBody(t *testing.T) {
	stringCmd := &redis.StringCmd{}
	stringCmd.SetErr(errors.New("test"))
	cmds := []redis.Cmder{stringCmd}
	_ = pipelineRspBody(cmds)
}

func Test_m007Err(t *testing.T) {
	t.Run("cas mismatch", func(t *testing.T) {
		err := errors.New("cas mismatch")
		_ = TRPCErr(err)
	})
	t.Run("cmd fail", func(t *testing.T) {
		err := errors.New("fail")
		_ = TRPCErr(err)
	})
}

func Test_pipeline(t *testing.T) {
	c := newMiniClient(t)
	invalidScript := "'set', 'key1', 'value1'); return 'ok'" // truncated Lua which will return a "compile error" error.
	t.Run("pipeline", func(t *testing.T) {
		validScript1 := "redis.call('set', 'key1', 'value1'); return 'ok'" // correct lua
		var results []*redis.Cmd
		_, _ = c.Pipelined(testCtx, func(pipeliner redis.Pipeliner) error {
			results = []*redis.Cmd{
				pipeliner.Eval(testCtx, validScript1, []string{"1"}),
				pipeliner.Eval(testCtx, invalidScript, []string{"1"}),
			}
			return nil
		})
		if err := results[0].Err(); err != nil {
			t.Fatalf("pipeline fail %v", err)
		}
	})
	t.Run("eval", func(t *testing.T) {
		if err := c.Eval(testCtx, invalidScript, []string{"1"}).Err(); err != nil {
			t.Logf("eval %v", err)
		}
	})

}

func TestWithCommonMetaData(t *testing.T) {
	msg := codec.Message(context.Background())
	cases := []struct {
		Name string
		Msg  codec.Msg
		Want string
	}{
		{"ok", msg, "[database].localhost:6379"},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			withCommonMetaData(c.Msg, "[database]", "localhost:6379")
			const appKey = "overrideCalleeApp"
			const serverKey = "overrideCalleeServer"
			meta := msg.CommonMeta()
			res := meta[appKey].(string) + "." + meta[serverKey].(string)
			assert.Equal(t, res, c.Want)
		})
	}
}

func doGetSetDel(t *testing.T, k, v string) {
	start := time.Now()
	c := newMiniClient(t)
	r1, err := c.Set(testCtx, k, v, 0).Result()
	if err != nil {
		t.Fatalf("Set fail err=[%v] cost=[%v]", err, time.Since(start))
	} else {
		t.Logf("Set ok r1=[%v]", r1)
	}
	v1, err := c.Get(testCtx, k).Result()
	if err != nil {
		t.Fatalf("Get fail err=[%v] cost=[%v]", err, time.Since(start))
	} else {
		t.Logf("Get ok v1=[%v]", v1)
	}
	del, err := c.Del(testCtx, k).Result()
	if err != nil {
		t.Fatalf("Del fail err=[%v] cost=[%v]", err, time.Since(start))
	} else {
		t.Logf("Del ok del=[%v]", del)
	}
	ctx, redisMsg := WithMessage(testCtx)
	redisMsg.Options = append(redisMsg.Options, client.WithTimeout(-1))
	v1, err = c.Get(ctx, k).Result()
	t.Logf("Get fail err=[%v] v1=[%v] cost=[%v]", err, v1, time.Since(start))
}

// newMiniClient client in memory.
func newMiniClient(t *testing.T) redis.UniversalClient {
	s := miniredis.RunT(t)
	target := fmt.Sprintf("redis://%s/0", s.Addr())
	c, err := New("trpc.gamecenter.test.redis", client.WithTarget(target))
	if err != nil {
		t.Fatalf("%+v", err)
	}
	return c
}

func TestDialHook(t *testing.T) {
	target := "polaris://trpc.trpcdatabase.goredis.redis/0?is_proxy=true"
	if _, err := New("trpc.gamecenter.test.redis", client.WithTarget(target)); err != nil {
		if strings.Contains(errs.Msg(err), "polaris") {
			t.Logf("polaris not exist")
			return
		}
		t.Fatalf("%+v", err)
	}
}
