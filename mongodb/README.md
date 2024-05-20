English | [中文](README.zh_CN.md)

# tRPC-Go mongodb plugin
[![BK Pipelines Status](https://api.bkdevops.qq.com/process/api/external/pipelines/projects/pcgtrpcproject/p-d7b163d3830a429e976bf77e2409c6d3/badge?X-DEVOPS-PROJECT-ID=pcgtrpcproject)](http://devops.oa.com/ms/process/api-html/user/builds/projects/pcgtrpcproject/pipelines/p-d7b163d3830a429e976bf77e2409c6d3/latestFinished?X-DEVOPS-PROJECT-ID=pcgtrpcproject)

Base on community [mongo](https://go.mongodb.org/mongo-driver/mongo), used with trpc.

## mongodb client
```yaml
client:                                            # Backend configuration for client calls.
  service:                                         # Configuration for the backend.
    - name: trpc.mongodb.xxx.xxx         
      target: mongodb://user:passwd@vip:port       # mongodb standard uri：mongodb://[username:password@]host1[:port1][,host2[:port2],...[,hostN[:portN]]][/[database][?options]]
      timeout: 800                                 # The maximum processing time of the current request.
    - name: trpc.mongodb.xxx.xxx1         
      target: mongodb+polaris://user:passwd@polaris_name  # mongodb+polaris means that the host in the mongodb uri will perform Polaris analysis.
      timeout: 800                                        # The maximum processing time of the current request.
```
```go
package main

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"trpc.group/trpc-go/trpc-database/mongodb"
	"trpc.group/trpc-go/trpc-go/log"
)

// BattleFlow is battle information.
type BattleInfo struct {
	Id    string `bson:"_id,omitempty"`
	Ctime uint32 `bson:"ctime,omitempty" json:"ctime,omitempty"`
}

func (s *server) SayHello(ctx context.Context, req *pb.ReqBody, rsp *pb.RspBody) (err error) {
	proxy := mongodb.NewClientProxy("trpc.mongodb.xxx.xxx") // Your custom service name，used for monitoring, reporting and mapping configuration.

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
		// The same proxy instance needs to be used during transaction execution.
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
	// Business logic.
}
```
## Frequently Asked Questions (FAQs)
- Q1: How to configure ClientOptions:
- A1: When creating a Transport, you can use WithOptionInterceptor to configure ClientOptions. You can refer to options_test.go for more information.
