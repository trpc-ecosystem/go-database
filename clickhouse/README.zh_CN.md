[English](README.md) | 中文

# tRPC-Go clickhouse 插件

[![Go Reference](https://pkg.go.dev/badge/trpc.group/trpc-go/trpc-database/clickhouse.svg)](https://pkg.go.dev/trpc.group/trpc-go/trpc-database/clickhouse)
[![Go Report Card](https://goreportcard.com/badge/trpc.group/trpc-go/trpc-database/clickhouse)](https://goreportcard.com/report/trpc.group/trpc-go/trpc-database/clickhouse)
[![Tests](https://github.com/trpc-ecosystem/go-database/actions/workflows/clickhouse.yml/badge.svg)](https://github.com/trpc-ecosystem/go-database/actions/workflows/clickhouse.yml)
[![Coverage](https://codecov.io/gh/trpc-ecosystem/go-database/branch/main/graph/badge.svg?flag=clickhouse&precision=2)](https://app.codecov.io/gh/trpc-ecosystem/go-database/tree/main/clickhouse)

## clickhouse client 配置
> 本项目依赖官方库 clickhouse-go 的 v2 版本，兼容了 [v1 版本的 dsn](https://github.com/ClickHouse/clickhouse-go/tree/v1#dsn)，为防止有疏漏，建议使用 [v2 版本的 dsn](https://github.com/ClickHouse/clickhouse-go#dsn)。

> 请注意 v2 版本对类型做了强限定，写入数据的类型必须完全对应表中定义的类型，如表定义的 UInt8，则写入数据类型必须是 uint8，其它数字类型也会报错。

> 此外 dsn 中增加了 3 个连接参数配置 ```max_idle , max_open, max_lifetime```, 便于用户自定义连接配置。

```yaml
client:  #  客户端调用的后端配置
  service:  #  针对后端的配置
    - name: trpc.clickhouse.ip.v1
      target: clickhouse://ip:port?username=*&password=*&database=*  #  clickhouse 标准 uri: tcp://host1[:port1][?options] , 参考 https://github.com/ClickHouse/clickhouse-go/tree/v1#dsn
      timeout: 800  #  当前这个请求最长处理时间
    - name: trpc.clickhouse.name.v1
      target: clickhouse+polaris://polaris_name?username=*&password=*&database=*  #  clickhouse+polaris 表示 clickhouse uri 中的 host 会进行北极星解析
      timeout: 800  #  当前这个请求最长处理时间
    - name: trpc.clickhouse.ip.v2
      target: clickhouse://username:password@ip:port/database?dial_timeout=200ms  #  clickhouse 标准 uri: clickhouse://username:password@host1[:port1]/database[?options], 参考 https://github.com/ClickHouse/clickhouse-go#dsn
      timeout: 800  #  当前这个请求最长处理时间
    - name: trpc.clickhouse.name.v2
      target: clickhouse+polaris://username:password@polaris_name/database?max_idle=10&max_open=100&max_lifetime=3m  #  clickhouse+polaris 表示 clickhouse uri 中的 host 会进行北极星解析
      timeout: 800  #  当前这个请求最长处理时间
```

## 测试（详细测试代码请查看 clickhouse_test.go 文件）
```go
package main

import (
    "fmt"
    "time"
    "context"

    "trpc.group/trpc-go/trpc-database/clickhouse"
)

type TableRow struct {
    Country    string    `db:"country_code"`
    Os         uint8     `db:"os_id"`
    Browser    uint8     `db:"browser_id"`
    Categories []int16   `db:"categories"`
    ActionDay  time.Time `db:"action_day"`
    ActionTime time.Time `db:"action_time"`
}

func (s *server) SayHello(ctx context.Context, req *pb.ReqBody, rsp *pb.RspBody) (err error) {
    proxy := clickhouse.NewClientProxy("trpc.clickhouse.xxx.xxx") // service name 自己随便填，主要用于监控上报和寻址配置项

    var results []*TableRow
    err := proxy.QueryToStructs(ctx, &results, "SELECT country_code, os_id, browser_id, categories, action_day, action_time FROM example LIMIT 5")
    if err != nil {
        return err
    }
    for _, val := range results {
        fmt.Printf("%+v", val)
    }

    // 业务逻辑
    // ......

    return
}
```

## Q&A
1. 错误信息：err: [hello] unexpected packet [89] from server
   答：target 中的端口配置错误，clickhouse 支持 3 种连接方式，分别使用不同端口，原生 tcp 默认端口 [9000]、MySQL 默认端口 [9004]、Http 默认端口 [8123]，本客户端插件仅支持原生 tcp[9000] 模式，需要调整服务 uri 端口。
2. 错误信息：err: type:framework, code:141, msg:driver: bad connection
   答：SQL 查询耗时超过了```clickhouse client 配置中的 timeout```，context 上下文在 timeout 超时后会被终止，导致查询请求停止而报错，可以视情况调整 timeout 的配置。
