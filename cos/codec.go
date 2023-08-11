/**
 * @Author: williamfqwu
 * @Date: 2021-01-06 15:05:08
 * @LastEditors: Please set LastEditors
 * @LastEditTime: 2021-01-31 20:08:32
 * @Description:
 * @Copyright: Tencent All rights reserved
 */

package cos

import (
	"errors"
	stdhttp "net/http"

	"trpc.group/trpc-go/trpc-go/codec"
	thttp "trpc.group/trpc-go/trpc-go/http"
	"trpc.group/trpc-go/trpc-go/transport"
)

func init() {
	DefaultServerCodec.ServerCodec = *thttp.DefaultServerCodec
	codec.Register("coshttp", DefaultServerCodec, DefaultClientCodec)
	transport.RegisterClientTransport("coshttp", thttp.DefaultClientTransport)
}

var (
	// DefaultClientCodec is default http client codec.
	DefaultClientCodec = &ClientCodec{}

	// DefaultServerCodec is default http server codec.
	DefaultServerCodec = &ServerCodec{}
)

// ClientCodec is the http client decoder.
type ClientCodec struct {
	thttp.ClientCodec
}

// ServerCodec is the http client decoder.
type ServerCodec struct {
	thttp.ServerCodec
}

// Decode decodes the metadata in the http client response.
func (c *ClientCodec) Decode(msg codec.Msg, data []byte) (rspBody []byte, err error) {
	rspHeader, ok := msg.ClientRspHead().(*thttp.ClientRspHeader)
	if !ok {
		return nil, errors.New("rsp header must be type of *http.ClientRspHeader")
	}

	// Rewrites StatusPartialContent to StatusOK to adapt to the framework.
	rsp := rspHeader.Response
	if rsp.StatusCode == stdhttp.StatusPartialContent {
		rsp.StatusCode = stdhttp.StatusOK
	}

	// Call the http decode method of framework.
	return c.ClientCodec.Decode(msg, data)
}
