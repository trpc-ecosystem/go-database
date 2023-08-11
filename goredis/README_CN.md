# tRPC-Go go-redis 插件

[English](README.md) | 中文

## 原理 
对开源的 github.com/go-redis/redis 利用hook技术进行扩展, 从而增加对于trpc生态的支持。

## 使用
### 1 配置 trpc_go.yaml 文件
```yaml
client:                                            #客户端调用的后端配置
  service:                                         #业务服务提供的service，可以有多个
    - name: trpc.gamecenter.test.redis      	   #后端服务的service name
      target: redis://127.0.0.1:6379               #请求服务地址格式：redis://<user>:<password>@<host>:<port>/<db_number>
      timeout: 60000
```

### 2 测试用例
```go
// Test_New redis 初始化
func Test_New(t *testing.T) {
	// 扩展接口
	cli, err := New("trpc.gamecenter.test.redis")
	if err != nil {
		t.Fatalf("new fail err=[%v]", err)
	}
	// go-redis Set
	result, err := cli.Set(trpc.BackgroundContext(), "key", "value", 0).Result()
	if err != nil {
		t.Fatalf("set fail err=[%v]", err)
	}
	t.Logf("Set result=[%v]", result)
	// go-redis Get
	value, err := cli.Get(trpc.BackgroundContext(), "key").Result()
	if err != nil {
		t.Fatalf("get fail err=[%v]", err)
	}
	t.Logf("Get value=[%v]", value)
}
```

# url 格式
```html
  A: <scheme>://<user>:<password>@<host>:<port>/<db_number>?<is_proxy=true><?min_idle_conns=10>
```

| 字段 | 解释                                                                                                                                                                                                                                                                                                               | 例子                                                     |
| --- |------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|--------------------------------------------------------|
| scheme | 名字解析模式                                                                                                                                                                                                                                                                                                           | redis，rediss，polaris，ip                                |
| user | 用户名                                                                                                                                                                                                                                                                                                              | 平台用作权限控制，没有不填，但是后面的`:`需要带上                             |
| password | 密码                                                                                                                                                                                                                                                                                                               | 有特殊字符需要进行 url encode                                   |
| host,port | ip和端口                                                                                                                                                                                                                                                                                                            | 服务地址，集群版多个地址用`,`分割，北极星直接填服务名                           |
| db_number | 数据库号码                                                                                                                                                                                                                                                                                                            | 不同的数据库数据是相互隔离的，可以用来区分`测试`或`正式`环境 或 不同的`业务`             |
| is_proxy | 代理模式                                                                                                                                                                                                                                                                                                             | 主要是为了兼容非标准的redis集群 |
| context_timeout_enabled | 是否启用context超时，默认开启。                                                                                                                                                                                                                                                                                              |  |
| master_name | 主机名                                                                                                                                                                                                                                                                                                              | 哨兵模式主机名 |
| client_name | will execute the `CLIENT SETNAME ClientName` command for each conn.                                                                                                                                                                                                                                              |  |
| max_retries | Maximum number of retries before giving up. <br>Default is 3 retries; -1 (not 0) disables retries.                                                                                                                                                                                                               |  |
| min_retry_backoff | Minimum backoff between each retry. <br>Default is 8 milliseconds; -1 disables backoff. 单位毫秒.                                                                                                                                                                                                                    |  |
| max_retry_backoff | Maximum backoff between each retry. <br>Default is 512 milliseconds; -1 disables backoff. 单位毫秒.                                                                                                                                                                                                                  |  |
| dial_timeout | Dial timeout for establishing new connections. <br>Default is 5 seconds. <br>单位毫秒.                                                                                                                                                                                                                               | |
| read_timeout | Timeout for socket reads. <br>If reached, commands will fail with a timeout instead of blocking. Supported values: <br>- `0` - default timeout (3 seconds). - <br>`-1` - no timeout (block indefinitely). - <br>`-2` - disables SetReadDeadline calls completely. <br>单位毫秒.                                      | |
| write_timeout | Timeout for socket writes. <br>If reached, commands will fail with a timeout instead of blocking. Supported values: <br>- `0` - default timeout (3 seconds). - <br>`-1` - no timeout (block indefinitely). - <br>`-2` - disables SetWriteDeadline calls completely. <br>单位毫秒.                                    | |
| pool_fifo | Type of connection pool.<br>true for FIFO pool, false for LIFO pool.<br>Note that FIFO has slightly higher overhead compared to LIFO,<br>but it helps closing idle connections faster reducing the pool size.                                                                                                    |  |
| pool_size | Maximum number of socket connections. Default is 250 connections per every available CPU as reported by runtime.GOMAXPROCS.                                                                                                                                                                                      | 默认 250 * cpu |
| pool_timeout | Amount of time client waits for connection if all connections <br>are busy before returning an error.<br>Default is ReadTimeout + 1 second.                                                                                                                                                                      ||
| min_idle_conns | Minimum number of idle connections which is useful when establishing<br>new connection is slow.                                                                                                                                                                                                                  ||
| max_idle_conns | Maximum number of idle connections.                                                                                                                                                                                                                                                                              ||
| conn_max_idle_time | ConnMaxIdleTime is the maximum amount of time a connection may be idle. Should be less than server's timeout. <br>Expired connections may be closed lazily before reuse. If d <= 0, connections are not closed due to a connection's idle time. <br>Default is 30 minutes. -1 disables idle timeout check. 单位毫秒. ||
| conn_max_lifetime | ConnMaxLifetime is the maximum amount of time a connection may be reused. <br>Expired connections may be closed lazily before reuse. If <= 0, connections are not closed due to a connection's age.<br> Default is to not close idle connections.                                                                ||
| max_redirects | The maximum number of retries before giving up. Command is retried <br>on network errors and MOVED/ASK redirects. <br>Default is 3 retries.                                                                                                                                                                      ||
| read_only | Enables read-only commands on slave nodes.                                                                                                                                                                                                                                                                       ||
| route_by_latency | Allows routing read-only commands to the closest master or slave node. It automatically enables ReadOnly.                                                                                                                                                                                                        ||
| route_randomly | Allows routing read-only commands to the random master or slave node. <br>It automatically enables ReadOnly.                                                                                                                                                                                                     ||

* 更多字段：https://trpc.group/trpc-go/trpc-database/blob/master/goredis/internal/proto/goredis.proto#L30
* 参考：https://github.com/redis/go-redis/blob/master/options.go#L221
* 例如：`redis://:password@9.xx.xx.252:6380/15?is_proxy=true`

# mock 参考
* https://github.com/go-redis/redismock

# 常见问题
* Q: cmd not support <br>
  A: ckv+，istore 等redis属于proxy模式，不是redis集群模式，解决方案是：<br>
     在target的最后增加选项 `?is_proxy=true`，如下：<br>
```yaml
target: polaris://:password@trpc.xxx_lite_test.redis.com/0?is_proxy=true
```
  注：代理模块支持只支持 polaris 北极星名字解析，代理模式主调监控中地址是固定的一个随机地址。
* Q: 密码中存在特殊字符例如：`@`,`:`,`?`,`=`等如何处理？<br>
  A: 对密码进行：url encode
* Q: galileo 监控没有数据，但是日志可以打印？<br>
  A: New 参数 name 字段需要四个字段组合而成，否则不兼容；
* Q：集群版地址如何输入？<br>
  A：集群版地址传输用","分割，如；`redis://user:password@127.0.0.1:6379,127.0.0.1:6380/0`
