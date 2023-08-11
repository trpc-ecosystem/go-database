// Package gorm encapsulates the tRPC plugin in GORM.
package gorm

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"
	"strings"
	"unsafe"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
	_ "trpc.group/trpc-go/trpc-selector-dsn" // Need import the dsn selector.
)

// OpEnum is the database operation enumeration type.
type OpEnum int

// Constants for database operation types. Starting from 1 to avoid using the default value 0.
const (
	OpPrepareContext OpEnum = iota + 1
	OpExecContext
	OpQueryContext
	OpQueryRowContext
	OpBeginTx
	OpPing
	OpCommit
	OpRollback
	OpGetDB
)

// String converts the operation type constant to human-readable text.
func (op OpEnum) String() string {
	return [...]string{
		"",
		"PrepareContext",
		"ExecContext",
		"QueryContext",
		"QueryRowContext",
		"BeginTx",
		"Ping",
		"Commit",
		"Rollback",
		"GetDB",
	}[op]
}

// ConnPool implements the gorm.ConnPool interface as well as transaction and Ping functionality.
type ConnPool interface {
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	// BeginTx implements ConnPoolBeginner.
	BeginTx(ctx context.Context, opts *sql.TxOptions) (gorm.ConnPool, error)
	// Ping implements pinger.
	Ping() error
	// GetDBConn implements GetDBConnector.
	GetDBConn() (*sql.DB, error)
}

// Request is the request passed to the tRPC framework.
type Request struct {
	// Op is the type of database operation performed.
	Op        OpEnum
	Query     string
	Args      []interface{}
	Tx        *sql.Tx
	TxOptions *sql.TxOptions
}

// Response is the result returned by the tRPC framework.
type Response struct {
	Result sql.Result
	Stmt   *sql.Stmt
	Row    *sql.Row
	Rows   *sql.Rows
	Tx     *sql.Tx
	DB     *sql.DB
}

// Client encapsulates the tRPC client and implements the ConnPoolBeginner interface.
type Client struct {
	ServiceName string
	Client      client.Client
	opts        []client.Option
}

// TxClient is the TRPC client with a transaction opened, and it implements the TxCommitter interface.
// The reason for separating the implementation of Client and TxClient
//
//		is that GORM defines two interfaces to support transaction operations:
//	 1. ConnPoolBeginner: including the method for opening transactions using BeginTx,
//	    corresponding to the Client object of this plugin.
//	 2. TxCommitter: including the methods for committing and rolling back transactions using Commit and Rollback,
//	    corresponding to the TxClient object of this plugin.
//
// These two interfaces correspond to two types of connection pools,
//
//	before and after opening transactions, which cannot be mixed.
//
// For example, under the GORM automatic transaction mechanism (SkipDefaultTransaction=false),
//
//	for operations such as Create and Delete,
//	if the connection pool has implemented the 'ConnPoolBeginner' interface, it will automatically open a transaction.
//	If the current connection itself has already opened a transaction,
//	it will cause an exception of 'lock wait timeout exceeded' due to the duplicate opening of the transaction,
//	as detailed in the GORM.DB.Begin method.
//
// Therefore, after opening a transaction,
//
//	you need to convert the Client object to a connection object
//	that has not implemented the 'ConnPoolBeginner' interface: TxClient.
//
// The calling logic is:
//  1. Use Client before opening a transaction. At this time,
//     the Client implements the ConnPoolBeginner interface and can call BeginTx to open a transaction.
//  2. Call Client.BeginTx to open a transaction and return TxClient.
type TxClient struct {
	Client *Client
	Tx     *sql.Tx
}

// DefaultTRPCLogger is the GORM logger that connects to trpc-log/log.
var DefaultTRPCLogger = NewTRPCLogger(logger.Config{
	LogLevel:                  logger.Warn,
	IgnoreRecordNotFoundError: true,
})

// NewConnPool generates a ConnPool that sends all requests through tRPC.
// This ConnPool can be used as a parameter to generate gorm.DB,
// making it easier to adjust other configurations of GORM.
var NewConnPool = func(name string, opts ...client.Option) ConnPool {
	c := &Client{
		ServiceName: name,
		Client:      client.DefaultClient,
	}
	c.opts = make([]client.Option, 0, len(opts)+3)
	c.opts = append(c.opts, opts...)
	c.opts = append(c.opts,
		client.WithProtocol("gorm"),
		client.WithDisableServiceRouter(),
		client.WithTimeout(0),
	)
	return c
}

// NewClientProxy generates a gorm.DB that can connect to a specified location and send all requests through tRPC.
var NewClientProxy = func(name string, opts ...client.Option) (*gorm.DB, error) {
	connPool := NewConnPool(name, opts...)
	// If you need to add other types of databases, add a switch...case statement here and in transport.go.
	splitServiceName := strings.Split(name, ".")
	if len(splitServiceName) < 2 {
		return gorm.Open(
			mysql.New(
				mysql.Config{
					Conn: connPool,
				}),
			&gorm.Config{
				Logger: getLogger(name),
			})
	}
	// Support for other DBs.
	// Compatibility logic, defaulting to MySQL.
	dbEngineType := splitServiceName[1]
	// Support postgresql.
	switch dbEngineType {
	case "postgres":
		// Set the dialect configuration. Using the wrong dialect may cause strange errors.
		return gorm.Open(
			postgres.New(
				postgres.Config{
					PreferSimpleProtocol: true,
					Conn:                 connPool,
				}),
			&gorm.Config{
				Logger: getLogger(name),
			})
	default:
		return gorm.Open(
			mysql.New(
				mysql.Config{
					Conn: connPool,
				}),
			&gorm.Config{
				Logger: getLogger(name),
			},
		)
	}
}

// handleReq abstracts multiple types of requests into one type and identifies the type using 'op'.
// It enters the TRPC-Go call chain and is restored to the original request in the transport module.
func handleReq(ctx context.Context, cp gorm.ConnPool, mreq *Request, mrsp *Response) error {
	var gc *Client
	if txgc, ok := cp.(*TxClient); ok {
		gc = txgc.Client
		mreq.Tx = txgc.Tx
	} else {
		gc = cp.(*Client)
	}
	mctx, msg := codec.WithCloneMessage(ctx)
	defer codec.PutBackMessage(msg)
	msg.WithClientRPCName(fmt.Sprintf("/%s/%s", gc.ServiceName, mreq.Op))
	msg.WithCalleeServiceName(gc.ServiceName)
	msg.WithSerializationType(-1) // Do not serialize.
	msg.WithCompressType(0)       // Do not compress.
	msg.WithClientReqHead(mreq)
	msg.WithClientRspHead(mrsp)
	// Handle request parameters.
	err := handleReqArgs(mreq)
	if err != nil {
		return err
	}
	// Pass the request to the tRPC framework.
	return gc.Client.Invoke(mctx, mreq, mrsp, gc.opts...)
}

// handleReqArgs handles the pass-through request parameters.
func handleReqArgs(mreq *Request) error {
	// When the request parameter of GORM is *schema.serializer (which implements the driver.Valuer interface),
	// and there is a circular reference in the schema.serializer structure,
	// when the request is passed to the later chain like opentelemtry,
	// stack overflow may occur during serialization, which will hang the process,
	// so here the value needs to be passed.
	for k, arg := range mreq.Args {
		if valuer, ok := arg.(driver.Valuer); ok {
			v, err := valuer.Value()
			if err != nil {
				return err
			}
			mreq.Args[k] = v
		}
	}
	return nil
}

// In order for the gorm.DB generated by ConnPool to be able to use all the native functionalities of GORM,
// it is necessary to implement the gorm.ConnPool, gorm.ConnPoolBeginner interfaces, and the Ping method.

// PrepareContext implements the ConnPool.PrepareContext method.
func (gc *Client) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	return prepareContext(gc, ctx, query)
}

func prepareContext(cp gorm.ConnPool, ctx context.Context, query string) (*sql.Stmt, error) {
	mreq := &Request{
		Op:    OpPrepareContext,
		Query: query,
	}
	mrsp := &Response{}

	if err := handleReq(ctx, cp, mreq, mrsp); err != nil {
		return nil, err
	}
	return mrsp.Stmt, nil
}

// ExecContext implements the ConnPool.ExecContext method.
func (gc *Client) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return execContext(gc, ctx, query, args...)
}

func execContext(cp gorm.ConnPool, ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	mreq := &Request{
		Op:    OpExecContext,
		Query: query,
		Args:  args,
	}
	mrsp := &Response{}

	if err := handleReq(ctx, cp, mreq, mrsp); err != nil {
		return nil, err
	}
	return mrsp.Result, nil
}

// QueryContext implements the ConnPool.QueryContext method.
func (gc *Client) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return queryContext(gc, ctx, query, args...)
}

func queryContext(cp gorm.ConnPool, ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	mreq := &Request{
		Op:    OpQueryContext,
		Query: query,
		Args:  args,
	}
	mrsp := &Response{}

	if err := handleReq(ctx, cp, mreq, mrsp); err != nil {
		return nil, err
	}
	return mrsp.Rows, nil
}

// QueryRowContext implements the ConnPool.QueryRowContext method.
func (gc *Client) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return queryRowContext(gc, ctx, query, args...)
}

func queryRowContext(cp gorm.ConnPool, ctx context.Context, query string, args ...interface{}) *sql.Row {
	mreq := &Request{
		Op:    OpQueryRowContext,
		Query: query,
		Args:  args,
	}
	mrsp := &Response{}
	if err := handleReq(ctx, cp, mreq, mrsp); err != nil {
		// An error occurred during execution,
		// and sql.Row.err needs to be assigned a value to avoid panic during chain calls to QueryRowContent().Scan().
		row := &sql.Row{}
		v := reflect.ValueOf(row)
		errField := v.Elem().FieldByName("err")
		errPointer := unsafe.Pointer(errField.UnsafeAddr())
		errVal := (*error)(errPointer)
		*errVal = err
		return row
	}
	return mrsp.Row
}

// BeginTx implements the ConnPoolBeginner.BeginTx method.
func (gc *Client) BeginTx(ctx context.Context, opts *sql.TxOptions) (gorm.ConnPool, error) {
	mreq := &Request{
		Op:        OpBeginTx,
		TxOptions: opts,
	}
	mrsp := &Response{}

	if err := handleReq(ctx, gc, mreq, mrsp); err != nil {
		return nil, err
	}

	// Returns a client with an opened transaction.
	txc := &TxClient{
		Client: gc,
		Tx:     mrsp.Tx,
	}
	return txc, nil
}

// Ping implements the Ping method.
func (gc *Client) Ping() error {
	mreq := &Request{
		Op: OpPing,
	}
	mrsp := &Response{}
	ctx := trpc.BackgroundContext()
	if err := handleReq(ctx, gc, mreq, mrsp); err != nil {
		return err
	}
	return nil
}

// GetDBConn implements the GetDBConn method.
func (gc *Client) GetDBConn() (*sql.DB, error) {
	mreq := &Request{
		Op: OpGetDB,
	}
	mrsp := &Response{}
	ctx := trpc.BackgroundContext()
	if err := handleReq(ctx, gc, mreq, mrsp); err != nil {
		return nil, err
	}
	return mrsp.DB, nil
}

// TxClient implements gorm.ConnPool.

// PrepareContext implements the ConnPool.PrepareContext method.
func (txgc *TxClient) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	return prepareContext(txgc, ctx, query)
}

// ExecContext implements the ConnPool.ExecContext method.
func (txgc *TxClient) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return execContext(txgc, ctx, query, args...)
}

// QueryContext implements the ConnPool.QueryContext method.
func (txgc *TxClient) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return queryContext(txgc, ctx, query, args...)
}

// QueryRowContext implements the ConnPool.QueryRowContext method.
func (txgc *TxClient) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return queryRowContext(txgc, ctx, query, args...)
}

// TxClient implements TxCommitter.

// Commit performs a commit.
func (txgc *TxClient) Commit() error {
	mreq := &Request{
		Op: OpCommit,
	}
	mrsp := &Response{}

	return handleReq(context.TODO(), txgc, mreq, mrsp)
}

// Rollback rolls back the transaction.
func (txgc *TxClient) Rollback() error {
	mreq := &Request{
		Op: OpRollback,
	}
	mrsp := &Response{}

	return handleReq(context.TODO(), txgc, mreq, mrsp)
}
