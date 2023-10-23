English | [中文](README.zh_CN.md)

# tRPC-Go go-redis plugin

[![Coverage](https://codecov.io/gh/trpc-ecosystem/go-database/goredis/branch/main/graph/badge.svg)](https://app.codecov.io/gh/trpc-ecosystem/go-databse/goredis/tree/main)

## Principle
Use hook technology to expand the open source github.com/go-redis/redis.

## Use
### 1 Configure trpc_go.yaml file
```yaml
client:                                            # Backend configuration for client.
  service:                                         # The service provided by the business service can have multiple.
    - name: trpc.gamecenter.test.redis            # The service name of the backend service.
      target: redis://127.0.0.1:6379              # Request service address format：redis://<user>:<password>@<host>:<port>/<db_number>
      timeout: 60000
```

### 2 Test case
```go
// Test_New redis initialization
func Test_New(t *testing.T) {
	// Extended interface.
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

# url format
```html
  A: <scheme>://<user>:<password>@<host>:<port>/<db_number>?<is_proxy=true><?min_idle_conns=10>
```

| Field                   | Explanation                                                                                                                                                                                                                                                                                                                 | Example                                                                                                                                           |
|-------------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|---------------------------------------------------------------------------------------------------------------------------------------------------|
| scheme                  | Name resolution mode                                                                                                                                                                                                                                                                                                        | redis，rediss，polaris，ip                                                                                                                           |
| user                    | User name                                                                                                                                                                                                                                                                                                                   | The platform is used as permission control, there is no blank, but the following `:` needs to be taken.                                           |
| password                | password                                                                                                                                                                                                                                                                                                                    | There are special characters that need to use url encode.                                                                                         |
| host,port               | ip and port                                                                                                                                                                                                                                                                                                                 | Service address, multiple addresses in the cluster version are separated by `,`, Polaris directly fills in the service name.                      |
| db_number               | Database number                                                                                                                                                                                                                                                                                                             | Different database data are isolated from each other, which can be used to distinguish `test` or `official` environment or different `business`.  |
| is_proxy                | Proxy mode                                                                                                                                                                                                                                                                                                                  | Mainly to be compatible with non-standard redis cluster. |
| context_timeout_enabled | Whether to enable context timeout, it is enabled by default.                                                                                                                                                                                                                                                                |                                                                                                                                                   |
| master_name             | host name                                                                                                                                                                                                                                                                                                                   | Sentry mode hostname.                                                                                                                             |
| client_name             | will execute the `CLIENT SETNAME ClientName` command for each conn.                                                                                                                                                                                                                                                         |                                                                                                                                                   |
| max_retries             | Maximum number of retries before giving up. <br>Default is 3 retries; -1 (not 0) disables retries.                                                                                                                                                                                                                          |                                                                                                                                                   |
| min_retry_backoff       | Minimum backoff between each retry. <br>Default is 8 milliseconds; -1 disables backoff. In milliseconds.                                                                                                                                                                                                                    |                                                                                                                                                   |
| max_retry_backoff       | Maximum backoff between each retry. <br>Default is 512 milliseconds; -1 disables backoff. In milliseconds.                                                                                                                                                                                                                  |                                                                                                                                                   |
| dial_timeout            | Dial timeout for establishing new connections. <br>Default is 5 seconds. <br>In milliseconds.                                                                                                                                                                                                                               |                                                                                                                                                   |
| read_timeout            | Timeout for socket reads. <br>If reached, commands will fail with a timeout instead of blocking. Supported values: <br>- `0` - default timeout (3 seconds). - <br>`-1` - no timeout (block indefinitely). - <br>`-2` - disables SetReadDeadline calls completely. <br>In milliseconds.                                      |                                                                                                                                                   |
| write_timeout           | Timeout for socket writes. <br>If reached, commands will fail with a timeout instead of blocking. Supported values: <br>- `0` - default timeout (3 seconds). - <br>`-1` - no timeout (block indefinitely). - <br>`-2` - disables SetWriteDeadline calls completely. <br>In milliseconds.                                    |                                                                                                                                                   |
| pool_fifo               | Type of connection pool.<br>true for FIFO pool, false for LIFO pool.<br>Note that FIFO has slightly higher overhead compared to LIFO,<br>but it helps closing idle connections faster reducing the pool size.                                                                                                               |                                                                                                                                                   |
| pool_size               | Maximum number of socket connections. Default is 250 connections per every available CPU as reported by runtime.GOMAXPROCS.                                                                                                                                                                                                 | default 250 * cpu                                                                                                                                 |
| pool_timeout            | Amount of time client waits for connection if all connections <br>are busy before returning an error.<br>Default is ReadTimeout + 1 second.                                                                                                                                                                                 ||
| min_idle_conns          | Minimum number of idle connections which is useful when establishing<br>new connection is slow.                                                                                                                                                                                                                             ||
| max_idle_conns          | Maximum number of idle connections.                                                                                                                                                                                                                                                                                         ||
| conn_max_idle_time      | ConnMaxIdleTime is the maximum amount of time a connection may be idle. Should be less than server's timeout. <br>Expired connections may be closed lazily before reuse. If d <= 0, connections are not closed due to a connection's idle time. <br>Default is 30 minutes. -1 disables idle timeout check. In milliseconds. ||
| conn_max_lifetime       | ConnMaxLifetime is the maximum amount of time a connection may be reused. <br>Expired connections may be closed lazily before reuse. If <= 0, connections are not closed due to a connection's age.<br> Default is to not close idle connections.                                                                           ||
| max_redirects           | The maximum number of retries before giving up. Command is retried <br>on network errors and MOVED/ASK redirects. <br>Default is 3 retries.                                                                                                                                                                                 ||
| read_only               | Enables read-only commands on slave nodes.                                                                                                                                                                                                                                                                                  ||
| route_by_latency        | Allows routing read-only commands to the closest master or slave node. It automatically enables ReadOnly.                                                                                                                                                                                                                   ||
| route_randomly          | Allows routing read-only commands to the random master or slave node. <br>It automatically enables ReadOnly.                                                                                                                                                                                                                ||

* More fields：https://trpc.group/trpc-go/trpc-database/blob/master/goredis/internal/proto/goredis.proto#L30
* See：https://github.com/redis/go-redis/blob/master/options.go#L221
* For example：`redis://:password@9.xx.xx.252:6380/15?is_proxy=true`

# mock reference
* https://github.com/go-redis/redismock

# Common problem
* Q: cmd not support <br>
  A: Redis such as ckv+ and istore belong to the proxy mode, not the redis cluster mode. The solution is:<br>
     Add the option `?is_proxy=true` at the end of the target, as follows:<br>
```yaml
target: polaris://:password@trpc.xxx_lite_test.redis.com/0?is_proxy=true
```
Note: The proxy module only supports polaris name resolution, and the calling address of proxy mode is a fixed random address.
* Q: How to deal with special characters such as `@`,`:`,`?`,`=` in the password? <br>
  A: Perform the password: url encode.
* Q: There is no data in galileo monitoring, but the logs can be printed? <br>
  A: The name field of the New parameter needs to be composed of four fields, otherwise it is not compatible;
* Q: How to enter the cluster version address? <br>
  A: The address transmission of the cluster version is divided by ",", such as; `redis://user:password@127.0.0.1:6379,127.0.0.1:6380/0`.
