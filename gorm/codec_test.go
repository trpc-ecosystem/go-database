package gorm

import (
	"context"
	"testing"

	"trpc.group/trpc-go/trpc-go/codec"

	. "github.com/smartystreets/goconvey/convey"
)

func TestUnit_ClientCodec_Encode_P0(t *testing.T) {
	Convey("TestUnit_ClientCodec_Encode_P0", t, func() {
		cc := new(ClientCodec)
		_, msg := codec.WithNewMessage(context.Background())
		res, err := cc.Encode(msg, []byte{})
		So(err, ShouldBeNil)
		So(res, ShouldBeNil)
		So(msg.CallerServiceName(), ShouldNotEqual, "")
	})
}

func TestUnit_ClientCodec_Decode_P0(t *testing.T) {
	Convey("TestUnit_ClientCodec_Decode_P0", t, func() {
		cc := new(ClientCodec)
		_, msg := codec.WithNewMessage(context.Background())
		res, err := cc.Decode(msg, []byte{})
		So(res, ShouldBeNil)
		So(err, ShouldBeNil)
	})
}
