package goes

import (
	"context"
	"net/http"

	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/transport"
)

const codecName = "goes"

func init() {
	transport.RegisterClientTransport(codecName, DefaultClientTransport)
}

// DefaultClientTransport default elasticsearch transport instance
var DefaultClientTransport = NewClientTransport()

// NewClientTransport create a new transport.ClientTransport
func NewClientTransport(opt ...OptClientTransport) transport.ClientTransport {
	ct := &ClientTransport{
		transport: http.DefaultTransport,
	}
	for _, o := range opt {
		o(ct)
	}

	return ct
}

// ClientTransport elasticsearch client transport
type ClientTransport struct {
	transport http.RoundTripper
}

// OptClientTransport client transport option
type OptClientTransport func(ct *ClientTransport)

// WithHTTPRoundTripper set http round tripper
func WithHTTPRoundTripper(rt http.RoundTripper) OptClientTransport {
	return func(ct *ClientTransport) {
		ct.transport = rt
	}
}

// RoundTrip impl transport.ClientTransport interface method
func (ct *ClientTransport) RoundTrip(ctx context.Context, _ []byte, callOpts ...transport.RoundTripOption) (
	[]byte, error) {
	msg := codec.Message(ctx)
	requestWrapper, ok := msg.ClientReqHead().(*requestWrapper)
	if !ok {
		return nil, errs.NewFrameError(errs.RetClientEncodeFail,
			"goes get requestWrapper failed")
	}

	if requestWrapper == nil || requestWrapper.request == nil {
		return nil, errs.NewFrameError(errs.RetClientEncodeFail,
			"goes request empty")
	}

	// reset request context
	requestWrapper.request = requestWrapper.request.WithContext(ctx)

	// extract response instance
	responseWrapper, ok := msg.ClientRspHead().(*responseWrapper)
	if !ok {
		return nil, errs.NewFrameError(errs.RetClientEncodeFail,
			"goes get responseWrapper failed")
	}

	// send http request
	response, err := ct.transport.RoundTrip(requestWrapper.request)
	if err != nil {
		return nil, err
	}

	// set es response
	responseWrapper.response = response

	return nil, nil
}
