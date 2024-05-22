# tRPC-Go mongodb 插件
[![BK Pipelines Status](https://api.bkdevops.qq.com/process/api/external/pipelines/projects/pcgtrpcproject/p-d7b163d3830a429e976bf77e2409c6d3/badge?X-DEVOPS-PROJECT-ID=pcgtrpcproject)](http://devops.oa.com/ms/process/api-html/user/builds/projects/pcgtrpcproject/pipelines/p-d7b163d3830a429e976bf77e2409c6d3/latestFinished?X-DEVOPS-PROJECT-ID=pcgtrpcproject)

封装社区的 [mongo](https://go.mongodb.org/mongo-driver/mongo) ，配合 trpc 使用。

## mongodb client
```yaml
client:                                            #客户端调用的后端配置
  service:                                         #针对后端的配置
    - name: trpc.mongodb.xxx.xxx         
      target: mongodb://user:passwd@vip:port       #mongodb 标准uri：mongodb://[username:password@]host1[:port1][,host2[:port2],...[,hostN[:portN]]][/[database][?options]]
      timeout: 800                                 #当前这个请求最长处理时间
    - name: trpc.mongodb.xxx.xxx1         
      target: mongodb+polaris://user:passwd@polaris_name  # mongodb+polaris表示mongodb uri中的host会进行北极星解析
      timeout: 800                                        # 当前这个请求最长处理时间
```
```go
package main

import (
	"time"
	"context"

	"trpc.group/trpc-go/trpc-database/mongodb"
	"trpc.group/trpc-go/trpc-go/client"
)

// BattleFlow 对局信息
type BattleInfo struct {
	Id    string `bson:"_id,omitempty" `
	Ctime uint32 `bson:"ctime,omitempty" json:"ctime,omitempty"`
}

func (s *server) SayHello(ctx context.Context, req *pb.ReqBody, rsp *pb.RspBody) (err error) {
	proxy := mongodb.NewClientProxy("trpc.mongodb.xxx.xxx") // service name自己随便填，主要用于监控上报和寻址配置项

	// mongodb insert
	_, err = proxy.InsertOne(sc, "database", "table", bson.M{"_id": "key2", "value": "v2"})

	// mongodb ReplaceOne
	opts := options.Replace().SetUpsert(true)
	filter := bson.D{{"_id", "key1"}}
	_, err := proxy.ReplaceOne(ctx, "database", "table", filter, &BattleInfo{}, opts)
	if err != nil {
		log.Errorf("err=%v, data=%v", err, *battleInfo)
		return err
	}

	// mongodb FindOne
	rst := proxy.FindOne(ctx, "database", "table", bson.D{{"_id", "key1"}})
	battleInfo = &BattleInfo{}
	err = rst.Decode(battleInfo)
	if err != nil {
		return nil, err
	}

	// mongodb transaction
	err = proxy.Transaction(ctx, func(sc mongo.SessionContext) error {
		//事务执行过程中需要使用同一proxy实例执行
		_, tErr := proxy.InsertOne(sc, "database", "table", bson.M{"_id": "key1", "value": "v1"})
		if tErr != nil {
			return tErr
		}
		_, tErr = proxy.InsertOne(sc, "database", "table", bson.M{"_id": "key2", "value": "v2"})
		if tErr != nil {
			return tErr
		}
		return nil
	}, nil)

	// mongodb RunCommand
	cmdDB := bson.D{}
	cmdDB = append(cmdDB, bson.E{Key: "enableSharding", Value: "dbName"})
	err = proxy.RunCommand(ctx, "admin", cmdDB).Err()
	if err != nil {
		return nil, err
	}

	cmdColl := bson.D{}
	cmdColl = append(cmdColl, bson.E{Key: "shardCollection", Value: "dbName.collectionName"})
	cmdColl = append(cmdColl, bson.E{Key: "key", Value: bson.D{{"openId", "hashed"}}})
	cmdColl = append(cmdColl, bson.E{Key: "unique", Value: false})
	cmdColl = append(cmdColl, bson.E{Key: "numInitialChunks", Value: 10})
	err = proxy.RunCommand(ctx, "admin", cmdColl).Err()
	if err != nil {
		return nil, err
	}
	// 业务逻辑
}
```
## 常见问题

- Q1: 如何配置 ClientOptions
- A1: 创建Transport时可以使用WithOptionInterceptor对ClientOptions进行配置，可以参考options_test.go
