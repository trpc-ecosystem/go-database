// Package clickhouse packages standard library clickhouse.
package clickhouse

import (
	"fmt"
	"os"
	"path"

	"trpc.group/trpc-go/trpc-go/codec"
)

func init() {
	codec.Register("clickhouse", nil, DefaultClientCodec)
}

// default codec
var (
	DefaultClientCodec = &ClientCodec{}
)

// ClientCodec decodes clickhouse client requests.
type ClientCodec struct{}

// Encode sets the metadata requested by the clickhouse client.
func (c *ClientCodec) Encode(msg codec.Msg, body []byte) (buffer []byte, err error) {

	// Itself.
	if msg.CallerServiceName() == "" {
		msg.WithCallerServiceName(fmt.Sprintf("trpc.clickhouse.%s.service", path.Base(os.Args[0])))
	}

	return nil, nil
}

// Decode parses the metadata in the clickhouse client return packet.
func (c *ClientCodec) Decode(msg codec.Msg, buffer []byte) (body []byte, err error) {

	return nil, nil
}
