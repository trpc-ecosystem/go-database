package timer

import (
	"trpc.group/trpc-go/trpc-go/codec"
)

func init() {
	codec.Register("timer", DefaultServerCodec, nil)
}

// DefaultServerCodec is default server codec.
var DefaultServerCodec = &ServerCodec{}

// ServerCodec is the timer server codec.
type ServerCodec struct {
}

// Decode codecs the binary request data, and service handler will automatically create a new empty msg.
func (s *ServerCodec) Decode(msg codec.Msg, reqbuf []byte) (reqbody []byte, err error) {
	// set upstream service name
	msg.WithCallerServiceName("trpc.timer.noserver.noservice")
	// set current rpc method name
	msg.WithServerRPCName("/trpc.timer.server.service/handle")
	// set caller
	msg.WithCalleeApp("timer")
	msg.WithCalleeServer(serverName)
	msg.WithCalleeService("service")

	return reqbody, nil
}

// Encode server encode rspbody into binary and return to client.
func (s *ServerCodec) Encode(msg codec.Msg, rspbody []byte) (rspbuf []byte, err error) {
	if err := msg.ServerRspErr(); err != nil {
		return nil, err
	}
	return nil, nil
}
