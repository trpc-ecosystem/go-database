package kafka

import (
	"context"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"trpc.group/trpc-go/trpc-go/codec"
)

func TestClientEncode(t *testing.T) {
	Convey("TestEncode", t, func() {
		cc := new(ClientCodec)
		_, msg := codec.WithNewMessage(context.Background())
		res, err := cc.Encode(msg, []byte{})
		So(err, ShouldBeNil)
		So(res, ShouldBeNil)
	})
}

func TestClientDecode(t *testing.T) {
	Convey("TestDecode", t, func() {
		cc := new(ClientCodec)
		_, msg := codec.WithNewMessage(context.Background())
		res, err := cc.Decode(msg, []byte{})
		So(res, ShouldBeNil)
		So(err, ShouldBeNil)
	})
}

func TestServerEncode(t *testing.T) {
	Convey("TestEncode", t, func() {
		sc := new(ServerCodec)
		_, msg := codec.WithNewMessage(context.Background())
		res, err := sc.Encode(msg, []byte{})
		So(err, ShouldBeNil)
		So(res, ShouldBeNil)
	})
}

func TestServerDecode(t *testing.T) {
	Convey("TestDecode", t, func() {
		sc := new(ServerCodec)
		_, msg := codec.WithNewMessage(context.Background())
		res, err := sc.Decode(msg, []byte{})
		So(res, ShouldBeNil)
		So(err, ShouldBeNil)
	})
}
