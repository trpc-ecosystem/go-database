# tRPC-GO hbase 插件

[![Go Reference](https://pkg.go.dev/badge/trpc.group/trpc-go/trpc-database/hbase.svg)](https://pkg.go.dev/trpc.group/trpc-go/trpc-database/hbase)
[![Go Report Card](https://goreportcard.com/badge/trpc.group/trpc-go/trpc-database/hbase)](https://goreportcard.com/report/trpc.group/trpc-go/trpc-database/hbase)
[![Tests](https://github.com/trpc-ecosystem/go-database/actions/workflows/hbase.yml/badge.svg)](https://github.com/trpc-ecosystem/go-database/actions/workflows/hbase.yml)
[![Coverage](https://codecov.io/gh/trpc-ecosystem/go-database/branch/main/graph/badge.svg?flag=hbase&precision=2)](https://app.codecov.io/gh/trpc-ecosystem/go-database/tree/main/hbase)

封装 [hbase](https://github.com/tsuna/gohbase)，配合 trpc 使用

## hbase client 框架配置
```yaml
- name: trpc.app.hbase.sevice
  callee: trpc.app.hbase.sevice
  protocol: hbase
  target: hbase://10.0.0.100:2181,10.0.0.101:2181?zookeeperRoot=/root&zookeeperTimeout=1000&regionLookupTimeout=1000&regionReadTimeout=1000&effectiveUser=root
```

## 注意事项

gohbase 使用了 logrus 会打一些 debug 日志，hbase.SetLogLevel(log.LevelXXX) 这个用来减少 hbase client 连接过程中的日志

## 用法
```go
proxy := hbase.NewClientProxy(req.Name) // 请定义为全局变量

// get
table := "tableName"
rowKey := "rowKey"
// 列族->查询列数组
family := make(map[string][]string)
result, err := hbase.ParseGetResult(proxy.Get(ctx, table, rowKey, family))

// put
table := "tableName"
rowKey := "rowKey"
// 列族 -> [列 -> 值]
family := make(map[string]map[string][]byte)
putRsp, err := proxy.Put(ctx, req.Table, req.RowKey, family)

// rawGet -------
family := map[string][]string{
  "family": {"qualifier1", "qualifier2"},
}
get, err := hrpc.NewGetStr(ctx, table, rowKey, hrpc.Families(family))
// judge err
result, err := proxy.RawGet(ctx, get)

// rawPut -------
values := map[string]map[string][]byte{
  "family": {
    "qualifier1": []byte("1"),
    "qualifier2": []byte("str"),
  },
}

put, err := hrpc.NewPutStr(ctx, table, rowKey, values)
// judge err
result, err := proxy.RawPut(ctx, put)

// rawInc -------
inc, err := hrpc.NewIncStrSingle(ctx, table, rowKey, "family", "qualifier1", 2)
// judge err
result, err := proxy.RawInc(ctx, inc)

// del -------
values := map[string]map[string][]byte{
  "family": {
    "qualifier1": nil,
  },
}

// way 1
result, err := proxy.Del(ctx, table, rowKey, values)

// way 2
del, err := hrpc.NewDelStr(ctx, table, rowKey, values)
// judge err
result, err = proxy.RawDel(ctx, del)
```