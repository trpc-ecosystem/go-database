package redcron

import (
	"context"
	"fmt"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	redis "github.com/redis/go-redis/v9"
	goredis "trpc.group/trpc-go/trpc-database/goredis"
	trpc "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/filter"
)

var (
	testCtx context.Context
)

const (
	testTarget = "trpc.gamecenter.test.cron"
)

func init() {
	trpc.ServerConfigPath = "../trpc_go.yaml"
	trpc.NewServer()
	testCtx = trpc.BackgroundContext()
}

func TestRedCron_AddFunc(t *testing.T) {
	c := newMiniClient(t)
	t.Run("ok", func(t *testing.T) {
		redCron, err := New(c)
		if err != nil {
			t.Fatalf("New fail %v", err)
		}
		// Add filters.
		opts := []client.Option{client.WithFilter(
			func(ctx context.Context, req interface{}, rsp interface{}, f filter.ClientHandleFunc) error {
				err = f(ctx, req, rsp)
				t.Logf("%v filter rsp=%+v req=%+v err=%v", time.Now(), rsp, req, err)
				return err
			})}
		// Add timing tasks.
		id, err := redCron.AddFunc(testTarget,
			func(ctx context.Context, req interface{}, rsp interface{}) (err error) {
				t.Logf("%v AddFunc cron", time.Now())
				return nil
			}, WithTRPCOption(opts...))
		if err != nil {
			t.Fatalf("AddFunc fail %v", err)
		}
		t.Logf("%v id %d", time.Now(), id)
		// Start a scheduled task.
		redCron.Start()
		defer redCron.Stop()
		time.Sleep(5500 * time.Millisecond)
	})
	t.Run("disable", func(t *testing.T) {
		targets := []string{"cron://", "", "http"}
		for _, target := range targets {
			redCron, err := New(c)
			if err != nil {
				t.Fatalf("New fail %v", err)
			}
			// Add timing tasks.
			id, err := redCron.AddFunc(testTarget,
				func(ctx context.Context, req interface{}, rsp interface{}) (err error) {
					t.Logf("%v AddFunc cron", time.Now())
					return nil
				}, WithTRPCOption(client.WithTarget(target)))
			t.Logf("%d %+v", id, err)
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
