[English](README.md) | 中文

# Gorm trpc插件

[![Go Reference](https://pkg.go.dev/badge/trpc.group/trpc-go/trpc-database/gorm.svg)](https://pkg.go.dev/trpc.group/trpc-go/trpc-database/gorm)
[![Go Report Card](https://goreportcard.com/badge/trpc.group/trpc-go/trpc-database/gorm)](https://goreportcard.com/report/trpc.group/trpc-go/trpc-database/gorm)
[![Tests](https://github.com/trpc-ecosystem/go-database/actions/workflows/gorm.yml/badge.svg)](https://github.com/trpc-ecosystem/go-database/actions/workflows/gorm.yml)
[![Coverage](https://codecov.io/gh/trpc-ecosystem/go-database/branch/coverage/graph/badge.svg?flag=gorm&precision=2)](https://app.codecov.io/gh/trpc-ecosystem/go-database/tree/coverage/gorm)

该插件实现了对Gorm ConnPool的trpc封装，可以使用原生Gorm接口同时复用trpc的插件生态

## 目录

[TOC]

## 背景

在中台的开发过程中，开发者经常需要和数据库进行交互。为了用上trpc-go的动态寻址、调用链监控等能力，需要使用trpc-go封装好的数据库公共组件。

用各种orm框架可以有效解决代码质量和工程效率的问题，但是由于没有做trpc-go的封装，无法使用trpc-go原生的框架配置、filter、寻址等功能。因此，我们需要封装一个orm插件。

目前go相关的ORM框架中gorm最为流行，社区活跃度也很高，团队内开发人员接受程度较好

## 快速上手

目前支持mysql/clickhouse，已经在代码中做出支持其他数据库的调整，可以快速迭代支持其他类型数据库

### trpc框架配置

和trpc-database/mysql的配置基本保持一致，插件中会将timeout强制设置为0，原因是trpc-go收到请求回包后会cancel context，和原生database/sql获取结果产生竞争，导致Context Cancelled错误。可以直接在地址中设置超时时间或设置context来实现timeout功能。
```yaml
...
client:                                     
  service:                                         
    - name: trpc.mysql.xxxx.xxxx # 初始化为mysql
      target: dsn://root:xxxxxxg@tcp(127.0.0.1:3306)/test?charset=utf8mb4&parseTime=True
      # timeout设置无效
    - name: trpc.mysql.xxxx.xxxx # 初始化为mysql
      target: gorm+polaris://root:xxxxxxg@tcp(trpc.mysql.xxxx.xxxx)/test?charset=utf8mb4&parseTime=True
      namespace: Production
    - name: trpc.clickhouse.xxxx.xxx  # 初始化为clickhouse
      target: dsn://tcp://localhost:9000/${database}?username=user&password=qwerty&read_timeout=10
...
```

### 连接池配置参数设置(可选)
可以通过插件配置的方式配置连接池参数
```yaml
plugins: #插件配置
  database:
    gorm:
      # 所有数据库连接默认的连接池配置
      max_idle: 20 # 最大空闲连接数
      max_open: 100 # 最大在线连接数
      max_lifetime: 180000 # 连接最大生命周期(单位：毫秒)
      # 指定数据库连接单独配置连接池
      service:
        - name: trpc.mysql.xxxx.xxxx
          max_idle: 10 # 最大空闲连接数
          max_open: 50 # 最大在线连接数
          max_lifetime: 180000 # 连接最大生命周期(单位：毫秒)
          driver_name: xxx # 连接使用的驱动（此项默认为空，如配置驱动名，应先导入对应的驱动）
```

### gorm 日志配置（可选）

可以通过插件配置的方式配置日志参数

```yaml
plugins: #插件配置
  database:
    gorm:
      # 所有数据库连接默认的日志配置
      logger:
        slow_threshold: 1000 # 慢查询阈值，单位 ms
        colorful: true # 日志是否着色
        ignore_record_not_found_error: true # 是否忽略记录不存在的错误
        log_level: 4 # 1: Silent, 2: Error, 3: Warn, 4: Info
      # 指定数据库连接单独配置日志
      service:
        - name: trpc.mysql.xxxx.xxxx
          logger:
            slow_threshold: 1000 # 慢查询阈值，单位 ms
            colorful: true # 日志是否着色
            ignore_record_not_found_error: true # 是否忽略记录不存在的错误
            log_level: 4 # 1: Silent, 2: Error, 3: Warn, 4: Info
```

### 编写代码

本插件的NewClientProxy方法可以返回一个所有请求通过trpc处理的gorm.DB指针，直接用这个指针调用gorm的原生方法即可。

原生gorm文档: https://gorm.io/zh_CN/docs/index.html

如果发现问题请在项目中提issue或联系开发者。

```go
// 简单使用方法，只支持mysql, gormDB是原生的gorm.DB类型指针
gormDB := gorm.NewClientProxy("trpc.mysql.test.test")
gormDB.Where("current_owners = ?", "xxxx").Where("id < ?", xxxx).Find(&owners)
```

如果需要对DB做额外设置，或需要使用mysql以外数据库时，可以使用原生的gorm.Open函数，并使用本插件提供的ConnPool
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
		Logger:  gormplugin.DefaultTRPCLogger, // 示例，传入自定义Logger
	},
)
```

### 日志

由于gorm的日志输出到stdout，不会输出到在 tRPC-Go 的日志中。本插件对tRPC log进行了一次封装，使得gorm的日志可以打印到tRPC log上。

使用NewClientProxy方法生成的gorm.DB已经封装好了tRPC logger，使用`gorm.Open`的方式打开ConnPool的话，可以采用上文示例的写法使用封装好的tRPC logger。

打印Log的示例:
```
gormDB := gorm.NewClientProxy("trpc.mysql.test.test")
gormDB.Debug().Where("current_owners = ?", "xxxx").Where("id < ?", xxxx).Find(&owners)
```
### Context
使用数据库插件时，可能需要上报链路追踪信息，需要带context发起请求，gorm可以使用WithContext的方法带上context
示例:
```
gormDB := gorm.NewClientProxy("trpc.mysql.test.test")
gormDB.WithContext(ctx).Where("current_owners = ?", "xxxx").Where("id < ?", xxxx).Find(&owners)
```

### 单元测试
使用`sqlmock`mock一个sql.DB，使用原生的gorm.Open打开，就得到了一个可以自己控制结果的gorm.DB。
```go
db, mock, _ := sqlmock.New()
defer db.Close()
gormDB, _ := gorm.Open(mysql.New(mysql.Config{Conn: db}))
```
要mock返回的结果，可以这么写
```go
mock.ExpectQuery(`SELECT * FROM "blogs"`).WillReturnRows(sqlmock.NewRows(nil))
mock.ExpectExec("INSERT INTO users").WithArgs("john", AnyTime{}).WillReturnResult(NewResult(1, 1))
```
具体可以查阅sqlmock的[文档](https://github.com/DATA-DOG/go-sqlmock)

如果不清楚具体的SQL语句，可以具体调用前添加`Debug()`函数把loglevel暂时调到Info，从log中获取SQL语句。

例子:
```
gormDB.Debug().Where("current_owners = ?", "xxxx").Where("id < ?", xxxx).Find(&owners)
```


## 实现思路

**gorm和数据库的交互也是通过`*sql.DB`库实现的。**
且gorm还支持自定义数据库连接的扩展，见官方文档：

[GORM 允许通过一个现有的数据库连接来初始化 *gorm.DB](https://gorm.io/zh_CN/docs/connecting_to_the_database.html)

## 具体实现

### 包结构
```
gorm
├── client.go      	#调用该插件的入口
├── codec.go      	#编解码模块
├── plugin.go			#插件配置的实现
└── transport.go		#实际发送接收数据 

```
### 主要逻辑说明

在gorm框架中所有与DB交互都是通过DB.ConnPool这个field来完成的，ConnPool 是一个interface，所以只要我们自己的Client实现了ConnPool接口，那么gorm就可以使用Client来与DB交互。

ConnPool的定义见：[gorm ConnPool](https://github.com/go-gorm/gorm/blob/master/interfaces.go)

实际使用过后发现gormCli仅实现ConnPool的方法还是不够，如无法使用事务的方式与DB交互，所以就全量检索了代码找出gorm调用ConnPool的所有方法，最后将其实现。

至此，Client对于gorm来说是一个满足要求的`自定义连接`了，类似 sql.DB， 只是实现了sql.DB 一部分功能。

此外，还有一个TxClient用于事务的处理，它实现了gorm定义的ConnPool和TxCommitter。


## 相关链接：

* gorm 自定义连接说明：https://gorm.io/zh_CN/docs/connecting_to_the_database.html

## FAQ
### 怎么打印具体的SQL语句与结果
本插件已经实现了gorm的TRPC Logger,具体可以看上文的"日志"部分。
如果使用了默认的NewClientProxy的话，在请求前加入Debug()就可以以Info的级别输出到trpc log
示例:
```
gormDB.Debug().Where("current_owners = ?", "xxxx").Where("id < ?", xxxx).Find(&owners)
```

### 怎么让请求带TraceID和全链路超时信息
使用WithContext方法可以把context传递给gorm
示例
```
gormDB.WithContext(ctx).Where("current_owners = ?", "xxxx").Where("id < ?", xxxx).Find(&owners)
```

### 怎么设置事务的隔离级别
启动事务时，Begin方法中可以带sql.TxOptions，在这里设置隔离级别

当然，启动事务之后tx.Exec手动设置也可以

### 目前只支持MySQL、ClickHouse、Postgresql

### 使用gorm事务时，只有Begin方法经过了tRPC调用链路
这是设计时为了减少复杂度做的妥协，本插件在调用BeginTx时，直接返回了sql.DB.BeginTx()的结果，即返回了一个已经打开的Transaction，后续事务操作由该Transaction处理。

考虑到本插件主要是为连接到mysql实例设计，这种处理方法可以在正常运行的情况下降低一些插件的复杂度。但是考虑到可能有服务需要让所有数据库请求通过trpc filter，将来会修改机制，让事务中的所有请求也走一遍trpc请求的流程。


如果是由于这样使得上报数据不准确的话,可以禁用gorm事务优化,这样能保证所有不显式使用事务的请求只走会上报的基础方法。
> 为了确保数据一致性，GORM 会在事务里执行写入操作（创建、更新、删除）。如果没有这方面的要求，您可以在初始化时禁用它。

示例:
```go
connPool := gormplugin.NewConnPool("trpc.mysql.test.test")
gormDB, err := gorm.Open(
	mysql.New(
		mysql.Config{
			Conn: connPool,
		}),
	&gorm.Config{
      Logger:  gormplugin.DefaultTRPCLogger, // 示例，传入自定义Logger
      SkipDefaultTransaction: true, // 示例，禁用gorm事务优化
	},
)
```


