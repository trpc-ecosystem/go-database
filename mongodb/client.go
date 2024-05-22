// Package mongodb encapsulates standard library mongodb.
package mongodb

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/log"
)

// TxFunc mongo is a transaction logic function,
// if an error is returned, it will be rolled back.
type TxFunc func(sc mongo.SessionContext) error

//go:generate mockgen -source=client.go -destination=./mockmongodb/client_mock.go -package=mockmongodb

// Client is mongodb request interface.
type Client interface {
	BulkWrite(ctx context.Context, database string, coll string, models []mongo.WriteModel,
		opts ...*options.BulkWriteOptions) (*mongo.BulkWriteResult, error)
	InsertOne(ctx context.Context, database string, coll string, document interface{},
		opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error)
	InsertMany(ctx context.Context, database string, coll string, documents []interface{},
		opts ...*options.InsertManyOptions) (*mongo.InsertManyResult, error)
	DeleteOne(ctx context.Context, database string, coll string, filter interface{},
		opts ...*options.DeleteOptions) (*mongo.DeleteResult, error)
	DeleteMany(ctx context.Context, database string, coll string, filter interface{},
		opts ...*options.DeleteOptions) (*mongo.DeleteResult, error)
	UpdateOne(ctx context.Context, database string, coll string, filter interface{}, update interface{},
		opts ...*options.UpdateOptions) (*mongo.UpdateResult, error)
	UpdateMany(ctx context.Context, database string, coll string, filter interface{}, update interface{},
		opts ...*options.UpdateOptions) (*mongo.UpdateResult, error)
	ReplaceOne(ctx context.Context, database string, coll string, filter interface{},
		replacement interface{}, opts ...*options.ReplaceOptions) (*mongo.UpdateResult, error)
	Aggregate(ctx context.Context, database string, coll string, pipeline interface{},
		opts ...*options.AggregateOptions) (*mongo.Cursor, error)
	CountDocuments(ctx context.Context, database string, coll string, filter interface{},
		opts ...*options.CountOptions) (int64, error)
	EstimatedDocumentCount(ctx context.Context, database string, coll string,
		opts ...*options.EstimatedDocumentCountOptions) (int64, error)
	Distinct(ctx context.Context, database string, coll string, fieldName string, filter interface{},
		opts ...*options.DistinctOptions) ([]interface{}, error)
	Find(ctx context.Context, database string, coll string, filter interface{},
		opts ...*options.FindOptions) (*mongo.Cursor, error)
	FindOne(ctx context.Context, database string, coll string, filter interface{},
		opts ...*options.FindOneOptions) *mongo.SingleResult
	FindOneAndDelete(ctx context.Context, database string, coll string, filter interface{},
		opts ...*options.FindOneAndDeleteOptions) *mongo.SingleResult
	FindOneAndReplace(ctx context.Context, database string, coll string, filter interface{},
		replacement interface{}, opts ...*options.FindOneAndReplaceOptions) *mongo.SingleResult
	FindOneAndUpdate(ctx context.Context, database string, coll string, filter interface{},
		update interface{}, opts ...*options.FindOneAndUpdateOptions) *mongo.SingleResult
	Watch(ctx context.Context, pipeline interface{},
		opts ...*options.ChangeStreamOptions) (*mongo.ChangeStream, error)
	WatchDatabase(ctx context.Context, database string, pipeline interface{},
		opts ...*options.ChangeStreamOptions) (*mongo.ChangeStream, error)
	WatchCollection(ctx context.Context, database string, collection string, pipeline interface{},
		opts ...*options.ChangeStreamOptions) (*mongo.ChangeStream, error)
	Transaction(ctx context.Context, sf TxFunc, tOpts []*options.TransactionOptions,
		opts ...*options.SessionOptions) error
	Disconnect(ctx context.Context) error
	RunCommand(ctx context.Context, database string, runCommand interface{},
		opts ...*options.RunCmdOptions) *mongo.SingleResult
	Indexes(ctx context.Context, database string, collection string) (mongo.IndexView, error)
	Database(ctx context.Context, database string) (*mongo.Database, error)
	Collection(ctx context.Context, database string, collection string) (*mongo.Collection, error)
	StartSession(ctx context.Context) (mongo.Session, error)
	//Deprecated
	Do(ctx context.Context, cmd string, db string, coll string, args map[string]interface{}) (interface{}, error)
}

// IndexViewer is the interface definition of the index.
// Refer to the naming of the community open source library,
// define the index interface separately, and divide the interface according to the function.
type IndexViewer interface {
	// CreateMany creates the interface definition of the index.
	CreateMany(ctx context.Context, database string, coll string,
		models []mongo.IndexModel, opts ...*options.CreateIndexesOptions) ([]string, error)
	CreateOne(ctx context.Context, database string, coll string,
		model mongo.IndexModel, opts ...*options.CreateIndexesOptions) (string, error)
	DropOne(ctx context.Context, database string, coll string,
		name string, opts ...*options.DropIndexesOptions) (bson.Raw, error)
	DropAll(ctx context.Context, database string, coll string,
		opts ...*options.DropIndexesOptions) (bson.Raw, error)
}

// mongodbCli is a backend request structure.
type mongodbCli struct {
	ServiceName string
	Client      client.Client
	opts        []client.Option
}

// NewClientProxy creates a new mongo backend request proxy.
// The required parameter mongo service name: trpc.mongo.xxx.xxx.
func NewClientProxy(name string, opts ...client.Option) Client {
	c := &mongodbCli{
		ServiceName: name,
		Client:      client.DefaultClient,
	}

	c.opts = make([]client.Option, 0, len(opts)+2)
	c.opts = append(c.opts, opts...)
	c.opts = append(c.opts, client.WithProtocol("mongodb"), client.WithDisableServiceRouter())
	return c
}

// NewInsertOneModel creates a new InsertOneModel.
// InsertOneModel is used to insert a single document in a BulkWrite operation.
func NewInsertOneModel() *mongo.InsertOneModel {
	return mongo.NewInsertOneModel()
}

// NewUpdateManyModel creates a new UpdateManyModel.
// UpdateManyModel is used to update multiple documents in a BulkWrite operation.
func NewUpdateManyModel() *mongo.UpdateManyModel {
	return mongo.NewUpdateManyModel()
}

// NewUpdateOneModel creates a new UpdateOneModel.
// UpdateOneModel is used to update at most one document in a BulkWrite operation.
func NewUpdateOneModel() *mongo.UpdateOneModel {
	return mongo.NewUpdateOneModel()
}

// NewReplaceOneModel  creates a new ReplaceOneModel.
// ReplaceOneModel is used to replace at most one document in a BulkWrite operation.
func NewReplaceOneModel() *mongo.ReplaceOneModel {
	return mongo.NewReplaceOneModel()
}

// NewSessionContext creates a new SessionContext associated with the given Context and Session parameters.
func NewSessionContext(ctx context.Context, sess mongo.Session) mongo.SessionContext {
	return mongo.NewSessionContext(ctx, sess)
}

// mongodb cmd definition
var (
	Find              = "find"
	FindOne           = "findone"
	FindOneAndReplace = "findoneandreplace"
	FindC             = "findc" // Return mongo.Cursor type interface, use cursor.All/Decode to parse to structure.
	DeleteOne         = "deleteone"
	DeleteMany        = "deletemany"
	FindOneAndDelete  = "findoneanddelete"
	FindOneAndUpdate  = "findoneandupdate"
	FindOneAndUpdateS = "findoneandupdates" // Return mongo.SingleResult type interface,
	// use Decode to parse to structure.
	InsertOne  = "insertone"
	InsertMany = "insertmany"
	UpdateOne  = "updateone"
	UpdateMany = "updatemany"
	ReplaceOne = "replaceone"
	Count      = "count"
	Aggregate  = "aggregate"  // Polymerization
	AggregateC = "aggregatec" // Return mongo.Cursor type interface,
	// use cursor.All/Decode to parse to structure.
	Distinct               = "distinct"
	BulkWrite              = "bulkwrite"
	CountDocuments         = "countdocuments"
	EstimatedDocumentCount = "estimateddocumentcount"
	Watch                  = "watch"
	WatchDatabase          = "watchdatabase"
	WatchCollection        = "watchcollection"
	Transaction            = "transaction"
	Disconnect             = "disconnect"
	RunCommand             = "runcommand"      // Execute commands sequentially
	IndexCreateOne         = "indexcreateone"  // Create index
	IndexCreateMany        = "indexcreatemany" // Create indexes in batches
	IndexDropOne           = "indexdropone"    // Delete index
	IndexDropAll           = "indexdropall"    // Delete all indexes
	Indexes                = "indexes"         // Get the original index object
	DatabaseCmd            = "database"        // Get the original database
	CollectionCmd          = "collection"      // Get the original collection
	StartSession           = "startsession"    // Create a new Session and SessionContext
)

// Request mongodb request body
type Request struct {
	Command    string
	Database   string
	Collection string
	Arguments  map[string]interface{}

	DriverProxy bool        //driver transparent transmission
	Filter      interface{} //driver filter
	CommArg     interface{} //general parameters
	Opts        interface{} //option parameter
}

// Response mongodb response body
type Response struct {
	Result   interface{}
	txClient *mongo.Client //Use transparent mongo client in transaction execution.
}

// Do is a general execution interface,
// which executes different curd operations according to cmd.
func (c *mongodbCli) Do(ctx context.Context, cmd string, database string, collection string,
	args map[string]interface{}) (interface{}, error) {
	cmd = strings.ToLower(cmd)
	req := &Request{
		Command:    cmd,
		Database:   database,
		Collection: collection,
		Arguments:  args,
	}
	rsp := &Response{}

	ctx, msg := codec.WithCloneMessage(ctx)
	defer codec.PutBackMessage(msg)
	msg.WithClientRPCName(fmt.Sprintf("/%s/%s", c.ServiceName, cmd))
	msg.WithCalleeServiceName(c.ServiceName)
	msg.WithSerializationType(-1) // Not serialized.
	msg.WithCompressType(0)       // Not compressed.
	msg.WithClientReqHead(req)
	msg.WithClientRspHead(rsp)

	err := c.Client.Invoke(ctx, req, rsp, c.opts...)
	return rsp.Result, err
}

// invoke is a universal execution interface, execute different curd operations according to cmd.
func (c *mongodbCli) invoke(ctx context.Context, req *Request) (
	interface{}, error) {
	req.DriverProxy = true

	// If there is a transparently transmitted response when executing a transaction,
	// use the transparently transmitted instance directly.
	var rsp *Response
	rspHead := trpc.Message(ctx).ClientRspHead()
	if rspHead != nil {
		if rspIns, ok := rspHead.(*Response); ok {
			rsp = rspIns
		}
	}

	// If there is no specified mongo instance, create a new response.
	if rsp == nil {
		rsp = &Response{}
	}

	ctx, msg := codec.WithCloneMessage(ctx)
	defer codec.PutBackMessage(msg)
	msg.WithClientRPCName(fmt.Sprintf("/%s/driver.%s", c.ServiceName, req.Command))
	msg.WithCalleeServiceName(c.ServiceName)
	msg.WithSerializationType(-1) //Not serialized.
	msg.WithCompressType(0)       //Not compressed.
	msg.WithClientReqHead(req)
	msg.WithClientRspHead(rsp)

	err := c.Client.Invoke(ctx, req, rsp, c.opts...)
	return rsp.Result, err
}

// Transaction executes a Transaction.
func (c *mongodbCli) Transaction(ctx context.Context, sf TxFunc, tOpts []*options.TransactionOptions,
	opts ...*options.SessionOptions) error {
	request := &Request{
		Command: Transaction,
		CommArg: sf,
		Filter:  tOpts,
		Opts:    opts,
	}

	_, err := c.invoke(ctx, request)
	if err != nil {
		return err
	}
	return nil
}

// InsertOne executes an insert command to insert a single document into the collection.
func (c *mongodbCli) InsertOne(ctx context.Context, database string, coll string, document interface{},
	opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error) {
	request := &Request{
		Command:    InsertOne,
		Database:   database,
		Collection: coll,
		CommArg:    document,
		Opts:       opts,
	}
	rsp, err := c.invoke(ctx, request)
	if err != nil {
		return nil, err
	}
	return rsp.(*mongo.InsertOneResult), nil
}

// InsertMany executes an insert command to insert multiple documents into the collection. If write errors occur
// during the operation (e.g. duplicate key error), this method returns a BulkWriteException error.
func (c *mongodbCli) InsertMany(ctx context.Context, database string, coll string, documents []interface{},
	opts ...*options.InsertManyOptions) (*mongo.InsertManyResult, error) {
	request := &Request{
		Command:    InsertMany,
		Database:   database,
		Collection: coll,
		CommArg:    documents,
		Opts:       opts,
	}
	rsp, err := c.invoke(ctx, request)
	if rsp != nil {
		return rsp.(*mongo.InsertManyResult), err
	}
	return nil, err
}

// DeleteOne executes a delete command to delete at most one document from the collection.
func (c *mongodbCli) DeleteOne(ctx context.Context, database string, coll string, filter interface{},
	opts ...*options.DeleteOptions) (*mongo.DeleteResult, error) {
	request := &Request{
		Command:    DeleteOne,
		Database:   database,
		Collection: coll,
		Filter:     filter,
		Opts:       opts,
	}
	rsp, err := c.invoke(ctx, request)
	if err != nil {
		return nil, err
	}
	return rsp.(*mongo.DeleteResult), nil
}

// DeleteMany executes a delete command to delete documents from the collection.
func (c *mongodbCli) DeleteMany(ctx context.Context, database string, coll string, filter interface{},
	opts ...*options.DeleteOptions) (*mongo.DeleteResult, error) {
	request := &Request{
		Command:    DeleteMany,
		Database:   database,
		Collection: coll,
		Filter:     filter,
		Opts:       opts,
	}
	rsp, err := c.invoke(ctx, request)
	if err != nil {
		return nil, err
	}
	return rsp.(*mongo.DeleteResult), nil
}

// UpdateOne executes an update command to update at most one document in the collection.
func (c *mongodbCli) UpdateOne(ctx context.Context, database string, coll string, filter interface{},
	update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
	request := &Request{
		Command:    UpdateOne,
		Database:   database,
		Collection: coll,
		Filter:     filter,
		CommArg:    update,
		Opts:       opts,
	}
	rsp, err := c.invoke(ctx, request)
	if err != nil {
		return nil, err
	}
	return rsp.(*mongo.UpdateResult), nil
}

// UpdateMany executes an update command to update documents in the collection.
func (c *mongodbCli) UpdateMany(ctx context.Context, database string, coll string, filter interface{},
	update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
	request := &Request{
		Command:    UpdateMany,
		Database:   database,
		Collection: coll,
		Filter:     filter,
		CommArg:    update,
		Opts:       opts,
	}
	rsp, err := c.invoke(ctx, request)
	if err != nil {
		return nil, err
	}
	return rsp.(*mongo.UpdateResult), nil
}

// ReplaceOne executes an update command to replace at most one document in the collection.
func (c *mongodbCli) ReplaceOne(ctx context.Context, database string, coll string, filter interface{},
	replacement interface{}, opts ...*options.ReplaceOptions) (*mongo.UpdateResult, error) {
	request := &Request{
		Command:    ReplaceOne,
		Database:   database,
		Collection: coll,
		Filter:     filter,
		CommArg:    replacement,
		Opts:       opts,
	}
	rsp, err := c.invoke(ctx, request)
	if err != nil {
		return nil, err
	}
	return rsp.(*mongo.UpdateResult), nil
}

// Aggregate executes an aggregate command against the collection and returns a cursor over the resulting documents.
func (c *mongodbCli) Aggregate(ctx context.Context, database string, coll string, pipeline interface{},
	opts ...*options.AggregateOptions) (*mongo.Cursor, error) {
	request := &Request{
		Command:    Aggregate,
		Database:   database,
		Collection: coll,
		CommArg:    pipeline,
		Opts:       opts,
	}
	rsp, err := c.invoke(ctx, request)
	if err != nil {
		return nil, err
	}
	return rsp.(*mongo.Cursor), nil
}

// CountDocuments returns the number of documents in the collection. For a fast count of the documents in the
// collection, see the EstimatedDocumentCount method.
func (c *mongodbCli) CountDocuments(ctx context.Context, database string, coll string, filter interface{},
	opts ...*options.CountOptions) (int64, error) {
	request := &Request{
		Command:    CountDocuments,
		Database:   database,
		Collection: coll,
		Filter:     filter,
		Opts:       opts,
	}
	rsp, err := c.invoke(ctx, request)
	if err != nil {
		return 0, err
	}
	return rsp.(int64), nil
}

// EstimatedDocumentCount executes a count command and returns an estimate of the number of documents in the collection
// using collection metadata.
func (c *mongodbCli) EstimatedDocumentCount(ctx context.Context, database string, coll string,
	opts ...*options.EstimatedDocumentCountOptions) (int64, error) {
	request := &Request{
		Command:    EstimatedDocumentCount,
		Database:   database,
		Collection: coll,
		Opts:       opts,
	}
	rsp, err := c.invoke(ctx, request)
	if err != nil {
		return 0, err
	}
	return rsp.(int64), nil
}

// Distinct executes a distinct command to find the unique values for a specified field in the collection.
func (c *mongodbCli) Distinct(ctx context.Context, database string, coll string, fieldName string, filter interface{},
	opts ...*options.DistinctOptions) ([]interface{}, error) {
	request := &Request{
		Command:    Distinct,
		Database:   database,
		Collection: coll,
		Filter:     filter,
		CommArg:    fieldName,
		Opts:       opts,
	}
	rsp, err := c.invoke(ctx, request)
	if err != nil {
		return nil, err
	}
	return rsp.([]interface{}), nil
}

// InsertOne executes an insert command to insert a single document into the collection.
func (c *mongodbCli) Find(ctx context.Context, database string, coll string, filter interface{},
	opts ...*options.FindOptions) (*mongo.Cursor, error) {
	request := &Request{
		Command:    Find,
		Database:   database,
		Collection: coll,
		Filter:     filter,
		Opts:       opts,
	}
	rsp, err := c.invoke(ctx, request)
	if err != nil {
		return nil, err
	}
	return rsp.(*mongo.Cursor), nil
}

// FindOne executes a find command and returns a SingleResult for one document in the collection.
func (c *mongodbCli) FindOne(ctx context.Context, database string, coll string, filter interface{},
	opts ...*options.FindOneOptions) *mongo.SingleResult {
	request := &Request{
		Command:    FindOne,
		Database:   database,
		Collection: coll,
		Filter:     filter,
		Opts:       opts,
	}
	rsp, err := c.invoke(ctx, request)
	if err != nil {
		log.Errorf("client invoke error: %v", err)
		return mongo.NewSingleResultFromDocument(bson.D{}, err, nil)
	}
	return rsp.(*mongo.SingleResult)
}

// FindOneAndDelete executes a findAndModify command to delete at most one document in the collection. and returns the
// document as it appeared before deletion.
func (c *mongodbCli) FindOneAndDelete(ctx context.Context, database string, coll string, filter interface{},
	opts ...*options.FindOneAndDeleteOptions) *mongo.SingleResult {
	request := &Request{
		Command:    FindOneAndDelete,
		Database:   database,
		Collection: coll,
		Filter:     filter,
		Opts:       opts,
	}
	rsp, err := c.invoke(ctx, request)
	if err != nil {
		log.Errorf("client invoke error: %v", err)
		return mongo.NewSingleResultFromDocument(bson.D{}, err, nil)
	}
	return rsp.(*mongo.SingleResult)
}

// FindOneAndReplace executes a findAndModify command to replace at most one document in the collection
// and returns the document as it appeared before replacement.
func (c *mongodbCli) FindOneAndReplace(ctx context.Context, database string, coll string, filter interface{},
	replacement interface{}, opts ...*options.FindOneAndReplaceOptions) *mongo.SingleResult {
	request := &Request{
		Command:    FindOneAndReplace,
		Database:   database,
		Collection: coll,
		Filter:     filter,
		CommArg:    replacement,
		Opts:       opts,
	}
	rsp, err := c.invoke(ctx, request)
	if err != nil {
		log.Errorf("client invoke error: %v", err)
		return mongo.NewSingleResultFromDocument(bson.D{}, err, nil)
	}
	return rsp.(*mongo.SingleResult)
}

// FindOneAndUpdate executes a findAndModify command to update at most one document in the collection and returns the
// document as it appeared before updating.
func (c *mongodbCli) FindOneAndUpdate(ctx context.Context, database string, coll string, filter interface{},
	update interface{}, opts ...*options.FindOneAndUpdateOptions) *mongo.SingleResult {
	request := &Request{
		Command:    FindOneAndUpdate,
		Database:   database,
		Collection: coll,
		Filter:     filter,
		CommArg:    update,
		Opts:       opts,
	}
	rsp, err := c.invoke(ctx, request)
	if err != nil {
		log.Errorf("client invoke error: %v", err)
		return mongo.NewSingleResultFromDocument(bson.D{}, err, nil)
	}
	return rsp.(*mongo.SingleResult)
}

// BulkWrite performs a bulk write operation (https://docs.mongodb.com/manual/core/bulk-write-operations/)
func (c *mongodbCli) BulkWrite(ctx context.Context, database string, coll string, models []mongo.WriteModel,
	opts ...*options.BulkWriteOptions) (*mongo.BulkWriteResult, error) {
	request := &Request{
		Command:    BulkWrite,
		Database:   database,
		Collection: coll,
		CommArg:    models,
		Opts:       opts,
	}
	rsp, err := c.invoke(ctx, request)
	if rsp != nil {
		return rsp.(*mongo.BulkWriteResult), err
	}
	return nil, err
}

// Watch returns a change stream for all changes on the deployment.
func (c *mongodbCli) Watch(ctx context.Context, pipeline interface{},
	opts ...*options.ChangeStreamOptions) (*mongo.ChangeStream, error) {

	request := &Request{
		Command: Watch,
		CommArg: pipeline,
		Opts:    opts,
	}
	rsp, err := c.invoke(ctx, request)
	if err != nil {
		return nil, err
	}
	return rsp.(*mongo.ChangeStream), nil
}

// WatchDatabase returns a change stream for all changes to the corresponding database.
func (c *mongodbCli) WatchDatabase(ctx context.Context, database string, pipeline interface{},
	opts ...*options.ChangeStreamOptions) (*mongo.ChangeStream, error) {
	request := &Request{
		Command:  WatchDatabase,
		Database: database,
		CommArg:  pipeline,
		Opts:     opts,
	}
	rsp, err := c.invoke(ctx, request)
	if err != nil {
		return nil, err
	}
	return rsp.(*mongo.ChangeStream), nil
}

// WatchCollection returns a change stream for all changes on the corresponding collection.
func (c *mongodbCli) WatchCollection(ctx context.Context, database string, collection string, pipeline interface{},
	opts ...*options.ChangeStreamOptions) (*mongo.ChangeStream, error) {
	request := &Request{
		Command:    WatchCollection,
		Database:   database,
		Collection: collection,
		CommArg:    pipeline,
		Opts:       opts,
	}
	rsp, err := c.invoke(ctx, request)
	if err != nil {
		return nil, err
	}
	return rsp.(*mongo.ChangeStream), nil
}

// Disconnect closes the mongo client under service name.
func (c *mongodbCli) Disconnect(ctx context.Context) error {
	request := &Request{
		Command: Disconnect,
	}
	_, err := c.invoke(ctx, request)
	return err
}

// RunCommand executes the given command against the database. This function does not obey the Database's read
// preference. To specify a read preference, the RunCmdOptions.ReadPreference option must be used.
// The runCommand parameter must be a document for the command to be executed. It cannot be nil.
// This must be an order-preserving type such as bson.D. Map types such as bson.M are not valid.
// The shardCollection command must be run against the admin database.
func (c *mongodbCli) RunCommand(ctx context.Context, database string, runCommand interface{},
	opts ...*options.RunCmdOptions) *mongo.SingleResult {
	request := &Request{
		Command:  RunCommand,
		Database: database,
		CommArg:  runCommand,
		Opts:     opts,
	}
	rsp, err := c.invoke(ctx, request)
	if err != nil {
		log.Errorf("client invoke error: %v", err)
		return mongo.NewSingleResultFromDocument(bson.D{}, err, nil)
	}
	return rsp.(*mongo.SingleResult)
}

// CreateMany executes a createIndexes command to create multiple indexes on the collection and returns
// the names of the new indexes.
func (c *mongodbCli) CreateMany(ctx context.Context, database string, collection string,
	models []mongo.IndexModel, opts ...*options.CreateIndexesOptions) ([]string, error) {
	request := &Request{
		Command:    IndexCreateMany,
		Database:   database,
		Collection: collection,
		CommArg:    models,
		Opts:       opts,
	}
	rsp, err := c.invoke(ctx, request)
	if err != nil {
		return nil, err
	}
	if sliceStr, ok := rsp.([]string); ok {
		return sliceStr, nil
	}
	return nil, buildUnMatchKindError(reflect.Slice, rsp)
}

// CreateOne executes a createIndexes command to create an index on the collection and returns the name of the new
// index. See the IndexView.CreateMany documentation for more information and an example.
func (c *mongodbCli) CreateOne(ctx context.Context, database string, collection string,
	model mongo.IndexModel, opts ...*options.CreateIndexesOptions) (string, error) {
	request := &Request{
		Command:    IndexCreateOne,
		Database:   database,
		Collection: collection,
		CommArg:    model,
		Opts:       opts,
	}
	rsp, err := c.invoke(ctx, request)
	if err != nil {
		return "", err
	}
	if s, ok := rsp.(string); ok {
		return s, nil
	}
	return "", buildUnMatchKindError(reflect.String, rsp)
}

// DropOne executes a dropIndexes operation to drop an index on the collection. If the operation succeeds, this returns
// a BSON document in the form {nIndexesWas: <int32>}. The "nIndexesWas" field in the response contains the number of
// indexes that existed prior to the drop.
//
// The name parameter should be the name of the index to drop. If the name is "*", ErrMultipleIndexDrop will be returned
// without running the command because doing so would drop all indexes.
//
// The opts parameter can be used to specify options for this operation (see the options.DropIndexesOptions
// documentation).
//
// For more information about the command, see https://docs.mongodb.com/manual/reference/command/dropIndexes/.
func (c *mongodbCli) DropOne(ctx context.Context, database string, collection string,
	name string, opts ...*options.DropIndexesOptions) (bson.Raw, error) {
	request := &Request{
		Command:    IndexDropOne,
		Database:   database,
		Collection: collection,
		CommArg:    name,
		Opts:       opts,
	}
	rsp, err := c.invoke(ctx, request)
	if err != nil {
		return nil, err
	}
	if raw, ok := rsp.(bson.Raw); ok {
		return raw, nil
	}
	return nil, buildUnMatchKindError(reflect.Slice, rsp)
}

// DropAll executes a dropIndexes operation to drop all indexes on the collection. If the operation succeeds, this
// returns a BSON document in the form {nIndexesWas: <int32>}. The "nIndexesWas" field in the response contains the
// number of indexes that existed prior to the drop.
//
// The opts parameter can be used to specify options for this operation (see the options.DropIndexesOptions
// documentation).
//
// For more information about the command, see https://docs.mongodb.com/manual/reference/command/dropIndexes/.
func (c *mongodbCli) DropAll(ctx context.Context, database string, collection string,
	opts ...*options.DropIndexesOptions) (bson.Raw, error) {
	request := &Request{
		Command:    IndexDropAll,
		Database:   database,
		Collection: collection,
		CommArg:    nil,
		Opts:       opts,
	}
	rsp, err := c.invoke(ctx, request)
	if err != nil {
		return nil, err
	}
	if raw, ok := rsp.(bson.Raw); ok {
		return raw, nil
	}
	return nil, buildUnMatchKindError(reflect.Slice, rsp)
}

// buildUnMatchKindError builds a type mismatch error,
// the result type returned when invoke is not what we expected.
func buildUnMatchKindError(want reflect.Kind, actual interface{}) error {
	val := reflect.ValueOf(actual)
	return fmt.Errorf("the result kind of got is not expect, expect is %s but actural is %s",
		want.String(), val.Kind().String())
}

// Indexes gets the original index operation object.
func (c *mongodbCli) Indexes(ctx context.Context, database string, collection string) (mongo.IndexView, error) {
	request := &Request{
		Command:    Indexes,
		Database:   database,
		Collection: collection,
		CommArg:    nil,
	}
	rsp, err := c.invoke(ctx, request)
	if err != nil {
		return mongo.IndexView{}, err
	}
	if raw, ok := rsp.(mongo.IndexView); ok {
		return raw, nil
	}
	return mongo.IndexView{}, buildUnMatchKindError(reflect.Struct, rsp)
}

// Database gets the original database used to call the original method.
func (c *mongodbCli) Database(ctx context.Context, database string) (*mongo.Database, error) {
	request := &Request{
		Command:  DatabaseCmd,
		Database: database,
		CommArg:  nil,
	}
	rsp, err := c.invoke(ctx, request)
	if err != nil {
		return nil, err
	}
	if raw, ok := rsp.(*mongo.Database); ok {
		return raw, nil
	}
	return nil, buildUnMatchKindError(reflect.Ptr, rsp)
}

// Collection gets the original collection used to call the original method.
func (c *mongodbCli) Collection(ctx context.Context, database string, collection string) (*mongo.Collection, error) {
	request := &Request{
		Command:    CollectionCmd,
		Database:   database,
		Collection: collection,
		CommArg:    nil,
	}
	rsp, err := c.invoke(ctx, request)
	if err != nil {
		return nil, err
	}
	if raw, ok := rsp.(*mongo.Collection); ok {
		return raw, nil
	}
	return nil, buildUnMatchKindError(reflect.Ptr, rsp)
}

// StartSession starts a new session configured with the given options.
// StartSession does not actually communicate with the server and will not error if the client is disconnected.
// If the DefaultReadConcern, DefaultWriteConcern, or DefaultReadPreference options are not set,
// the client's read concern, write concern, or read preference will be used, respectively.
func (c *mongodbCli) StartSession(ctx context.Context) (mongo.Session, error) {
	request := &Request{
		Command: StartSession,
		CommArg: nil,
	}
	rsp, err := c.invoke(ctx, request)
	if err != nil {
		return nil, err
	}
	if raw, ok := rsp.(mongo.Session); ok {
		return raw, nil
	}
	return nil, buildUnMatchKindError(reflect.Ptr, rsp)
}
