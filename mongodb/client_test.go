package mongodb

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/client/mockclient"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/transport"
)

const fakeError = "fake error"

func TestUnitNewClientProxy(t *testing.T) {
	Convey("TestUnit_NewClientProxy_P0", t, func() {
		mysqlClient := NewClientProxy("trpc.mongo.xxx.xxx")
		rawClient, ok := mysqlClient.(*mongodbCli)
		So(ok, ShouldBeTrue)
		So(rawClient.Client, ShouldResemble, client.DefaultClient)
		So(rawClient.ServiceName, ShouldEqual, "trpc.mongo.xxx.xxx")
		So(len(rawClient.opts), ShouldEqual, 2)
	})
}

func TestInsert(t *testing.T) {
	defer gomonkey.ApplyMethod(reflect.TypeOf(new(ClientTransport)), "GetMgoClient",
		func(transport *ClientTransport, ctx context.Context, dsn string) (*mongo.Client, error) {
			return &mongo.Client{}, nil
		},
	).Reset()
	t.Run("InsertOne success", func(t *testing.T) {
		defer gomonkey.ApplyMethod(reflect.TypeOf(new(mongo.Collection)), "InsertOne",
			func(coll *mongo.Collection, ctx context.Context, document interface{},
				opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error) {
				return nil, nil
			},
		).Reset()
		_, err := NewClientProxy("trpc.mongodb.app.demo").InsertOne(context.Background(), "demo", "test", nil)
		assert.Nil(t, err)
	})
	t.Run("InsertOne fail", func(t *testing.T) {
		defer gomonkey.ApplyMethod(reflect.TypeOf(new(mongo.Collection)), "InsertOne",
			func(coll *mongo.Collection, ctx context.Context, document interface{},
				opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error) {
				return nil, errors.New(fakeError)
			},
		).Reset()
		_, err := NewClientProxy("trpc.mongodb.app.demo").InsertOne(context.Background(), "demo", "test", nil)
		assert.NotNil(t, err)
	})
	t.Run("InsertMany succ", func(t *testing.T) {
		defer gomonkey.ApplyMethod(reflect.TypeOf(new(mongo.Collection)), "InsertMany",
			func(coll *mongo.Collection, ctx context.Context, documents []interface{},
				opts ...*options.InsertManyOptions) (*mongo.InsertManyResult, error) {
				return nil, nil
			},
		).Reset()
		_, err := NewClientProxy("trpc.mongodb.app.demo").InsertMany(context.Background(), "demo", "test", nil)
		assert.Nil(t, err)
	})
	t.Run("InsertMany fail", func(t *testing.T) {
		defer gomonkey.ApplyMethod(reflect.TypeOf(new(mongo.Collection)), "InsertMany",
			func(coll *mongo.Collection, ctx context.Context, documents []interface{},
				opts ...*options.InsertManyOptions) (*mongo.InsertManyResult, error) {
				return nil, errors.New(fakeError)
			},
		).Reset()
		_, err := NewClientProxy("trpc.mongodb.app.demo").InsertMany(context.Background(), "demo", "test", nil)
		assert.NotNil(t, err)
	})
	t.Run("InsertMany duplicate succ", func(t *testing.T) {
		defer gomonkey.ApplyMethod(reflect.TypeOf(new(mongo.Collection)), "InsertMany",
			func(coll *mongo.Collection, ctx context.Context, documents []interface{},
				opts ...*options.InsertManyOptions) (*mongo.InsertManyResult, error) {
				return &mongo.InsertManyResult{
						InsertedIDs: []interface{}{"test"},
					}, mongo.BulkWriteException{
						WriteConcernError: &mongo.WriteConcernError{Name: "name", Code: 100, Message: "bar"},
						WriteErrors: []mongo.BulkWriteError{
							{
								WriteError: mongo.WriteError{Code: 11000, Message: "blah E11000 blah"},
								Request:    &mongo.InsertOneModel{}},
						},
						Labels: []string{"otherError"},
					}
			},
		).Reset()
		result, err := NewClientProxy("trpc.mongodb.app.demo").InsertMany(context.Background(), "demo", "test", nil)
		assert.NotNil(t, err)
		assert.NotNil(t, result)
	})

	t.Run("BulkWrite duplicate succ", func(t *testing.T) {
		defer gomonkey.ApplyMethod(reflect.TypeOf(new(mongo.Collection)), "BulkWrite",
			func(coll *mongo.Collection, ctx context.Context, models []mongo.WriteModel,
				opts ...*options.BulkWriteOptions) (*mongo.BulkWriteResult, error) {
				return &mongo.BulkWriteResult{
						InsertedCount: 1,
					}, mongo.BulkWriteException{
						WriteConcernError: &mongo.WriteConcernError{Name: "name", Code: 100, Message: "bar"},
						WriteErrors: []mongo.BulkWriteError{
							{
								WriteError: mongo.WriteError{Code: 11000, Message: "blah E11000 blah"},
								Request:    &mongo.InsertOneModel{}},
						},
						Labels: []string{"otherError"},
					}
			},
		).Reset()
		models := make([]mongo.WriteModel, 0, 2)
		model := mongo.NewUpdateOneModel()
		model.SetUpsert(true)
		model.SetFilter(bson.M{"test": 1})
		model.SetUpdate(bson.M{
			"$set":         bson.M{"test": 1},
			"$setOnInsert": bson.M{"test": 1},
		})
		models = append(models, model)
		model = mongo.NewUpdateOneModel()
		model.SetUpsert(true)
		model.SetFilter(bson.M{"test": 2})
		model.SetUpdate(bson.M{
			"$set":         bson.M{"test": 2},
			"$setOnInsert": bson.M{"test": 2},
		})
		models = append(models, model)
		result, err := NewClientProxy("trpc.mongodb.app.demo").BulkWrite(context.Background(), "demo", "test", models)
		assert.NotNil(t, err)
		assert.NotNil(t, result)
	})
	t.Run("Transaction succ", func(t *testing.T) {
		client, _ := mongo.Connect(trpc.BackgroundContext(),
			options.Client().ApplyURI("mongodb://127.0.0.1:27017"))
		assert.NotNil(t, client)

		sess, _ := client.StartSession()
		assert.NotNil(t, sess)

		defer gomonkey.ApplyMethod(reflect.TypeOf(new(mongo.Client)), "StartSession",
			func(coll *mongo.Client, opts ...*options.SessionOptions) (mongo.Session, error) {
				return sess, nil
			},
		).Reset()
		err := NewClientProxy("trpc.mongodb.app.demo").Transaction(context.Background(),
			func(sc mongo.SessionContext) error {
				return nil
			}, nil)
		assert.Nil(t, err)

		err = NewClientProxy("trpc.mongodb.app.demo").Transaction(context.Background(),
			func(sc mongo.SessionContext) error {
				return errs.New(-1, "failed test")
			}, nil)
		assert.NotNil(t, err)

	})

	t.Run("Transaction fail", func(t *testing.T) {
		client, _ := mongo.Connect(trpc.BackgroundContext(),
			options.Client().ApplyURI("mongodb://127.0.0.1:27017"))
		assert.NotNil(t, client)

		sess, _ := client.StartSession()

		defer gomonkey.ApplyMethod(reflect.TypeOf(new(mongo.Client)), "StartSession",
			func(coll *mongo.Client, opts ...*options.SessionOptions) (mongo.Session, error) {
				return sess, nil
			},
		).Reset()
		err := NewClientProxy("trpc.mongodb.app.demo").Transaction(context.Background(),
			func(sc mongo.SessionContext) error {
				return nil
			}, nil)
		assert.Nil(t, err)
		err = NewClientProxy("trpc.mongodb.app.demo").Transaction(context.Background(),
			func(sc mongo.SessionContext) error {
				return errs.New(-1, "failed test")
			}, nil)
		assert.NotNil(t, err)
	})
}

func TestDelete(t *testing.T) {
	defer gomonkey.ApplyMethod(reflect.TypeOf(new(ClientTransport)), "GetMgoClient",
		func(transport *ClientTransport, ctx context.Context, dsn string) (*mongo.Client, error) {
			return &mongo.Client{}, nil
		},
	).Reset()
	t.Run("DeleteOne succ", func(t *testing.T) {
		defer gomonkey.ApplyMethod(reflect.TypeOf(new(mongo.Collection)), "DeleteOne",
			func(coll *mongo.Collection, ctx context.Context, filter interface{},
				opts ...*options.DeleteOptions) (*mongo.DeleteResult, error) {
				return nil, nil
			},
		).Reset()
		_, err := NewClientProxy("trpc.mongodb.app.demo").DeleteOne(context.Background(), "demo", "test", nil)
		assert.Nil(t, err)
	})
	t.Run("DeleteOne fail", func(t *testing.T) {
		defer gomonkey.ApplyMethod(reflect.TypeOf(new(mongo.Collection)), "DeleteOne",
			func(coll *mongo.Collection, ctx context.Context, filter interface{},
				opts ...*options.DeleteOptions) (*mongo.DeleteResult, error) {
				return nil, errors.New(fakeError)
			},
		).Reset()
		_, err := NewClientProxy("trpc.mongodb.app.demo").DeleteOne(context.Background(), "demo", "test", nil)
		assert.NotNil(t, err)
	})
	t.Run("DeleteMany succ", func(t *testing.T) {
		defer gomonkey.ApplyMethod(reflect.TypeOf(new(mongo.Collection)), "DeleteMany",
			func(coll *mongo.Collection, ctx context.Context, filter interface{},
				opts ...*options.DeleteOptions) (*mongo.DeleteResult, error) {
				return nil, nil
			},
		).Reset()
		_, err := NewClientProxy("trpc.mongodb.app.demo").DeleteMany(context.Background(), "demo", "test", nil)
		assert.Nil(t, err)
	})
	t.Run("DeleteMany fail", func(t *testing.T) {
		defer gomonkey.ApplyMethod(reflect.TypeOf(new(mongo.Collection)), "DeleteMany",
			func(coll *mongo.Collection, ctx context.Context, filter interface{},
				opts ...*options.DeleteOptions) (*mongo.DeleteResult, error) {
				return nil, errors.New(fakeError)
			},
		).Reset()
		_, err := NewClientProxy("trpc.mongodb.app.demo").DeleteMany(context.Background(), "demo", "test", nil)
		assert.NotNil(t, err)
	})
}

func TestUpdate(t *testing.T) {
	defer gomonkey.ApplyMethod(reflect.TypeOf(new(ClientTransport)), "GetMgoClient",
		func(transport *ClientTransport, ctx context.Context, dsn string) (*mongo.Client, error) {
			return &mongo.Client{}, nil
		},
	).Reset()
	t.Run("UpdateOne succ", func(t *testing.T) {
		defer gomonkey.ApplyMethod(reflect.TypeOf(new(mongo.Collection)), "UpdateOne",
			func(coll *mongo.Collection, ctx context.Context, filter interface{}, update interface{},
				opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
				return nil, nil
			},
		).Reset()
		_, err := NewClientProxy("trpc.mongodb.app.demo").UpdateOne(context.Background(), "demo", "test", nil, nil)
		assert.Nil(t, err)
	})
	t.Run("UpdateOne fail", func(t *testing.T) {
		defer gomonkey.ApplyMethod(reflect.TypeOf(new(mongo.Collection)), "UpdateOne",
			func(coll *mongo.Collection, ctx context.Context, filter interface{}, update interface{},
				opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
				return nil, errors.New(fakeError)
			},
		).Reset()
		_, err := NewClientProxy("trpc.mongodb.app.demo").UpdateOne(context.Background(), "demo", "test", nil, nil)
		assert.NotNil(t, err)
	})
	t.Run("UpdateMany succ", func(t *testing.T) {
		defer gomonkey.ApplyMethod(reflect.TypeOf(new(mongo.Collection)), "UpdateMany",
			func(coll *mongo.Collection, ctx context.Context, filter interface{}, update interface{},
				opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
				return nil, nil
			},
		).Reset()
		_, err := NewClientProxy("trpc.mongodb.app.demo").UpdateMany(context.Background(), "demo", "test", nil, nil)
		assert.Nil(t, err)
	})
	t.Run("UpdateMany fail", func(t *testing.T) {
		defer gomonkey.ApplyMethod(reflect.TypeOf(new(mongo.Collection)), "UpdateMany",
			func(coll *mongo.Collection, ctx context.Context, filter interface{}, update interface{},
				opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
				return nil, errors.New(fakeError)
			},
		).Reset()
		_, err := NewClientProxy("trpc.mongodb.app.demo").UpdateMany(context.Background(), "demo", "test", nil, nil)
		assert.NotNil(t, err)
	})
}

func TestFind(t *testing.T) {
	defer gomonkey.ApplyMethod(reflect.TypeOf(new(ClientTransport)), "GetMgoClient",
		func(transport *ClientTransport, ctx context.Context, dsn string) (*mongo.Client, error) {
			return &mongo.Client{}, nil
		},
	).Reset()
	t.Run("Find succ", func(t *testing.T) {
		defer gomonkey.ApplyMethod(reflect.TypeOf(new(mongo.Collection)), "Find",
			func(coll *mongo.Collection, ctx context.Context, filter interface{},
				opts ...*options.FindOptions) (*mongo.Cursor, error) {
				return nil, nil
			},
		).Reset()
		_, err := NewClientProxy("trpc.mongodb.app.demo").Find(context.Background(), "demo", "test", nil)
		assert.Nil(t, err)
	})
	t.Run("Find fail", func(t *testing.T) {
		defer gomonkey.ApplyMethod(reflect.TypeOf(new(mongo.Collection)), "Find",
			func(coll *mongo.Collection, ctx context.Context, filter interface{},
				opts ...*options.FindOptions) (*mongo.Cursor, error) {
				return nil, errors.New(fakeError)
			},
		).Reset()
		_, err := NewClientProxy("trpc.mongodb.app.demo").Find(context.Background(), "demo", "test", nil)
		assert.NotNil(t, err)
	})
	t.Run("FindOne succ", func(t *testing.T) {
		defer gomonkey.ApplyMethod(reflect.TypeOf(new(mongo.Collection)), "FindOne",
			func(coll *mongo.Collection, ctx context.Context, filter interface{},
				opts ...*options.FindOneOptions) *mongo.SingleResult {
				return nil
			},
		).Reset()
		rst := NewClientProxy("trpc.mongodb.app.demo").FindOne(context.Background(), "demo", "test", nil)
		assert.Nil(t, rst)
	})
	t.Run("FindOne fail", func(t *testing.T) {
		defer gomonkey.ApplyMethod(reflect.TypeOf(new(ClientTransport)), "RoundTrip",
			func(ct *ClientTransport, ctx context.Context, _ []byte, callOpts ...transport.RoundTripOption) ([]byte,
				error) {
				return nil, errors.New(fakeError)
			},
		).Reset()
		rst := NewClientProxy("trpc.mongodb.app.demo").FindOne(context.Background(), "demo", "test", nil)
		assert.Equal(t, rst.Err().Error(), fakeError)
	})
	t.Run("FindOneAndDelete succ", func(t *testing.T) {
		defer gomonkey.ApplyMethod(reflect.TypeOf(new(mongo.Collection)), "FindOneAndDelete",
			func(coll *mongo.Collection, ctx context.Context, filter interface{},
				opts ...*options.FindOneAndDeleteOptions) *mongo.SingleResult {
				return nil
			},
		).Reset()
		rst := NewClientProxy("trpc.mongodb.app.demo").FindOneAndDelete(context.Background(), "demo", "test", nil)
		assert.Nil(t, rst)
	})
	t.Run("FindOneAndDelete fail", func(t *testing.T) {
		defer gomonkey.ApplyMethod(reflect.TypeOf(new(ClientTransport)), "RoundTrip",
			func(ct *ClientTransport, ctx context.Context, _ []byte, callOpts ...transport.RoundTripOption) ([]byte,
				error) {
				return nil, errors.New(fakeError)
			},
		).Reset()
		rst := NewClientProxy("trpc.mongodb.app.demo").FindOneAndDelete(context.Background(), "demo", "test", nil)
		assert.Equal(t, rst.Err().Error(), fakeError)
	})
	t.Run("FindOneAndReplace succ", func(t *testing.T) {
		defer gomonkey.ApplyMethod(reflect.TypeOf(new(mongo.Collection)), "FindOneAndReplace",
			func(coll *mongo.Collection, ctx context.Context, filter interface{},
				replacement interface{}, opts ...*options.FindOneAndReplaceOptions) *mongo.SingleResult {
				return nil
			},
		).Reset()
		rst := NewClientProxy("trpc.mongodb.app.demo").FindOneAndReplace(context.Background(), "demo", "test", nil, nil)
		assert.Nil(t, rst)
	})
	t.Run("FindOneAndReplace fail", func(t *testing.T) {
		defer gomonkey.ApplyMethod(reflect.TypeOf(new(ClientTransport)), "RoundTrip",
			func(ct *ClientTransport, ctx context.Context, _ []byte, callOpts ...transport.RoundTripOption) ([]byte,
				error) {
				return nil, errors.New(fakeError)
			},
		).Reset()
		rst := NewClientProxy("trpc.mongodb.app.demo").FindOneAndReplace(context.Background(), "demo", "test", nil, nil)
		assert.Equal(t, rst.Err().Error(), fakeError)
	})
	t.Run("FindOneAndUpdate succ", func(t *testing.T) {
		defer gomonkey.ApplyMethod(reflect.TypeOf(new(mongo.Collection)), "FindOneAndUpdate",
			func(coll *mongo.Collection, ctx context.Context, filter interface{},
				replacement interface{}, opts ...*options.FindOneAndUpdateOptions) *mongo.SingleResult {
				return nil
			},
		).Reset()
		rst := NewClientProxy("trpc.mongodb.app.demo").FindOneAndUpdate(context.Background(), "demo", "test", nil, nil)
		assert.Nil(t, rst)
	})
	t.Run("FindOneAndUpdate fail", func(t *testing.T) {
		defer gomonkey.ApplyMethod(reflect.TypeOf(new(ClientTransport)), "RoundTrip",
			func(ct *ClientTransport, ctx context.Context, _ []byte, callOpts ...transport.RoundTripOption) ([]byte,
				error) {
				return nil, errors.New(fakeError)
			},
		).Reset()
		rst := NewClientProxy("trpc.mongodb.app.demo").FindOneAndUpdate(context.Background(), "demo", "test", nil, nil)
		assert.Equal(t, rst.Err().Error(), fakeError)
	})

}
func TestUnitMongodbCli_DriverProxy(t *testing.T) {
	defer gomonkey.ApplyMethod(reflect.TypeOf(new(ClientTransport)), "GetMgoClient",
		func(transport *ClientTransport, ctx context.Context, dsn string) (*mongo.Client, error) {
			return &mongo.Client{}, nil
		},
	).Reset()
	defer gomonkey.ApplyMethod(reflect.TypeOf(new(mongo.Collection)), "BulkWrite",
		func(coll *mongo.Collection, ctx context.Context, models []mongo.WriteModel,
			opts ...*options.BulkWriteOptions) (*mongo.BulkWriteResult, error) {
			return nil, nil
		},
	).Reset()
	_, err := NewClientProxy("trpc.mongodb.app.demo").BulkWrite(context.Background(), "demo", "test", nil)
	assert.Nil(t, err)

	defer gomonkey.ApplyMethod(reflect.TypeOf(new(mongo.Collection)), "ReplaceOne",
		func(coll *mongo.Collection, ctx context.Context, filter interface{},
			replacement interface{}, opts ...*options.ReplaceOptions) (*mongo.UpdateResult, error) {
			return nil, nil
		},
	).Reset()
	_, err = NewClientProxy("trpc.mongodb.app.demo").ReplaceOne(context.Background(), "demo", "test", nil, nil)
	assert.Nil(t, err)

	defer gomonkey.ApplyMethod(reflect.TypeOf(new(mongo.Collection)), "Aggregate",
		func(coll *mongo.Collection, ctx context.Context, pipeline interface{},
			opts ...*options.AggregateOptions) (*mongo.Cursor, error) {
			return nil, nil
		},
	).Reset()
	_, err = NewClientProxy("trpc.mongodb.app.demo").Aggregate(context.Background(), "demo", "test", nil)
	assert.Nil(t, err)

	defer gomonkey.ApplyMethod(reflect.TypeOf(new(mongo.Collection)), "CountDocuments",
		func(coll *mongo.Collection, ctx context.Context, filter interface{},
			opts ...*options.CountOptions) (int64, error) {
			return 0, nil
		},
	).Reset()
	_, err = NewClientProxy("trpc.mongodb.app.demo").CountDocuments(context.Background(), "demo", "test", nil)
	assert.Nil(t, err)

	defer gomonkey.ApplyMethod(reflect.TypeOf(new(mongo.Collection)), "EstimatedDocumentCount",
		func(coll *mongo.Collection, ctx context.Context,
			opts ...*options.EstimatedDocumentCountOptions) (int64, error) {
			return 0, nil
		},
	).Reset()
	_, err = NewClientProxy("trpc.mongodb.app.demo").EstimatedDocumentCount(context.Background(), "demo", "test", nil)
	assert.Nil(t, err)

	defer gomonkey.ApplyMethod(reflect.TypeOf(new(mongo.Collection)), "Distinct",
		func(coll *mongo.Collection, ctx context.Context, fieldName string, filter interface{},
			opts ...*options.DistinctOptions) ([]interface{}, error) {
			return nil, nil
		},
	).Reset()
	_, err = NewClientProxy("trpc.mongodb.app.demo").Distinct(context.Background(), "demo", "test", "", nil)
	assert.Nil(t, err)

	defer gomonkey.ApplyMethod(reflect.TypeOf(new(mongo.Client)), "Watch",
		func(coll *mongo.Client, ctx context.Context, pipeline interface{},
			opts ...*options.ChangeStreamOptions) (*mongo.ChangeStream, error) {
			return nil, nil
		},
	).Reset()
	_, err = NewClientProxy("trpc.mongodb.app.demo").Watch(context.Background(), nil, nil)
	assert.Nil(t, err)

	defer gomonkey.ApplyMethod(reflect.TypeOf(new(mongo.Database)), "Watch",
		func(coll *mongo.Database, ctx context.Context, pipeline interface{},
			opts ...*options.ChangeStreamOptions) (*mongo.ChangeStream, error) {
			return nil, nil
		},
	).Reset()
	_, err = NewClientProxy("trpc.mongodb.app.demo").WatchDatabase(context.Background(), "demo", nil, nil)
	assert.Nil(t, err)

	defer gomonkey.ApplyMethod(reflect.TypeOf(new(mongo.Collection)), "Watch",
		func(coll *mongo.Collection, ctx context.Context, pipeline interface{},
			opts ...*options.ChangeStreamOptions) (*mongo.ChangeStream, error) {
			return nil, nil
		},
	).Reset()
	_, err = NewClientProxy("trpc.mongodb.app.demo").WatchCollection(context.Background(), "demo", "test", nil, nil)
	assert.Nil(t, err)

	defer gomonkey.ApplyMethod(reflect.TypeOf(new(mongo.Client)), "Disconnect",
		func(coll *mongo.Client, ctx context.Context) error {
			return nil
		},
	).Reset()
	err = NewClientProxy("trpc.mongodb.app.demo").Disconnect(context.Background())
	assert.Nil(t, err)
}

// Test the watch interface.
func TestUnitMongodbCli_Watch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCli := mockclient.NewMockClient(ctrl)
	mockCli.EXPECT().Invoke(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("cli err")).AnyTimes()

	cli := &mongodbCli{
		Client: mockCli,
	}
	_, err := cli.Watch(context.Background(), nil, nil)
	assert.NotNil(t, err)

	_, err = cli.WatchDatabase(context.Background(), "demo", nil, nil)
	assert.NotNil(t, err)

	_, err = cli.WatchCollection(context.Background(), "demo", "test", nil, nil)
	assert.NotNil(t, err)

}

func TestUnitMongodbCli_Do(t *testing.T) {

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCli := mockclient.NewMockClient(ctrl)
	m1 := mockCli.EXPECT().Invoke(gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("err"))
	m2 := mockCli.EXPECT().Invoke(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, reqbody interface{}, rspbody interface{}, opt ...client.Option) error {
			r, _ := rspbody.(*Response)
			r.Result = "result"
			return nil
		})
	gomock.InOrder(m1, m2)

	type fields struct {
		ServiceName string
		Client      client.Client
		opts        []client.Option
	}
	type args struct {
		ctx        context.Context
		cmd        string
		database   string
		collection string
		args       map[string]interface{}
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    interface{}
		wantErr bool
	}{
		{"err", fields{Client: mockCli},
			args{ctx: context.Background()}, nil, true},
		{"succ", fields{Client: mockCli},
			args{ctx: context.Background()}, "result", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &mongodbCli{
				ServiceName: tt.fields.ServiceName,
				Client:      tt.fields.Client,
				opts:        tt.fields.opts,
			}
			got, err := c.Do(tt.args.ctx, tt.args.cmd, tt.args.database, tt.args.collection, tt.args.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Do() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Do() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_mongodbCli_CreateMany(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCli := mockclient.NewMockClient(ctrl)
	m1 := mockCli.EXPECT().Invoke(gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("err"))
	m2 := mockCli.EXPECT().Invoke(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, reqbody interface{}, rspbody interface{}, opt ...client.Option) error {
			r, _ := rspbody.(*Response)
			r.Result = []string{"index_1", "index_2"}
			return nil
		})
	m3 := mockCli.EXPECT().Invoke(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, reqbody interface{}, rspbody interface{}, opt ...client.Option) error {
			r, _ := rspbody.(*Response)
			r.Result = "not match kind"
			return nil
		})
	gomock.InOrder(m1, m2, m3)

	type fields struct {
		ServiceName string
		Client      client.Client
		opts        []client.Option
	}
	type args struct {
		ctx        context.Context
		database   string
		collection string
		models     []mongo.IndexModel
		opts       []*options.CreateIndexesOptions
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []string
		wantErr bool
	}{
		{"err", fields{Client: mockCli},
			args{ctx: context.Background()}, nil, true},
		{"succ", fields{Client: mockCli},
			args{ctx: context.Background()}, []string{"index_1", "index_2"}, false},
		{"not match kind", fields{Client: mockCli},
			args{ctx: context.Background()}, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &mongodbCli{
				ServiceName: tt.fields.ServiceName,
				Client:      tt.fields.Client,
				opts:        tt.fields.opts,
			}
			got, err := c.CreateMany(tt.args.ctx, tt.args.database, tt.args.collection, tt.args.models, tt.args.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateMany() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CreateMany() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_mongodbCli_RunCommand(t *testing.T) {
	patch := gomonkey.ApplyMethod(reflect.TypeOf(new(ClientTransport)), "GetMgoClient",
		func(transport *ClientTransport, ctx context.Context, dsn string) (*mongo.Client, error) {
			return &mongo.Client{}, nil
		},
	)
	defer patch.Reset()

	t.Run("RunCommand succ", func(t *testing.T) {
		patch2 := gomonkey.ApplyMethod(reflect.TypeOf(new(mongo.Database)), "RunCommand",
			// There are more than 126 columns here,
			// which does not meet the specification and blocks the upload of the code,
			// and the line break is processed.
			func(db *mongo.Database, ctx context.Context,
				runCommand interface{}, opts ...*options.RunCmdOptions) *mongo.SingleResult {
				return &mongo.SingleResult{}
			},
		)
		defer patch2.Reset()
		rst := NewClientProxy("trpc.mongodb.app.demo").RunCommand(context.Background(), "admin", nil)
		assert.NotNil(t, rst)
	})
	t.Run("RunCommand fail", func(t *testing.T) {
		patch3 := gomonkey.ApplyMethod(reflect.TypeOf(new(ClientTransport)), "RoundTrip",
			func(ct *ClientTransport, ctx context.Context, _ []byte, callOpts ...transport.RoundTripOption) ([]byte,
				error) {
				return nil, errors.New(fakeError)
			},
		)
		defer patch3.Reset()
		rst := NewClientProxy("trpc.mongodb.app.demo").RunCommand(context.Background(), "admin", nil)
		assert.Equal(t, rst.Err().Error(), fakeError)
	})

}

func Test_mongodbCli_CreateOne(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCli := mockclient.NewMockClient(ctrl)
	m1 := mockCli.EXPECT().Invoke(gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("err"))
	m2 := mockCli.EXPECT().Invoke(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, reqbody interface{}, rspbody interface{}, opt ...client.Option) error {
			r, _ := rspbody.(*Response)
			r.Result = "create_one"
			return nil
		})
	m3 := mockCli.EXPECT().Invoke(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, reqbody interface{}, rspbody interface{}, opt ...client.Option) error {
			r, _ := rspbody.(*Response)
			r.Result = 123
			return nil
		})
	gomock.InOrder(m1, m2, m3)
	type fields struct {
		ServiceName string
		Client      client.Client
		opts        []client.Option
	}
	type args struct {
		ctx        context.Context
		database   string
		collection string
		model      mongo.IndexModel
		opts       []*options.CreateIndexesOptions
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{"err", fields{Client: mockCli},
			args{ctx: context.Background()}, "", true},
		{"succ", fields{Client: mockCli},
			args{ctx: context.Background()}, "create_one", false},
		{"not match kind", fields{Client: mockCli},
			args{ctx: context.Background()}, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &mongodbCli{
				ServiceName: tt.fields.ServiceName,
				Client:      tt.fields.Client,
				opts:        tt.fields.opts,
			}
			got, err := c.CreateOne(tt.args.ctx, tt.args.database, tt.args.collection, tt.args.model, tt.args.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateOne() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("CreateOne() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_mongodbCli_DropOne(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCli := mockclient.NewMockClient(ctrl)
	m1 := mockCli.EXPECT().Invoke(gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("err"))
	m2 := mockCli.EXPECT().Invoke(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, reqbody interface{}, rspbody interface{}, opt ...client.Option) error {
			r, _ := rspbody.(*Response)
			r.Result = bson.Raw("drop_one")
			return nil
		})
	m3 := mockCli.EXPECT().Invoke(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, reqbody interface{}, rspbody interface{}, opt ...client.Option) error {
			r, _ := rspbody.(*Response)
			r.Result = 123
			return nil
		})
	gomock.InOrder(m1, m2, m3)
	type fields struct {
		ServiceName string
		Client      client.Client
		opts        []client.Option
	}
	type args struct {
		ctx        context.Context
		database   string
		collection string
		name       string
		opts       []*options.DropIndexesOptions
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bson.Raw
		wantErr bool
	}{
		{"err", fields{Client: mockCli},
			args{ctx: context.Background()}, nil, true},
		{"succ", fields{Client: mockCli},
			args{ctx: context.Background()}, bson.Raw("drop_one"), false},
		{"not match kind", fields{Client: mockCli},
			args{ctx: context.Background()}, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &mongodbCli{
				ServiceName: tt.fields.ServiceName,
				Client:      tt.fields.Client,
				opts:        tt.fields.opts,
			}
			got, err := c.DropOne(tt.args.ctx, tt.args.database, tt.args.collection, tt.args.name, tt.args.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("DropOne() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DropOne() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_mongodbCli_DropAll(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCli := mockclient.NewMockClient(ctrl)
	m1 := mockCli.EXPECT().Invoke(gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("err"))
	m2 := mockCli.EXPECT().Invoke(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, reqbody interface{}, rspbody interface{}, opt ...client.Option) error {
			r, _ := rspbody.(*Response)
			r.Result = bson.Raw("drop_all")
			return nil
		})
	m3 := mockCli.EXPECT().Invoke(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, reqbody interface{}, rspbody interface{}, opt ...client.Option) error {
			r, _ := rspbody.(*Response)
			r.Result = 123
			return nil
		})
	type fields struct {
		ServiceName string
		Client      client.Client
		opts        []client.Option
	}
	type args struct {
		ctx        context.Context
		database   string
		collection string
		opts       []*options.DropIndexesOptions
	}
	gomock.InOrder(m1, m2, m3)
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bson.Raw
		wantErr bool
	}{
		{"err", fields{Client: mockCli},
			args{ctx: context.Background()}, nil, true},
		{"succ", fields{Client: mockCli},
			args{ctx: context.Background()}, bson.Raw("drop_all"), false},
		{"not match kind", fields{Client: mockCli},
			args{ctx: context.Background()}, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &mongodbCli{
				ServiceName: tt.fields.ServiceName,
				Client:      tt.fields.Client,
				opts:        tt.fields.opts,
			}
			got, err := c.DropAll(tt.args.ctx, tt.args.database, tt.args.collection, tt.args.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("DropAll() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DropAll() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_mongodbCli_Collection(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCli := mockclient.NewMockClient(ctrl)
	m1 := mockCli.EXPECT().Invoke(gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("err"))
	m2 := mockCli.EXPECT().Invoke(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, reqbody interface{}, rspbody interface{}, opt ...client.Option) error {
			r, _ := rspbody.(*Response)
			r.Result = &mongo.Collection{}
			return nil
		})
	m3 := mockCli.EXPECT().Invoke(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, reqbody interface{}, rspbody interface{}, opt ...client.Option) error {
			r, _ := rspbody.(*Response)
			r.Result = 123
			return nil
		})
	gomock.InOrder(m1, m2, m3)

	type fields struct {
		ServiceName string
		Client      client.Client
		opts        []client.Option
	}
	type args struct {
		database   string
		collection string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *mongo.Collection
		wantErr bool
	}{
		{"err", fields{Client: mockCli}, args{}, nil, true},
		{"suc", fields{Client: mockCli}, args{}, &mongo.Collection{}, false},
		{"not match kind", fields{Client: mockCli}, args{}, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &mongodbCli{
				ServiceName: tt.fields.ServiceName,
				Client:      tt.fields.Client,
				opts:        tt.fields.opts,
			}
			got, err := c.Collection(trpc.BackgroundContext(), tt.args.database, tt.args.collection)
			if (err != nil) != tt.wantErr {
				t.Errorf("Collection() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Collection() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_mongodbCli_Database(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCli := mockclient.NewMockClient(ctrl)
	m1 := mockCli.EXPECT().Invoke(gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("err"))
	m2 := mockCli.EXPECT().Invoke(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, reqbody interface{}, rspbody interface{}, opt ...client.Option) error {
			r, _ := rspbody.(*Response)
			r.Result = &mongo.Database{}
			return nil
		})
	m3 := mockCli.EXPECT().Invoke(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, reqbody interface{}, rspbody interface{}, opt ...client.Option) error {
			r, _ := rspbody.(*Response)
			r.Result = 123
			return nil
		})
	gomock.InOrder(m1, m2, m3)

	type fields struct {
		ServiceName string
		Client      client.Client
		opts        []client.Option
	}
	type args struct {
		database string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *mongo.Database
		wantErr bool
	}{
		{"err", fields{Client: mockCli}, args{}, nil, true},
		{"suc", fields{Client: mockCli}, args{}, &mongo.Database{}, false},
		{"not match kind", fields{Client: mockCli}, args{}, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &mongodbCli{
				ServiceName: tt.fields.ServiceName,
				Client:      tt.fields.Client,
				opts:        tt.fields.opts,
			}
			got, err := c.Database(trpc.BackgroundContext(), tt.args.database)
			if (err != nil) != tt.wantErr {
				t.Errorf("Database() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Database() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_mongodbCli_Indexes(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCli := mockclient.NewMockClient(ctrl)
	m1 := mockCli.EXPECT().Invoke(gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("err"))
	m2 := mockCli.EXPECT().Invoke(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, reqbody interface{}, rspbody interface{}, opt ...client.Option) error {
			r, _ := rspbody.(*Response)
			r.Result = mongo.IndexView{}
			return nil
		})
	m3 := mockCli.EXPECT().Invoke(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, reqbody interface{}, rspbody interface{}, opt ...client.Option) error {
			r, _ := rspbody.(*Response)
			r.Result = 123
			return nil
		})
	gomock.InOrder(m1, m2, m3)

	type fields struct {
		ServiceName string
		Client      client.Client
		opts        []client.Option
	}
	type args struct {
		database   string
		collection string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    mongo.IndexView
		wantErr bool
	}{
		{"err", fields{Client: mockCli}, args{}, mongo.IndexView{}, true},
		{"suc", fields{Client: mockCli}, args{}, mongo.IndexView{}, false},
		{"not match kind", fields{Client: mockCli}, args{}, mongo.IndexView{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &mongodbCli{
				ServiceName: tt.fields.ServiceName,
				Client:      tt.fields.Client,
				opts:        tt.fields.opts,
			}
			got, err := c.Indexes(trpc.BackgroundContext(), tt.args.database, tt.args.collection)
			if (err != nil) != tt.wantErr {
				t.Errorf("Indexes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Indexes() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_mongodbCli_StartSession(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCli := mockclient.NewMockClient(ctrl)
	m1 := mockCli.EXPECT().Invoke(gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("err"))
	m3 := mockCli.EXPECT().Invoke(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, reqbody interface{}, rspbody interface{}, opt ...client.Option) error {
			r, _ := rspbody.(*Response)
			r.Result = 123
			return nil
		})
	gomock.InOrder(m1, m3)

	type fields struct {
		ServiceName string
		Client      client.Client
		opts        []client.Option
	}
	tests := []struct {
		name    string
		fields  fields
		want    mongo.Session
		wantErr bool
	}{
		{"err", fields{Client: mockCli}, nil, true},
		{"not match kind", fields{Client: mockCli}, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &mongodbCli{
				ServiceName: tt.fields.ServiceName,
				Client:      tt.fields.Client,
				opts:        tt.fields.opts,
			}
			got, err := c.StartSession(trpc.BackgroundContext())
			if (err != nil) != tt.wantErr {
				t.Errorf("StartSession() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("StartSession() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewInsertOneModel(t *testing.T) {
	mc := gomonkey.ApplyFunc(mongo.NewInsertOneModel, func() *mongo.InsertOneModel { return &mongo.InsertOneModel{} })
	defer mc.Reset()

	tests := []struct {
		name string
		want *mongo.InsertOneModel
	}{
		{"suc", &mongo.InsertOneModel{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewInsertOneModel(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewInsertOneModel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewReplaceOneModel(t *testing.T) {
	mc := gomonkey.ApplyFunc(mongo.NewReplaceOneModel, func() *mongo.ReplaceOneModel { return &mongo.ReplaceOneModel{} })
	defer mc.Reset()

	tests := []struct {
		name string
		want *mongo.ReplaceOneModel
	}{
		{"suc", &mongo.ReplaceOneModel{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewReplaceOneModel(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewReplaceOneModel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewSessionContext(t *testing.T) {
	type args struct {
		ctx  context.Context
		sess mongo.Session
	}
	tests := []struct {
		name string
		args args
		want mongo.SessionContext
	}{
		{"suc", args{ctx: trpc.BackgroundContext(), sess: nil}, mongo.NewSessionContext(trpc.BackgroundContext(), nil)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewSessionContext(tt.args.ctx, tt.args.sess); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewSessionContext() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewUpdateManyModel(t *testing.T) {
	mc := gomonkey.ApplyFunc(mongo.NewUpdateManyModel, func() *mongo.UpdateManyModel { return &mongo.UpdateManyModel{} })
	defer mc.Reset()

	tests := []struct {
		name string
		want *mongo.UpdateManyModel
	}{
		{"suc", &mongo.UpdateManyModel{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewUpdateManyModel(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewUpdateManyModel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewUpdateOneModel(t *testing.T) {
	mc := gomonkey.ApplyFunc(mongo.NewUpdateOneModel, func() *mongo.UpdateOneModel { return &mongo.UpdateOneModel{} })
	defer mc.Reset()

	tests := []struct {
		name string
		want *mongo.UpdateOneModel
	}{
		{"suc", &mongo.UpdateOneModel{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewUpdateOneModel(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewUpdateOneModel() = %v, want %v", got, tt.want)
			}
		})
	}
}
