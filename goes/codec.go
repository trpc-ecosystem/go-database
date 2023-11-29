package goes

import "trpc.group/trpc-go/trpc-go/codec"

func init() {
	codec.Register("goes", nil, DefaultClientCodec)
}

var (
	// DefaultClientCodec default client codec instance
	DefaultClientCodec = &ClientCodec{}
)

// ClientCodec client codec instance for request encode/decode
type ClientCodec struct{}

// Encode no need to encode anything
func (c *ClientCodec) Encode(msg codec.Msg, _ []byte) ([]byte, error) {
	return nil, nil
}

// Decode no need to decode anything
func (c *ClientCodec) Decode(msg codec.Msg, _ []byte) ([]byte, error) {
	return nil, nil
}
