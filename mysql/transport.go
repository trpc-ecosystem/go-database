package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"

	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/naming/selector"
	"trpc.group/trpc-go/trpc-go/transport"
	dsn "trpc.group/trpc-go/trpc-selector-dsn"
)

func init() {
	transport.RegisterClientTransport("mysql", DefaultClientTransport)
	selector.Register("mysql+polaris", dsn.NewResolvableSelector("polaris", &dsn.URIHostExtractor{}))
}

const defaultDriverName = "mysql"

// ClientTransport client side mysql transport.
type ClientTransport struct {
	opts   *transport.ClientTransportOptions
	dbs    map[string]*sql.DB
	dblock sync.RWMutex

	MaxIdle     int
	MaxOpen     int
	MaxLifetime time.Duration
	DriverName  string
}

// DefaultClientTransport default client mysql transport.
var DefaultClientTransport = NewClientTransport()

// NewClientTransport create mysql transport.
func NewClientTransport(opt ...transport.ClientTransportOption) transport.ClientTransport {
	opts := &transport.ClientTransportOptions{}
	for _, o := range opt {
		o(opts)
	}
	return &ClientTransport{
		opts:        opts,
		dbs:         make(map[string]*sql.DB),
		MaxIdle:     10,
		MaxOpen:     10000,
		MaxLifetime: 3 * time.Minute,
		DriverName:  defaultDriverName,
	}
}

// RoundTrip send and receive mysql packets, return mysql response to ctx inside, no need to return rspbuf here.
func (ct *ClientTransport) RoundTrip(ctx context.Context, reqBuf []byte,
	callOpts ...transport.RoundTripOption) (response []byte, err error) {
	msg := codec.Message(ctx)
	defer func() {
		err = generateTRPCError(err)
	}()

	req, ok := msg.ClientReqHead().(*Request)
	if !ok {
		return nil, errs.NewFrameError(errs.RetClientEncodeFail,
			"mysql client transport: ReqHead should be type of *mysql.Request")
	}
	rsp, ok := msg.ClientRspHead().(*Response)
	if !ok {
		return nil, errs.NewFrameError(errs.RetClientEncodeFail,
			"mysql client transport: RspHead should be type of *mysql.Response")
	}

	opts := &transport.RoundTripOptions{}
	for _, o := range callOpts {
		o(opts)
	}
	dsn := opts.Address
	db, err := ct.GetDB(dsn)
	if err != nil {
		err = fmt.Errorf(
			`err: %w,
current dsn(masked from opts.Address): %s,
if it is not what you want, it is possible that your client config is not loaded correctly`,
			err, mask(dsn)) // Mask out the credentials.
		return
	}

	err = ct.runCommand(ctx, db, req, rsp)
	postProcessing(msg, dsn)
	return
}

func generateTRPCError(err error) error {
	if _, ok := err.(*errs.Error); ok || err == nil {
		return err
	}

	switch err {
	case sql.ErrNoRows:
		return ErrNoRows
	case context.Canceled:
		return errs.NewFrameError(errs.RetClientCanceled, err.Error())
	case context.DeadlineExceeded:
		return errs.NewFrameError(errs.RetClientTimeout, err.Error())
	default:
	}

	var mySQLError *mysql.MySQLError
	if errors.As(err, &mySQLError) {
		return errs.New(int(mySQLError.Number), mySQLError.Message)
	}

	return errs.NewFrameError(errs.RetClientNetErr, err.Error())
}

// mask masks the given string with '*' characters in the middle to prevent security vulnerabilities.
func mask(s string) string {
	const (
		preLen = 6
		sufLen = 6
		stars  = "***...***"
	)
	n := len(s)
	if n < preLen+sufLen {
		return s
	}
	return s[:preLen] + stars + s[n-sufLen:n]
}

// postProcessing Processing Fix msg.
func postProcessing(msg codec.Msg, dsn string) {
	cfg, err := getCfg(dsn)
	if err != nil {
		// Non-critical logic can be returned directly.
		log.Warnf("getCfg(%s) err: %v", dsn, err)
		return
	}
	fixRemoteIP(msg, cfg)
	withCommonMetaCalleeAppServer(msg, cfg)
}

// getCfg Parse out config from dsn.
func getCfg(dsn string) (*mysql.Config, error) {
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil || cfg.Net != "tcp" {
		return nil, fmt.Errorf("invalid dsn: %s", dsn)
	}
	return cfg, nil
}

// fixRemoteIP Fix the transferred IP.
func fixRemoteIP(msg codec.Msg, cfg *mysql.Config) {
	// If RemoteAddr has already been set, return directly to.
	if msg.RemoteAddr() != nil {
		return
	}
	ipPort, err := cfgToEndpoint(cfg, modeIP)
	if err != nil {
		return
	}
	// cfgToEndpoint() must return a legal ipPort if err is nil.
	addr, _ := net.ResolveTCPAddr("tcp", ipPort)
	msg.WithRemoteAddr(addr)
}

type mode string

const (
	defaultMySQLCalleeApp      = "[mysql]"
	modeIP                mode = "ip"   // Prefer IP mode
	modeHost              mode = "host" // Prefer hostname mode
)

// withCommonMetaCalleeAppServer Populate the called app and the called server with real DB instances
// in CommonMeta for fault location.
func withCommonMetaCalleeAppServer(msg codec.Msg, cfg *mysql.Config) {
	endPoint, err := cfgToEndpoint(cfg, modeHost)
	// Because it is a bypass post-processing logic, parsing cfg errors can be returned directly,
	// explicitly ignoring errors.
	if err != nil {
		log.Warnf("cfgToEndpoint() err: %v", err)
		return
	}
	meta := msg.CommonMeta()
	if meta == nil {
		meta = codec.CommonMeta{}
	}
	const (
		appKey    = "overrideCalleeApp"
		serverKey = "overrideCalleeServer"
	)
	meta[appKey] = defaultMySQLCalleeApp
	meta[serverKey] = endPoint
	msg.WithCommonMeta(meta)
}

// cfgToEndpoint cfg string to real end point information, optionally preferring host or ip mode.
func cfgToEndpoint(cfg *mysql.Config, m mode) (string, error) {
	if cfg == nil {
		return "", errors.New("nil *mysql.Config")
	}
	switch m {
	case modeHost:
		// The host mode does not try to resolve the IP and returns example.com:3306 or 1.2.3.4:3306 format directly.
		return cfg.Addr, nil
	case modeIP:
		const validLen = 2
		arr := strings.Split(cfg.Addr, ":")
		if len(arr) != validLen {
			return "", fmt.Errorf("invalid mysql config dsn address: %s", cfg.Addr)
		}
		host, port := arr[0], arr[1]
		if ip := net.ParseIP(host); ip != nil {
			// Already a legitimate IP Direct return.
			return host + ":" + port, nil
		}
		if ips, err := net.LookupIP(host); err == nil && len(ips) == 1 {
			return ips[0].String() + ":" + port, nil
		}
		// If the resolution fails or if a domain name corresponds to multiple IPs, the IP is not repaired.
		return "", fmt.Errorf("cannot resolve domain or resolved too many IP: %s", host)
	default:
		return "", fmt.Errorf("unsupported mode: %s", m)
	}
}

func (ct *ClientTransport) runCommand(ctx context.Context, db *sql.DB, req *Request, rsp *Response) error {
	var err error
	switch req.op {
	case opTransaction:
		return handleTransaction(ctx, db, req)
	case opTransactionx:
		return ct.handleTransactionx(ctx, db, req)
	case opQuery:
		return handleQuery(ctx, db, req)
	case opQueryRow:
		return handleQueryRow(ctx, db, req)
	case opQueryToStruct:
		return ct.handleQueryToStruct(ctx, db, req)
	case opQueryToStructs:
		return handleQueryToStructs(ctx, db, req)
	case opExec:
		rsp.Result, err = handleExec(ctx, db, req)
		return err
	case opNamedExec:
		rsp.Result, err = ct.handleNamedExec(ctx, db, req)
		return err
	case opSelect:
		return ct.handleSelect(ctx, db, req)
	case opGet:
		return ct.handleGet(ctx, db, req)
	case opNamedQuery:
		rsp.SqlxRows, err = ct.handleNamedQuery(ctx, db, req)
		return err
	default:
		return errs.NewFrameError(errs.RetUnknown, "trpc-mysql: undefined op type")
	}
}

func handleTransaction(ctx context.Context, db *sql.DB, req *Request) (err error) {
	var tx *sql.Tx
	if tx, err = db.BeginTx(ctx, req.txOpts); err != nil {
		return
	}
	if err := req.tx(tx); err != nil {
		if e := tx.Rollback(); e != nil {
			return e
		}
		return err
	}
	if err := tx.Commit(); err != nil {
		if e := tx.Rollback(); e != nil {
			return e
		}
		return err
	}
	return
}

func (ct *ClientTransport) newSqlxDB(db *sql.DB, unsafe bool) *sqlx.DB {
	sqlxdb := sqlx.NewDb(db, ct.DriverName)
	if unsafe {
		return sqlxdb.Unsafe()
	}
	return sqlxdb
}

func (ct *ClientTransport) handleTransactionx(ctx context.Context, db *sql.DB, req *Request) error {
	tx, err := ct.newSqlxDB(db, req.unsafe).BeginTxx(ctx, req.txOpts)
	if err != nil {
		return fmt.Errorf("begin transaction error: %w", err)
	}
	if err := req.txx(tx); err != nil {
		if e := tx.Rollback(); e != nil {
			return fmt.Errorf("transaction error: %s, and rollback error: %w", err.Error(), e)
		}
		return err
	}
	if err := tx.Commit(); err != nil {
		if e := tx.Rollback(); e != nil {
			return fmt.Errorf("commit transaction error: %s, and rollback error: %w", err.Error(), e)
		}
		return fmt.Errorf("commit transaction error: %w", err)
	}
	return nil
}

func handleExec(ctx context.Context, db *sql.DB, req *Request) (sql.Result, error) {
	return db.ExecContext(ctx, req.Exec, req.Args...)
}

func (ct *ClientTransport) handleNamedExec(ctx context.Context, db *sql.DB, req *Request) (sql.Result, error) {
	sqlxdb := ct.newSqlxDB(db, req.unsafe)
	if len(req.Args) == 0 || req.Args[0] == nil { // Compatible with parameterless, direct full SQL format scenarios.
		return sqlxdb.NamedExecContext(ctx, req.Exec, Args{{"1": 1}})
	}
	return sqlxdb.NamedExecContext(ctx, req.Exec, req.Args[0])
}

func handleQueryRow(ctx context.Context, db *sql.DB, req *Request) (err error) {
	row := db.QueryRowContext(ctx, req.Query, req.Args...)
	return row.Scan(req.QueryRowDest...)
}

func handleQuery(ctx context.Context, db *sql.DB, req *Request) (err error) {
	var rows *sql.Rows
	rows, err = db.QueryContext(ctx, req.Query, req.Args...)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		err = req.next(rows)
		if err == ErrBreak {
			break
		}
		if err != nil {
			return
		}
	}
	err = rows.Err()
	return
}

func (ct *ClientTransport) handleQueryToStruct(ctx context.Context, db *sql.DB, req *Request) error {
	return ct.newSqlxDB(db, req.unsafe).QueryRowxContext(ctx, req.Query, req.Args...).StructScan(req.QueryToStructDest)
}

func handleQueryToStructs(ctx context.Context, db *sql.DB, req *Request) (err error) {
	rows, err := db.QueryContext(ctx, req.Query, req.Args...)
	if err != nil {
		return
	}
	defer rows.Close()
	err = sqlx.StructScan(rows, req.QueryToStructsDest)
	return
}

func (ct *ClientTransport) handleSelect(ctx context.Context, db *sql.DB, req *Request) (err error) {
	return ct.newSqlxDB(db, req.unsafe).SelectContext(ctx, req.QueryToDest, req.Query, req.Args...)
}

func (ct *ClientTransport) handleGet(ctx context.Context, db *sql.DB, req *Request) error {
	return ct.newSqlxDB(db, req.unsafe).GetContext(ctx, req.QueryToDest, req.Query, req.Args...)
}

func (ct *ClientTransport) handleNamedQuery(ctx context.Context, db *sql.DB, req *Request) (*sqlx.Rows, error) {
	sqlxdb := ct.newSqlxDB(db, req.unsafe)
	if len(req.Args) == 0 || req.Args[0] == nil { // Compatible with parameterless, direct full SQL format scenarios.
		return sqlxdb.QueryxContext(ctx, req.Query)
	}
	return sqlxdb.NamedQueryContext(ctx, req.Query, req.Args[0])
}

// GetDB Get mysql link.
func (ct *ClientTransport) GetDB(dsn string) (*sql.DB, error) {
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

	db, err := sql.Open(ct.DriverName, dsn)
	if err != nil {
		return nil, err
	}

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
