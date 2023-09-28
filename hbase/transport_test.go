package hbase

import (
	"context"
	"testing"

	"github.com/agiledragon/gomonkey"
	"github.com/tsuna/gohbase"
	"github.com/tsuna/gohbase/hrpc"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/transport"

	. "github.com/smartystreets/goconvey/convey"
)

func TestClientTransport_RoundTrip(t *testing.T) {
	ct := NewClientTransport()
	addr := "zk_addr?zookeeperRoot=/xx/xxx&zookeeperTimeout=1&regionLookupTimeout=1&regionReadTimeout=1"
	Convey("ClientTransport.RoundTrip", t, func() {
		ctx := context.Background()
		mock := gomonkey.ApplyFunc(gohbase.NewClient, func(addr string, option ...gohbase.Option) gohbase.Client {
			return &mockClient{}
		}).ApplyFunc(hrpc.NewGetStr, func(ctx context.Context, table, key string,
			options ...func(hrpc.Call) error) (*hrpc.Get, error) {
			return &hrpc.Get{}, nil
		}).ApplyFunc(hrpc.NewPutStr, func(ctx context.Context, table, key string,
			values map[string]map[string][]byte, options ...func(hrpc.Call) error) (*hrpc.Mutate, error) {
			return &hrpc.Mutate{}, nil
		})
		defer mock.Reset()
		Convey("otherCmd-err", func() {
			newCtx, msg := codec.WithCloneMessage(ctx)
			defer codec.PutBackMessage(msg)
			msg.WithClientReqHead(&Request{Command: "unsupported cmd"})
			msg.WithClientRspHead(&Response{})
			rsp, err := ct.RoundTrip(newCtx, nil, transport.WithDialAddress(addr))
			So(err, ShouldNotBeNil)
			So(rsp, ShouldBeNil)
		})

	})
}
