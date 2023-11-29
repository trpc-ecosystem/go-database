[English](README.md) | 中文

# tRPC-Go ElasticSearch 插件
封装社区的 [es](https://github.com/elastic/go-elasticsearch)，配合 trpc 框架使用。

# 使用示例

## yaml 配置
全部的配置项都放置在 plugins 中进行设置，client 配置项中仅保留 service 名称。

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
        - name: trpc.app.service.es # 和 client 中的 service name 保持一致
          url: http://127.0.0.1:9200 #es地址
          user: username #用户名
          password: password #密码
          timeout: 1000 #超时时间
          log:
    		enabled: true #是否开启日志
    		request_enabled: true #是否打印请求日志
    		response_enabled: true #是否打印响应日志
```

## API 调用

goes 支持 elastic V7 和 V8 两个版本的API，不再支持 V6 版本的API。SDK 支持三种调用方式, 详情可见代码使用示例

### 通过`Client.Method`方式调用

#### elastic V7调用示例

```go
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"trpc.group/trpc-go/trpc-database/goes"
	"github.com/elastic/go-elasticsearch/esapi"

	pb "trpc.test.helloworld"
)

var body = map[string]interface{}{
	"account_number": 111111,
	"firstname":      "User",
	"city":           "Shenzhen",
}

func (s *greeterImpl) SayHello(ctx context.Context, req *pb.HelloRequest) (rsp *pb.HelloReply, err error) {
	// 使用 es v7 API 执行请求
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

#### elastic V8调用示例

```go
package main

func (s *greeterImpl) SayHello(ctx context.Context, req *pb.HelloRequest) (rsp *pb.HelloReply, err error) {
	// 使用 es v8 API 执行请求
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

### 通过`MethodRequest.Do`调用

#### elastic V7调用示例

```go
package main

func (s *greeterImpl) SayHello(ctx context.Context, req *pb.HelloRequest) (rsp *pb.HelloReply, err error) {
	// 使用 es v7 API 执行请求 
	proxyV7, err := goes.NewElasticClientV7("trpc.app.service.es")
    if err != nil {
        return nil, err
    }
	
    // 使用es api构建请求
    indexRequest := esapi.IndexRequest{
        Index:      "bank", 
        DocumentID: strconv.Itoa(1),
        Body:       bytes.NewReader(jsonBody),
    }
	
    // 使用trpc goes client执行请求
    _, err = indexRequest.Do(ctx, proxyV7)
    if err != nil {
        return nil, err
    }
    return &pb.HelloReply{}, nil
}
```
#### elastic V8调用示例
```go
package main

func (s *greeterImpl) SayHello(ctx context.Context, req *pb.HelloRequest) (rsp *pb.HelloReply, err error) { 
    // 使用esv8 api执行请求
    proxyV8, err := goes.NewElasticClientV8("trpc.app.service.es")
    if err != nil {
        return nil, err
    }
    
    // 使用es api构建请求 
    indexRequestV8 := esapi.IndexRequest{
        Index:      "bank", 
        DocumentID: strconv.Itoa(1), 
        Body:       bytes.NewReader(jsonBody),
    }
    
    // 使用trpc goes client执行请求
    _, err = indexRequestV8.Do(ctx, proxyV8)
    if err != nil {
        return nil, err
    }
    return &pb.HelloReply{}, nil
}
```

### 通过`RoundTriper.RoundTrip`方式调用

这种方式仅在需要完全自定义 es HTTP 请求时使用，大部分场景下优先使用 es API 进行调用。

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

## 常见问题

1.`NewElasticClientV7`和`NewElasticClientV8`两个方法都会创建新的连接，同时会向es server端发送一次GET请求校验server端信息，建议复用同一个client达到复用连接的效果。