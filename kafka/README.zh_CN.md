# tRPC-Go kafka 插件

[English](README.md) | 中文

封装社区的 [sarama](https://github.com/Shopify/sarama) ，配合 trpc 使用。

## producer client

```yaml
client:                                            #客户端调用的后端配置
  service:                                         #针对单个后端的配置
    - name: trpc.app.server.producer               #生产者服务名自己随便定义         
      target: kafka://ip1:port1,ip2:port2?topic=YOUR_TOPIC&clientid=xxx&compression=xxx
      timeout: 800                                 #当前这个请求最长处理时间
```

```go
package main

import (
  "time"
  "context"

  "trpc.group/trpc-go/trpc-database/kafka"
  "trpc.group/trpc-go/trpc-go/client"
)

func (s *server) SayHello(ctx context.Context, req *pb.ReqBody, rsp *pb.RspBody)( err error ) {
  proxy := kafka.NewClientProxy("trpc.app.server.producer") // service name 自己随便填，主要用于监控上报和寻找配置项

  // 如使用自己配置字符串，可在 NewClientProxy 中使用 client.WithTarget 指定。
  // kafka 命令
  err := proxy.Produce(ctx, key, value)

  // 生产原生 sarama 消息，返回 offset、partition
  partition, offset, err := proxy.SendSaramaMessage(ctx, sarama.ProducerMessage{
    Topic: "your_topic",
    Value: sarama.ByteEncoder(msg),
  })

  // 业务逻辑
}
```

## consumer service

```yaml
server:                                                                                   #服务端配置
  service:                                                                                #业务服务提供的 service，可以有多个
    - name: trpc.app.server.consumer                                                      #service 的路由名称，需要使用 trpc.${app}.${server}.consumer  
      address: ip1:port1,ip2:port2?topics=topic1,topic2&group=xxx&version=x.x.x.x        #kafka consumer broker address，version 如果不设置则为 1.1.1.0，部分 ckafka 需要指定 0.10.2.0
      protocol: kafka                                                                     #应用层协议 
      timeout: 1000                                                                       #框架配置，与 sarama 配置无关

```

```go
package main

import (
  "context"

  "trpc.group/trpc-go/trpc-database/kafka"
  trpc "trpc.group/trpc-go/trpc-go"
  "github.com/Shopify/sarama"
)

func main() {
  s := trpc.NewServer()
  // 使用自定义 addr，需在启动 server 前调用
  /*
    cfg := kafka.GetDefaultConfig()
    cfg.ClientID = "newClientID"
    kafka.RegisterAddrConfig("address", cfg)
  */
  // 启动多个消费者的情况，可以配置多个 service，然后这里任意匹配 kafka.RegisterHandlerService(s.Service("name"), handle)，没有指定 name 的情况，代表所有 service 共用同一个 handler
  kafka.RegisterKafkaHandlerService(s, handle) 
  s.Serve()
}

// 只有返回成功 nil，才会确认消费成功，返回 err 不会确认成功，会等待 3s 重新消费，会有重复消息，一定要保证处理函数幂等性
func handle(ctx context.Context, msg *sarama.ConsumerMessage) error {
  return nil
}
```

### 批量消费

```go
import (
  "context"
 
  "trpc.group/trpc-go/trpc-database/kafka"
  trpc "trpc.group/trpc-go/trpc-go"
  "github.com/Shopify/sarama"
)

func main() {
  s := trpc.NewServer()
  kafka.RegisterBatchHandlerService(s, handle) 
  s.Serve()
}

// 注意：必须要配置 batch 参数 (>0), 如果不配置 batch 参数会发生消费处理函数不匹配导致消费失败
// 完整使用例子参考 examples/batchconsumer
// 只有返回成功 nil，才会确认消费成功，返回 err 整个批次所有消息重新消费
func handle(ctx context.Context, msgArray []*sarama.ConsumerMessage) error {
  // ...
  return nil
}
```

如果需要配置自己的参数，可以使用：

```go
cfg := kafka.GetDefaultConfig()
// 更新自己的 cfg 属性
kafka.RegisterAddrConfig("address", cfg) // address 为你配置中填写的 address
```

## 参数说明

### 生产者

```yaml
client:
  service:
    - name: trpc.app.server.producer
      target: kafka://ip1:port1,ip2:port2?topic=YOUR_TOPIC&clientid=xxx&compression=xxx
      timeout: 800
    - name: trpc.app.server.producer1   # 北极星接入，不支持动态变更，如果北极星 IP 有变更，需要重启服务才能生效。如果需要获取所有北极星节点，需要关注 Q14
      target: kafka://YOUR_SERVICE_NAME?topic=YOUR_TOPIC&clientid=xxx&compression=xxx&discover=polaris&namespace=Development 
      timeout: 800 
```

|    参数名称     |               含义               | 可选值&说明                                                                                                                                                                                                                                                                                                 |
| :-------------: | :------------------------------: | :---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
|     ip:port     |             地址列表             | 复数地址使用逗号分割，支持 域名:port，ip:port。暂不支持 cl5                                                                                                                                                                                                                                                 |
|    clientid     |            生产者 ID             | 若为虫洞 kafka 需要到管理页面注册                                                                                                                                                                                                                                                                           |
|      topic      |           生产的 topic           | 调用 Produce 必需。若为虫洞 kafka 需要到管理页面注册                                                                                                                                                                                                                                                        |
|     version     |            客户端版本            | 支持以下两种格式的版本号 0.1.1.0 / 1.1.0                                                                                                                                                                                                                                                                    |
|   partitioner   |           消息分片方式           | <div> random: sarama.NewRandomPartitioner(默认值);<br>roundrobin: sarama.NewRoundRobinPartitioner;<br>hash: sarama.NewHashPartitioner 暂未自定义 hash 方式; </div>                                                                                                                                          |
|   compression   |             压缩方式             | <div> none : sarama.CompressionNone;<br>gzip : sarama.CompressionGZIP(默认值);<br>snappy : sarama.CompressionSnappy;<br>lz4 :sarama.CompressionLZ4;<br>zstd 等于 sarama.CompressionZSTD; </div>                                                                                                             |
| maxMessageBytes |           Msg 最大长度           | 默认 131072                                                                                                                                                                                                                                                                                                 |
|  requiredAcks   |           是否需要回执           | <div>生产消息时，broker 返回消息回执（ack）的模式，支持以下 3 个值（0/1/-1):<br>0: NoResponse，不需要等待 broker 响应。<br>1: WaitForLocal，等待本地（leader 节点）的响应即可返回。<br>-1: WaitForAll，等待所有节点（leader 节点和所有 In Sync Replication 的 follower 节点）均响应后返回。（默认值）</div> |
|    maxRetry     |         失败最大重试次数         | 生产消息最大重试次数，默认为 3 次，注意：必须大于等于 0，否则会报错                                                                                                                                                                                                                                         |
|  retryInterval  |           失败重试间隔           | 单位毫秒，默认 100ms                                                                                                                                                                                                                                                                                        |
|    trpcMeta     | 透传 trpc 元数据到 sarama header | true 表示开启透传，false 表示不透传，默认 false                                                                                                                                                                                                                                                             |
|    discover     |  用于服务发现的 discovery 类型   | 例如：polaris                                                                                                                                                                                                                                                                                               |
|    namespace    |           服务命名空间           | 例如：Development                                                                                                                                                                                                                                                                                           |
|   idempotent    |        是否开始生成者幂等        | true/false，默认 false ｜                                                                                                                                                                                                                                                                                   |

### 消费者

```yaml
server:
  service:
    - name: trpc.app.server.consumer
      address: ip1:port1,ip2:port2?topics=topic1,topic2&group=xxx&version=x.x.x.x
      protocol: kafka
      timeout: 1000
    - name: trpc.app.server.consumer1   # 北极星接入，不支持动态变更，如果北极星 IP 有变更，需要重启服务才能生效。如果需要获取所有北极星节点，需要关注 Q14
      address: YOUR_SERVICE_NAME?topics=topic1,topic2&group=xxx&version=x.x.x.x&discover=polaris&namespace=Development
      protocol: kafka                                            
      timeout: 1000    #请求最长处理时间 单位 毫秒
```

|       参数名称        |                                                                含义                                                                 |                                                                          可选值&说明                                                                           |
| :-------------------: | :---------------------------------------------------------------------------------------------------------------------------------: | :------------------------------------------------------------------------------------------------------------------------------------------------------------: |
|     ip:port 列表      |                                                                地址                                                                 |                                                   复数地址使用逗号分割，支持域名:port，ip:port。暂不支持 cl5                                                   |
|         group         |                                                              消费者组                                                               |                                                               若为虫洞 kafka 需要到管理页面注册                                                                |
|       clientid        |                                                              客户端 id                                                              |                                                                     连接 kafka 的客户端 id                                                                     |
|        topics         |                                                           消费的的 toipc                                                            |                                                                         复数用逗号分割                                                                         |
|      compression      |                                                              压缩方式                                                               |                                                                                                                                                                |
|       strategy        |                                                                策略                                                                 |           <div>sticky: sarama.BalanceStrategySticky;<br>range : sarama.BalanceStrategyRange;<br>roundrobin: sarama.BalanceStrategyRoundRobin;</div>            |
|     fetchDefault      | 拉取消息的默认大小（字节），如果消息实际大小大于此值，需要重新分配内存空间，会影响性能，等价于 sarama.Config.Consumer.Fetch.Default |                                                                          默认 524288                                                                           |
|       fetchMax        |              拉取消息的最大大小（字节），如果消息实际大小大于此值，会直接报错，等价于 sarama.Config.Consumer.Fetch.Max              |                                                                          默认 1048576                                                                          |
|         batch         |                                                             每批次数目                                                              |                        使用批量消费时必填，注册批量消费函数时，batch 不填会发生参数不匹配导致消费失败，使用参考 examples/batchconsumer                         |
|      batchFlush       |                                                            批次消费间隔                                                             |                                               默认 2 秒，单位 ms, 表示当批量消费不满足最大条数时，强制消费的间隔                                               |
|        initial        |                                                            初始消费位置                                                             |                                     <div>新消费者组第一次连到集群消费的位置<br>newest: 最新位置<br> oldest: 最老位置</div>                                     |
|      maxWaitTime      |                                                    单次消费拉取请求最长等待时间                                                     |                                                        最长等待时间仅在没有最新数据时才会等待，默认 1s                                                         |
|       maxRetry        |                                                          失败最大重试次数                                                           | 超过后直接确认并继续消费下一条消息，默认 0:没有次数限制，一直重试、负数表示不重试，直接确认并继续消费下一条消息，正数表示如果一直错误最终执行次数为 maxRetry+1 |
|  netMaxOpenRequests   |                                                           最大同时请求数                                                            |                                                               网络层配置，最大同时请求数，默认 5                                                               |
|   maxProcessingTime   |                                                       消费者单条最大请求时间                                                        |                                                                      单位 ms，默认 1000ms                                                                      |
|    netDailTimeout     |                                                            链接超时时间                                                             |                                                        网络层配置，链接超时时间，单位 ms，默认 30000ms                                                         |
|    netReadTimeout     |                                                             读超时时间                                                              |                                                         网络层配置，读超时时间，单位 ms，默认 30000ms                                                          |
|    netWriteTimeout    |                                                             写超时时间                                                              |                                                         网络层配置，写超时时间，单位 ms，默认 30000ms                                                          |
|  groupSessionTimeout  |                                                       消费组 session 超时时间                                                       |                                                                     单位 ms，默认 10000ms                                                                      |
| groupRebalanceTimeout |                                                      消费者 rebalance 超时时间                                                      |                                                                     单位 ms，默认 60000ms                                                                      |
|       mechanism       |                                                         使用密码时加密方式                                                          |                                                               可选值 SCRAM-SHA-512/SCRAM-SHA-256                                                               |
|         user          |                                                               用户名                                                                |                                                                                                                                                                |
|       password        |                                                                密码                                                                 |                                                                                                                                                                |
|     retryInterval     |                                                              重试间隔                                                               |                                                                     单位毫秒，默认 3000ms                                                                      |
|    isolationLevel     |                                                              隔离级别                                                               |                                                              可选值 ReadUncommitted/ReadCommitted                                                              |
|       trpcMeta        |                                         透传 trpc 元数据，读取 sarama header 设置 trpc meta                                         |                                                        true 表示开启透传，false 表示不透传，默认 false                                                         |
|       discover        |                                                    用于服务发现的 discovery 类型                                                    |                                                                         例如：polaris                                                                          |
|       namespace       |                                                            服务命名空间                                                             |                                                                       例如：Development                                                                        |

## 常见问题

- Q1: 消费者的 service.name 该怎么写
- A1: 如果只有消费者一个 service，则名字可以任意起（trpc 框架默认会把实现注册到 server 里面的所有 service），完整见 examples/consumer

``` yaml
server:                                                                  
  service:                                                               
    - name: trpc.anyname.will.works
      address: 9.134.192.186:9092?topics=test_topic&group=uzuki_consumer 
      protocol: kafka                                                    
      timeout: 1000                                                      
```

``` go
s := trpc.NewServer()
kafka.RegisterKafkaHandlerService(s, handle) 
```

如果有多个 service，则需要在注册时指定与配置文件相同的名字，完整见 examples/consumer_with_mulit_service

``` yaml
server:                                                                   
  service:                                                                
    - name: trpc.databaseDemo.kafka.consumer1                             
      address: 9.134.192.186:9092?topics=test_topic&group=uzuki_consumer1 
      protocol: kafka                                                     
      timeout: 1000                                                       
    - name: trpc.databaseDemo.kafka.consumer2                             
      address: 9.134.192.186:9092?topics=test_topic&group=uzuki_consumer2 
      protocol: kafka                                                     
      timeout: 1000     
```

``` go
s := trpc.NewServer()    
kafka.RegisterKafkaConsumerService(s.Service("trpc.databaseDemo.kafka.consumer1"), &Consumer{})
kafka.RegisterKafkaConsumerService(s.Service("trpc.databaseDemo.kafka.consumer2"), &Consumer{})
```

- Q2: 如果消费时 handle 返回了非 nil 会发生什么

- A2: 会休眠 3s 后重新消费，不建议这么做，失败的应该由业务做重试逻辑

- Q3: 使用 ckafka 生产消息时，提示

``` log
err:type:framework, code:141, msg:kafka client transport SendMessage: kafka server: Message contents does not match its CRC.
```

- A3: 默认启用了 gzip 压缩，优先考虑在 target 上加上参数**compression=none**

``` yaml
target: kafka://ip1:port1,ip2:port2?clientid=xxx&compression=none
```

- Q4: 使用 ckafka 消费消息时，提示

``` log
kafka server transport: consume fail:kafka: client has run out of available brokers to talk to (Is your cluster reachable?)
```

- A4: 优先检查 brokers 是否可达，然后检查支持的 kafka 客户端版本，尝试在配置文件 address 中加上参数例如**version=0.10.2.0**

``` yaml
address: ip1:port1,ip2:port2?topics=topic1,topic2&group=xxx&version=0.10.2.0 
```

- Q5: 消费消息时，提示

``` log
kafka server transport: consume fail:kafka server: The provider group protocol type is incompatible with the other members.
```

- A5: 同一消费者组的客户端重分组策略不一样，可修改参数**strategy**，可选：sticky(默认)，range，roundrobin

``` yaml
address: ip1:port1,ip2:port2?topics=topic1,topic2&group=xxx&strategy=range
```

- Q6: 生产时同一用户需要有序，如何配置
- A6: 客户端增加参数**partitioner**，可选 random（默认），roundrobin，hash（按 key 分区）

``` yaml
target: kafka://ip1:port1,ip2:port2?clientid=xxx&partitioner=hash
```

- Q7: 如何异步生产
- A7: 客户端增加参数**async=1**

``` yaml
target: kafka://ip1:port1,ip2:port2?clientid=xxx&async=1
```

- Q8: 遇到北极星路由问题 "Polaris-1006(ErrCodeServerError)" "not found service"
- A8: 确认一下 trpc 配置中的 service.name 是  trpc.**${app}.${server}**.AnyUniqNameWillWork 而不是 trpc.**app.server**.AnyUniqNameWillWork, 必须要用占位符
错误现场：

``` log
type:framework, code:131, msg:client Select: get source service route rule err: Polaris-1006(ErrCodeServerError): Response from {ID: 2079470528, Service: {ServiceKey: {namespace: "Polaris", service: "polaris.discover"}, ClusterType: discover}, Address: 9.97.76.132:8081}: not found service
```

- Q9: 如何使用账号密码链接
- A9: 需要在链接参数中配置加密方式、用户名和密码
例如：

``` yaml
address: ip1:port1,ip2:port2?topics=topic1,topic2&mechanism=SCRAM-SHA-512&user={user}&password={password}
```

- Q10: 如何使用异步生产写数据回调
- A10: 需要在代码中重写异步生产写数据的成功/失败的回掉函数，例如

```go
import ( 
  // ... 
  "trpc.group/trpc-go/trpc-database/kafka"
  // ...
)

func init() {
  // 重写默认的异步生产写数据错误回调
  kafka.AsyncProducerErrorCallback = func(err error, topic string, key, value []byte, headers []sarama.RecordHeader) {
    // do something if async producer occurred error.
  }

  // 重写默认的异步生产写数据成功回调
  kafka.AsyncProducerSuccCallback = funcfunc(topic string, key, value []byte, headers []sarama.RecordHeader) {
    // do something if async producer succeed.
  }
}
```

- Q11: 如何注入自定义配置 (远端配置)
- A11: 在`trpc_go.yaml`中配置`fake_address`，然后配合`kafka.RegisterAddrConfig`方法注入
`trpc_go.yaml`配置如下

``` yaml
address: fake_address 
```

在服务启动前，注入自定义配置

```go
func main() {
  s := trpc.NewServer()
  // 使用自定义 addr，需在启动 server 前注入
  cfg := kafka.GetDefaultConfig()
  cfg.Brokers = []string{"127.0.0.1:9092"}
  cfg.Topics = []string{"test_topic"}
  kafka.RegisterAddrConfig("fake_address", cfg)
  kafka.RegisterKafkaConsumerService(s, &Consumer{})

  s.Serve()
}
```

- Q12: 如何透传 trpc 元数据
- A12: 生产者和消费者增加参数 trpcMeta=true，默认不开启透传；如果原生产接口已经设置了 header，需要注意 header 冲突、重复、覆盖等问题；

生产者

``` yaml
target: kafka://11.135.87.176:9092?clientid=test_producer&partitioner=hash&topic=test_topic&trpcMeta=true
```

消费者

``` yaml
address: 11.135.87.176:9092?topics=test_topic&group=test_group&trpcMeta=true     #kafka consumer dsn
```

- Q13: 如何获取底层 sarama 的上下文信息
  - A13:通过 kafka.GetRawSaramaContext 可以获取底层 sarama ConsumerGroupSession 和 ConsumerGroupClaim。但是此处暴露这两个接口只是方便用户做监控日志，应该只使用其读方法，调用任何写方法在这里都是未定义行为，可能造成未知结果

```go
// RawSaramaContext 存放 sarama ConsumerGroupSession 和 ConsumerGroupClaim
// 导出此结构体是为了方便用户实现监控，提供的内容仅用于读，调用任何写方法属于未定义行为
type RawSaramaContext struct {
    Session  sarama.ConsumerGroupSession
    Claim sarama.ConsumerGroupClaim
}
```

使用实例

```go
func example(ctx context.Context){
    if rawContext, ok := kafka.GetRawSaramaContext(ctx); ok {
        log.Infof("InitialOffset:%d", rawContext.Claim.InitialOffset())
    }
}

```

- Q14: 服务端北极星寻址如何获取所有节点
  - A14: 在`trpc_go.yaml`中开启`service_router.need_return_all_nodes = true`

```yaml
plugins:     #插件配置
  selector:
    polaris:
      service_router:
        need_return_all_nodes: true
```
