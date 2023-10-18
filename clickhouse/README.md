English | [中文](README.zh_CN.md)

# tRPC-Go clickhouse plugin

## Clickhouse client configuration
> This project depends on clickhouse-go's v2 version, which is compatible with [v1 version dsn](https://github.com/ClickHouse/clickhouse-go /tree/v1#dsn), in order to prevent omissions, it is recommended to use [v2 version of dsn](https://github.com/ClickHouse/clickhouse-go#dsn).

> Please note that the v2 version has a strong limitation on the type. The type of the written data must completely correspond to the type defined in the table. For example, if the table defines UInt8, the written data type must be uint8, and other digital types will also report errors.

> In addition, 3 connection parameter configurations ```max_idle, max_open, max_lifetime``` have been added to dsn, which is convenient for users to customize connection configuration.

```yaml
client:  #  Backend configuration for client calls.
  service:  #  Configuration for the backend.
    - name: trpc.clickhouse.ip.v1
      target: clickhouse://ip:port?username=*&password=*&database=*  #  clickhouse standard uri: tcp://host1[:port1][?options] , see https://github.com/ClickHouse/clickhouse-go/tree/v1#dsn
      timeout: 800  #  The maximum processing time of the current request.
    - name: trpc.clickhouse.name.v1
      target: clickhouse+polaris://polaris_name?username=*&password=*&database=*  #  clickhouse+polaris indicates that the host in the clickhouse uri will perform Polaris analysis.
      timeout: 800  #  The maximum processing time of the current request.
    - name: trpc.clickhouse.ip.v2
      target: clickhouse://username:password@ip:port/database?dial_timeout=200ms  #  clickhouse standard uri: clickhouse://username:password@host1[:port1]/database[?options], see https://github.com/ClickHouse/clickhouse-go#dsn
      timeout: 800  #  The maximum processing time of the current request.
    - name: trpc.clickhouse.name.v2
      target: clickhouse+polaris://username:password@polaris_name/database?max_idle=10&max_open=100&max_lifetime=3m  #  clickhouse+polaris indicates that the host in the clickhouse uri will perform Polaris analysis.
      timeout: 800  #  The maximum processing time of the current request.
```

## Test (see clickhouse_test.go file for detailed test code)
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
    proxy := clickhouse.NewClientProxy("trpc.clickhouse.xxx.xxx") // The service name is customized 
	                                                              // and is mainly used for monitoring reporting and addressing configuration items。

    var results []*TableRow
    err := proxy.QueryToStructs(ctx, &results, "SELECT country_code, os_id, browser_id, categories, action_day, action_time FROM example LIMIT 5")
    if err != nil {
        return err
    }
    for _, val := range results {
        fmt.Printf("%+v", val)
    }

    // Business logic
    // ......

    return
}
```

## Q&A
1. Error message: err: [hello] unexpected packet [89] from server
   Answer: The port configuration in the target is wrong. Clickhouse supports 3 connection methods, using different ports respectively. The native tcp default port [9000], the MySQL default port [9004], and the Http default port [8123]. This client plug-in only supports native In tcp[9000] mode, the service uri port needs to be adjusted.
2. Error message: err: type:framework, code:141, msg:driver: bad connection
   Answer: The SQL query takes longer than the timeout in the ```clickhouse client configuration```, the context context will be terminated after the timeout expires, causing the query request to stop and an error is reported, and the timeout configuration can be adjusted according to the situation.
