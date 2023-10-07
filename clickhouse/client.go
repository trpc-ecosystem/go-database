// Package clickhouse 封装标准库clickhouse
package clickhouse

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-utils/copyutils"
)

// Operation type enumeration.
const (
	opExec = iota + 1
	opQuery
	opQueryRow
	opQueryToStructs
	opTransaction
)

// Client is client structure.
type Client interface {
	Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	Query(ctx context.Context, next NextFunc, query string, args ...interface{}) error
	QueryRow(ctx context.Context, dest []interface{}, query string, args ...interface{}) error
	QueryToStructs(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	Transaction(ctx context.Context, fn TxFunc) error
}

// Client is backend request structure.
type clickhouseCli struct {
	ServiceName string
	Client      client.Client
	opts        []client.Option
}

// NewClientProxy creates a new clickhouse backend request proxy.
// Required parameter clickhouse service name: trpc.clickhouse.xxx.xxx.
func NewClientProxy(name string, opts ...client.Option) Client {
	c := &clickhouseCli{
		ServiceName: name,
		Client:      client.DefaultClient,
	}

	c.opts = make([]client.Option, 0, len(opts)+2)
	c.opts = append(c.opts, opts...)
	c.opts = append(c.opts, client.WithProtocol("clickhouse"), client.WithDisableServiceRouter())
	return c
}

// Request clickhouse request body
type Request struct {
	Query string
	Exec  string
	Args  []interface{}

	op   int
	next NextFunc
	tx   TxFunc

	queryToStructsDest interface{}
	queryRowDest       []interface{}
}

// Copy returns a new Request.
//
// Read-only fields in Request are only shallow copied.
// For fields that may be written concurrently, a deep copy is required.
// It should be noted that NextFunc and TxFunc are closures, we can only copy them normally.
// But this necessarily breaks concurrency safety.
// Therefore, Copy returns an error for a non-empty next or tx.
func (r *Request) Copy() (interface{}, error) {
	if r.next != nil {
		return nil, errors.New("request with non nil next closure does not support Copy")
	}
	if r.tx != nil {
		return nil, errors.New("request with non nil tx closure does not support Copy")
	}

	newReq := Request{
		Query: r.Query,
		Exec:  r.Exec,
		Args:  r.Args,
		op:    r.op,
		next:  r.next,
		tx:    r.tx,
	}

	queryToStructDest, err := copyutils.DeepCopy(r.queryToStructsDest)
	if err != nil {
		return nil, err
	}
	newReq.queryToStructsDest = queryToStructDest

	queryRowDest, err := copyutils.DeepCopy(r.queryRowDest)
	if err != nil {
		return nil, err
	}
	var ok bool
	newReq.queryRowDest, ok = queryRowDest.([]interface{})
	if !ok {
		return nil, errors.New("copyutils.DeepCopy queryRowDest need []interface{}")
	}

	return &newReq, nil
}

// CopyTo copies a Request to another Request. dst must be of type *Request.
// Similar to Copy, for non-empty next or tx, CopyTo will return an error.
func (r *Request) CopyTo(dst interface{}) error {
	if r.next != nil {
		return errors.New("request with non nil next closure does not support CopyTo")
	}
	if r.tx != nil {
		return errors.New("request with non nil tx closure does not support CopyTo")
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

	// If queryToStructsDest or queryRowDest exists, they must be provided by the application layer.
	// We must ensure that the memory addresses pointed to by these fields remain unchanged,
	// that is, they cannot be simply assigned.
	// For concrete content, a shallow copy is sufficient.
	if dr.queryToStructsDest != nil {
		if err := copyutils.ShallowCopy(dr.queryToStructsDest, r.queryToStructsDest); err != nil {
			return err
		}
	}
	if len(dr.queryRowDest) != len(r.queryRowDest) {
		return fmt.Errorf(
			"the length of queryRowDest does not match, expect %d, got %d",
			len(r.queryRowDest), len(dr.queryRowDest),
		)
	}
	for i := range r.queryRowDest {
		if err := copyutils.ShallowCopy(dr.queryRowDest[i], r.queryRowDest[i]); err != nil {
			return err
		}
	}

	return nil
}

// Response clickhouse response body
type Response struct {
	Result sql.Result
}

type (
	// NextFunc query makes a request, and the logic executed by each row of data records is required.
	// Wrapped at the bottom of the framework, it can prevent users from missing rows.Close, rows.Err, etc.,
	// and can also prevent the context from being canceled during the scan process.
	// return value error ==nil to continue to the next row, ==ErrBreak to end the loop early !=nil return failure.
	NextFunc func(*sql.Rows) error

	// TxFunc is the processing function after the user initiates the transaction request, and it is required.
	// The transaction call can be simplified, the transaction is automatically rolled back when error != nil,
	// otherwise the transaction is automatically committed.
	TxFunc func(*sql.Tx) error
)

var (
	// ErrBreak early breaks to end the scan loop.
	ErrBreak = errors.New("clickhouse scan rows break")
)

// Exec executes the clickhouse insert delete update command.
func (c *clickhouseCli) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	creq := &Request{
		op:   opExec,
		Exec: query,
		Args: args,
	}
	crsp := &Response{}

	cctx, msg := codec.WithCloneMessage(ctx)
	defer codec.PutBackMessage(msg)
	msg.WithClientRPCName(fmt.Sprintf("/%s/Exec", c.ServiceName))
	msg.WithCalleeServiceName(c.ServiceName)
	msg.WithSerializationType(-1) // Not serialized.
	msg.WithCompressType(0)       // Not compressed.
	msg.WithClientReqHead(creq)
	msg.WithClientRspHead(crsp)

	err := c.Client.Invoke(cctx, creq, crsp, c.opts...)
	if err != nil {
		return nil, err
	}

	return crsp.Result, nil
}

// Query executes the clickhouse select command.
func (c *clickhouseCli) Query(ctx context.Context, next NextFunc, query string, args ...interface{}) error {
	creq := &Request{
		op:    opQuery,
		Query: query,
		Args:  args,
		next:  next,
	}
	crsp := &Response{}

	cctx, msg := codec.WithCloneMessage(ctx)
	defer codec.PutBackMessage(msg)
	msg.WithClientRPCName(fmt.Sprintf("/%s/Query", c.ServiceName))
	msg.WithCalleeServiceName(c.ServiceName)
	msg.WithSerializationType(-1) // Not serialized.
	msg.WithCompressType(0)       // Not compressed.
	msg.WithClientReqHead(creq)
	msg.WithClientRspHead(crsp)

	err := c.Client.Invoke(cctx, creq, crsp, c.opts...)
	if err != nil {
		return err
	}

	return nil
}

// QueryRow executes the clickhouse QueryRow command.
func (c *clickhouseCli) QueryRow(ctx context.Context, dest []interface{}, query string, args ...interface{}) error {
	creq := &Request{
		op:           opQueryRow,
		Query:        query,
		Args:         args,
		queryRowDest: dest,
	}
	crsp := &Response{}

	cctx, msg := codec.WithCloneMessage(ctx)
	defer codec.PutBackMessage(msg)
	msg.WithClientRPCName(fmt.Sprintf("/%s/QueryRow", c.ServiceName))
	msg.WithCalleeServiceName(c.ServiceName)
	msg.WithSerializationType(-1) // Not serialized.
	msg.WithCompressType(0)       // Not compressed.
	msg.WithClientReqHead(creq)
	msg.WithClientRspHead(crsp)

	err := c.Client.Invoke(cctx, creq, crsp, c.opts...)
	if err != nil {
		return err
	}

	return nil
}

// QueryToStructs executes the clickhouse select command,
// scans the result to the dest slice, and supports structure slices.
func (c *clickhouseCli) QueryToStructs(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	creq := &Request{
		op:                 opQueryToStructs,
		Query:              query,
		Args:               args,
		queryToStructsDest: dest,
	}
	crsp := &Response{}

	ctx, msg := codec.WithCloneMessage(ctx)
	defer codec.PutBackMessage(msg)
	msg.WithClientRPCName(fmt.Sprintf("/%s/QueryToStructs", c.ServiceName))
	msg.WithCalleeServiceName(c.ServiceName)
	msg.WithSerializationType(-1) // Not serialized.
	msg.WithCompressType(0)       // Not compressed.
	msg.WithClientReqHead(creq)
	msg.WithClientRspHead(crsp)

	err := c.Client.Invoke(ctx, creq, crsp, c.opts...)
	if err != nil {
		return err
	}

	return nil
}

// Transaction executes the clickhouse transaction,
// fn is a callback function that receives *sql.Tx as a parameter.
// When error != nil is returned in fn, the transaction is automatically rolled back,
// otherwise the transaction is automatically committed.
func (c *clickhouseCli) Transaction(ctx context.Context, fn TxFunc) error {
	creq := &Request{
		op: opTransaction,
		tx: fn,
	}
	crsp := &Response{}

	ctx, msg := codec.WithCloneMessage(ctx)
	defer codec.PutBackMessage(msg)
	msg.WithClientRPCName(fmt.Sprintf("/%s/Transaction", c.ServiceName))
	msg.WithCalleeServiceName(c.ServiceName)
	msg.WithSerializationType(-1) // Not serialized.
	msg.WithCompressType(0)       // Not compressed.
	msg.WithClientReqHead(creq)
	msg.WithClientRspHead(crsp)

	err := c.Client.Invoke(ctx, creq, crsp, c.opts...)
	if err != nil {
		return err
	}

	return nil
}
