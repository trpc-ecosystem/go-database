package mongodb_test

import (
	"context"
	"flag"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"trpc.group/trpc-go/trpc-database/mongodb"
	"trpc.group/trpc-go/trpc-go/client"
)

var ctx = context.Background()

var database = "test"
var collection = "test"

// var target = flag.String("target", "mongodb://user:passwd@127.0.0.1:27017", "mongodb server target dsn address")
var cmd = flag.String("cmd", "find", "cmd")
var timeout = flag.Duration("timeout", 1*time.Millisecond, "timeout")
var findArgs = map[string]interface{}{
	"filter": map[string]interface{}{
		"name": "tony",
		"age": map[string]interface{}{
			"$gt": 17,
		},
	},
	"sort": map[string]interface{}{
		"age": -1,
	},
}
var deleteArgs = map[string]interface{}{
	"filter": map[string]interface{}{
		"age": map[string]interface{}{
			"$gt": 220,
		},
		"name": "teacher",
	},
}

var findOneAndUpdateArgs = map[string]interface{}{
	"filter": map[string]interface{}{
		"name": "lucy",
		"age": map[string]interface{}{
			"$lt": 17,
		},
	},
	"update": map[string]interface{}{
		"$set": map[string]interface{}{
			"name": "tommmmmmmmmmmmm",
			"age":  00000000,
		},
	},
	"sort": map[string]interface{}{
		"age": -1,
	},
	"returnDocument": "After",
}

var inserOneArgs = map[string]interface{}{
	"document": map[string]interface{}{
		"name": "tony",
		"age":  18,
		"sex":  "female",
	},
}

var document1 = map[string]interface{}{
	"name": "lucy",
	"age":  0,
	"sex":  "m",
}
var document2 = map[string]interface{}{
	"name": "lucy2",
	"age":  1,
	"sex":  "m",
}
var document = map[string]interface{}{
	"name": "lucy3",
	"age":  2,
	"sex":  "m",
}
var insertManyArgs = map[string]interface{}{
	"documents": []interface{}{document, document1, document2},
}

var updateOneArgs = map[string]interface{}{
	"filter": map[string]interface{}{
		"name": "tom",
		"age": map[string]interface{}{
			"$et": 17,
		},
	},
	"update": map[string]interface{}{
		"$set": map[string]interface{}{
			"sex": "male",
		},
	},
}
var updateManyArgs = map[string]interface{}{
	"filter": map[string]interface{}{
		"name": "tom",
		"age": map[string]interface{}{
			"$gt": 17,
		},
	},
	"update": map[string]interface{}{
		"$set": map[string]interface{}{
			"sex": "male",
		},
	},
}

var distinctArgs = map[string]interface{}{
	"filter": map[string]interface{}{
		"age": map[string]interface{}{
			"$gt": 17,
		},
	},
	"fieldName": "name",
}

func TestMongodbDo(t *testing.T) {
	flag.Parse()
	var args = map[string]interface{}{}
	switch strings.ToLower(*cmd) {
	case "find":
		args = findArgs
	case "deleteone":
		args = deleteArgs
	case "deletemany":
		args = deleteArgs
	case "findoneanddelete":
		args = findArgs
	case "findoneandupdate":
		args = findOneAndUpdateArgs
	case "insertone":
		args = inserOneArgs
	case "insertmany":
		args = insertManyArgs
	case "updateone":
		args = updateOneArgs
	case "updatemany":
		args = updateManyArgs
	case "count":
		args = findArgs
	case "distinct":
		args = distinctArgs
	}
	proxy := mongodb.NewClientProxy("trpc.mongodb.server.service", client.WithTimeout(*timeout))

	_, err := proxy.Do(ctx, *cmd, database, collection, args)
	assert.NotNil(t, err)
}
