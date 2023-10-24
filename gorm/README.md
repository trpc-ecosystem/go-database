English | [中文](README.zh_CN.md)

# Gorm trpc plugin

[![Go Reference](https://pkg.go.dev/badge/trpc.group/trpc-go/trpc-database/gorm.svg)](https://pkg.go.dev/trpc.group/trpc-go/trpc-database/gorm)
[![Go Report Card](https://goreportcard.com/badge/trpc.group/trpc-go/trpc-database/gorm)](https://goreportcard.com/report/trpc.group/trpc-go/trpc-database/gorm)
[![Tests](https://github.com/trpc-ecosystem/go-database/actions/workflows/gorm.yml/badge.svg)](https://github.com/trpc-ecosystem/go-database/actions/workflows/gorm.yml)
[![Coverage](https://codecov.io/gh/trpc-ecosystem/go-database/branch/coverage/graph/badge.svg?flag=gorm&precision=2)](https://app.codecov.io/gh/trpc-ecosystem/go-database/tree/coverage/gorm)

This plugin provides a wrapper for the Gorm in trpc, allowing you to use the native Gorm interface while reusing trpc's plugin ecosystem.

## Table of Contents

[TOC]

## Background

During the development process of the middleware, developers often need to interact with databases. In order to leverage the capabilities of trpc-go, such as dynamic addressing and trace and monitoring, it is recommand to use the this gorm plugin.

Various ORM frameworks can effectively solve code quality and engineering efficiency issues. However, without trpc-go integration, it is not possible to use the native trpc-go framework configuration, filters, addressing, and other features. Therefore, we need to create an ORM plugin.

Currently, the most popular ORM framework in Go is Gorm, which has a high level of community activity and good acceptance among team developers.

## Quick Start

Currently, it supports MySQL/ClickHouse and has made adjustments in the code to quickly iterate and support other types of databases.

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
      max_idle: 20 # Maximum number of idle connections
      max_open: 100 # Maximum number of open connections
      max_lifetime: 180000 # Maximum connection lifetime (in milliseconds)
      # Individual connection pool configuration for specific database connections
      service:
        - name: trpc.mysql.xxxx.xxxx
          max_idle: 10 
          max_open: 50 
          max_lifetime: 180000 
          driver_name: xxx # Driver used for the connection (empty by default, import the corresponding driver if specifying)

```

### gorm Logger Configuration (Optional)

You can configure logging parameters using plugins. Here is an example of configuring logging parameters using YAML syntax:


```yaml
plugins: # Plugin configuration
  database:
    gorm:
      # Default logging configuration for all database connections
      logger:
        slow_threshold: 1000 # Slow query threshold in milliseconds
        colorful: true # Whether to colorize the logs
        ignore_record_not_found_error: true # Whether to ignore errors when records are not found
        log_level: 4 # 1: Silent, 2: Error, 3: Warn, 4: Info
      # Individual logging configuration for specific database connections
      service:
        - name: trpc.mysql.xxxx.xxxx
          logger:
            slow_threshold: 1000 
            colorful: true 
            ignore_record_not_found_error: true 
            log_level: 4 

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


## Related Links：

* Custom Connection in GORM：https://gorm.io/zh_CN/docs/connecting_to_the_database.html

## FAQ
### How to print specific SQL statements and results?
This plugin has implemented the TRPC Logger for GORM, as mentioned in the "Logging" section above. If you are using the default NewClientProxy, you can use the Debug() method before the request to output the SQL statements to the TRPC log with the Info level.

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

### Currently, only MySQL, ClickHouse, and PostgreSQL are supported.

### When using GORM transactions, only the Begin method goes through the tRPC call chain.
This is a design compromise made to reduce complexity. In this plugin, when calling BeginTx, it directly returns the result of sql.DB.BeginTx(), which is an already opened transaction. Subsequent transaction operations are handled by that transaction.

Considering that this plugin is mainly designed for connecting to MySQL instances, this approach can reduce some complexity while ensuring normal operation. However, considering that there may be services that require all database requests to go through the tRPC filter, the mechanism will be modified in the future to make all requests within the transaction go through the tRPC request flow.

If inaccurate reporting of data is caused by this behavior, you can disable the GORM transaction optimization to ensure that all requests that do not explicitly use transactions go through the basic reporting methods.

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


