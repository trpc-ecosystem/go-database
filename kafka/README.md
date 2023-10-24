English | [中文](README.zh_CN.md)

# tRPC-Go kafka plugin

[![Go Reference](https://pkg.go.dev/badge/trpc.group/trpc-go/trpc-database/kafka.svg)](https://pkg.go.dev/trpc.group/trpc-go/trpc-database/kafka)
[![Go Report Card](https://goreportcard.com/badge/trpc.group/trpc-go/trpc-database/kafka)](https://goreportcard.com/report/trpc.group/trpc-go/trpc-database/kafka)
[![Tests](https://github.com/trpc-ecosystem/go-database/actions/workflows/kafka.yml/badge.svg)](https://github.com/trpc-ecosystem/go-database/actions/workflows/kafka.yml)
[![Coverage](https://codecov.io/gh/trpc-ecosystem/go-database/branch/coverage/graph/badge.svg?flag=kafka&precision=2)](https://app.codecov.io/gh/trpc-ecosystem/go-database/tree/coverage/kafka)

wrapping community [sarama](https://github.com/Shopify/sarama), used with trpc.

## producer client

```yaml
client:                                            # Backend configuration for client calls.
  service:                                         # Configuration for the backend.
    - name: trpc.app.server.producer               # producer service name,define by yourself.        
      target: kafka://ip1:port1,ip2:port2?topic=YOUR_TOPIC&clientid=xxx&compression=xxx
      timeout: 800                                 # The maximum processing time of the current request.
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
  proxy := kafka.NewClientProxy("trpc.app.server.producer") // The service name is customized 
  // and is mainly used for monitoring reporting and addressing configuration items.
  
  // kafka function call.
  err := proxy.Produce(ctx, key, value)

  // producer native interface, return offset、partition.
  partition, offset, err := proxy.SendSaramaMessage(ctx, sarama.ProducerMessage{
    Topic: "your_topic",
    Value: sarama.ByteEncoder(msg),
  })

  // Business logic
  // ...
}
```

## consumer service

```yaml
server:                                                                                   # Server configuration.
  service:                                                                                # The service provided by the business service can have multiple.
    - name: trpc.app.server.consumer                                                      # The routing name of the service is currently fixed.
      address: ip1:port1,ip2:port2?topics=topic1,topic2&group=xxx&version=x.x.x.x         # kafka consumer broker address，version default 1.1.1.0, partial version ckafka need to specify 0.10.2.0.
      protocol: kafka                                                                     # Application layer protocol.  
      timeout: 1000                                                                       # The maximum request processing time, in milliseconds. Framework configuration, not related to sarama configuration.

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
  // To use a custom addr, it needs to be called before starting the server
  /*
    cfg := kafka.GetDefaultConfig()
    cfg.ClientID = "newClientID"
    kafka.RegisterAddrConfig("address", cfg)
  */
  // In the case of starting multiple consumers, multiple services can be configured, and then any matching here kafka.RegisterHandlerService(s.Service("name"), handle), If no name is specified, it means that all services share the same handler
  kafka.RegisterKafkaHandlerService(s, handle) 
  s.Serve()
}

// Only when nil is returned successfully will the consumption be confirmed successfully.
// If the return fails, it will be determined whether to repeat the consumption according to the message attribute.
func handle(ctx context.Context, msg *sarama.ConsumerMessage) error {
  return nil
}
```

### batch consumer

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

// Note: The batch parameter (>0) must be configured, if the batch parameter is not configured, the consumption processing function will not match and the consumption will fail.
// For complete usage examples, refer to examples/batchconsumer.
// Only when nil is returned successfully will the consumption be confirmed successfully, and all messages in the entire batch will be re-consumed if err is returned.
func handle(ctx context.Context, msgArray []*sarama.ConsumerMessage) error {
  // ...
  return nil
}
```

If you need to configure your own parameters, you can use:

```go
cfg := kafka.GetDefaultConfig()
// Update own cfg properties.
kafka.RegisterAddrConfig("address", cfg) // address is the address filled in your configuration.
```

## Parameter Description

### producer

```yaml
client:
  service:
    - name: trpc.app.server.producer
      target: kafka://ip1:port1,ip2:port2?topic=YOUR_TOPIC&clientid=xxx&compression=xxx
      timeout: 800
    - name: trpc.app.server.producer1   # Polaris Naming access does not support dynamic changes. If the Polaris IP changes, the service needs to be restarted to take effect. If you need to get all Polaris nodes, you need to pay attention to Q14
      target: kafka://YOUR_SERVICE_NAME?topic=YOUR_TOPIC&clientid=xxx&compression=xxx&discover=polaris&namespace=Development 
      timeout: 800 
```

| parameter name  |               meaning               | Optional value & description                                                                                                                                                                                                                                                                                                                                                                                             |
| :-------------: | :---------------------------------: | :----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
|     ip:port     |            address list             | Separate multiple addresses with commas(','),support domain:port, ip:porta. Not supported yet cl5.                                                                                                                                                                                                                                                                                                                       |
|    clientid     |             producer ID             | If it is [wormhole] kafka (PCG internal version), you need to register on the management page.                                                                                                                                                                                                                                                                                                                           |
|      topic      |           producer topic            | call Produce required. If it is [wormhole] kafka, need to register on the management page.                                                                                                                                                                                                                                                                                                                               |
|     version     |           client version            | Version numbers in the following two formats are supported: 0.1.1.0 / 1.1.0                                                                                                                                                                                                                                                                                                                                              |
|   partitioner   |          message partition          | <div> random: sarama.NewRandomPartitioner(Defaults);<br>roundrobin: sarama.NewRoundRobinPartitioner;<br>hash: sarama.NewHashPartitioner No custom hash method yet; </div>                                                                                                                                                                                                                                                |
|   compression   |         compression method          | <div> none : sarama.CompressionNone;<br>gzip : sarama.CompressionGZIP(Defaults);<br>snappy : sarama.CompressionSnappy;<br>lz4 :sarama.CompressionLZ4;<br>zstd equal sarama.CompressionZSTD; </div>                                                                                                                                                                                                                       |
| maxMessageBytes |       The maximum length Msg        | Defaults 131072                                                                                                                                                                                                                                                                                                                                                                                                          |
|  requiredAcks   |        need a return receipt        | <div>When producing a message, the broker returns the message acknowledgment (ack) mode, which supports the following 3 values（0/1/-1):<br>0: NoResponse，No need to wait for broker response.<br>1: WaitForLocal, wait for the response from the local (leader node) to return.<br>-1: WaitForAll, wait for all nodes (leader node and all In Sync Replication follower nodes) to respond and return. (Defaults)</div> |
|    maxRetry     |         failed max retries          | The maximum number of retries for production messages, the default is 3 times, note: must be greater than or equal to 0, otherwise an error will be reported.                                                                                                                                                                                                                                                            |
|  retryInterval  |       failure retry interval        | The unit is milliseconds, the default is 100ms.                                                                                                                                                                                                                                                                                                                                                                          |
|    trpcMeta     | transfer trpc meta to sarama header | true means enable transfer, false means not transfer, the default is false                                                                                                                                                                                                                                                                                                                                               |
|    discover     |     for service discovery type      | For example：polaris                                                                                                                                                                                                                                                                                                                                                                                                     |
|    namespace    |          service namespace          | For example：Development                                                                                                                                                                                                                                                                                                                                                                                                 |
|   idempotent    |   to start generators idempotent    | true/false,the default is false ｜                                                                                                                                                                                                                                                                                                                                                                                       |

### consumer

```yaml
server:
  service:
    - name: trpc.app.server.consumer
      address: ip1:port1,ip2:port2?topics=topic1,topic2&group=xxx&version=x.x.x.x
      protocol: kafka
      timeout: 1000
    - name: trpc.app.server.consumer1   # Polaris Naming access does not support dynamic changes. If the Polaris IP changes, the service needs to be restarted to take effect. If you need to get all Polaris nodes, you need to pay attention to Q14.
      address: YOUR_SERVICE_NAME?topics=topic1,topic2&group=xxx&version=x.x.x.x&discover=polaris&namespace=Development
      protocol: kafka                                            
      timeout: 1000    # Maximum request processing time unit milliseconds
```

|    parameter name     |                                                                                                                    meaning                                                                                                                     |                                                                                                                                        Optional value & description                                                                                                                                        |
| :-------------------: | :--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------: | :--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------: |
|     ip:port list      |                                                                                                                    address                                                                                                                     |                                                                                                     Separate multiple addresses with commas(','),support domain:port, ip:porta. Not supported yet cl5.                                                                                                     |
|         group         |                                                                                                                 consumer group                                                                                                                 |                                                                                                                    If it is [wormhole] kafka, need to register on the management page.                                                                                                                     |
|       clientid        |                                                                                                                   client id                                                                                                                    |                                                                                                                                          connect kafka client id                                                                                                                                           |
|        topics         |                                                                                                                 consumer toipc                                                                                                                 |                                                                                                                                        multiple separated by commas                                                                                                                                        |
|      compression      |                                                                                                               compression method                                                                                                               |                                                                                                                                                                                                                                                                                                            |
|       strategy        |                                                                                                                    strategy                                                                                                                    |                                                                                 <div>sticky: sarama.BalanceStrategySticky;<br>range : sarama.BalanceStrategyRange;<br>roundrobin: sarama.BalanceStrategyRoundRobin;</div>                                                                                  |
|     fetchDefault      | The default size (bytes) of the pulled message. If the actual size of the message is larger than this value, the memory space needs to be reallocated, which will affect performance. It is equivalent to sarama.Config.Consumer.Fetch.Default |                                                                                                                                               default 524288                                                                                                                                               |
|       fetchMax        |                The maximum size (bytes) of the message to be pulled. If the actual size of the message is greater than this value, an error will be reported directly, which is equivalent to sarama.Config.Consumer.Fetch.Max                 |                                                                                                                                              default 1048576                                                                                                                                               |
|         batch         |                                                                                                                  batch number                                                                                                                  |                                        It is required when using batch consumption. When registering a batch consumption function, if the batch is not filled in, the parameters will not match and the consumption will fail. Use reference examples/batchconsumer                                        |
|      batchFlush       |                                                                                                                batch flust time                                                                                                                |                                                                       The default is 2 seconds, the unit is ms, which means the interval of forced consumption when the batch consumption does not meet the maximum number of items                                                                        |
|        initial        |                                                                                                           initial consumer location                                                                                                            |                                                                      <div>The location where the new consumer group connects to the cluster consumer for the first time<br>newest: latest location<br> oldest: oldest position</div>                                                                       |
|      maxWaitTime      |                                                                                         The maximum waiting time for a single consumption pull request                                                                                         |                                                                                                            The maximum waiting time will only wait when there is no latest data, the default 1s                                                                                                            |
|       maxRetry        |                                                                                                               failed max retries                                                                                                               | After exceeding, directly confirm and continue to consume the next message, default 0: no limit, keep retrying, negative number means no retry, directly confirm and continue to consume the next message, positive number means if there is always an error, the final number of executions is maxRetry+1 |
|  netMaxOpenRequests   |                                                                                                    Maximum number of simultaneous requests                                                                                                     |                                                                                                              Network layer configuration, maximum number of simultaneous requests, default 5                                                                                                               |
|   maxProcessingTime   |                                                                                                 The maximum request time for a single consumer                                                                                                 |                                                                                                                                          Unit ms, default 1000ms                                                                                                                                           |
|    netDailTimeout     |                                                                                                                connect timeout                                                                                                                 |                                                                                                                   Network layer configuration, connect timeout, unit ms, default 30000ms                                                                                                                   |
|    netReadTimeout     |                                                                                                                  read timeout                                                                                                                  |                                                                                                                    Network layer configuration, read timeout, unit ms, default 30000ms                                                                                                                     |
|    netWriteTimeout    |                                                                                                                 write timeout                                                                                                                  |                                                                                                                    Network layer configuration, write timeout, unit ms, default 30000ms                                                                                                                    |
|  groupSessionTimeout  |                                                                                                         consumer group session timeout                                                                                                         |                                                                                                                                          Unit ms, default 10000ms                                                                                                                                          |
| groupRebalanceTimeout |                                                                                                           consumer rebalance timeoue                                                                                                           |                                                                                                                                          Unit ms, default 60000ms                                                                                                                                          |
|       mechanism       |                                                                                                     Encryption method when using password                                                                                                      |                                                                                                                                 optional value SCRAM-SHA-512/SCRAM-SHA-256                                                                                                                                 |
|         user          |                                                                                                                      user                                                                                                                      |                                                                                                                                                                                                                                                                                                            |
|       password        |                                                                                                                    password                                                                                                                    |                                                                                                                                                                                                                                                                                                            |
|     retryInterval     |                                                                                                                 retry interval                                                                                                                 |                                                                                                                                           Unit ms, default3000ms                                                                                                                                           |
|    isolationLevel     |                                                                                                                isolation level                                                                                                                 |                                                                                                                                optional value ReadUncommitted/ReadCommitted                                                                                                                                |
|       trpcMeta        |                                                                                            transfer trpc meta, read sarama header to set trpc meta                                                                                             |                                                                                                                 true means enable transfer, false means not transfer, the default is false                                                                                                                 |
|       discover        |                                                                                                 The discovery type used for service discovery                                                                                                  |                                                                                                                                            For example：polaris                                                                                                                                            |
|       namespace       |                                                                                                               service namespace                                                                                                                |                                                                                                                                          For example：Development                                                                                                                                          |

## Q&A

- Q1: How to write the service.name of the consumer
- A1: If there is only one consumer service, The name can be arbitrary (the trpc framework will register the implementation to all services in the server by default), see complete examples/consumer

``` yaml
server:                                                                  
  service:                                                               
    - name: trpc.anyname.will.works                                
      address: 127.0.0.1:9092?topics=test_topic&group=uzuki_consumer 
      protocol: kafka                                                    
      timeout: 1000                                                      
```

``` go
s := trpc.NewServer()
kafka.RegisterKafkaHandlerService(s, handle) 
```

If there are multiple services, you need to specify the same name as the configuration file when registering, see complete examples/consumer_with_mulit_service

``` yaml
server:                                                                   
  service:                                                                
    - name: trpc.databaseDemo.kafka.consumer1                             
      address: 127.0.0.1:9092?topics=test_topic&group=uzuki_consumer1 
      protocol: kafka                                                     
      timeout: 1000                                                       
    - name: trpc.databaseDemo.kafka.consumer2                             
      address: 127.0.0.1:9092?topics=test_topic&group=uzuki_consumer2 
      protocol: kafka                                                     
      timeout: 1000     
```

``` go
s := trpc.NewServer()    
kafka.RegisterKafkaConsumerService(s.Service("trpc.databaseDemo.kafka.consumer1"), &Consumer{})
kafka.RegisterKafkaConsumerService(s.Service("trpc.databaseDemo.kafka.consumer2"), &Consumer{})
```

- Q2: What happens if handle returns non-nil when consuming

- A2: It will resume consumption after sleeping for 3s. It is not recommended to do so. If it fails, the business should do the retry logic

- Q3: When using ckafka to produce messages, error hint

``` log
err:type:framework, code:141, msg:kafka client transport SendMessage: kafka server: Message contents does not match its CRC.
```

- A3: By default, gzip compression is enabled, and it is preferred to add parameters to the target**compression=none**

``` yaml
target: kafka://ip1:port1,ip2:port2?clientid=xxx&compression=none
```

- Q4: When using ckafka to consume messages, error hint

``` log
kafka server transport: consume fail:kafka: client has run out of available brokers to talk to (Is your cluster reachable?)
```

- A4: First check whether the brokers are reachable, and then check the supported kafka client version, try to add parameters in the configuration file address, for example**version=0.10.2.0**

``` yaml
address: ip1:port1,ip2:port2?topics=topic1,topic2&group=xxx&version=0.10.2.0 
```

- Q5: When consuming messages, error hint

``` log
kafka server transport: consume fail:kafka server: The provider group protocol type is incompatible with the other members.
```

- A5: The client regrouping strategy of the same consumer group is different, the parameter **strategy** can be modified, optional value:sticky(default)，range，roundrobin

``` yaml
address: ip1:port1,ip2:port2?topics=topic1,topic2&group=xxx&strategy=range
```

- Q6: The same user needs to be ordered in production, how to configure
- A6: The client adds the parameter **partitioner**, optional random (default), roundrobin, hash (partitioned by key)

``` yaml
target: kafka://ip1:port1,ip2:port2?clientid=xxx&partitioner=hash
```

- Q7: How to produce asynchronously
- A7: The client config adds the parameter **async=1**

``` yaml
target: kafka://ip1:port1,ip2:port2?clientid=xxx&async=1
```

- Q8: Having error with Polaris routing "Polaris-1006(ErrCodeServerError)" "not found service"
- A8: Make sure that the service.name in the trpc configuration is trpc.**${app}.${server}**.AnyUniqNameWillWork instead of trpc.**app.server**.AnyUniqNameWillWork, a placeholder must be used
Error site:

``` log
type:framework, code:131, msg:client Select: get source service route rule err: Polaris-1006(ErrCodeServerError): Response from {ID: 2079470528, Service: {ServiceKey: {namespace: "Polaris", service: "polaris.discover"}, ClusterType: discover}, Address: 127.0.0.1:8081}: not found service
```

- Q9: How to use account and password
- A9: The encryption method, username and password need to be configured in the connection parameters
For example ：

``` yaml
address: ip1:port1,ip2:port2?topics=topic1,topic2&mechanism=SCRAM-SHA-512&user={user}&password={password}
```

- Q10: How to use asynchronous production write data callback
- A10: It is necessary to rewrite the success/failure callback function of asynchronous production write data in the code, for example

```go
import ( 
  // ... 
  "trpc.group/trpc-go/trpc-database/kafka"
  // ...
)

func init() {
  // Override the default asynchronous production write data error callback
  kafka.AsyncProducerErrorCallback = func(err error, topic string, key, value []byte, headers []sarama.RecordHeader) {
    // do something if async producer occurred error.
  }

  // Override the default asynchronous production write data success callback
  kafka.AsyncProducerSuccCallback = funcfunc(topic string, key, value []byte, headers []sarama.RecordHeader) {
    // do something if async producer succeed.
  }
}
```

- Q11: How to inject custom configuration (remote configuration)
- A11: Configure `fake_address` in `trpc_go.yaml`, and then inject it with `kafka.RegisterAddrConfig` method
`trpc_go.yaml` The configuration is as follows

``` yaml
address: fake_address 
```

Before the service starts, inject custom configuration

```go
func main() {
  s := trpc.NewServer()
  // Use a custom addr, which needs to be injected before starting the server
  cfg := kafka.GetDefaultConfig()
  cfg.Brokers = []string{"127.0.0.1:9092"}
  cfg.Topics = []string{"test_topic"}
  kafka.RegisterAddrConfig("fake_address", cfg)
  kafka.RegisterKafkaConsumerService(s, &Consumer{})

  s.Serve()
}
```

- Q12: How to transfer trpc metadata
- A12: Producers and consumers add the parameter trpcMeta=true, and transparent transmission is not enabled by default; if the original production interface has already set headers, you need to pay attention to header conflicts, duplication, overwriting and other issues;

producer

``` yaml
target: kafka://127.0.0.1:9092?clientid=test_producer&partitioner=hash&topic=test_topic&trpcMeta=true
```

consumer

``` yaml
address: 127.0.0.1:9092?topics=test_topic&group=test_group&trpcMeta=true     #kafka consumer dsn
```

- Q13: How to get the context information of the underlying sarama
- A13: The underlying sarama ConsumerGroupSession and ConsumerGroupClaim can be obtained through kafka.GetRawSaramaContext. However, the exposure of these two interfaces here is only for the convenience of users to monitor logs, and only the read method should be used. Calling any write method is undefined behavior here, which may cause unknown results

```go
// RawSaramaContext deposit sarama ConsumerGroupSession and ConsumerGroupClaim
// This structure is exported for the convenience of users to implement monitoring, the content provided is only for reading, calling any write method is an undefined behavior
type RawSaramaContext struct {
    Session  sarama.ConsumerGroupSession
    Claim sarama.ConsumerGroupClaim
}
```

use case

```go
func example(ctx context.Context){
    if rawContext, ok := kafka.GetRawSaramaContext(ctx); ok {
        log.Infof("InitialOffset:%d", rawContext.Claim.InitialOffset())
    }
}

```

- Q14: How to obtain all nodes in server-side Polaris addressing
- A14: Enable `service_router.need_return_all_nodes=true` in `trpc_go.yaml`

```yaml
plugins:     # plugin configuration
  selector:
    polaris:
      service_router:
        need_return_all_nodes: true
```
