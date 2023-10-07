// Package clickhouse encapsulates standard library clickhouse.
package clickhouse

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/jmoiron/sqlx"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/naming/selector"
	"trpc.group/trpc-go/trpc-go/transport"
	dsn "trpc.group/trpc-go/trpc-selector-dsn"
)

func init() {
	transport.RegisterClientTransport("clickhouse", DefaultClientTransport)
	selector.Register("clickhouse+polaris", dsn.NewResolvableSelector("polaris", &URIHostExtractor{}))
	selector.Register("clickhouse", dsn.DefaultSelector)
}

// ClientTransport is client-side clickhouse transport.
type ClientTransport struct {
	opts   *transport.ClientTransportOptions
	dbs    map[string]*sql.DB
	dblock sync.RWMutex
	options
}

// options is client-side transport configuration parameter.
type options struct {
	MaxIdle     int
	MaxOpen     int
	MaxLifetime time.Duration
}

// DefaultClientTransport is a default client clickhouse transport.
var DefaultClientTransport = NewClientTransport()

// NewClientTransport creates a clickhouse transport.
func NewClientTransport(opt ...transport.ClientTransportOption) transport.ClientTransport {
	opts := &transport.ClientTransportOptions{}
	for _, o := range opt {
		o(opts)
	}

	return &ClientTransport{
		opts: opts,
		dbs:  make(map[string]*sql.DB),
		options: options{
			MaxIdle:     10,
			MaxOpen:     10000,
			MaxLifetime: 3 * time.Minute,
		},
	}
}

// RoundTrip sends and receives clickhouse packets,
// returns the clickhouse response and puts it in ctx, there is no need to return rspbuf here.
func (ct *ClientTransport) RoundTrip(
	ctx context.Context, reqBuf []byte, callOpts ...transport.RoundTripOption,
) (rspBuf []byte, err error) {
	msg := codec.Message(ctx)
	defer func() {
		switch e := err.(type) {
		case *clickhouse.Exception:
			err = errs.New(int(e.Code), e.Message)
		case *errs.Error, nil:
		default:
			err = errs.NewFrameError(errs.RetClientNetErr, err.Error())
		}
	}()

	req, ok := msg.ClientReqHead().(*Request)
	if !ok {
		return nil, errs.NewFrameError(
			errs.RetClientEncodeFail,
			"clickhouse client transport: ReqHead should be type of *clickhouse.Request",
		)
	}
	rsp, ok := msg.ClientRspHead().(*Response)
	if !ok {
		return nil, errs.NewFrameError(
			errs.RetClientEncodeFail,
			"clickhouse client transport: RspHead should be type of *clickhouse.Response",
		)
	}

	opts := &transport.RoundTripOptions{}
	for _, o := range callOpts {
		o(opts)
	}
	dsn := opts.Address
	db, err := ct.GetDB(dsn)
	if err != nil {
		return
	}

	err = runCommand(ctx, db, req, rsp)
	withRemoteAddr(msg, dsn)
	withCommonMeta(msg, filledCommonMeta(msg, calleeApp, msg.RemoteAddr().String()))
	return
}

func runCommand(ctx context.Context, db *sql.DB, req *Request, rsp *Response) error {
	switch req.op {
	case opExec:
		var err error
		rsp.Result, err = handleExec(ctx, db, req)
		return err
	case opQuery:
		return handleQuery(ctx, db, req)
	case opQueryRow:
		return handleQueryRow(ctx, db, req)
	case opQueryToStructs:
		return handleQueryToStructs(ctx, db, req)
	case opTransaction:
		return handleTransaction(ctx, db, req)
	default:
		return errs.NewFrameError(errs.RetUnknown, "trpc-clickhouse: undefined op type")
	}
}

func handleExec(ctx context.Context, db *sql.DB, req *Request) (sql.Result, error) {
	return db.ExecContext(ctx, req.Exec, req.Args...)
}

func handleQuery(ctx context.Context, db *sql.DB, req *Request) error {
	rows, err := db.QueryContext(ctx, req.Query, req.Args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		err = req.next(rows)
		if err == ErrBreak {
			break
		}
		if err != nil {
			return err
		}
	}
	return rows.Err()
}

func handleQueryRow(ctx context.Context, db *sql.DB, req *Request) error {
	return db.QueryRowContext(ctx, req.Query, req.Args...).Scan(req.queryRowDest...)
}

func handleQueryToStructs(ctx context.Context, db *sql.DB, req *Request) error {
	rows, err := db.QueryContext(ctx, req.Query, req.Args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	return sqlx.StructScan(rows, req.queryToStructsDest)
}

func handleTransaction(ctx context.Context, db *sql.DB, req *Request) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := req.tx(tx); err != nil {
		if e := tx.Rollback(); e != nil {
			return fmt.Errorf("tx err: %v, rollback err: %w", err, e)
		}
		return err
	}
	return tx.Commit()
}

// GetDB gets clickhouse link.
func (ct *ClientTransport) GetDB(dsn string) (*sql.DB, error) {
	dsn = toV2DSN(dsn)
	ct.dblock.RLock()
	db, ok := ct.dbs[dsn]
	ct.dblock.RUnlock()

	if ok {
		return db, nil
	}

	ct.dblock.Lock()
	defer ct.dblock.Unlock()

	db, ok = ct.dbs[dsn]
	if ok {
		return db, nil
	}
	opts, realDSN := parseOptionsFromDSN(dsn)
	db, err := sql.Open("clickhouse", "tcp://"+realDSN)
	if err != nil {
		return nil, err
	}
	ct.setOptions(opts)
	if ct.MaxIdle > 0 {
		db.SetMaxIdleConns(ct.MaxIdle)
	}
	if ct.MaxOpen > 0 {
		db.SetMaxOpenConns(ct.MaxOpen)
	}
	if ct.MaxLifetime > 0 {
		db.SetConnMaxLifetime(ct.MaxLifetime)
	}

	ct.dbs[dsn] = db
	return db, nil
}

// setOptions sets configuration parameters.
func (ct *ClientTransport) setOptions(opt *options) {
	if opt == nil {
		return
	}
	if opt.MaxIdle > 0 {
		ct.MaxIdle = opt.MaxIdle
	}
	if opt.MaxOpen > 0 {
		ct.MaxOpen = opt.MaxOpen
	}
	if opt.MaxLifetime > 0 {
		ct.MaxLifetime = opt.MaxLifetime
	}
}

// withRemoteAddr sets access address.
func withRemoteAddr(msg codec.Msg, dsn string) {
	// If RemoteAddr has been set, return directly.
	if validNetAddr(msg.RemoteAddr()) {
		return
	}
	begin, length, _ := (&URIHostExtractor{}).Extract(dsn)
	// Take the first address as remoteAddr.
	firstHost := dsn[begin : begin+length]
	if i := strings.IndexByte(firstHost, ','); i > -1 {
		firstHost = firstHost[0:i]
	}
	addr, _ := net.ResolveTCPAddr("tcp", firstHost)
	msg.WithRemoteAddr(addr)
}

// validNetAddr checks whether net.Addr is valid.
func validNetAddr(addr net.Addr) bool {
	switch v := addr.(type) {
	case nil:
		return false
	case *net.TCPAddr:
		return v != nil
	case *net.UDPAddr:
		return v != nil
	case *net.UnixAddr:
		return v != nil
	case *net.IPAddr:
		return v != nil
	case *net.IPNet:
		return v != nil
	default:
		// Guaranteed logic, basically impossible to enter.
		return !reflect.ValueOf(v).IsNil()
	}
}

const calleeApp = "[clickhouse]" // clickhouse called app.

// filledCommonMeta fills the called app and called server in codec.
// CommonMeta as the actual clickhouse endpoint to facilitate fault location.
func filledCommonMeta(msg codec.Msg, calleeApp, calleeServer string) codec.CommonMeta {
	meta := msg.CommonMeta()
	if meta == nil {
		meta = codec.CommonMeta{}
	}
	const (
		appKey    = "overrideCalleeApp"
		serverKey = "overrideCalleeServer"
	)
	meta[appKey] = calleeApp
	meta[serverKey] = calleeServer
	return meta
}

// withCommonMeta sets CommonMeta.
func withCommonMeta(msg codec.Msg, meta codec.CommonMeta) {
	msg.WithCommonMeta(meta)
}
