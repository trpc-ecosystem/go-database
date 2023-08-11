package options

import (
	"testing"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/naming/selector"
)

func init() {
	testing.Init()
	selector.Register("redis", selector.NewIPSelector())
	selector.Register("cron", selector.NewIPSelector())
	trpc.ServerConfigPath = "../../trpc_go.yaml"
	trpc.NewServer()
}

func Test_calcPoolSize(t *testing.T) {
	t.Run("", func(t *testing.T) {
		poolSize := calcPoolSize()
		if poolSize <= 0 {
			t.Fatalf("calcPoolSize fail")
		}
		t.Logf("poolSize %d", poolSize)
	})
}

func TestNew(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		tOptions := &client.Options{
			Target: "redis://user:password@127.0.0.1:6379/0?is_proxy=false&min_idle_conns=10&conn_max_lifetime=100",
		}
		options, err := New(tOptions)
		if err != nil {
			t.Fatalf("%+v", err)
		}
		t.Logf("%+v", options)
	})
}
