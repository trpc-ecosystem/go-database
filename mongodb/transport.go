package mongodb

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/naming/selector"
	"trpc.group/trpc-go/trpc-go/transport"
	dsn "trpc.group/trpc-go/trpc-selector-dsn"
)

func init() {
	selector.Register("mongodb", dsn.NewDsnSelector(true))
	selector.Register("mongodb+polaris", dsn.NewResolvableSelectorWithOpts("polaris",
		dsn.WithEnableParseAddr(true), dsn.WithExtractor(&hostExtractor{})))
	transport.RegisterClientTransport("mongodb", DefaultClientTransport)
}

// ClientTransport is a client-side mongodb transport.
type ClientTransport struct {
	mongoDB           map[string]*mongo.Client
	optionInterceptor func(dsn string, opts *options.ClientOptions)
	mongoDBLock       sync.RWMutex
	MaxOpenConns      uint64
	MinOpenConns      uint64
	MaxConnIdleTime   time.Duration
	ReadPreference    *readpref.ReadPref
	ServiceNameURIs   map[string][]string
}

// DefaultClientTransport is a default client mongodb transport.
var DefaultClientTransport = NewMongoTransport()

// NewClientTransport creates a mongodb transport.
// Deprecated,use NewMongoTransport instead.
func NewClientTransport(opt ...transport.ClientTransportOption) transport.ClientTransport {
	return NewMongoTransport()
}

// NewMongoTransport creates a mongodb transport.
func NewMongoTransport(opt ...ClientTransportOption) transport.ClientTransport {
	ct := &ClientTransport{
		optionInterceptor: func(dsn string, opts *options.ClientOptions) {},
		mongoDB:           make(map[string]*mongo.Client),
		MaxOpenConns:      100, // The maximum number of connections in the connection pool.
		MinOpenConns:      5,
		MaxConnIdleTime:   5 * time.Minute, // Connection pool idle connection time.
		ReadPreference:    readpref.Primary(),
		ServiceNameURIs:   make(map[string][]string),
	}
	for _, o := range opt {
		o(ct)
	}
	return ct
}

// RoundTrip sends and receives mongodb packets,
// returns the mongodb response and puts it in ctx, there is no need to return rspbuf here.
func (ct *ClientTransport) RoundTrip(ctx context.Context, _ []byte,
	callOpts ...transport.RoundTripOption) (rspBytes []byte,
	err error) {

	msg := codec.Message(ctx)

	req, ok := msg.ClientReqHead().(*Request)
	if !ok {
		return nil, errs.NewFrameError(errs.RetClientEncodeFail,
			"mongodb client transport: ReqHead should be type of *mongodb.Request")
	}
	rsp, ok := msg.ClientRspHead().(*Response)
	if !ok {
		return nil, errs.NewFrameError(errs.RetClientEncodeFail,
			"mongodb client transport: RspHead should be type of *mongodb.Response")
	}

	opts := &transport.RoundTripOptions{}
	for _, o := range callOpts {
		o(opts)
	}

	if req.Command == Disconnect {
		return nil, ct.disconnect(ctx)
	}
	// Determine whether to use the mgo instance specified by the client.
	var mgo *mongo.Client
	if rsp.txClient != nil {
		mgo = rsp.txClient
	} else {
		mgo, err = ct.GetMgoClient(ctx, opts.Address)
		if err != nil {
			return nil, errs.NewFrameError(errs.RetClientNetErr,
				fmt.Sprintf("get mongo client failed: %s", err.Error()))
		}
	}

	var result interface{}
	if req.DriverProxy {
		result, err = handleDriverReq(ctx, mgo, req)
	} else {
		result, err = handleReq(ctx, mgo, req)
	}
	rsp.Result = result
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil, errs.Wrap(err, RetDuplicateKeyErr, err.Error())
		} else if mongo.IsTimeout(err) {
			return nil, errs.Wrap(err, errs.RetClientTimeout, err.Error())
		} else if mongo.IsNetworkError(err) {
			return nil, errs.Wrap(err, errs.RetClientNetErr, err.Error())
		} else {
			return nil, errs.Wrap(err, errs.RetUnknown, err.Error())
		}
	}
	return nil, nil
}

// handleDriverReq handles transparent transmission.
func handleDriverReq(ctx context.Context, mgoCli *mongo.Client, req *Request) (result interface{}, err error) {
	collection := mgoCli.Database(req.Database).Collection(req.Collection)
	switch req.Command {
	case InsertOne:
		return collection.InsertOne(ctx, req.CommArg, req.Opts.([]*options.InsertOneOptions)...)
	case InsertMany:
		return collection.InsertMany(ctx, req.CommArg.([]interface{}), req.Opts.([]*options.InsertManyOptions)...)
	case DeleteOne:
		return collection.DeleteOne(ctx, req.Filter, req.Opts.([]*options.DeleteOptions)...)
	case DeleteMany:
		return collection.DeleteMany(ctx, req.Filter, req.Opts.([]*options.DeleteOptions)...)
	case UpdateOne:
		return collection.UpdateOne(ctx, req.Filter, req.CommArg, req.Opts.([]*options.UpdateOptions)...)
	case UpdateMany:
		return collection.UpdateMany(ctx, req.Filter, req.CommArg, req.Opts.([]*options.UpdateOptions)...)
	case ReplaceOne:
		return collection.ReplaceOne(ctx, req.Filter, req.CommArg, req.Opts.([]*options.ReplaceOptions)...)
	case Aggregate:
		return collection.Aggregate(ctx, req.CommArg, req.Opts.([]*options.AggregateOptions)...)
	case CountDocuments:
		return collection.CountDocuments(ctx, req.Filter, req.Opts.([]*options.CountOptions)...)
	case EstimatedDocumentCount:
		return collection.EstimatedDocumentCount(ctx, req.Opts.([]*options.EstimatedDocumentCountOptions)...)
	case Distinct:
		return collection.Distinct(ctx, req.CommArg.(string), req.Filter, req.Opts.([]*options.DistinctOptions)...)
	case Find:
		return collection.Find(ctx, req.Filter, req.Opts.([]*options.FindOptions)...)
	case FindOne:
		return extractSingleResultErr(collection.FindOne(ctx, req.Filter, req.Opts.([]*options.FindOneOptions)...))
	case FindOneAndDelete:
		return extractSingleResultErr(collection.FindOneAndDelete(ctx, req.Filter,
			req.Opts.([]*options.FindOneAndDeleteOptions)...))
	case FindOneAndReplace:
		return extractSingleResultErr(collection.FindOneAndReplace(ctx, req.Filter, req.CommArg,
			req.Opts.([]*options.FindOneAndReplaceOptions)...))
	case FindOneAndUpdate:
		return extractSingleResultErr(collection.FindOneAndUpdate(ctx, req.Filter, req.CommArg,
			req.Opts.([]*options.FindOneAndUpdateOptions)...))
	case BulkWrite:
		return collection.BulkWrite(ctx, req.CommArg.([]mongo.WriteModel), req.Opts.([]*options.BulkWriteOptions)...)
	case Watch:
		return mgoCli.Watch(ctx, req.CommArg, req.Opts.([]*options.ChangeStreamOptions)...)
	case WatchDatabase:
		return mgoCli.Database(req.Database).Watch(ctx, req.CommArg, req.Opts.([]*options.ChangeStreamOptions)...)
	case WatchCollection:
		return mgoCli.Database(req.Database).Collection(req.Collection).Watch(ctx,
			req.CommArg, req.Opts.([]*options.ChangeStreamOptions)...)
	case Transaction:
		return execMongoTransaction(ctx, mgoCli, req.CommArg.(TxFunc), req.Filter.([]*options.TransactionOptions),
			req.Opts.([]*options.SessionOptions)...)
	case RunCommand:
		return extractSingleResultErr(mgoCli.Database(req.Database).RunCommand(ctx,
			req.CommArg, req.Opts.([]*options.RunCmdOptions)...))
	case IndexCreateOne:
		return collection.Indexes().CreateOne(ctx,
			req.CommArg.(mongo.IndexModel), req.Opts.([]*options.CreateIndexesOptions)...)
	case IndexCreateMany:
		return collection.Indexes().CreateMany(ctx,
			req.CommArg.([]mongo.IndexModel), req.Opts.([]*options.CreateIndexesOptions)...)
	case IndexDropOne:
		return collection.Indexes().DropOne(ctx,
			req.CommArg.(string), req.Opts.([]*options.DropIndexesOptions)...)
	case IndexDropAll:
		return collection.Indexes().DropAll(ctx,
			req.Opts.([]*options.DropIndexesOptions)...)
	case Indexes:
		return collection.Indexes(), nil
	case DatabaseCmd:
		return mgoCli.Database(req.Database), nil
	case CollectionCmd:
		return collection, nil
	case StartSession:
		return mgoCli.StartSession()
	default:
		return nil, errs.New(errs.RetClientDecodeFail, "error mongo command")
	}
}

// execMongoTransaction executes mongo transactions.
func execMongoTransaction(ctx context.Context, mgoCli *mongo.Client, sf TxFunc, tOpts []*options.TransactionOptions,
	opts ...*options.SessionOptions) (result interface{}, err error) {

	rspHead := trpc.Message(ctx).ClientRspHead()
	// Bind client before transaction execution.
	if rspHead == nil {
		return nil, errs.New(errs.RetClientDecodeFail, "rspHead can not be nil")
	}
	mCliRsp, ok := rspHead.(*Response)
	if !ok {
		return nil, errs.New(errs.RetClientDecodeFail, "conversion from rspHead to Respons failed")
	}
	mCliRsp.txClient = mgoCli

	// Obtain session.
	sess, err := mgoCli.StartSession(opts...)
	if err != nil {
		return nil, err
	}

	// Close session when finished.
	defer func() {
		sess.EndSession(ctx)
	}()

	transactionFn := func(sessCtx mongo.SessionContext) (interface{}, error) {
		return nil, sf(sessCtx)
	}

	return sess.WithTransaction(ctx, transactionFn, tOpts...)
}

// handleReq is an auxiliary function that handles the Req passed in by RoundTrip.
func handleReq(ctx context.Context, mgoCli *mongo.Client,
	req *Request) (result interface{}, err error) {
	collection := mgoCli.Database(req.Database).Collection(req.Collection)

	switch strings.ToLower(req.Command) {
	case Find:
		result, err = executeFind(ctx, collection, req.Arguments)
	case FindC:
		result, err = executeFindC(ctx, collection, req.Arguments)
	case DeleteOne:
		result, err = executeDeleteOne(ctx, collection, req.Arguments)
	case DeleteMany:
		result, err = executeDeleteMany(ctx, collection, req.Arguments)
	case FindOneAndDelete:
		result, err = executeFindOneAndDelete(ctx, collection, req.Arguments)
	case FindOneAndUpdate:
		result, err = executeFindOneAndUpdate(ctx, collection, req.Arguments)
	case FindOneAndUpdateS:
		result, err = executeFindOneAndUpdateS(ctx, collection, req.Arguments)
	case InsertOne:
		result, err = executeInsertOne(ctx, collection, req.Arguments)
	case InsertMany:
		result, err = executeInsertMany(ctx, collection, req.Arguments)
	case UpdateOne:
		result, err = executeUpdateOne(ctx, collection, req.Arguments)
	case UpdateMany:
		result, err = executeUpdateMany(ctx, collection, req.Arguments)
	case Count:
		result, err = executeCount(ctx, collection, req.Arguments)
	case Aggregate:
		result, err = executeAggregate(ctx, collection, req.Arguments)
	case AggregateC:
		result, err = executeAggregateC(ctx, collection, req.Arguments)
	case Distinct:
		result, err = executeDistinct(ctx, collection, req.Arguments)
	case BulkWrite:
		result, err = executeBulkWrite(ctx, collection, req.Arguments)
	case IndexCreateOne:
		result, err = executeIndexCreateOne(ctx, collection.Indexes(), req.Arguments)
	case IndexCreateMany:
		result, err = executeIndexCreateMany(ctx, collection.Indexes(), req.Arguments)
	case IndexDropOne:
		result, err = executeIndexDropOne(ctx, collection.Indexes(), req.Arguments)
	case IndexDropAll:
		result, err = executeIndexDropAll(ctx, collection.Indexes(), req.Arguments)
	case Indexes:
		result = collection.Indexes()
	case DatabaseCmd:
		result = mgoCli.Database(req.Database)
	case CollectionCmd:
		result = collection
	case StartSession:
		result, err = mgoCli.StartSession()
	default:
		err = fmt.Errorf("error mongo command")
	}
	return result, err
}

// GetMgoClient obtains mongodb client, cache dsn=>client,
// save some initialization steps such as reparsing parameters, generating topology server, etc.
func (ct *ClientTransport) GetMgoClient(ctx context.Context, dsn string) (*mongo.Client, error) {
	ct.mongoDBLock.RLock()
	mgo, ok := ct.mongoDB[dsn]
	ct.mongoDBLock.RUnlock()

	if ok {
		return mgo, nil
	}
	ct.mongoDBLock.Lock()
	defer ct.mongoDBLock.Unlock()

	mgo, ok = ct.mongoDB[dsn]
	if ok {
		return mgo, nil
	}
	clientOptions := ct.getClientOptions(dsn)

	// Based on the uri parameter, if it is not set, use the default value.
	if clientOptions.MaxPoolSize == nil {
		clientOptions.SetMaxPoolSize(ct.MaxOpenConns)
	}
	if clientOptions.MinPoolSize == nil {
		clientOptions.SetMinPoolSize(ct.MinOpenConns)
	}
	if clientOptions.MaxConnIdleTime == nil {
		clientOptions.SetMaxConnIdleTime(ct.MaxConnIdleTime)
	}
	if clientOptions.ReadPreference == nil {
		clientOptions.SetReadPreference(ct.ReadPreference)
	}

	if ct.optionInterceptor != nil {
		ct.optionInterceptor(dsn, clientOptions)
	}

	// The mongo-driver manages the connection itself, once Connect is initialized and used multiple times.
	mgo, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, err
	}

	err = mgo.Ping(ctx, ct.ReadPreference)
	if err != nil {
		_ = mgo.Disconnect(ctx)
		return nil, fmt.Errorf("ping mongo failed: %w", err)
	}

	ct.mongoDB[dsn] = mgo
	serviceName := codec.Message(ctx).CalleeServiceName()
	ct.ServiceNameURIs[serviceName] = append(ct.ServiceNameURIs[serviceName], dsn)
	return mgo, nil
}

func (ct *ClientTransport) getClientOptions(dsn string) *options.ClientOptions {
	uri := "mongodb://" + dsn
	clientOptions := options.Client().ApplyURI(uri)
	if clientOptions.MaxPoolSize == nil {
		clientOptions.SetMaxPoolSize(ct.MaxOpenConns)
	}
	if clientOptions.MinPoolSize == nil {
		clientOptions.SetMinPoolSize(ct.MinOpenConns)
	}
	if clientOptions.MaxConnIdleTime == nil {
		clientOptions.SetMaxConnIdleTime(ct.MaxConnIdleTime)
	}
	return clientOptions
}

func (ct *ClientTransport) disconnect(ctx context.Context) error {
	serviceName := codec.Message(ctx).CalleeServiceName()
	ct.mongoDBLock.RLock()
	uris, ok := ct.ServiceNameURIs[serviceName]
	ct.mongoDBLock.RUnlock()
	if !ok {
		return nil
	}

	ct.mongoDBLock.Lock()
	defer ct.mongoDBLock.Unlock()
	var funcs []func() error
	for _, uri := range uris {
		mgoCli, ok := ct.mongoDB[uri]
		if !ok {
			continue
		}
		delete(ct.mongoDB, uri)
		funcs = append(funcs, func() error {
			return mgoCli.Disconnect(ctx)
		})
	}

	delete(ct.ServiceNameURIs, serviceName)
	return trpc.GoAndWait(funcs...)
}

type hostExtractor struct {
}

// Extract extractHost is used to remove the "://" of uri and the part before it.
func (e *hostExtractor) Extract(uri string) (begin int, length int, err error) {
	// mongodb+polaris://user:pswd@xxx.mongodb.com
	offset := 0

	if idx := strings.Index(uri, "@"); idx != -1 {
		uri = uri[idx+1:]
		offset += idx + 1
	}

	begin = offset
	length = len(uri)
	if idx := strings.IndexAny(uri, "/?@"); idx != -1 {
		if uri[idx] == '@' {
			return 0, 0, errs.NewFrameError(errs.RetClientRouteErr, "unescaped @ sign in user info")
		}
		if uri[idx] == '?' {
			return 0, 0, errs.NewFrameError(errs.RetClientRouteErr, "must have a / before the query ?")
		}
		length = idx
	}

	return
}

func extractSingleResultErr(ms *mongo.SingleResult) (*mongo.SingleResult, error) {
	if ms != nil && ms.Err() != nil {
		return nil, ms.Err()
	}

	return ms, nil
}
