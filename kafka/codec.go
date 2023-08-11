package kafka

import (
	"fmt"
	"os"
	"path"

	"trpc.group/trpc-go/trpc-go/codec"
)

func init() {
	codec.Register("kafka", DefaultServerCodec, DefaultClientCodec)
}

// default codec
var (
	DefaultServerCodec = &ServerCodec{}
	DefaultClientCodec = &ClientCodec{}

	serverName = path.Base(os.Args[0])
)

// ServerCodec Server codec
type ServerCodec struct{}

// Decode when the server receives the binary request data from the client and unpacks it into reqbody,
// the service handler will automatically create a new empty msg as the initial general message body
func (s *ServerCodec) Decode(_ codec.Msg, _ []byte) ([]byte, error) {
	return nil, nil
}

// Encode the server packs rspbody into a binary and sends it back to the client
func (s *ServerCodec) Encode(_ codec.Msg, _ []byte) ([]byte, error) {
	return nil, nil
}

// ClientCodec decode kafka client requests
type ClientCodec struct{}

// Encode set metadata for kafka client requests
func (c *ClientCodec) Encode(kafkaMsg codec.Msg, _ []byte) ([]byte, error) {
	if kafkaMsg.CallerServiceName() == "" {
		kafkaMsg.WithCallerServiceName(fmt.Sprintf("trpc.kafka.%s.service", serverName))
	}
	return nil, nil
}

// Decode parse the metadata in the kafka client return packet
func (c *ClientCodec) Decode(kafkaMsg codec.Msg, _ []byte) ([]byte, error) {
	return nil, nil
}
