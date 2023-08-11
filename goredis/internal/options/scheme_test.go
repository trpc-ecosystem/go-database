package options

import (
	"testing"
)

func Test_parseScheme(t *testing.T) {
	f := func(t *testing.T, scheme, namespace, host string, isProxy bool) {
		addrs, err := parseScheme(scheme, namespace, host, isProxy)
		if err != nil {
			t.Fatalf("%+v", err)
		}
		if len(addrs) == 0 {
			t.Fatalf("service empty")
		}
		t.Logf("%+v", addrs)
	}
	const service = "trpc.gamecenter.trpcproxy.TRPCProxy"
	t.Run("redis", func(t *testing.T) {
		f(t, "redis", "Production", "127.0.0.1:6379", false)
	})
	t.Run("polaris", func(t *testing.T) {
		addrs, err := parseScheme("polaris", "Production", service, false)
		t.Logf("%+v %v", addrs, err)
	})
	t.Run("proxy", func(t *testing.T) {
		f(t, "polaris", "Production", service, true)
	})
}

func Test_parseSchemeCover(t *testing.T) {
	t.Run("scheme", func(t *testing.T) {
		_, _ = parseScheme("tcp", "", "", false)
	})
	t.Run("addr", func(t *testing.T) {
		_, _ = parseScheme("ip", "", "", false)
	})
}
