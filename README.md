English | [中文](README.zh_CN.md)

# tRPC-Go Database Plugin

[![LICENSE](https://img.shields.io/badge/license-Apache--2.0-green.svg)](https://github.com/trpc-ecosystem/go-database/blob/main/LICENSE)

In the daily development process, developers often need to access various storage systems such as MySQL, Redis, Kafka, etc., for database operations. While using open-source SDKs can fulfill the requirements for accessing databases, developers are responsible for handling naming routing, monitoring, and configuration themselves.

Considering that tRPC-Go provides a variety of plugins for naming routing, monitoring, and configuration management, it is possible to wrap open-source SDKs to reuse tRPC-Go's capabilities and reduce redundant code. tRPC-Go offers encapsulation for some open-source SDKs, allowing you to directly leverage tRPC-Go's features for naming routing, monitoring, and more.

| Database | Description |
| :-------: | :---------- |
| bigcache | Wraps the open-source local caching database [Bigcache](https://github.com/allegro/bigcache) |
| clickhouse | Wraps the open-source database [Clickhouse SDK](https://github.com/ClickHouse/clickhouse-go) |
| cos | Wraps Tencent Cloud Object Storage [COS SDK](https://github.com/tencentyun/cos-go-sdk-v5) |
| goes | Wraps the open-source official Go [ElasticSearch client](https://github.com/elastic/go-elasticsearch) |
| goredis | Wraps the in-memory database [Redis SDK](https://github.com/redis/go-redis) |
| gorm | Wraps the Golang ORM library [GORM](https://github.com/go-gorm/gorm) |
| hbase | Wraps the open-source database [HBase SDK](https://github.com/tsuna/gohbase) |
| kafka | Wraps the open-source Kafka message queue SDK [Sarama](https://github.com/IBM/sarama) |
| mongodb | Wraps the open-source database [MongoDB Driver](https://go.mongodb.org/mongo-driver/mongo) |
| mysql | Wraps the open-source database [MySQL Driver](https://github.com/go-sql-driver/mysql) |
| timer | Local/distributed timer functionality |

## Copyright

The copyright notice pertaining to the Tencent code in this repo was previously in the name of “THL A29 Limited.”  That entity has now been de-registered.  You should treat all previously distributed copies of the code as if the copyright notice was in the name of “Tencent.”
