package hbase

import (
	"fmt"
	"os"
	"path"

	"trpc.group/trpc-go/trpc-go/codec"
)

func init() {
	codec.Register("hbase", nil, DefaultClientCodec)
}

// default codec
var (
	DefaultClientCodec = &ClientCodec{}
)

// ClientCodec 解码hbase client请求
type ClientCodec struct{}

// Encode 设置hbase client请求的元数据
func (c *ClientCodec) Encode(msg codec.Msg, body []byte) (buffer []byte, err error) {
	//自身
	if msg.CallerServiceName() == "" {
		msg.WithCallerServiceName(fmt.Sprintf("trpc.hbase.%s.service", path.Base(os.Args[0])))
	}

	return nil, nil
}

// Decode 解析hbase client回包里的元数据
func (c *ClientCodec) Decode(msg codec.Msg, buffer []byte) (body []byte, err error) {

	return nil, nil
}
