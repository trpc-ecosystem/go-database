package mongodb

import (
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ClientTransportOption sets client transport parameter.
type ClientTransportOption func(opt *ClientTransport)

// WithOptionInterceptor returns an ClientTransportOption which sets mongo client option interceptor
func WithOptionInterceptor(f func(dsn string, opts *options.ClientOptions)) ClientTransportOption {
	return func(ct *ClientTransport) {
		ct.optionInterceptor = f
	}
}
