[English](README.md) | 中文

# tRPC-Go Database 插件

[![LICENSE](https://img.shields.io/badge/license-Apache--2.0-green.svg)](https://github.com/trpc-ecosystem/go-database/blob/main/LICENSE)

在日常的开发过程中，开发者经常会访问 MySQL、Redis、Kafka 等存储进行数据库的读写。直接使用开源的 SDK 虽然可以满足访问数据库的需求，但是用户需要自己负责路由寻址、监控上报、配置的开发。

考虑到 tRPC-Go 提供了多种多样的路由寻址、监控上报、配置管理的插件，我们可以封装一下开源的 SDK，复用 tRPC-Go 插件的能力，减少重复代码。tRPC-Go 提供了部分开源 SDK 的封装，可以直接复用 tRPC-Go 的路由寻址、监控上报等功能。

| 数据库 | 描述 |
| :----: | :----   |
| bigcache | 封装开源本地缓存数据库 [Bigcache](https://github.com/allegro/bigcache) |
| clickhouse | 封装开源数据库 [Clickhouse SDK](https://github.com/ClickHouse/clickhouse-go) |
| cos | 封装腾讯云对象存储 [COS SDK](https://github.com/tencentyun/cos-go-sdk-v5) |
| goredis | 封装内存数据库 [Redis SDK](https://github.com/redis/go-redis) |
| gorm | 封装 Golang ORM 库 [GORM](https://github.com/go-gorm/gorm) |
| hbase | 封装开源数据库 [HBase SDK](https://github.com/tsuna/gohbase) |
| kafka | 封装开源消息队列 Kafka SDK [Sarama](https://github.com/IBM/sarama) |
| mysql | 封装开源数据库 [Mysql Driver](https://github.com/go-sql-driver/mysql) |
| timer | 本地/分布式定时器 |