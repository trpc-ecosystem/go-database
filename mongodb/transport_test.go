package mongodb

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
)

func TestUnitExtractHost(t *testing.T) {
	assert := assert.New(t)

	cases := []struct {
		uri  string // The uri here has removed "://" and the part before it.
		host string
		err  string
	}{
		{
			uri:  "localhost",
			host: "localhost",
		},
		{
			uri:  "admin:123456@localhost/",
			host: "localhost",
		},
		{
			uri:  "admin:123456@localhost",
			host: "localhost",
		},
		{
			uri:  "example1.com:27017,example2.com:27017",
			host: "example1.com:27017,example2.com:27017",
		},
		{
			uri:  "host1,host2,host3/?slaveOk=true",
			host: "host1,host2,host3",
		},
		{
			uri: "admin:@123456@localhost/",
			err: errs.NewFrameError(errs.RetClientRouteErr, "unescaped @ sign in user info").Error(),
		},
		{
			uri: "admin:123456@localhost?",
			err: errs.NewFrameError(errs.RetClientRouteErr, "must have a / before the query ?").Error(),
		},
	}

	for _, c := range cases {
		pos, length, err := new(hostExtractor).Extract(c.uri)
		if len(c.err) != 0 {
			assert.EqualErrorf(err, c.err, "case: %+v ", c)
		} else {
			assert.Equalf(c.host, c.uri[pos:pos+length], "case: %+v", c)
		}
	}
}

func TestUnitClientTransport_GetMgoClient(t *testing.T) {
	Convey("TestUnitClientTransport_GetMgoClient", t, func() {
		pm := &event.PoolMonitor{}
		i := func(dsn string, opts *options.ClientOptions) {
			opts.SetPoolMonitor(pm)
		}
		cli := &ClientTransport{
			optionInterceptor: i,
			mongoDB:           make(map[string]*mongo.Client),
			MaxOpenConns:      100, // The maximum number of connections in the connection pool.
			MinOpenConns:      5,
			MaxConnIdleTime:   5 * time.Minute, // Connection pool idle connection time.
			ServiceNameURIs:   make(map[string][]string),
		}
		Convey("err", func() {
			defer gomonkey.ApplyFunc(mongo.NewClient, func(opts ...*options.ClientOptions) (*mongo.Client, error) {
				return nil, fmt.Errorf("err")
			}).Reset()
			ctx := context.TODO()
			mCli, err := cli.GetMgoClient(ctx, "addr1")
			So(mCli, ShouldBeNil)
			So(err, ShouldNotBeNil)
		})
		Convey("succ", func() {
			defer gomonkey.ApplyFunc(mongo.NewClient, func(opts ...*options.ClientOptions) (*mongo.Client, error) {
				return new(mongo.Client), nil
			}).Reset()
			defer gomonkey.ApplyMethod(reflect.TypeOf(new(mongo.Client)), "Ping",
				func(client *mongo.Client, ctx context.Context, rp *readpref.ReadPref) error {
					return nil
				},
			).Reset()
			ctx := context.TODO()
			mCli1, err := cli.GetMgoClient(ctx, "addr1")
			So(mCli1, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(len(cli.mongoDB), ShouldEqual, 1)
			mCli2, err := cli.GetMgoClient(ctx, "addr2")
			So(mCli2, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(len(cli.mongoDB), ShouldEqual, 2)
			mCli3, err := cli.GetMgoClient(ctx, "addr1")
			So(mCli3, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(len(cli.mongoDB), ShouldEqual, 2)
			So(mCli1, ShouldEqual, mCli3)
			So(mCli1, ShouldNotEqual, mCli2)
		})
	})
}

// TestUnitRoundTrip tests the RoundTrip function.
func TestUnitRoundTrip(t *testing.T) {
	defer gomonkey.ApplyMethod(reflect.TypeOf(new(ClientTransport)), "GetMgoClient",
		func(ct *ClientTransport, ctx context.Context, dsn string) (*mongo.Client, error) {
			return &mongo.Client{}, nil
		},
	).Reset()

	clientTransport := DefaultClientTransport
	ctx, msg := codec.WithNewMessage(trpc.BackgroundContext())
	_, err := clientTransport.RoundTrip(ctx, nil)
	assert.NotNil(t, err)

	msg.WithClientReqHead(&Request{})
	_, err = clientTransport.RoundTrip(ctx, nil)
	assert.NotNil(t, err)

	msg.WithClientRspHead(&Response{})
	_, err = clientTransport.RoundTrip(ctx, nil)
	assert.NotNil(t, err)

	msg.WithClientReqHead(&Request{
		Command: Find,
	})
	defer gomonkey.ApplyMethod(reflect.TypeOf(new(mongo.Collection)), "Find",
		func(coll *mongo.Collection, ctx context.Context, filter interface{},
			opts ...*options.FindOptions) (*mongo.Cursor, error) {
			return nil, errs.New(-1, "find cover")
		},
	).Reset()
	_, err = clientTransport.RoundTrip(ctx, nil)
	assert.NotNil(t, err)

	msg.WithClientReqHead(&Request{
		Command: Find,
	})
	defer gomonkey.ApplyMethod(reflect.TypeOf(new(mongo.Collection)), "Find",
		func(coll *mongo.Collection, ctx context.Context, filter interface{},
			opts ...*options.FindOptions) (*mongo.Cursor, error) {
			return nil, mongo.WriteException{WriteErrors: mongo.WriteErrors{mongo.WriteError{Code: 11000}}}
		},
	).Reset()
	_, err = clientTransport.RoundTrip(ctx, nil)
	assert.NotNil(t, err)

	msg.WithClientReqHead(&Request{
		Command: Find,
	})
	defer gomonkey.ApplyMethod(reflect.TypeOf(new(mongo.Collection)), "Find",
		func(coll *mongo.Collection, ctx context.Context, filter interface{},
			opts ...*options.FindOptions) (*mongo.Cursor, error) {
			return nil, context.DeadlineExceeded
		},
	).Reset()
	_, err = clientTransport.RoundTrip(ctx, nil)
	assert.NotNil(t, err)

	msg.WithClientReqHead(&Request{
		Command: Find,
	})
	defer gomonkey.ApplyMethod(reflect.TypeOf(new(mongo.Collection)), "Find",
		func(coll *mongo.Collection, ctx context.Context, filter interface{},
			opts ...*options.FindOptions) (*mongo.Cursor, error) {
			return nil, mongo.CommandError{Labels: []string{"NetworkError"}}
		},
	).Reset()
	_, err = clientTransport.RoundTrip(ctx, nil)
	assert.NotNil(t, err)

	msg.WithClientReqHead(&Request{
		Command: FindC,
	})
	defer gomonkey.ApplyMethod(reflect.TypeOf(new(mongo.Collection)), "Find",
		func(coll *mongo.Collection, ctx context.Context, filter interface{},
			opts ...*options.FindOptions) (*mongo.Cursor, error) {
			return &mongo.Cursor{}, nil
		},
	).Reset()
	_, err = clientTransport.RoundTrip(ctx, nil)
	assert.Nil(t, err)

	msg.WithClientReqHead(&Request{
		Command: DeleteOne,
	})
	defer gomonkey.ApplyMethod(reflect.TypeOf(new(mongo.Collection)), "DeleteOne",
		func(*mongo.Collection, context.Context, interface{}, ...*options.DeleteOptions) (*mongo.DeleteResult, error) {
			return &mongo.DeleteResult{}, nil
		},
	).Reset()
	_, err = clientTransport.RoundTrip(ctx, nil)
	assert.Nil(t, err)
}

func TestClientTransport_disconnect(t *testing.T) {
	defer gomonkey.ApplyMethod(reflect.TypeOf(new(mongo.Client)), "Disconnect",
		func(coll *mongo.Client, ctx context.Context) error {
			return nil
		},
	).Reset()
	type fields struct {
		mongoDB         map[string]*mongo.Client
		MaxOpenConns    uint64
		MinOpenConns    uint64
		MaxConnIdleTime time.Duration
		ReadPreference  *readpref.ReadPref
		ServiceNameURIs map[string][]string
	}

	ctx := trpc.BackgroundContext()
	msg := trpc.Message(ctx)
	msg.WithCalleeServiceName("uri")

	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "succ",
			fields: fields{
				ServiceNameURIs: map[string][]string{
					"uri": {"uri1"},
				},
				mongoDB: map[string]*mongo.Client{
					"uri1": nil,
				},
			},
			args: args{
				ctx: ctx,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ct := &ClientTransport{
				mongoDB:         tt.fields.mongoDB,
				MaxOpenConns:    tt.fields.MaxOpenConns,
				MinOpenConns:    tt.fields.MinOpenConns,
				MaxConnIdleTime: tt.fields.MaxConnIdleTime,
				ReadPreference:  tt.fields.ReadPreference,
				ServiceNameURIs: tt.fields.ServiceNameURIs,
			}
			if err := ct.disconnect(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("ClientTransport.disconnect() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_handleReq(t *testing.T) {
	c := &mongo.Client{}
	db := &mongo.Database{}
	col := &mongo.Collection{}
	m1 := gomonkey.ApplyMethod(c, "Database",
		func(c *mongo.Client, name string, opts ...*options.DatabaseOptions) *mongo.Database {
			return db
		})
	defer m1.Reset()
	m2 := gomonkey.ApplyMethod(db, "Collection",
		func(c *mongo.Database, name string, opts ...*options.CollectionOptions) *mongo.Collection {
			return col
		})
	defer m2.Reset()
	ctx := trpc.BackgroundContext()
	t.Run("indexes", func(t *testing.T) {
		mm := gomonkey.ApplyMethod(col, "Indexes", func(c *mongo.Collection) mongo.IndexView { return mongo.IndexView{} })
		defer mm.Reset()
		gotResult, err := handleReq(ctx, c, &Request{Command: Indexes})
		if err != nil {
			t.Errorf("handleReq() error = %v, wantErr %v", err, nil)
			return
		}
		if !reflect.DeepEqual(gotResult, mongo.IndexView{}) {
			t.Errorf("handleReq() gotResult = %v, want %v", gotResult, mongo.IndexView{})
		}
	})
	t.Run("DatabaseCmd", func(t *testing.T) {
		gotResult, err := handleReq(ctx, c, &Request{Command: DatabaseCmd})
		if err != nil {
			t.Errorf("handleReq() error = %v, wantErr %v", err, nil)
			return
		}
		if !reflect.DeepEqual(gotResult, db) {
			t.Errorf("handleReq() gotResult = %v, want %v", gotResult, db)
		}
	})
	t.Run("CollectionCmd", func(t *testing.T) {
		gotResult, err := handleReq(ctx, c, &Request{Command: CollectionCmd})
		if err != nil {
			t.Errorf("handleReq() error = %v, wantErr %v", err, nil)
			return
		}
		if !reflect.DeepEqual(gotResult, col) {
			t.Errorf("handleReq() gotResult = %v, want %v", gotResult, col)
		}
	})
	t.Run("StartSession", func(t *testing.T) {
		mm := gomonkey.ApplyMethod(c, "StartSession",
			func(c *mongo.Client, opts ...*options.SessionOptions) (mongo.Session, error) { return nil, nil })
		defer mm.Reset()
		gotResult, err := handleReq(ctx, c, &Request{Command: StartSession})
		if err != nil {
			t.Errorf("handleReq() error = %v, wantErr %v", err, nil)
			return
		}
		if !reflect.DeepEqual(gotResult, nil) {
			t.Errorf("handleReq() gotResult = %v, want %v", gotResult, nil)
		}
	})
}

func Test_handleDriverReq(t *testing.T) {
	c := &mongo.Client{}
	db := &mongo.Database{}
	col := &mongo.Collection{}
	m1 := gomonkey.ApplyMethod(c, "Database",
		func(c *mongo.Client, name string, opts ...*options.DatabaseOptions) *mongo.Database {
			return db
		})
	defer m1.Reset()
	m2 := gomonkey.ApplyMethod(db, "Collection",
		func(c *mongo.Database, name string, opts ...*options.CollectionOptions) *mongo.Collection {
			return col
		})
	defer m2.Reset()
	ctx := trpc.BackgroundContext()

	t.Run("indexes", func(t *testing.T) {
		mm := gomonkey.ApplyMethod(col, "Indexes", func(c *mongo.Collection) mongo.IndexView { return mongo.IndexView{} })
		defer mm.Reset()
		gotResult, err := handleDriverReq(ctx, c, &Request{Command: Indexes})
		if err != nil {
			t.Errorf("handleReq() error = %v, wantErr %v", err, nil)
			return
		}
		if !reflect.DeepEqual(gotResult, mongo.IndexView{}) {
			t.Errorf("handleReq() gotResult = %v, want %v", gotResult, mongo.IndexView{})
		}
	})
	t.Run("DatabaseCmd", func(t *testing.T) {
		gotResult, err := handleDriverReq(ctx, c, &Request{Command: DatabaseCmd})
		if err != nil {
			t.Errorf("handleReq() error = %v, wantErr %v", err, nil)
			return
		}
		if !reflect.DeepEqual(gotResult, db) {
			t.Errorf("handleReq() gotResult = %v, want %v", gotResult, db)
		}
	})
	t.Run("CollectionCmd", func(t *testing.T) {
		gotResult, err := handleDriverReq(ctx, c, &Request{Command: CollectionCmd})
		if err != nil {
			t.Errorf("handleReq() error = %v, wantErr %v", err, nil)
			return
		}
		if !reflect.DeepEqual(gotResult, col) {
			t.Errorf("handleReq() gotResult = %v, want %v", gotResult, col)
		}
	})
	t.Run("StartSession", func(t *testing.T) {
		mm := gomonkey.ApplyMethod(c, "StartSession",
			func(c *mongo.Client, opts ...*options.SessionOptions) (mongo.Session, error) { return nil, nil })
		defer mm.Reset()
		gotResult, err := handleDriverReq(ctx, c, &Request{Command: StartSession})
		if err != nil {
			t.Errorf("handleReq() error = %v, wantErr %v", err, nil)
			return
		}
		if !reflect.DeepEqual(gotResult, nil) {
			t.Errorf("handleReq() gotResult = %v, want %v", gotResult, nil)
		}
	})
}
