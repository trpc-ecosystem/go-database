package mongodb

import (
	"fmt"
	"os"
	"path"

	"trpc.group/trpc-go/trpc-go/codec"
)

func init() {
	codec.Register("mongodb", nil, defaultClientCodec)
}

// default codec
var (
	defaultClientCodec = &ClientCodec{}
)

// ClientCodec decodes mongodb client request.
type ClientCodec struct{}

// Encode sets the metadata requested by the mongodb client.
func (c *ClientCodec) Encode(msg codec.Msg, _ []byte) ([]byte, error) {

	//Itself.
	if msg.CallerServiceName() == "" {
		msg.WithCallerServiceName(fmt.Sprintf("trpc.mongodb.%s.service", path.Base(os.Args[0])))
	}

	return nil, nil
}

// Decode parses the metadata in the mongodb client return packet.
func (c *ClientCodec) Decode(msg codec.Msg, _ []byte) ([]byte, error) {

	return nil, nil
}
