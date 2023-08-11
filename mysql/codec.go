package mysql

import (
	"fmt"
	"os"
	"path"

	"trpc.group/trpc-go/trpc-go/codec"
)

func init() {
	codec.Register("mysql", nil, DefaultClientCodec)
}

// default codec
var (
	DefaultClientCodec = &ClientCodec{}
)

// ClientCodec Decode mysql client requests.
type ClientCodec struct{}

// Encode Set metadata for mysql client requests.
func (c *ClientCodec) Encode(msg codec.Msg, body []byte) (buffer []byte, err error) {

	//self
	if msg.CallerServiceName() == "" {
		msg.WithCallerServiceName(fmt.Sprintf("trpc.mysql.%s.service", path.Base(os.Args[0])))
	}

	return nil, nil
}

// Decode Parsing metadata in mysql client packet returns.
func (c *ClientCodec) Decode(msg codec.Msg, buffer []byte) (body []byte, err error) {

	return nil, nil
}
