// Package mysql Wrapping standard library mysql
package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"

	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-utils/copyutils"

	// anonymous import package
	_ "trpc.group/trpc-go/trpc-selector-dsn"
)

const (
	opExec = iota + 1
	opGet
	opQuery
	opQueryRow
	opQueryToStruct
	opQueryToStructs
	opSelect
	opTransaction
	opNamedExec
	opNamedQuery
	opTransactionx
)

// Client client Structure
//
//go:generate mockgen -source=client.go -destination=mockmysql/mysql_mock.go -package=mockmysql.
type Client interface {
	// DB implementation via native sql.
	Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	Query(ctx context.Context, next NextFunc, query string, args ...interface{}) error
	QueryRow(ctx context.Context, dest []interface{}, query string, args ...interface{}) error
	Transaction(ctx context.Context, fn TxFunc, opts ...TxOption) error

	// Implementation via sqlx.
	Get(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	Select(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	NamedExec(ctx context.Context, query string, args interface{}) (sql.Result, error)
	NamedQuery(ctx context.Context, query string, args interface{}) (*sqlx.Rows, error)
	QueryToStruct(ctx context.Context, dst interface{}, query string, args ...interface{}) error
	QueryToStructs(ctx context.Context, dst interface{}, query string, args ...interface{}) error
	Transactionx(ctx context.Context, fn TxxFunc, opts ...TxOption) error
}

// Client back-end request structure.
type mysqlCli struct {
	serviceName string
	client      client.Client
	opts        []client.Option
	// if unsafe mode is on, it will silently succeed to scan when
	// columns in the SQL result have no fields in the destination struct.
	unsafe bool
}

func newMysqlCli(name string, opts ...client.Option) *mysqlCli {
	return &mysqlCli{
		serviceName: name,
		client:      client.DefaultClient,
		opts:        append([]client.Option{client.WithProtocol("mysql"), client.WithDisableServiceRouter()}, opts...),
	}
}

// NewClientProxy Create a new mysql backend request proxy Mandatory parameter mysql service name: trpc.mysql.xxx.xxx.
var NewClientProxy = func(name string, opts ...client.Option) Client {
	return newMysqlCli(name, opts...)
}

// NewUnsafeClient is similar to NewClientProxy, except that it enables sqlx's unsafe mode,
// which only affects sqlx-related operations.
func NewUnsafeClient(name string, opts ...client.Option) Client {
	c := newMysqlCli(name, opts...)
	c.unsafe = true // turns on sqlx's unsafe mode
	return c
}

// Request mysql request body.
type Request struct {
	Query string
	Exec  string
	Args  []interface{}

	// if unsafe mode is on, it will silently succeed to scan when
	// columns in the SQL result have no fields in the destination struct.
	unsafe bool

	op     int
	next   NextFunc
	tx     TxFunc
	txx    TxxFunc
	txOpts *sql.TxOptions

	QueryToDest        interface{}
	QueryToStructDest  interface{}
	QueryToStructsDest interface{}
	QueryRowDest       []interface{}
}

// Copy returns a new Request.
//
// Read-only fields in Request only make a shallow copy. For fields that
// may be written concurrently, a deep copy is required.
// Note that NextFunc and TxFunc are closures and we can only make a normal copy of them.
// But this necessarily breaks concurrency safety. Therefore, Copy returns an error for a non-empty next or tx.
func (r *Request) Copy() (interface{}, error) {
	if r.next != nil {
		return nil, errors.New("request with non nil next closure does not support Copy")
	}
	if r.tx != nil {
		return nil, errors.New("request with non nil tx closure does not support Copy")
	}

	newReq := Request{
		Query:  r.Query,
		Exec:   r.Exec,
		Args:   r.Args,
		op:     r.op,
		next:   r.next,
		tx:     r.tx,
		txOpts: r.txOpts,
		unsafe: r.unsafe,
	}

	replica, err := copyutils.DeepCopy(r.QueryToDest)
	if err != nil {
		return nil, err
	}
	newReq.QueryToDest = replica
	replica, err = copyutils.DeepCopy(r.QueryToStructsDest)
	if err != nil {
		return nil, err
	}
	newReq.QueryToStructsDest = replica
	replica, err = copyutils.DeepCopy(r.QueryToStructDest)
	if err != nil {
		return nil, err
	}
	newReq.QueryToStructDest = replica
	replica, err = copyutils.DeepCopy(r.QueryRowDest)
	if err != nil {
		return nil, err
	}
	newReq.QueryRowDest = replica.([]interface{})

	return &newReq, nil
}

// CopyTo copies a Request into another Request. dst must be of type *Request.
// Similar to Copy, CopyTo returns an error for a non-empty next or tx.
func (r *Request) CopyTo(dst interface{}) error {
	if r.next != nil {
		return fmt.Errorf("request with non nil next closure does not support CopyTo")
	}
	if r.tx != nil {
		return fmt.Errorf("Request with non nil tx closure does not support CopyTo")
	}

	dr, ok := dst.(*Request)
	if !ok {
		return fmt.Errorf("expect dst type %T, got %T", r, dst)
	}

	dr.Query = r.Query
	dr.Exec = r.Exec
	dr.Args = r.Args
	dr.op = r.op
	dr.next = r.next
	dr.tx = r.tx
	dr.txOpts = r.txOpts
	dr.unsafe = r.unsafe

	// If QueryToStructsDest or QueryRowDest exist, then
	// they must be provided by the application layer.
	// We must ensure that the memory addresses pointed to by these fields remain unchanged, that is,
	// they cannot be simply assigned.
	// For specific content, it is sufficient to use a shallow copy.
	if dr.QueryToDest != nil {
		if err := copyutils.ShallowCopy(dr.QueryToDest, r.QueryToDest); err != nil {
			return nil
		}
	}
	if dr.QueryToStructsDest != nil {
		if err := copyutils.ShallowCopy(dr.QueryToStructsDest, r.QueryToStructsDest); err != nil {
			return err
		}
	}
	if dr.QueryToStructDest != nil {
		if err := copyutils.ShallowCopy(dr.QueryToStructDest, r.QueryToStructDest); err != nil {
			return err
		}
	}
	if len(dr.QueryRowDest) != len(r.QueryRowDest) {
		return fmt.Errorf("the length of QueryRowDest does not match, expect %d, got %d",
			len(r.QueryRowDest), len(dr.QueryRowDest))
	}
	for i := range r.QueryRowDest {
		if err := copyutils.ShallowCopy(dr.QueryRowDest[i], r.QueryRowDest[i]); err != nil {
			return err
		}
	}

	return nil
}

// Response mysql response body.
type Response struct {
	Result   sql.Result
	SqlxRows *sqlx.Rows
}

// Args mysql named args.
type Args []map[string]interface{}

type (
	// NextFunc query request, each row of data records to execute the logic, required.
	// Wrapped in the bottom of the framework to prevent the user from missing rows.Close, rows.Err, etc., and
	// to prevent the context from being cancelled during scan.
	// Return value error ==nil continue to the next row, ==ErrBreak end the loop early ! =nil return failure.
	NextFunc func(*sql.Rows) error

	// TxFunc is a user transaction function that returns err ! = nil when
	// the transaction is automatically rolled back, otherwise the transaction is automatically committed.
	// TxFunc receives native sql.Tx.
	TxFunc func(*sql.Tx) error

	// TxxFunc is a user transaction function that returns err ! = nil, otherwise
	// the transaction is automatically rolled back, otherwise the transaction is automatically committed.
	// TxxFunc receives sqlx.Tx as an argument, providing more helper methods.
	TxxFunc func(*sqlx.Tx) error
)

var (
	// ErrBreak ends the scan loop by breaking early.
	ErrBreak = errors.New("mysql scan rows break")
)

// Query Execute the mysql select command.
func (c *mysqlCli) Query(ctx context.Context, next NextFunc, query string, args ...interface{}) error {
	mreq := &Request{
		Query:  query,
		Args:   args,
		next:   next,
		op:     opQuery,
		unsafe: c.unsafe,
	}
	mrsp := &Response{}

	mctx, msg := codec.WithCloneMessage(ctx)
	defer codec.PutBackMessage(msg)
	msg.WithClientRPCName(fmt.Sprintf("/%s/Query", c.serviceName))
	msg.WithCalleeServiceName(c.serviceName)
	msg.WithSerializationType(codec.SerializationTypeUnsupported)
	msg.WithCompressType(codec.CompressTypeNoop)
	msg.WithClientReqHead(mreq)
	msg.WithClientRspHead(mrsp)

	err := c.client.Invoke(mctx, mreq, mrsp, c.opts...)
	if err != nil {
		return err
	}

	return nil
}

// QueryRow Execute the mysql QueryRow command.
func (c *mysqlCli) QueryRow(ctx context.Context, dest []interface{}, query string, args ...interface{}) error {
	mreq := &Request{
		Query:        query,
		Args:         args,
		QueryRowDest: dest,
		op:           opQueryRow,
		unsafe:       c.unsafe,
	}
	mrsp := &Response{}

	mctx, msg := codec.WithCloneMessage(ctx)
	defer codec.PutBackMessage(msg)
	msg.WithClientRPCName(fmt.Sprintf("/%s/QueryRow", c.serviceName))
	msg.WithCalleeServiceName(c.serviceName)
	msg.WithSerializationType(codec.SerializationTypeUnsupported)
	msg.WithCompressType(codec.CompressTypeNoop)
	msg.WithClientReqHead(mreq)
	msg.WithClientRspHead(mrsp)

	err := c.client.Invoke(mctx, mreq, mrsp, c.opts...)
	if err != nil {
		return err
	}

	return nil
}

// QueryToStruct executes the mysql select command and scans the results into the dst struct.
func (c *mysqlCli) QueryToStruct(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	mreq := &Request{
		Query:             query,
		Args:              args,
		QueryToStructDest: dest,
		op:                opQueryToStruct,
		unsafe:            c.unsafe,
	}
	mrsp := &Response{}

	ctx, msg := codec.WithCloneMessage(ctx)
	msg.WithClientRPCName(fmt.Sprintf("/%s/QueryToStruct", c.serviceName))
	msg.WithCalleeServiceName(c.serviceName)
	msg.WithSerializationType(codec.SerializationTypeUnsupported)
	msg.WithCompressType(codec.CompressTypeNoop)
	msg.WithClientReqHead(mreq)
	msg.WithClientRspHead(mrsp)

	err := c.client.Invoke(ctx, mreq, mrsp, c.opts...)
	if err != nil {
		return err
	}
	return nil
}

// QueryToStructs executes mysql select command to scan the result into dst slices, supporting structure slices.
func (c *mysqlCli) QueryToStructs(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	mreq := &Request{
		Query:              query,
		Args:               args,
		QueryToStructsDest: dest,
		op:                 opQueryToStructs,
		unsafe:             c.unsafe,
	}
	mrsp := &Response{}

	ctx, msg := codec.WithCloneMessage(ctx)
	defer codec.PutBackMessage(msg)
	msg.WithClientRPCName(fmt.Sprintf("/%s/QueryToStructs", c.serviceName))
	msg.WithCalleeServiceName(c.serviceName)
	msg.WithSerializationType(codec.SerializationTypeUnsupported)
	msg.WithCompressType(codec.CompressTypeNoop)
	msg.WithClientReqHead(mreq)
	msg.WithClientRspHead(mrsp)

	err := c.client.Invoke(ctx, mreq, mrsp, c.opts...)
	if err != nil {
		return err
	}

	return nil
}

// Exec execute mysql insert delete update command.
func (c *mysqlCli) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	mreq := &Request{
		Exec:   query,
		Args:   args,
		op:     opExec,
		unsafe: c.unsafe,
	}
	mrsp := &Response{}

	ctx, msg := codec.WithCloneMessage(ctx)
	defer codec.PutBackMessage(msg)
	msg.WithClientRPCName(fmt.Sprintf("/%s/Exec", c.serviceName))
	msg.WithCalleeServiceName(c.serviceName)
	msg.WithSerializationType(codec.SerializationTypeUnsupported)
	msg.WithCompressType(codec.CompressTypeNoop)
	msg.WithClientReqHead(mreq)
	msg.WithClientRspHead(mrsp)

	err := c.client.Invoke(ctx, mreq, mrsp, c.opts...)
	if err != nil {
		return nil, err
	}

	return mrsp.Result, nil
}

// NamedExec performs MySQL write operations (insert, update, delete), similar to Exec, but with bound name mapping.
// The mapping form supports the use of []map or struct,
// for examples and descriptions see：https://jmoiron.github.io/sqlx/#namedParams.
// If the field is in struct format, the mapped field should be tagged with
// db in the tag of the struct field, example: struct { Name string `db: "name"` }.
// If you use the []map format, the example：[]map[string]interface{}{ {"first_name": "Ardie", "last_name": "Savea" }.
func (c *mysqlCli) NamedExec(ctx context.Context, query string, args interface{}) (sql.Result, error) {
	ctx, msg := codec.WithCloneMessage(ctx)
	defer codec.PutBackMessage(msg)
	msg.WithClientRPCName(fmt.Sprintf("/%s/NamedExec", c.serviceName))
	msg.WithCalleeServiceName(c.serviceName)
	msg.WithSerializationType(codec.SerializationTypeUnsupported)
	msg.WithCompressType(codec.CompressTypeNoop)

	request := &Request{
		Exec:   query,
		Args:   []interface{}{args},
		op:     opNamedExec,
		unsafe: c.unsafe,
	}
	response := new(Response)
	msg.WithClientReqHead(request)
	msg.WithClientRspHead(response)

	err := c.client.Invoke(ctx, request, response, c.opts...)
	if err != nil {
		return nil, err
	}

	return response.Result, nil
}

// Transaction executes a mysql transaction, fn is a multi-callback function that receives *sql.Tx.
// fn returns error ! = nil the transaction is automatically rolled back, otherwise the transaction
// is automatically committed.
func (c *mysqlCli) Transaction(ctx context.Context, fn TxFunc, opts ...TxOption) error {
	txOpts := new(sql.TxOptions)
	for _, o := range opts {
		o(txOpts)
	}
	mreq := &Request{
		tx:     fn,
		txOpts: txOpts,
		op:     opTransaction,
		unsafe: c.unsafe,
	}
	mrsp := &Response{}
	ctx, msg := codec.WithCloneMessage(ctx)
	defer codec.PutBackMessage(msg)
	msg.WithClientRPCName(fmt.Sprintf("/%s/Transaction", c.serviceName))
	msg.WithCalleeServiceName(c.serviceName)
	msg.WithSerializationType(codec.SerializationTypeUnsupported)
	msg.WithCompressType(codec.CompressTypeNoop)
	msg.WithClientReqHead(mreq)
	msg.WithClientRspHead(mrsp)

	err := c.client.Invoke(ctx, mreq, mrsp, c.opts...)
	if err != nil {
		return err
	}
	return nil
}

// Transactionx executes mysql transactions via sqlx, fn is a multi-callback function that receives *sqlx.
// fn returns error ! = nil the transaction is automatically rolled back,
// otherwise the transaction is automatically committed.
func (c *mysqlCli) Transactionx(ctx context.Context, fn TxxFunc, opts ...TxOption) error {
	txOpts := new(sql.TxOptions)
	for _, o := range opts {
		o(txOpts)
	}
	mreq := &Request{
		txx:    fn,
		txOpts: txOpts,
		op:     opTransactionx,
		unsafe: c.unsafe,
	}

	mrsp := &Response{}
	ctx, msg := codec.WithCloneMessage(ctx)
	defer codec.PutBackMessage(msg)
	msg.WithClientRPCName(fmt.Sprintf("/%s/Transactionx", c.serviceName))
	msg.WithCalleeServiceName(c.serviceName)
	msg.WithSerializationType(codec.SerializationTypeUnsupported)
	msg.WithCompressType(codec.CompressTypeNoop)
	msg.WithClientReqHead(mreq)
	msg.WithClientRspHead(mrsp)

	err := c.client.Invoke(ctx, mreq, mrsp, c.opts...)
	if err != nil {
		return err
	}
	return nil
}

// Get Query a single data item Scan the result to dest.
func (c *mysqlCli) Get(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	mreq := &Request{
		Query:       query,
		Args:        args,
		QueryToDest: dest,
		op:          opGet,
		unsafe:      c.unsafe,
	}
	mrsp := &Response{}

	ctx, msg := codec.WithCloneMessage(ctx)
	defer codec.PutBackMessage(msg)
	msg.WithClientRPCName(fmt.Sprintf("/%s/Get", c.serviceName))
	msg.WithCalleeServiceName(c.serviceName)
	msg.WithSerializationType(codec.SerializationTypeUnsupported)
	msg.WithCompressType(codec.CompressTypeNoop)
	msg.WithClientReqHead(mreq)
	msg.WithClientRspHead(mrsp)

	err := c.client.Invoke(ctx, mreq, mrsp, c.opts...)
	if err != nil {
		return err
	}

	return nil
}

// Select Query multiple data Scan results to dest.
func (c *mysqlCli) Select(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	mreq := &Request{
		Query:       query,
		Args:        args,
		QueryToDest: dest,
		op:          opSelect,
		unsafe:      c.unsafe,
	}
	mrsp := &Response{}

	ctx, msg := codec.WithCloneMessage(ctx)
	defer codec.PutBackMessage(msg)
	msg.WithClientRPCName(fmt.Sprintf("/%s/Select", c.serviceName))
	msg.WithCalleeServiceName(c.serviceName)
	msg.WithSerializationType(codec.SerializationTypeUnsupported)
	msg.WithCompressType(codec.CompressTypeNoop)
	msg.WithClientReqHead(mreq)
	msg.WithClientRspHead(mrsp)

	err := c.client.Invoke(ctx, mreq, mrsp, c.opts...)
	if err != nil {
		return err
	}

	return nil
}

// NamedQuery executes mysql query query condition binding name mapping.
// Parameter support map, struct Reference https://jmoiron.github.io/sqlx/#namedParams.
func (c *mysqlCli) NamedQuery(ctx context.Context, query string, args interface{}) (*sqlx.Rows, error) {
	mreq := &Request{
		Query:  query,
		Args:   []interface{}{args},
		op:     opNamedQuery,
		unsafe: c.unsafe,
	}
	mrsp := &Response{}

	ctx, msg := codec.WithCloneMessage(ctx)
	defer codec.PutBackMessage(msg)
	msg.WithClientRPCName(fmt.Sprintf("/%s/NamedQuery", c.serviceName))
	msg.WithCalleeServiceName(c.serviceName)
	msg.WithSerializationType(codec.SerializationTypeUnsupported)
	msg.WithCompressType(codec.CompressTypeNoop)
	msg.WithClientReqHead(mreq)
	msg.WithClientRspHead(mrsp)

	err := c.client.Invoke(ctx, mreq, mrsp, c.opts...)
	if err != nil {
		return nil, err
	}

	return mrsp.SqlxRows, nil
}
