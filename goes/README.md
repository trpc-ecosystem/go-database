English | [中文](README.zh_CN.md)

# tRPC-Go ElasticSearch Plugin
wrapping community [es](https://github.com/elastic/go-elasticsearch), used with trpc.

# Usage Example

## YAML Configuration
All configuration options are placed within the plugins section, with only the service name retained in the client configuration.

```yaml
client:                                          
  timeout: 1000                                  
  namespace: Development                         
  service:                                         
    - name: trpc.app.service.es

plugins:                                          
  database:
    goes:
      clientoptions:
        - name: trpc.app.service.es # Match the service name in the client configuration
          url: http://127.0.0.1:9200 # Elasticsearch address
          user: username # Username
          password: password # Password
          timeout: 1000 # Timeout
          log:
            enabled: true # Enable logging
            request_enabled: true # Print request logs
            response_enabled: true # Print response logs
```

## API Invocation

goes supports both Elastic V7 and V8 API versions, and no longer supports V6 API version. The SDK supports three calling methods; details can be found in the code usage examples.

### Calling through `Client.Method`

#### Example for Elastic V7

```go
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"git.woa.com/trpc-go/trpc-database/goes"
	"github.com/elastic/go-elasticsearch/esapi"

	pb "git.code.oa.com/trpcprotocol/test/helloworld"
)

var body = map[string]interface{}{
	"account_number": 111111,
	"firstname":      "User",
	"city":           "Shenzhen",
}

func (s *greeterImpl) SayHello(ctx context.Context, req *pb.HelloRequest) (rsp *pb.HelloReply, err error) {
	// Use es v7 API to execute the request
	proxyV7, err := goes.NewElasticClientV7("trpc.app.service.es")
	if err != nil {
		return nil, err
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	_, err = proxyV7.Index("bank", bytes.NewBuffer(jsonBody), proxyV7.Index.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	return &pb.HelloReply{}, nil
}
```

#### Example for Elastic V8

```go
package main

func (s *greeterImpl) SayHello(ctx context.Context, req *pb.HelloRequest) (rsp *pb.HelloReply, err error) {
	// Use es v8 API to execute the request
	proxyV8, err := goes.NewElasticClientV8("trpc.app.service.es")
    if err != nil {
        return nil, err
    }
    jsonBody, err = json.Marshal(body)
	if err != nil {
        return nil, err
    }
    _, err = proxyV8.Index("bank", bytes.NewBuffer(jsonBody), proxyV8.Index.WithContext(ctx))
	if err != nil {
        return nil, err
    }
	
    return &pb.HelloReply{}, nil
}
```

### Calling through `MethodRequest.Do`

#### Example for Elastic V7

```go
package main

func (s *greeterImpl) SayHello(ctx context.Context, req *pb.HelloRequest) (rsp *pb.HelloReply, err error) {
	// Use es v7 API to execute the request 
	proxyV7, err := goes.NewElasticClientV7("trpc.app.service.es")
    if err != nil {
        return nil, err
    }
	
    // Build request using es API
    indexRequest := esapi.IndexRequest{
        Index:      "bank", 
        DocumentID: strconv.Itoa(1),
        Body:       bytes.NewReader(jsonBody),
    }
	
    // Execute request using trpc goes client
    _, err = indexRequest.Do(ctx, proxyV7)
    if err != nil {
        return nil, err
    }
    return &pb.HelloReply{}, nil
}
```

#### Example for Elastic V8

```go
package main

func (s *greeterImpl) SayHello(ctx context.Context, req *pb.HelloRequest) (rsp *pb.HelloReply, err error) { 
    // Use es v8 API to execute the request
    proxyV8, err := goes.NewElasticClientV8("trpc.app.service.es")
    if err != nil {
        return nil, err
    }
    
    // Build request using es API 
    indexRequestV8 := esapi.IndexRequest{
        Index:      "bank", 
        DocumentID: strconv.Itoa(1), 
        Body:       bytes.NewReader(jsonBody),
    }
    
    // Execute request using trpc goes client
    _, err = indexRequestV8.Do(ctx, proxyV8)
    if err != nil {
        return nil, err
    }
    return &pb.HelloReply{}, nil
}
```

### Calling through `RoundTriper.RoundTrip`

This method should only be used when fully customizing the es HTTP request is necessary, and in most cases, it is preferable to use the es API for the call.

```go
package main

func (s *greeterImpl) SayHello(ctx context.Context, req *pb.HelloRequest) (rsp *pb.HelloReply, err error) {
    url := &url.URL{Path: "/", Scheme: "http"}
    proxy := goes.NewClientProxy("trpc.app.service.es")
    _, err = proxy.RoundTrip(&http.Request{Method: "GET", Proto: "HTTP/1.1",URL:url, ProtoMajor: 1, ProtoMinor: 1})
    if err != nil {
        return nil, err
    }
    return &pb.HelloReply{}, nil
}
```

## Frequently Asked Questions

1. Both `NewElasticClientV7` and `NewElasticClientV8` methods create a new connection and send a GET request to the es server to verify server information. It is recommended to reuse the same client to achieve connection reuse.
