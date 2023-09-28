package hbase

import (
	"context"
	"os"
	"testing"

	"github.com/agiledragon/gomonkey"
	"trpc.group/trpc-go/trpc-go"

	. "github.com/smartystreets/goconvey/convey"
)

func TestClientCodec_Decode(t *testing.T) {
	c := &ClientCodec{}
	Convey("ClientCodec.Decode", t, func() {
		msg := trpc.Message(context.Background())
		buf, err := c.Decode(msg, nil)
		So(err, ShouldBeNil)
		So(buf, ShouldBeNil)
	})
}

func TestClientCodec_Encode(t *testing.T) {
	c := &ClientCodec{}
	Convey("ClientCodec.Encode", t, func() {
		mock := gomonkey.ApplyGlobalVar(&os.Args, []string{"business"})
		defer mock.Reset()
		msg := trpc.Message(context.Background())
		buf, err := c.Encode(msg, nil)
		So(err, ShouldBeNil)
		So(buf, ShouldBeNil)
		So(msg.CallerServiceName(), ShouldEqual, "trpc.hbase.business.service")
	})
}
