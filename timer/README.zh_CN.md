[English](README.md) | 中文

# tRPC-Go timer 定时器插件

[![Coverage](https://codecov.io/gh/trpc-ecosystem/go-database/branch/coverage/graph/badge.svg?flag=timer&precision=2)](https://app.codecov.io/gh/trpc-ecosystem/go-database/tree/coverage/timer)

## timer service

```yaml
server:                                            #服务端配置
  service:                                         #业务服务提供的service，可以有多个
    - name: trpc.app.server.service                #service的路由名称,自己随便定义，由于监控上报，如果是使用123平台，需要使用trpc.${app}.${server}.service
      nic: eth1                                    #网卡 ipport随便填，主要用于分布式定时器的互斥
      port: 9001                                   #服务监听端口 可使用占位符 ${port}
      network: "0 */5 * * * *"                     #定时器cron表达式: [second minute hour day month weekday] 如 "0 */5 * * * *" 每隔5分钟
      protocol: timer                              #应用层协议 
      timeout: 1000                                #请求最长处理时间 单位 毫秒
```

```go
package main

import (
	"context"
	
	"trpc.group/trpc-go/trpc-database/timer"

	trpc "trpc.group/trpc-go/trpc-go"
)

func main() {

	s := trpc.NewServer()
	// 启动多个定时器的情况，可以配置多个service，然后这里任意匹配 timer.RegisterHandlerService(s.Service("name"), handle)，没有指定name的情况，代表所有service共用同一个handler
	timer.RegisterHandlerService(s, handle)
	s.Serve()
}

func handle(ctx context.Context) error {

	return nil
}
```

## 任务调度器
timer默认是本地定时器，即所有节点都会启动定时任务，各个节点都是相互独立的

定时任务采用cron秒级语法，如需在程序启动时马上执行需要配置network = "0 */5 * * * *?startAtOnce=1"，这样首次执行失败会阻止程序启动起来

如需分布式互斥任务，即同一个服务，同一个时刻，只会有一个节点执行，可以自己实现Scheduler并注册进来
```golang
// Scheduler 定时器调度器
type Scheduler struct {
}

// Schedule 自己通过数据存储实现互斥任务定时器
// serviceName 服务进程名，表示一个服务，一个服务下面可以部署多个节点
// newNode 各个节点的唯一id  ip:port_pid_timestamp
// holdTime 任务抢占的有效时间 1s
// nowNode 返回抢占成功的节点id
// err 抢占失败则返回err，当前节点不执行任务
func (s *Scheduler) Schedule(serviceName string, newNode string, holdTime time.Duration) (nowNode string, err error) {
	// 如 可使用redis的setnx来实现
	// setnx serviceName newNode ex expireSeconds
	// if fail, nowNode = get serviceName
	return nowNode, nil
}
```
```golang
func main() {
	timer.RegisterScheduler("name", &Scheduler{})

	s := trpc.NewServer()
	timer.RegisterHandlerService(s, handle)
	s.Serve()
}
```
```yaml
server:                                            #服务端配置
  service:                                         #业务服务提供的service，可以有多个
    - name: trpc.app.server.service
      network: "0 */1 * * * *?scheduler=name"
      protocol: timer                              #应用层协议
```

## 参数说明


| 参数名称        | 含义         | 可选值        |
|:-----------:|:----------:|:----------:|
| Schedule    | 任务定时器      | 自行注册       |
| ServiceName | 服务进程名称     | 自定义        |
| holdTime    | 任务抢占的有效时间  | 单位：秒       |
| StartAtOnce | 是否在启动程序时执行 | 1:执行；0:不执行 |
