package mongodb

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"trpc.group/trpc-go/trpc-go"
)

var succArgs = map[string]interface{}{
	"filter": map[string]interface{}{
		"key": "123",
	},
	"sort": map[string]interface{}{
		"order": "asc",
	},
	"skip":  2.1,
	"limit": 3.2,
	"projection": map[string]interface{}{
		"key": "123",
	},
	"batchSize": 4.2,
	"collation": map[string]interface{}{
		"key": "123",
	},
	"update":         []interface{}{1, 2},
	"arrayFilters":   []interface{}{1, 2},
	"upsert":         true,
	"returnDocument": "After",
	"fieldName":      "123",
}

var sortArgs = map[string]interface{}{
	"sort": bson.D{{Key: "value", Value: 1}, {Key: "name", Value: -1}},
}

var errArgs = map[string]interface{}{
	"filter":         2,
	"sort":           2,
	"skip":           2,
	"limit":          2,
	"projection":     2,
	"batchSize":      2,
	"collation":      2,
	"update":         2,
	"arrayFilters":   2,
	"upsert":         2,
	"returnDocument": 2,
	"fieldName":      123,
}

func TestUnitExecuteFind(t *testing.T) {
	ops := options.Find()
	defer gomonkey.ApplyMethod(reflect.TypeOf(new(mongo.Collection)), "Find",
		func(coll *mongo.Collection, ctx context.Context, filter interface{},
			opts ...*options.FindOptions) (*mongo.Cursor, error) {
			if ctx == nil {
				return nil, fmt.Errorf("err")
			}
			*ops = *opts[0]
			return new(mongo.Cursor), nil
		},
	).Reset()
	seqDoc := bsoncore.BuildDocument(
		bsoncore.BuildDocument(
			nil,
			bsoncore.AppendDoubleElement(nil, "pi", 3.14159),
		),
		bsoncore.AppendStringElement(nil, "hello", "world"),
	)
	bs := &bsoncore.DocumentSequence{
		Style: bsoncore.SequenceStyle,
		Data:  seqDoc,
		Pos:   0,
	}

	var index int
	defer gomonkey.ApplyMethod(reflect.TypeOf(new(mongo.Cursor)), "Next",
		func(cursor *mongo.Cursor, ctx context.Context) bool {
			if index == 2 {
				return false
			}
			d, _ := bs.Next()
			cursor.Current = bson.Raw(d)
			index++
			return true
		},
	).Reset()

	defer gomonkey.ApplyMethod(reflect.TypeOf(new(mongo.Cursor)), "Close",
		func(cursor *mongo.Cursor, ctx context.Context) error {
			return nil
		},
	).Reset()

	type args struct {
		ctx  context.Context
		coll *mongo.Collection
		args map[string]interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    []map[string]interface{}
		opts    *options.FindOptions
		wantErr bool
	}{
		{"err", args{
			ctx:  trpc.BackgroundContext(),
			coll: &mongo.Collection{},
		}, nil, options.Find(), true},
		{"args err", args{
			ctx:  context.Background(),
			coll: &mongo.Collection{},
			args: errArgs,
		}, []map[string]interface{}{{
			"pi": 3.14159,
		}, {
			"hello": "world",
		}}, &options.FindOptions{}, false},
		{"succ", args{
			ctx:  context.Background(),
			coll: &mongo.Collection{},
			args: succArgs,
		}, []map[string]interface{}{}, &options.FindOptions{
			Sort: map[string]interface{}{
				"order": "asc",
			},
			Skip:  getInt64Point(int64(2)),
			Limit: getInt64Point(int64(3)),
			Projection: map[string]interface{}{
				"key": "123",
			},
			BatchSize: getInt32Point(4),
			Collation: new(options.Collation),
		}, false},
		{"succ", args{
			ctx:  context.Background(),
			coll: &mongo.Collection{},
			args: sortArgs,
		}, []map[string]interface{}{}, &options.FindOptions{
			Sort: bson.D{{Key: "value", Value: 1}, {Key: "name", Value: -1}},
			Skip: getInt64Point(int64(2)),
		}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _ = executeFindC(tt.args.ctx, tt.args.coll, tt.args.args)
			_, err := executeFind(tt.args.ctx, tt.args.coll, tt.args.args)
			assert.Nil(t, err)
		})
	}
}

func getInt64Point(i int64) *int64 {
	return &i
}
func getInt32Point(i int32) *int32 {
	return &i
}

func TestUnitExecuteDeleteMany(t *testing.T) {
	ops := options.Delete()
	defer gomonkey.ApplyMethod(reflect.TypeOf(new(mongo.Collection)), "DeleteMany",
		func(coll *mongo.Collection, ctx context.Context, filter interface{},
			opts ...*options.DeleteOptions) (*mongo.DeleteResult, error) {
			if ctx == nil {
				return nil, fmt.Errorf("err")
			}
			*ops = *opts[0]
			return &mongo.DeleteResult{
				DeletedCount: 1,
			}, nil
		},
	).Reset()

	type args struct {
		ctx  context.Context
		coll *mongo.Collection
		args map[string]interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]interface{}
		opts    *options.DeleteOptions
		wantErr bool
	}{
		{"err", args{
			ctx:  nil,
			coll: new(mongo.Collection),
		}, nil, options.Delete(), true},
		{"type err", args{
			ctx:  context.Background(),
			coll: new(mongo.Collection),
			args: errArgs,
		}, map[string]interface{}{
			"DeletedCount": float64(1),
		}, options.Delete(), false},
		{"succ", args{
			ctx:  context.Background(),
			coll: new(mongo.Collection),
			args: succArgs,
		}, map[string]interface{}{
			"DeletedCount": float64(1),
		}, &options.DeleteOptions{
			Collation: new(options.Collation),
		}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := executeDeleteMany(tt.args.ctx, tt.args.coll, tt.args.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("executeDeleteMany() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("executeDeleteMany() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(ops, tt.opts) {
				t.Errorf("executeFind() got = %v, want %v", ops, tt.opts)
			}
		})
	}
}

func TestUnitExecute(t *testing.T) {
	ops := options.Delete()
	defer gomonkey.ApplyMethod(reflect.TypeOf(new(mongo.Collection)), "DeleteOne",
		func(coll *mongo.Collection, ctx context.Context, filter interface{},
			opts ...*options.DeleteOptions) (*mongo.DeleteResult, error) {
			if ctx == nil {
				return nil, fmt.Errorf("err")
			}
			*ops = *opts[0]
			return &mongo.DeleteResult{
				DeletedCount: 1,
			}, nil
		},
	).Reset()

	type args struct {
		ctx  context.Context
		coll *mongo.Collection
		args map[string]interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]interface{}
		opts    *options.DeleteOptions
		wantErr bool
	}{
		{"err", args{
			ctx:  nil,
			coll: new(mongo.Collection),
		}, nil, options.Delete(), true},
		{"type err", args{
			ctx:  context.Background(),
			coll: new(mongo.Collection),
			args: errArgs,
		}, map[string]interface{}{
			"DeletedCount": float64(1),
		}, options.Delete(), false},
		{"succ", args{
			ctx:  context.Background(),
			coll: new(mongo.Collection),
			args: succArgs,
		}, map[string]interface{}{
			"DeletedCount": float64(1),
		}, &options.DeleteOptions{
			Collation: new(options.Collation),
		}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := executeDeleteOne(tt.args.ctx, tt.args.coll, tt.args.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("executeDeleteOne() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("executeDeleteOne() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(ops, tt.opts) {
				t.Errorf("executeFind() got = %v, want %v", ops, tt.opts)
			}
		})
	}
}

func TestUnitExecuteDeleteOne(t *testing.T) {
	ops := options.Delete()
	defer gomonkey.ApplyMethod(reflect.TypeOf(new(mongo.Collection)), "DeleteOne",
		func(coll *mongo.Collection, ctx context.Context, filter interface{},
			opts ...*options.DeleteOptions) (*mongo.DeleteResult, error) {
			if ctx == nil {
				return nil, fmt.Errorf("err")
			}
			*ops = *opts[0]
			return &mongo.DeleteResult{
				DeletedCount: 1,
			}, nil
		},
	).Reset()

	type args struct {
		ctx  context.Context
		coll *mongo.Collection
		args map[string]interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]interface{}
		opts    *options.DeleteOptions
		wantErr bool
	}{
		{"err", args{
			ctx:  nil,
			coll: new(mongo.Collection),
		}, nil, options.Delete(), true},
		{"type err", args{
			ctx:  context.Background(),
			coll: new(mongo.Collection),
			args: errArgs,
		}, map[string]interface{}{
			"DeletedCount": float64(1),
		}, options.Delete(), false},
		{"succ", args{
			ctx:  context.Background(),
			coll: new(mongo.Collection),
			args: succArgs,
		}, map[string]interface{}{
			"DeletedCount": float64(1),
		}, &options.DeleteOptions{
			Collation: new(options.Collation),
		}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := executeDeleteOne(tt.args.ctx, tt.args.coll, tt.args.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("executeDeleteOne() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("executeDeleteOne() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(ops, tt.opts) {
				t.Errorf("executeFind() got = %v, want %v", ops, tt.opts)
			}
		})
	}
}

func TestUnitExecuteFindOneAndDelete(t *testing.T) {
	ops := options.FindOneAndDelete()
	defer gomonkey.ApplyMethod(reflect.TypeOf(new(mongo.Collection)), "FindOneAndDelete",
		func(coll *mongo.Collection, ctx context.Context, filter interface{},
			opts ...*options.FindOneAndDeleteOptions) *mongo.SingleResult {
			*ops = *opts[0]
			return new(mongo.SingleResult)
		},
	).Reset()
	defer gomonkey.ApplyMethod(reflect.TypeOf(new(mongo.SingleResult)), "Decode",
		func(coll *mongo.SingleResult, v interface{}) error {
			value, _ := v.(*map[string]interface{})
			*value = map[string]interface{}{
				"pi":    3.1415,
				"hello": "world",
			}
			return nil
		},
	).Reset()
	type args struct {
		ctx  context.Context
		coll *mongo.Collection
		args map[string]interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]interface{}
		opts    *options.FindOneAndDeleteOptions
		wantErr bool
	}{
		{"type err", args{
			args: errArgs,
		}, map[string]interface{}{
			"pi":    3.1415,
			"hello": "world",
		}, options.FindOneAndDelete(), false},
		{"succ", args{
			args: succArgs,
		}, map[string]interface{}{
			"pi":    3.1415,
			"hello": "world",
		}, &options.FindOneAndDeleteOptions{
			Sort: map[string]interface{}{
				"order": "asc",
			},
			Projection: map[string]interface{}{
				"key": "123",
			},
			Collation: new(options.Collation),
		}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := executeFindOneAndDelete(tt.args.ctx, tt.args.coll, tt.args.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("executeFindOneAndDelete() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("executeFindOneAndDelete() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(ops, tt.opts) {
				t.Errorf("executeFind() got = %v, want %v", ops, tt.opts)
			}
		})
	}
}

func TestUnitExecuteFindOneAndUpdate(t *testing.T) {
	convey.Convey("TestUnitExecuteFindOneAndUpdate", t, func() {
		ops := options.FindOneAndUpdate()
		defer gomonkey.ApplyMethod(reflect.TypeOf(new(mongo.Collection)), "FindOneAndUpdate",
			func(coll *mongo.Collection, ctx context.Context, filter interface{},
				update interface{}, opts ...*options.FindOneAndUpdateOptions) *mongo.SingleResult {
				*ops = *opts[0]
				return new(mongo.SingleResult)
			},
		).Reset()
		defer gomonkey.ApplyMethod(reflect.TypeOf(new(mongo.SingleResult)), "Decode",
			func(coll *mongo.SingleResult, v interface{}) error {
				value, _ := v.(*map[string]interface{})
				*value = map[string]interface{}{
					"pi":    3.1415,
					"hello": "world",
				}
				return nil
			},
		).Reset()
		type args struct {
			ctx  context.Context
			coll *mongo.Collection
			args map[string]interface{}
		}
		tests := []struct {
			name    string
			args    args
			want    map[string]interface{}
			opts    *options.FindOneAndUpdateOptions
			wantErr bool
		}{
			{"type err", args{
				args: errArgs,
			}, map[string]interface{}{
				"pi":    3.1415,
				"hello": "world",
			}, options.FindOneAndUpdate(), false},
			{"succ", args{
				args: succArgs,
			}, map[string]interface{}{
				"pi":    3.1415,
				"hello": "world",
			}, &options.FindOneAndUpdateOptions{
				ArrayFilters: &options.ArrayFilters{
					Filters: []interface{}{1, 2},
				},
				Upsert:         getBoolPoint(true),
				ReturnDocument: getReturnDocumentPoint(options.After),
				Collation:      new(options.Collation),
				Projection: map[string]interface{}{
					"key": "123",
				},
				Sort: map[string]interface{}{
					"order": "asc",
				},
			}, false},
		}
		for _, tt := range tests {
			convey.Convey(tt.name, func() {
				_, _ = executeFindOneAndUpdateS(tt.args.ctx, tt.args.coll, tt.args.args)
				got, err := executeFindOneAndUpdate(tt.args.ctx, tt.args.coll, tt.args.args)
				if (err != nil) != tt.wantErr {
					t.Errorf("executeFindOneAndUpdate() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("executeFindOneAndUpdate() got = %v, want %v", got, tt.want)
				}
				convey.So(ops, convey.ShouldResemble, tt.opts)

			})
		}
	})
}

func getReturnDocumentPoint(document options.ReturnDocument) *options.ReturnDocument {
	return &document
}

func getBoolPoint(b bool) *bool {
	return &b
}

func TestUnitExecuteInsertOne(t *testing.T) {
	defer gomonkey.ApplyMethod(reflect.TypeOf(new(mongo.Collection)), "InsertOne",
		func(coll *mongo.Collection, ctx context.Context, document interface{},
			opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error) {
			if ctx == nil {
				return nil, fmt.Errorf("err")
			}
			return &mongo.InsertOneResult{
				InsertedID: "123",
			}, nil
		},
	).Reset()
	type args struct {
		ctx  context.Context
		coll *mongo.Collection
		args map[string]interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]interface{}
		wantErr bool
	}{
		{"not document", args{
			ctx: nil,
		}, nil, true},
		{"document type error", args{
			ctx: context.Background(),
			args: map[string]interface{}{
				"document": "123",
			},
		}, nil, true},
		{"insert error", args{
			ctx: nil,
			args: map[string]interface{}{
				"document": map[string]interface{}{
					"pi":    3.1415,
					"hello": "world",
				},
			},
		}, nil, true},
		{"succ", args{
			ctx: context.Background(),
			args: map[string]interface{}{
				"document": map[string]interface{}{
					"pi":    3.1415,
					"hello": "world",
				},
			},
		}, map[string]interface{}{
			"InsertedID": "123",
		}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := executeInsertOne(tt.args.ctx, tt.args.coll, tt.args.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("executeInsertOne() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("executeInsertOne() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUnitExecuteInsertMany(t *testing.T) {
	defer gomonkey.ApplyMethod(reflect.TypeOf(new(mongo.Collection)), "InsertMany",
		func(coll *mongo.Collection, ctx context.Context, documents []interface{},
			opts ...*options.InsertManyOptions) (*mongo.InsertManyResult, error) {
			if ctx == nil {
				return nil, fmt.Errorf("err")
			}
			return &mongo.InsertManyResult{
				InsertedIDs: []interface{}{
					"123",
					"456",
				},
			}, nil
		},
	).Reset()

	type args struct {
		ctx  context.Context
		coll *mongo.Collection
		args map[string]interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]interface{}
		wantErr bool
	}{
		{"not documents", args{
			ctx: nil,
		}, nil, true},
		{"documents type error", args{
			ctx: context.Background(),
			args: map[string]interface{}{
				"documents": "123",
			},
		}, nil, true},
		{"document type error", args{
			ctx: context.Background(),
			args: map[string]interface{}{
				"documents": []interface{}{
					123,
				},
			},
		}, nil, true},
		{"insert err", args{
			ctx: nil,
			args: map[string]interface{}{
				"documents": []interface{}{
					map[string]interface{}{
						"pi":    3.1415,
						"hello": "world",
					},
				},
			},
		}, nil, true},
		{"succ", args{
			ctx: context.Background(),
			args: map[string]interface{}{
				"documents": []interface{}{
					map[string]interface{}{
						"pi":    3.1415,
						"hello": "world",
					},
				},
			},
		}, map[string]interface{}{
			"InsertedIDs": []interface{}{"123", "456"},
		}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := executeInsertMany(tt.args.ctx, tt.args.coll, tt.args.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("executeInsertMany() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("executeInsertMany() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUnitExecuteUpdateOne(t *testing.T) {
	ops := options.Update()
	defer gomonkey.ApplyMethod(reflect.TypeOf(new(mongo.Collection)), "UpdateOne",
		func(coll *mongo.Collection, ctx context.Context, filter interface{}, update interface{},
			opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
			if ctx == nil {
				return nil, fmt.Errorf("err")
			}
			*ops = *opts[0]
			return &mongo.UpdateResult{
				MatchedCount:  1,
				ModifiedCount: 2,
				UpsertedCount: 3,
			}, nil
		},
	).Reset()

	type args struct {
		ctx  context.Context
		coll *mongo.Collection
		args map[string]interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]interface{}
		opts    *options.UpdateOptions
		wantErr bool
	}{
		{"err", args{
			ctx: nil,
		}, nil, options.Update(), true},
		{"type err", args{
			ctx:  context.Background(),
			args: errArgs,
		}, map[string]interface{}{
			"MatchedCount":  float64(1),
			"ModifiedCount": float64(2),
			"UpsertedCount": float64(3),
			"UpsertedID":    nil,
		}, options.Update(), false},
		{"succ", args{
			ctx:  context.Background(),
			args: succArgs,
		}, map[string]interface{}{
			"MatchedCount":  float64(1),
			"ModifiedCount": float64(2),
			"UpsertedCount": float64(3),
			"UpsertedID":    nil,
		}, &options.UpdateOptions{
			ArrayFilters: &options.ArrayFilters{
				Filters: []interface{}{1, 2},
			},
			Upsert:    getBoolPoint(true),
			Collation: new(options.Collation),
		}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := executeUpdateOne(tt.args.ctx, tt.args.coll, tt.args.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("executeUpdateOne() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("executeUpdateOne() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(ops, tt.opts) {
				t.Errorf("executeFind() got = %v, want %v", ops, tt.opts)
			}
		})
	}
}

func TestUnitExecuteUpdateMany(t *testing.T) {
	ops := options.Update()
	defer gomonkey.ApplyMethod(reflect.TypeOf(new(mongo.Collection)), "UpdateMany",
		func(coll *mongo.Collection, ctx context.Context, filter interface{}, update interface{},
			opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
			if ctx == nil {
				return nil, fmt.Errorf("err")
			}
			*ops = *opts[0]
			return &mongo.UpdateResult{
				MatchedCount:  1,
				ModifiedCount: 2,
				UpsertedCount: 3,
			}, nil
		},
	).Reset()

	type args struct {
		ctx  context.Context
		coll *mongo.Collection
		args map[string]interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]interface{}
		opts    *options.UpdateOptions
		wantErr bool
	}{
		{"err", args{
			ctx: nil,
		}, nil, options.Update(), true},
		{"type err", args{
			ctx:  context.Background(),
			args: errArgs,
		}, map[string]interface{}{
			"MatchedCount":  float64(1),
			"ModifiedCount": float64(2),
			"UpsertedCount": float64(3),
			"UpsertedID":    nil,
		}, options.Update(), false},
		{"succ", args{
			ctx:  context.Background(),
			args: succArgs,
		}, map[string]interface{}{
			"MatchedCount":  float64(1),
			"ModifiedCount": float64(2),
			"UpsertedCount": float64(3),
			"UpsertedID":    nil,
		}, &options.UpdateOptions{
			ArrayFilters: &options.ArrayFilters{
				Filters: []interface{}{1, 2},
			},
			Upsert:    getBoolPoint(true),
			Collation: new(options.Collation),
		}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := executeUpdateMany(tt.args.ctx, tt.args.coll, tt.args.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("executeUpdateMany() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("executeUpdateMany() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(ops, tt.opts) {
				t.Errorf("executeFind() got = %v, want %v", ops, tt.opts)
			}
		})
	}
}

func TestUnitExecuteCount(t *testing.T) {
	ops := options.Count()
	defer gomonkey.ApplyMethod(reflect.TypeOf(new(mongo.Collection)), "CountDocuments",
		func(coll *mongo.Collection, ctx context.Context, filter interface{},
			opts ...*options.CountOptions) (int64, error) {
			if ctx == nil {
				return 0, fmt.Errorf("err")
			}
			*ops = *opts[0]
			return 1, nil
		},
	).Reset()

	type args struct {
		ctx  context.Context
		coll *mongo.Collection
		args map[string]interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    int64
		opts    *options.CountOptions
		wantErr bool
	}{
		{"err", args{
			ctx: nil,
		}, 0, options.Count(), true},
		{"type err", args{
			ctx:  context.Background(),
			args: errArgs,
		}, 1, options.Count(), false},
		{"succ", args{
			ctx:  context.Background(),
			args: succArgs,
		}, 1, &options.CountOptions{
			Skip:      getInt64Point(2),
			Limit:     getInt64Point(3),
			Collation: new(options.Collation),
		}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := executeCount(tt.args.ctx, tt.args.coll, tt.args.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("executeCount() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("executeCount() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(ops, tt.opts) {
				t.Errorf("executeFind() got = %v, want %v", ops, tt.opts)
			}
		})
	}
}

func TestUnitExecuteDistinct(t *testing.T) {
	ops := options.Distinct()
	defer gomonkey.ApplyMethod(reflect.TypeOf(new(mongo.Collection)), "Distinct",
		func(coll *mongo.Collection, ctx context.Context, fieldName string, filter interface{},
			opts ...*options.DistinctOptions) ([]interface{}, error) {
			if ctx == nil {
				return nil, fmt.Errorf("err")
			}
			*ops = *opts[0]
			return []interface{}{"1", 23}, nil
		},
	).Reset()

	type args struct {
		ctx  context.Context
		coll *mongo.Collection
		args map[string]interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    []interface{}
		opts    *options.DistinctOptions
		wantErr bool
	}{
		{"err", args{
			ctx: nil,
		}, nil, options.Distinct(), true},
		{"type err", args{
			ctx:  context.Background(),
			args: errArgs,
		}, []interface{}{"1", 23}, options.Distinct(), false},
		{"succ", args{
			ctx:  context.Background(),
			args: succArgs,
		}, []interface{}{"1", 23}, &options.DistinctOptions{
			Collation: new(options.Collation),
		}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := executeDistinct(tt.args.ctx, tt.args.coll, tt.args.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("executeDistinct() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("executeDistinct() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(ops, tt.opts) {
				t.Errorf("executeFind() got = %v, want %v", ops, tt.opts)
			}
		})
	}
}

func TestUnitExecuteAggregate(t *testing.T) {
	ops := options.Aggregate()
	defer gomonkey.ApplyMethod(reflect.TypeOf(new(mongo.Collection)), "Aggregate",
		func(coll *mongo.Collection, ctx context.Context, filter interface{},
			opts ...*options.AggregateOptions) (*mongo.Cursor, error) {
			if ctx == nil {
				return nil, fmt.Errorf("err")
			}
			*ops = *opts[0]
			return &mongo.Cursor{}, nil
		},
	).Reset()

	type args struct {
		ctx  context.Context
		coll *mongo.Collection
		args map[string]interface{}
	}
	tArgs := make(map[string]interface{})
	tArgs["pipeline"] = make([]int, 0)
	tArgs["batchSize"] = float64(1)
	tArgs["collation"] = make(map[string]interface{})
	tArgs["maxTimeMS"] = float64(1)
	tests := []struct {
		name    string
		args    args
		want    []interface{}
		opts    *options.AggregateOptions
		wantErr bool
	}{
		{"err", args{
			ctx:  nil,
			args: tArgs,
		}, nil, options.Aggregate(), true},
		{"err_not_nil", args{
			ctx:  trpc.BackgroundContext(),
			args: tArgs,
		}, nil, options.Aggregate(), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _ = executeAggregateC(tt.args.ctx, tt.args.coll, tt.args.args)
			_, err := executeAggregate(tt.args.ctx, tt.args.coll, tt.args.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("executeAggregate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

		})
	}
}
func TestUnitCollationFromMap(t *testing.T) {
	type args struct {
		m map[string]interface{}
	}
	tests := []struct {
		name string
		args args
		want *options.Collation
	}{
		{"nil map", args{m: nil}, new(options.Collation)},
		{"empty map", args{m: map[string]interface{}{}}, new(options.Collation)},
		{"type err", args{m: map[string]interface{}{
			"locale":          1,
			"caseLevel":       2,
			"caseFirst":       3,
			"strength":        "str",
			"numericOrdering": 1,
			"alternate":       2,
			"maxVariable":     3,
			"normalization":   4,
			"backwards":       1,
		}}, new(options.Collation)},
		{"succ", args{m: map[string]interface{}{
			"locale":          "locale",
			"caseLevel":       true,
			"caseFirst":       "caseFirst",
			"strength":        3.15,
			"numericOrdering": false,
			"alternate":       "alternate",
			"maxVariable":     "maxVariable",
			"normalization":   true,
			"backwards":       false,
		}}, &options.Collation{
			Locale:          "locale",
			CaseLevel:       true,
			CaseFirst:       "caseFirst",
			Strength:        3,
			NumericOrdering: false,
			Alternate:       "alternate",
			MaxVariable:     "maxVariable",
			Normalization:   true,
			Backwards:       false,
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := collationFromMap(tt.args.m); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("collationFromMap() = %v, want %v", got, tt.want)
			}
		})
	}
}
func TestUnitBulkWrite(t *testing.T) {
	ops := options.BulkWrite()
	defer gomonkey.ApplyMethod(reflect.TypeOf(new(mongo.Collection)), "BulkWrite",
		func(coll *mongo.Collection, ctx context.Context, model []mongo.WriteModel,
			opts ...*options.BulkWriteOptions) (*mongo.BulkWriteResult, error) {
			*ops = *opts[0]
			return &mongo.BulkWriteResult{}, nil
		},
	).Reset()

	tArgs := make(map[string]interface{})
	tArgs["documents"] = []mongo.WriteModel{}
	tArgs["BulkWriteOptions"] = &options.BulkWriteOptions{}
	type args struct {
		ctx  context.Context
		coll *mongo.Collection
		args map[string]interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    []interface{}
		opts    *options.BulkWriteOptions
		wantErr bool
	}{
		{"err", args{
			ctx:  nil,
			args: tArgs,
		}, nil, options.BulkWrite(), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := executeBulkWrite(tt.args.ctx, tt.args.coll, tt.args.args)
			if err != nil {
				t.Errorf("executeBulkWrite() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
