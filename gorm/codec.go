package gorm

import (
	"fmt"
	"os"
	"path"

	"trpc.group/trpc-go/trpc-go/codec"
)

func init() {
	codec.Register("gorm", nil, DefaultClientCodec)
}

// default codec
var (
	DefaultClientCodec = &ClientCodec{}
)

// ClientCodec decodes db client requests.
type ClientCodec struct{}

// Encode sets the metadata of the db client request.
func (c *ClientCodec) Encode(msg codec.Msg, body []byte) (buffer []byte, err error) {
	// The request is from this service.
	if msg.CallerServiceName() == "" {
		msg.WithCallerServiceName(fmt.Sprintf("trpc.gorm.%s.service", path.Base(os.Args[0])))
	}

	return nil, nil
}

// Decode parses the metadata in the response from the db client.
func (c *ClientCodec) Decode(msg codec.Msg, buffer []byte) (body []byte, err error) {
	return nil, nil
}
