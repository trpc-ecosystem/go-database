package gorm

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/go-sql-driver/mysql"
	"golang.org/x/sync/singleflight"
	dsn "trpc.group/trpc-go/trpc-selector-dsn"

	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/naming/selector"
	"trpc.group/trpc-go/trpc-go/transport"
)

func init() {
	transport.RegisterClientTransport("gorm", defaultClientTransport)
	selector.Register("gorm+polaris", dsn.NewResolvableSelectorWithOpts("polaris",
		dsn.WithEnableParseAddr(true), dsn.WithExtractor(&dsn.URIHostExtractor{})))
}

// PoolConfig is the configuration of the database connection pool.
type PoolConfig struct {
	MaxIdle     int
	MaxOpen     int
	MaxLifetime time.Duration
	DriverName  string
}

// ClientTransport is a struct that implements the trpc ClientTransport interface
// and is responsible for sending requests.
type ClientTransport struct {
	opener            func(driverName, dataSourceName string) (*sql.DB, error)
	opts              *transport.ClientTransportOptions
	SQLDB             map[string]*sql.DB
	SQLDBLock         sync.RWMutex
	sfg               singleflight.Group
	DefaultPoolConfig PoolConfig
	PoolConfigs       map[string]PoolConfig
}

// defaultClientTransport is the default client transport.
// Can get the ClientTransport through plugin.GetClientTransport if necessary.
var defaultClientTransport = NewClientTransport()

// NewClientTransport creates transport.
func NewClientTransport(opt ...transport.ClientTransportOption) *ClientTransport {
	opts := &transport.ClientTransportOptions{}
	// Write the incoming func option into the opts field.
	for _, o := range opt {
		o(opts)
	}
	return &ClientTransport{
		opener: sql.Open,
		opts:   opts,
		SQLDB:  make(map[string]*sql.DB),
		DefaultPoolConfig: PoolConfig{
			MaxIdle:     defaultMaxIdle,
			MaxOpen:     defaultMaxOpen,
			MaxLifetime: defaultMaxLifetime,
		},
	}
}

// RoundTrip sends a SQL request and handles the SQL response.
func (ct *ClientTransport) RoundTrip(ctx context.Context, reqBuf []byte,
	callOpts ...transport.RoundTripOption) (rspBuf []byte, err error) {
	msg := codec.Message(ctx)
	defer func() {
		// Currently only supports MySQL,
		// adding other types of databases requires adding the corresponding error types.
		switch sqlErr := err.(type) {
		case *mysql.MySQLError:
			err = errs.Wrap(sqlErr, int(sqlErr.Number), sqlErr.Message)
		case *clickhouse.Exception:
			err = errs.Wrap(sqlErr, int(sqlErr.Code), sqlErr.Message)
		case *errs.Error, nil:
		default:
			err = errs.Wrap(err, errs.RetUnknown, err.Error())
		}
	}()

	req, ok := msg.ClientReqHead().(*Request)
	if !ok {
		return nil, errs.NewFrameError(errs.RetClientEncodeFail,
			"sql client transport: ReqHead should be type of *gormCli.Request")
	}
	rsp, ok := msg.ClientRspHead().(*Response)
	if !ok {
		return nil, errs.NewFrameError(errs.RetClientEncodeFail,
			"sql client transport: RspHead should be type of *gormCli.Response")
	}

	// If a transaction has already been started, execute the command directly using Tx.
	if req.Tx != nil {
		err = runTxCommand(ctx, req.Tx, req, rsp)
		return
	}

	sqlOpts := &transport.RoundTripOptions{}
	for _, o := range callOpts {
		o(sqlOpts)
	}
	withCommonMetaCalleeAppServer(msg, ct.getDBType(msg.CalleeServiceName()), mask(sqlOpts.Address))

	err = ct.getDBAndRunCommand(ctx, sqlOpts.Address, req, rsp)
	if err == nil {
		return
	}

	// DB might be closed by gorm.io/gorm. If the failure is due to the closure of DB, recreate DB and retry.
	// internal repository issues/525
	if isDBClosedErr(err) {
		ct.deleteDB(sqlOpts.Address)
		err = ct.getDBAndRunCommand(ctx, sqlOpts.Address, req, rsp)
	}
	return
}

func (ct *ClientTransport) getDBAndRunCommand(ctx context.Context, address string, req *Request, rsp *Response) error {
	// If a new type of database is added, the database type needs to be passed in here.
	// The CalleeServcieName can be read from sqlOpts.Msg,
	// which is the service name in the trpc framework configuration.
	// The database type can be obtained based on the second segment of the service name.
	msg := codec.Message(ctx)
	db, err := ct.GetDB(msg.CalleeServiceName(), address)
	if err != nil {
		return wrapGetDbError(err, address)
	}
	return runCommand(ctx, db, req, rsp)
}

// withCommonMetaCalleeAppServer Populate the called app and the called server with real DB instances
// in CommonMeta for fault location.
func withCommonMetaCalleeAppServer(msg codec.Msg, calleeApp, calleeServer string) {
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
	msg.WithCommonMeta(meta)
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

func runTxCommand(ctx context.Context, tx *sql.Tx, req *Request, rsp *Response) error {
	switch req.Op {
	case OpPrepareContext:
		stmt, err := tx.PrepareContext(ctx, req.Query)
		if err != nil {
			return err
		}
		rsp.Stmt = stmt
	case OpExecContext:
		result, err := tx.ExecContext(ctx, req.Query, req.Args...)
		if err != nil {
			return err
		}
		rsp.Result = result
	case OpQueryContext:
		rows, err := tx.QueryContext(ctx, req.Query, req.Args...)
		if err != nil {
			return err
		}
		rsp.Rows = rows
	case OpQueryRowContext:
		row := tx.QueryRowContext(ctx, req.Query, req.Args...)
		rsp.Row = row
	case OpCommit:
		err := tx.Commit()
		if err != nil {
			return err
		}
	case OpRollback:
		err := tx.Rollback()
		if err != nil {
			return err
		}
	default:
		return errs.NewFrameError(errs.RetServerNoFunc, "Illegal Method")
	}
	return nil
}

// runCommand restores the original request and executes it.
func runCommand(ctx context.Context, db *sql.DB, req *Request, rsp *Response) error {
	switch req.Op {
	case OpPrepareContext:
		stmt, err := db.PrepareContext(ctx, req.Query)
		if err != nil {
			return err
		}
		rsp.Stmt = stmt
	case OpExecContext:
		result, err := db.ExecContext(ctx, req.Query, req.Args...)
		if err != nil {
			return err
		}
		rsp.Result = result
	case OpQueryContext:
		rows, err := db.QueryContext(ctx, req.Query, req.Args...)
		if err != nil {
			return err
		}
		rsp.Rows = rows
	case OpQueryRowContext:
		// The definition of sql.Row contains an error, so this operation does not handle err,
		// but passes it to the upstream in the result.
		row := db.QueryRowContext(ctx, req.Query, req.Args...)
		if row != nil && isDBClosedErr(row.Err()) {
			return row.Err()
		}
		rsp.Row = row
	case OpBeginTx:
		// Default level is sql.LevelDefault.
		tx, err := db.BeginTx(ctx, req.TxOptions)
		if err != nil {
			return err
		}
		rsp.Tx = tx
	case OpPing:
		return db.Ping()
	case OpGetDB:
		rsp.DB = db
		return nil
	default:
		return errs.NewFrameError(errs.RetServerNoFunc, "Illegal Method")
	}
	return nil
}

var errDBAreadyExist = errors.New("the db is already exist")

// GetDB retrieves the database connection, currently supports mysql/clickhouse,
// can be extended for other types of databases.
func (ct *ClientTransport) GetDB(serviceName, dsn string) (*sql.DB, error) {
	// Singleton pattern with lock.
	ct.SQLDBLock.RLock()
	db, ok := ct.SQLDB[dsn]
	ct.SQLDBLock.RUnlock()
	if ok {
		return db, nil
	}

	iDB, err, _ := ct.sfg.Do(dsn, func() (interface{}, error) {
		ct.SQLDBLock.RLock()
		db, ok = ct.SQLDB[dsn]
		ct.SQLDBLock.RUnlock()
		if ok {
			return db, nil
		}
		// Pass the database type as part of the serviceName, such as trpc.mysql.xxx.xxx/trpc.clickhouse.xxx.xxx,
		// and use different drivers internally based on different types.
		db, err := ct.initDB(serviceName, dsn)
		if err != nil {
			return nil, wrapperSQLOpenError(err)
		}
		poolConfig, ok := ct.PoolConfigs[serviceName]
		if !ok {
			poolConfig = ct.DefaultPoolConfig
		}
		db.SetMaxIdleConns(poolConfig.MaxIdle)
		db.SetMaxOpenConns(poolConfig.MaxOpen)
		db.SetConnMaxLifetime(poolConfig.MaxLifetime)
		ct.SQLDBLock.Lock()
		ct.SQLDB[dsn] = db
		ct.SQLDBLock.Unlock()
		return db, nil
	})
	if err == nil || errors.Is(err, errDBAreadyExist) {
		return iDB.(*sql.DB), nil
	}
	return nil, err
}

func (ct *ClientTransport) deleteDB(dsn string) {
	ct.SQLDBLock.Lock()
	delete(ct.SQLDB, dsn)
	ct.SQLDBLock.Unlock()
}

func (ct *ClientTransport) getDBType(s string) string {
	splitServiceName := strings.Split(s, ".")
	// Compatibility logic, keeps MySQL as the default.
	// internal repository issues/235
	if len(splitServiceName) < 2 {
		return "mysql"
	}
	return splitServiceName[1]
}

func (ct *ClientTransport) initDB(s, dsn string) (*sql.DB, error) {
	// The format of the serviceName: trpc.${db type}.xxx.xxx.
	// When the serviceName is not standard, mysql is used by default.
	// The code is borrowed from the getAppServerService(string) (string, string, string,bool) method
	// in trpc.group/trpc-go/trpc-go/codec/message_impl.go.
	if conf, ok := ct.PoolConfigs[s]; ok && conf.DriverName != "" {
		return ct.opener(conf.DriverName, dsn)
	}
	if ct.DefaultPoolConfig.DriverName != "" {
		return ct.opener(ct.DefaultPoolConfig.DriverName, dsn)
	}
	switch ct.getDBType(s) {
	case "clickhouse":
		return ct.opener("clickhouse", dsn)
	case "postgres":
		// The driver registered for postgres is pgx by default.
		return ct.opener("pgx", dsn)
	default:
		return ct.opener("mysql", dsn)
	}
}

func wrapperSQLOpenError(err error) error {
	errStr := err.Error()
	if strings.HasPrefix(errStr, "sql: unknown driver") {
		return fmt.Errorf("error: %s, should register before open driver,"+
			"please refer: https://pkg.go.dev/database/sql#Register", errStr)
	}
	return err
}

func wrapGetDbError(err error, address string) error {
	return fmt.Errorf(
		`err: %w, 
current masked sqlOpts.Address: %s,
if it is not what you want, it is possible that your client config is not loaded correctly`,
		err, mask(address)) // Mask out the credentials.
}

func isDBClosedErr(err error) bool {
	return err != nil && err.Error() == "sql: database is closed"
}
