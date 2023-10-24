English | [中文](README.zh_CN.md)

# tRPC-Go timer plugin

[![Go Reference](https://pkg.go.dev/badge/trpc.group/trpc-go/trpc-database/timer.svg)](https://pkg.go.dev/trpc.group/trpc-go/trpc-database/timer)
[![Go Report Card](https://goreportcard.com/badge/trpc.group/trpc-go/trpc-database/timer)](https://goreportcard.com/report/trpc.group/trpc-go/trpc-database/timer)
[![Tests](https://github.com/trpc-ecosystem/go-database/actions/workflows/timer.yml/badge.svg)](https://github.com/trpc-ecosystem/go-database/actions/workflows/timer.yml)
[![Coverage](https://codecov.io/gh/trpc-ecosystem/go-database/branch/coverage/graph/badge.svg?flag=timer&precision=2)](https://app.codecov.io/gh/trpc-ecosystem/go-database/tree/coverage/timer)

## timer service

```yaml
server:                                            # Server configuration.
  service:                                         # The service provided by the business service can have multiple.
    - name: trpc.app.server.service                # The routing name of the service can be defined by yourself. For monitor reports, you need to use the name of trpc.${app}.${server}.service if you use the 123 platform.
      nic: eth1                                    # Network can be filled in casually, mainly used for mutual exclusion of distributed.
      port: 9001                                   # Service listening port can use placeholder ${port}.
      network: "0 */5 * * * *"                     # Timer cron expression: [second minute hour day month weekday], like: "0 */5 * * * *" means every 5 minutes.
      protocol: timer                              # Application layer protocol.
      timeout: 1000                                # Maximum request processing time, in milliseconds.
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
    // When starts multiple timers, you can configure multiple services, and use timer.RegisterHandlerService(s.Service("name"), handle) to configure the matching relationship between service and handler.
    // If no name is specified, it means that all services will share the same handler.
	timer.RegisterHandlerService(s, handle)
	s.Serve()
}

func handle(ctx context.Context) error {

	return nil
}
```

## task scheduler
The default timer is a local timer, which all nodes will start timer tasks, and each node is independent of each other.

Timer task uses cron second-level syntax. If you want to execute immediately when the program starts, you need to configure: network = "0 */5 * * * *?startAtOnce=1", so that the first execution failure will prevent the program to start.

If you need distributed mutual exclusion tasks, which the same service, the same time, only one node will execute, you can implement the Scheduler and register it.
```golang
// Scheduler is the timer scheduler.
type Scheduler struct {
}

// Schedule uses data storage to implement mutual exclusion task timer.
// serviceName is the service process name, it represents a service, and multiple nodes can be deployed under a service.
// newNode is the unique id of each node, and its format is: ip:port_pid_timestamp.
// holdTime is the effective time of task preemption, in seconds.
// nowNode returns the node id which preempts successfully.
// err returns err when preempts failed, and the current node does not execute tasks.
func (s *Scheduler) Schedule(serviceName string, newNode string, holdTime time.Duration) (nowNode string, err error) {
    // For example, it can be realized by using setnx of redis.
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
server:                                            # Server configuration.
  service:                                         # The service provided by the business service can have multiple.
    - name: trpc.app.server.service
      network: "0 */1 * * * *?scheduler=name"
      protocol: timer                              # Application layer protocol.
```

## parameter description


| parameter name        | meaning         | optional value        |
|:-----------:|:----------:|:----------:|
| Schedule    | task timer      | self-registration       |
| ServiceName | service process name     | self-defined        |
| holdTime    | effective time of task preemption  | unit: seconds       |
| StartAtOnce | whether to execute when starting the program | 1: execute; 0: not execute |
