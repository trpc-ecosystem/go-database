English | [中文](README.zh_CN.md)

# Gorm trpc plugin

[![Go Reference](https://pkg.go.dev/badge/trpc.group/trpc-go/trpc-database/gorm.svg)](https://pkg.go.dev/trpc.group/trpc-go/trpc-database/gorm)
[![Go Report Card](https://goreportcard.com/badge/trpc.group/trpc-go/trpc-database/gorm)](https://goreportcard.com/report/trpc.group/trpc-go/trpc-database/gorm)
[![Tests](https://github.com/trpc-ecosystem/go-database/actions/workflows/gorm.yml/badge.svg)](https://github.com/trpc-ecosystem/go-database/actions/workflows/gorm.yml)
[![Coverage](https://codecov.io/gh/trpc-ecosystem/go-database/branch/main/graph/badge.svg?flag=gorm&precision=2)](https://app.codecov.io/gh/trpc-ecosystem/go-database/tree/main/gorm)

This plugin provides a wrapper for the Gorm in trpc, allowing you to use the native Gorm interface while reusing trpc's plugin ecosystem.

## Table of Contents

[TOC]

## Background

During the development process of the middleware, developers often need to interact with databases. In order to leverage the capabilities of trpc-go, such as dynamic addressing and trace and monitoring, it is recommand to use the this gorm plugin.

Various ORM frameworks can effectively solve code quality and engineering efficiency issues. However, without trpc-go integration, it is not possible to use the native trpc-go framework configuration, filters, addressing, and other features. Therefore, we need to create an ORM plugin.

Currently, the most popular ORM framework in Go is Gorm, which has a high level of community activity and good acceptance among team developers.

## Quick Start

Add trpc_go.yaml framework configuration:

```yaml
client:                                            
  service: 
    - name: trpc.mysql.server.service
      # Reference: https://github.com/go-sql-driver/mysql?tab=readme-ov-file#dsn-data-source-name
      target: dsn://root:123456@tcp(127.0.0.1:3306)/mydb?charset=utf8mb4&parseTime=True
```

Code implementation:

```go
package main

import (
    "github.com/trpc-group/trpc-database/gorm"
    "trpc.group/trpc-go/trpc-go"
    "trpc.group/trpc-go/trpc-go/log"
)

type User struct {
    ID       int
    Username string
}

func main() {
    _ = trpc.NewServer()

    cli, err := gorm.NewClientProxy("trpc.mysql.server.service")
    if err != nil {
        panic(err)
    }

    // Create record
    insertUser := User{Username: "gorm-client"}
    result := cli.Create(&insertUser)
    log.Infof("inserted data's primary key: %d, err: %v", insertUser.ID, result.Error)

    // Query record
    var queryUser User
    if err := cli.First(&queryUser).Error; err != nil {
        panic(err)
    }
    log.Infof("query user: %+v", queryUser)

    // Delete record
    deleteUser := User{ID: insertUser.ID}
    if err := cli.Delete(&deleteUser).Error; err != nil {
        panic(err)
    }
    log.Info("delete record succeed")

    // For more use cases, see https://gorm.io/docs/create.html
}
```

Complete example: [mysql-example](examples/mysql/main.go)

## Complete Configuration

### tRPC-Go Framework Configuration

The plugin enforces setting the timeout to 0. This is because trpc-go cancels the context after receiving a request response, leading to competition with native database/sql for results and causing "Context Cancelled" errors. You can set the timeout directly in the address or use the context to implement timeout functionality.

```yaml
...
client:                                     
  service:                                         
    - name: trpc.mysql.xxxx.xxxx # initialized as mysql
      target: dsn://root:xxxxxxg@tcp(127.0.0.1:3306)/test?charset=utf8mb4&parseTime=True
      # timeout setting is ineffective
    - name: trpc.mysql.xxxx.xxxx # initialized as mysql
      target: gorm+polaris://root:xxxxxxg@tcp(trpc.mysql.xxxx.xxxx)/test?charset=utf8mb4&parseTime=True
      namespace: Production
    - name: trpc.clickhouse.xxxx.xxx  # initialized as clickhouse
      target: dsn://tcp://localhost:9000/${database}?username=user&password=qwerty&read_timeout=10
```

### Connection Pool Configuration Parameters (Optional)

You can configure connection pool parameters through the plugin's configuration.

```yaml
plugins: # Plugin configuration
  database:
    gorm:
      # Default connection pool configuration for all database connections
      max_idle: 20 # Maximum number of idle connections (default 10 if not set or set to 0); if negative, no idle connections are retained
      max_open: 100 # Maximum number of open connections (default 10000 if not set or set to 0); if negative, no limit on open connections
      max_lifetime: 180000 # Maximum connection lifetime in milliseconds (default 3min); if negative, connections are not closed due to age
      driver_name: mysql # Driver used for the connection (empty by default, import the corresponding driver if specifying)
      logger: # this feature is supported in versions >= v0.2.2
        slow_threshold: 200 # Slow query threshold in milliseconds, 0 means no slow query logging (default 0)
        colorful: false # Whether to colorize the logs (default false)
        ignore_record_not_found_error: false # Whether to ignore errors when records are not found (default false)
        log_level: 4 # Log level: 1:Silent, 2:Error, 3:Warn, 4:Info (default no logging)
        max_sql_size: 100 # Maximum SQL statement length for truncation, 0 means no limit (default 0)
      # Individual connection pool configuration for specific database connections
      service:
        - name: trpc.mysql.server.service
          max_idle: 10 
          max_open: 50 
          max_lifetime: 180000 
          driver_name: mysql # Driver used for the connection (empty by default, import the corresponding driver if specifying)
          logger:
            slow_threshold: 1000 
            colorful: true 
            ignore_record_not_found_error: true 
            log_level: 4 
```

## Advanced Features

### Polaris Routing and Addressing

If you need to use Polaris routing and addressing, first add:

`import _ "github.com/trpc-group/trpc-naming-polaris"`

Then use the `gorm+polaris` scheme in the framework configuration target, and write the Polaris service name in the original `host:port` position.

```yaml
client:                                            
  service:                                         
    - name: trpc.mysql.xxxx.xxxx
      namespace: Production
      # Use gorm+polaris scheme, write Polaris service name in original host:port position
      target: gorm+polaris://root:123456@tcp(${polaris_name})/mydb?parseTime=True
```

### ClickHouse Integration

Add trpc_go.yaml framework configuration:

Note that service.name uses the four-segment format `trpc.${app}.${server}.${service}`, where `${app}` should be `clickhouse`.

```yaml
client:                                            
  service: 
    - name: trpc.clickhouse.server.service
      # Reference: https://github.com/ClickHouse/clickhouse-go?tab=readme-ov-file#dsn
      target: dsn://clickhouse://default:@127.0.0.1:9000/mydb?dial_timeout=200ms&max_execution_time=60
      # In gorm/v0.2.2 and earlier, using this DSN format would prompt username or password errors, 
      # you should use the pre-change DSN format:
      # target: dsn://tcp://localhost:9000?username=user&password=qwerty&database=clicks&read_timeout=10&write_timeout=20
```

Complete example: [clickhouse-example](examples/clickhouse/main.go)

### SQLite Integration

Add trpc_go.yaml framework configuration:

Note that service.name uses the four-segment format `trpc.${app}.${server}.${service}`, where `${app}` should be `sqlite`.

```yaml
client:                                            
  service: 
    - name: trpc.sqlite.server.service
      # Reference: https://github.com/mattn/go-sqlite3?tab=readme-ov-file#dsn-examples
      # Execute "sqlite3 mysqlite.db" in the current directory to create the database
      target: dsn://file:mysqlite.db
```

Add plugin configuration, keep the service name consistent, and configure `driver_name: sqlite3`.

```yaml
plugins:
  database:
    gorm:
      service:
        # Configuration effective for trpc.sqlite.server.service client
        - name: trpc.sqlite.server.service
          driver_name: sqlite3 # Requires import "github.com/mattn/go-sqlite3"
```

Complete example: [sqlite-example](examples/sqlite/main.go)

### PostgreSQL Integration

Add trpc_go.yaml framework configuration.

Note that service.name uses the four-segment format `trpc.${app}.${server}.${service}`, where `${app}` should be `postgres`.

```yaml
client:                                            
  service: 
    - name: trpc.postgres.server.service
      # Reference: https://github.com/jackc/pgx?tab=readme-ov-file#example-usage
      target: dsn://postgres://username:password@localhost:5432/database_name
```

###  Code Implementation

The NewClientProxy method of this plugin can return a gorm.DB pointer that handles all requests through trpc. You can use this pointer to directly call native gorm methods.

Native gorm documentation: https://gorm.io/zh_CN/docs/index.html

If you encounter any issues, please open an issue in the project or contact the developers.

```go
// Simple usage example, only supports MySQL. gormDB is the native gorm.DB pointer.
gormDB := gorm.NewClientProxy("trpc.mysql.test.test")
gormDB.Where("current_owners = ?", "xxxx").Where("id < ?", xxxx).Find(&owners)
```

If you need to make additional settings to the DB or use a database other than MySQL, you can use the native gorm.Open function and the ConnPool provided by this plugin.
```go
import (
	"gorm.io/gorm"
	"gorm.io/gorm/mysql"
  gormplugin "trpc.group/trpc-go/trpc-database/gorm"
)

connPool := gormplugin.NewConnPool("trpc.mysql.test.test")
gormDB := gorm.Open(
	mysql.New(
		mysql.Config{
			Conn: connPool,
		}),
	&gorm.Config{
		Logger:  gormplugin.DefaultTRPCLogger, // For example: pass a custom logger
	},
)
```

### Logging

Due to gorm's logging being output to stdout, it doesn't output to the tRPC-Go logs. This plugin wraps the tRPC log, allowing gorm logs to be printed in the tRPC log.

The gorm.DB generated by the NewClientProxy method is already equipped with the tRPC logger. If you open the ConnPool using gorm.Open, you can use the previously mentioned example to utilize the wrapped tRPC logger.

Example of logging:
```go
gormDB := gorm.NewClientProxy("trpc.mysql.test.test")
gormDB.Debug().Where("current_owners = ?", "xxxx").Where("id < ?", xxxx).Find(&owners)
```

### Context
When using the database plugin, you may need to report trace information and pass a context with the request. Gorm provides the WithContext method to include a context.

Example:
```
gormDB := gorm.NewClientProxy("trpc.mysql.test.test")
gormDB.WithContext(ctx).Where("current_owners = ?", "xxxx").Where("id < ?", xxxx).Find(&owners)
```

### Unit Testing
You can use sqlmock to mock a sql.DB and open it with the native gorm.Open function to obtain a gorm.DB where you can control the results.
```go
db, mock, _ := sqlmock.New()
defer db.Close()
gormDB, _ := gorm.Open(mysql.New(mysql.Config{Conn: db}))
```
To mock the returned results, you can use the following syntax:
```go
mock.ExpectQuery(`SELECT * FROM "blogs"`).WillReturnRows(sqlmock.NewRows(nil))
mock.ExpectExec("INSERT INTO users").WithArgs("john", AnyTime{}).WillReturnResult(NewResult(1, 1))
```
For more details, you can refer to the documentation of sqlmock: https://github.com/DATA-DOG/go-sqlmock

If you are not sure about the specific SQL statements, you can temporarily set the log level to Info using the Debug() function to retrieve the SQL statements from the logs.

Example:
```
gormDB.Debug().Where("current_owners = ?", "xxxx").Where("id < ?", xxxx).Find(&owners)
```

### Implementation Details

See the specific [implementation details](docs/architecture.md) of the gorm plugin.

## Implementation Approach

**gorm interacts with the database using the `*sql.DB` library.**
gorm also supports custom database connections, as mentioned in the official documentation:

[GORM allows initializing *gorm.DB with an existing database connection](https://gorm.io/zh_CN/docs/connecting_to_the_database.html)

## Specific Implementation

### Package Structure
```
gorm
├── client.go      	# Entry point for using this plugin
├── codec.go      	# Encoding and decoding module
├── plugin.go			# Implementation of plugin configuration
└── transport.go		# Actual data sending and receiving

```
### Main Logic Explanation

In the gorm framework, all interactions with the DB are done through DB.ConnPool, which is an interface. So, as long as our Client implements the ConnPool interface, gorm can use the Client to interact with the DB.

The definition of ConnPool can be found here: [ConnPool](https://github.com/go-gorm/gorm/blob/master/interfaces.go)

After practical usage, it was found that the implementation of ConnPool methods in gormCli alone is not enough. For example, it was not possible to interact with the database using transactions. Therefore, a comprehensive code search was performed to identify all the methods in gorm that call ConnPool, and they were implemented accordingly.

Thus, the Client becomes a custom connection that satisfies gorm's requirements, similar to sql.DB, but implementing only a subset of sql.DB functionality.

Additionally, there is a TxClient used for transaction handling, which implements both the ConnPool and TxCommitter interfaces defined by gorm.

## Notes

### Timeout Settings Not Effective

If users add timeout in the framework configuration, the tRPC-Go framework will create a new context when making calls and cancel the context after the call ends. This feature will cause "Context Canceled" errors when reading data from the `Row` interface, so the current plugin will set timeout to zero.

```yaml
client:                                     
  service:                                         
    - name: trpc.mysql.xxxx.xxxx # initialized as mysql
      timeout: 1000 # timeout configuration will not take effect
```

If you need to configure request timeout, you can use context.WithTimeout() to control the context yourself.
If you want to configure connection establishment timeout, connection read/write timeout, you can consider configuring related parameters in the DSN, for example, for MySQL you can configure [`readTimeout`, `writeTimeout` and `timeout`](https://github.com/go-sql-driver/mysql?tab=readme-ov-file#connection-pool-and-timeouts) in the DSN.

```yaml
client:                                            
  service: 
    - name: trpc.mysql.server.service
      # Add readTimeout to DSN parameters, reference: https://github.com/go-sql-driver/mysql?tab=readme-ov-file#dsn-data-source-name
      target: dsn://root:123456@tcp(127.0.0.1:3306)/mydb?readTimeout=100ms
```

## Related Links：

* Custom Connection in GORM：https://gorm.io/zh_CN/docs/connecting_to_the_database.html

## FAQ

### How to print specific SQL statements and results?
This plugin has implemented the TRPC Logger for GORM. You only need to configure `plugin.database.gorm.logger` configuration (requires plugin version >= v0.2.2) to print specific SQL statements to tRPC-Go logs.
Or you can add `Debug()` before the request to output to logs at Info level.

Example:
```
gormDB.Debug().Where("current_owners = ?", "xxxx").Where("id < ?", xxxx).Find(&owners)
```

### How to include TraceID and full-chain timeout information in requests?
You can use the WithContext method to pass the context to GORM.

Example:
```
gormDB.WithContext(ctx).Where("current_owners = ?", "xxxx").Where("id < ?", xxxx).Find(&owners)
```

### How to set the isolation level for transactions?
When starting a transaction, you can provide sql.TxOptions in the Begin method to set the isolation level. Alternatively, you can set it manually after starting the transaction using tx.Exec.

### Soft delete not working after switching from native gorm to tRPC gorm plugin

This is because gorm changed the soft delete method after the major version upgrade from jinzhu/gorm to gorm.io/gorm.

For the new soft delete method, see the documentation https://gorm.io/docs/delete.html#Soft-Delete

Using gorm.DeleteAt can directly be compatible with jinzhu/gorm soft delete.

### How to handle Context Canceled errors

**Update to the latest version to resolve this issue. The latest version will force timeout to 0.**

This problem is generally caused by setting a global timeout in the trpc framework client. A common trpc framework configuration example is as follows:

```yaml
client:
  timeout: 1000 # Need to remove this
  service:
    - name: xxxxx
      protocol: trpc
      timeout: 1000 # Set timeout separately for other services
```

The client-level timeout is a global timeout. All trpc clients that don't have a separate timeout setting will use this setting. The old version of this plugin would report errors due to context being canceled as long as timeout was set.

### Currently, only MySQL, ClickHouse, and PostgreSQL are supported.

Parse the `client.service.name` name, which needs to use the standard four-segment name format. For example, `trpc.mysql.xxx.xxx` initializes the `mysql driver`; `trpc.clickhouse.xxx.xxx` initializes the `clickhouse driver`; `trpc.postgres.xxx.xxx` initializes the `postgres driver(pgx)`.

For non-standard four-segment names, it defaults to initializing the `mysql driver`.

### When using GORM transactions, only the Begin method goes through the tRPC call chain.

**Update to the latest version to resolve this issue. v0.1.3 version has resolved this issue.**

This is a design compromise made to reduce complexity. In this plugin, when calling BeginTx, it directly returns the result of sql.DB.BeginTx(), which is an already opened transaction. Subsequent transaction operations are handled by that transaction.

Considering that this plugin is mainly designed for connecting to MySQL instances, this approach can reduce some complexity while ensuring normal operation. However, considering that there may be services that require all database requests to go through the tRPC filter, the mechanism will be modified in the future to make all requests within the transaction go through the tRPC request flow.

If inaccurate reporting of data is caused by this behavior, you can disable the GORM transaction optimization to ensure that all requests that do not explicitly use transactions go through the basic reporting methods.

> To ensure data consistency, GORM performs write operations (create, update, delete) within transactions. If you don't have this requirement, you can disable it during initialization.

Example:
```go
connPool := gormplugin.NewConnPool("trpc.mysql.test.test")
gormDB, err := gorm.Open(
	mysql.New(
		mysql.Config{
			Conn: connPool,
		}),
	&gorm.Config{
      Logger:  gormplugin.DefaultTRPCLogger, // Pass in a custom Logger
      SkipDefaultTransaction: true, // Disable GORM transaction optimization
	},
)
```