package goes

import (
	"fmt"
	"net/http"

	elasticv7 "github.com/elastic/go-elasticsearch/v7"
	elasticv8 "github.com/elastic/go-elasticsearch/v8"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
)

// RoundTriper es client interface
type RoundTriper interface {
	// RoundTrip elastic client transport method implementation
	RoundTrip(request *http.Request) (*http.Response, error)
}

// NewClientProxy create a new elastic client proxy
func NewClientProxy(name string, opts ...client.Option) RoundTriper {
	return &esHTTPTransport{serviceName: name, c: client.DefaultClient,
		opts: append(opts, client.WithProtocol(codecName), client.WithTarget("passthrough://passthrough"))}
}

// NewElasticClientV8 create a new client
func NewElasticClientV8(serviceName string, opts ...client.Option) (*elasticv8.Client, error) {
	serviceConfig, ok := dbConfigMap[serviceName]
	if !ok {
		return nil, errs.New(errs.RetServerNoService, "service has no config")
	}

	return elasticv8.NewClient(elasticv8.Config{
		Addresses: []string{serviceConfig.URL},
		Username:  serviceConfig.User,
		Password:  serviceConfig.Password,
		Transport: NewClientProxy(serviceName, opts...),
		Logger: &esLogger{
			Enable:             serviceConfig.Log.Enabled,
			EnableRequestBody:  serviceConfig.Log.RequestEnabled,
			EnableResponseBody: serviceConfig.Log.ResponseEnabled,
		},
	})
}

// NewElasticTypedClientV8 create a new typed client
func NewElasticTypedClientV8(serviceName string, opts ...client.Option) (*elasticv8.TypedClient, error) {
	serviceConfig, ok := dbConfigMap[serviceName]
	if !ok {
		return nil, errs.New(errs.RetServerNoService, "service has no config")
	}

	return elasticv8.NewTypedClient(elasticv8.Config{
		Addresses: []string{serviceConfig.URL},
		Username:  serviceConfig.User,
		Password:  serviceConfig.Password,
		Transport: NewClientProxy(serviceName, opts...),
		Logger: &esLogger{
			Enable:             serviceConfig.Log.Enabled,
			EnableRequestBody:  serviceConfig.Log.RequestEnabled,
			EnableResponseBody: serviceConfig.Log.ResponseEnabled,
		},
	})
}

// NewElasticClientV7 create a new client
func NewElasticClientV7(serviceName string, opts ...client.Option) (*elasticv7.Client, error) {
	serviceConfig, ok := dbConfigMap[serviceName]
	if !ok {
		return nil, errs.New(errs.RetServerNoService, "service has no config")
	}

	return elasticv7.NewClient(elasticv7.Config{
		Addresses: []string{serviceConfig.URL},
		Username:  serviceConfig.User,
		Password:  serviceConfig.Password,
		Transport: NewClientProxy(serviceName, opts...),
		Logger: &esLogger{
			Enable:             serviceConfig.Log.Enabled,
			EnableRequestBody:  serviceConfig.Log.RequestEnabled,
			EnableResponseBody: serviceConfig.Log.ResponseEnabled,
		},
	})
}

// esHTTPTransport implements the RoundTriper interface
type esHTTPTransport struct {
	serviceName string
	c           client.Client
	opts        []client.Option
}

// responseWrapper wraps a http.Response
type responseWrapper struct {
	response *http.Response
}

// requestWrapper wraps a http.Request
type requestWrapper struct {
	request *http.Request
}

// simpleRequestInfo is used in client.Client.Invoke
type simpleRequestInfo struct {
	method string // http method
	URI    string // http uri info
}

// RoundTrip executes a single HTTP transaction, returning a Response for the provided Request.
func (t *esHTTPTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	ctx, msg := codec.WithCloneMessage(r.Context())
	msg.WithClientRPCName(fmt.Sprintf("/%s/%s", t.serviceName, r.Method))
	msg.WithCalleeServiceName(t.serviceName)
	rspWrapper := &responseWrapper{}
	reqWrapper := &requestWrapper{request: r}
	msg.WithClientReqHead(reqWrapper)
	msg.WithClientRspHead(rspWrapper)
	msg.WithCompressType(codec.CompressTypeNoop)
	msg.WithSerializationType(codec.SerializationTypeUnsupported)
	err := t.c.Invoke(ctx, simpleRequestInfo{method: r.Method, URI: r.URL.Path}, nil, t.opts...)
	if err != nil {
		return nil, err
	}
	return rspWrapper.response, nil
}
