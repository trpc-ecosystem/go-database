package mysql

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/transport"
)

var dbDsn = "user:password@tcp(localhost:5555)/dbname"

// TestUnit_NewClientTransport_P0 NewClientTransport test case.
func TestUnit_NewClientTransport_P0(t *testing.T) {
	Convey("TestUnit_NewClientTransport_P0", t, func() {
		opt := transport.WithClientUDPRecvSize(128)
		ct := NewClientTransport(opt)
		So(ct, ShouldResemble, &ClientTransport{
			opts:        &transport.ClientTransportOptions{UDPRecvSize: 128},
			dbs:         make(map[string]*sql.DB),
			dblock:      sync.RWMutex{},
			MaxIdle:     10,
			MaxOpen:     10000,
			MaxLifetime: 3 * time.Minute,
			DriverName:  defaultDriverName,
		})
	})
}

func mockTxFunc(*sql.Tx) error {
	return nil
}

// TestUnit_ClientTransport_GetDB_P0 ClientTransport.GetDB test case.
func TestUnit_ClientTransport_GetDB_P0(t *testing.T) {
	Convey("TestUnit_ClientTransport_GetDB_P0", t, func() {
		ct := new(ClientTransport)
		ct.dbs = make(map[string]*sql.DB)
		ct.MaxIdle = 10
		ct.MaxLifetime = 3 * time.Minute
		ct.MaxOpen = 1000
		ct.DriverName = defaultDriverName
		Convey("DSN Already Exists", func() {
			ct.dbs[dbDsn] = new(sql.DB)
			db, err := ct.GetDB(dbDsn)
			So(db, ShouldNotBeNil)
			So(err, ShouldBeNil)
		})
		Convey("Open Fail", func() {
			openMock := gomonkey.ApplyFunc(sql.Open, func(driverName string, dataSourceName string) (*sql.DB, error) {
				return nil, fmt.Errorf("fake error")
			})
			defer openMock.Reset()
			db, err := ct.GetDB(dbDsn)
			So(db, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldEqual, "fake error")
		})
		Convey("GetDB success", func() {
			openMock := gomonkey.ApplyFunc(sql.Open, func(driverName string, dataSourceName string) (*sql.DB, error) {
				return new(sql.DB), nil
			})
			defer openMock.Reset()
			db, err := ct.GetDB(dbDsn)
			So(db, ShouldNotBeNil)
			So(err, ShouldBeNil)
		})
	})
}

// TestUnit_ClientTransport_RT_P0 ClientTransport.RoundTrip test case.
func TestUnit_ClientTransport_RT_P0(t *testing.T) {
	Convey("TestUnit_ClientTransport_RT_P0", t, func() {
		ctx, msg := codec.WithNewMessage(context.Background())
		opts := []transport.RoundTripOption{
			transport.WithDialAddress(dbDsn),
		}
		ct := new(ClientTransport)
		ct.DriverName = defaultDriverName
		Convey("ClientReqHead() not *Request", func() {
			rspBuf, err := ct.RoundTrip(ctx, nil, opts...)
			So(rspBuf, ShouldBeNil)
			So(err, ShouldResemble, errs.NewFrameError(errs.RetClientEncodeFail,
				"mysql client transport: ReqHead should be type of *mysql.Request"))
		})

		request := new(Request)
		request.op = opExec
		msg.WithClientReqHead(request)

		Convey("msg.ClientRspHead() not *Response", func() {
			rspBuf, err := ct.RoundTrip(ctx, nil, opts...)
			So(rspBuf, ShouldBeNil)
			So(err, ShouldResemble, errs.NewFrameError(errs.RetClientEncodeFail,
				"mysql client transport: RspHead should be type of *mysql.Response"))
		})

		response := new(Response)
		msg.WithClientRspHead(response)

		Convey("GetDB Fail", func() {
			openMock := gomonkey.ApplyFunc(sql.Open, func(driverName string, dataSourceName string) (*sql.DB, error) {
				return nil, fmt.Errorf("fake error")
			})
			defer openMock.Reset()
			rspBuf, err := ct.RoundTrip(ctx, nil, opts...)
			So(rspBuf, ShouldBeNil)
			So(err.Error(), ShouldContainSubstring, "fake error")
		})

		// make sure GetDB success.
		ct.dbs = map[string]*sql.DB{
			dbDsn: new(sql.DB),
		}

		Convey("Do Cx", func() {

		})

		roundTripDoQueryTest(t, ct, opts...)
		roundTripDoTransactionTest(t, ct, opts...)
		roundTripGetTest(t, ct, opts...)
		roundTripSelectTest(t, ct, opts...)
		roundTripDoTransactionxTest(t, ct, opts...)

		Convey("Do ExecContext", func() {
			db := new(sql.DB)
			Convey("ExecContext Fail", func() {
				execContextMock := gomonkey.ApplyMethod(reflect.TypeOf(db), "ExecContext",
					func(*sql.DB, context.Context, string, ...interface{}) (sql.Result, error) {
						return nil, &mysql.MySQLError{
							Number:  1,
							Message: "fake error",
						}
					},
				)
				defer execContextMock.Reset()
				rspBuf, err := ct.RoundTrip(ctx, nil, opts...)
				So(rspBuf, ShouldBeNil)
				So(err, ShouldResemble, errs.New(int(1), "fake error"))
			})
			Convey("ExecContext Success", func() {
				execContextMock := gomonkey.ApplyMethod(reflect.TypeOf(db), "ExecContext",
					func(*sql.DB, context.Context, string, ...interface{}) (sql.Result, error) {
						return driver.RowsAffected(1), nil
					},
				)
				defer execContextMock.Reset()

				rspBuf, err := ct.RoundTrip(ctx, nil, opts...)
				So(rspBuf, ShouldBeNil)
				So(err, ShouldBeNil)
				So(response.Result, ShouldResemble, driver.RowsAffected(1))
			})
		})
	})
	t.Run("context is Canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		ctx, msg := codec.WithNewMessage(ctx)
		msg.WithClientReqHead(&Request{op: opExec})
		msg.WithClientRspHead(&Response{})
		ct := &ClientTransport{
			DriverName: defaultDriverName,
			dbs: map[string]*sql.DB{
				dbDsn: new(sql.DB),
			},
		}

		rspBuf, err := ct.RoundTrip(ctx, make([]byte, 0), transport.WithDialAddress(dbDsn))

		require.Nil(t, rspBuf)
		require.EqualError(t, err, errs.NewFrameError(errs.RetClientCanceled, context.Canceled.Error()).Error())
	})
	t.Run("context is DeadlineExceeded", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
		defer cancel()
		ctx, msg := codec.WithNewMessage(ctx)
		msg.WithClientReqHead(&Request{op: opExec})
		msg.WithClientRspHead(&Response{})
		ct := &ClientTransport{
			DriverName: defaultDriverName,
			dbs: map[string]*sql.DB{
				dbDsn: new(sql.DB),
			},
		}

		rspBuf, err := ct.RoundTrip(ctx, make([]byte, 0), transport.WithDialAddress(dbDsn))

		require.Nil(t, rspBuf)
		require.EqualError(t, err, errs.NewFrameError(errs.RetClientTimeout, context.DeadlineExceeded.Error()).Error())
	})
}

func roundTripDoQueryTest(t *testing.T, ct *ClientTransport, opts ...transport.RoundTripOption) {
	ctx, msg := codec.WithNewMessage(context.Background())
	request := new(Request)
	msg.WithClientReqHead(request)

	response := new(Response)
	msg.WithClientRspHead(response)
	ct.dbs = map[string]*sql.DB{
		dbDsn: new(sql.DB),
	}

	Convey("Do Query", func() {
		request.Query = "select * from table limit 1"
		request.op = opQuery
		db := new(sql.DB)
		Convey("QueryContext Fail", func() {
			queryContextMock := gomonkey.ApplyMethod(reflect.TypeOf(db), "QueryContext",
				func(*sql.DB, context.Context, string, ...interface{}) (*sql.Rows, error) {
					return nil, fmt.Errorf("fake error")
				},
			)
			defer queryContextMock.Reset()
			rspBuf, err := ct.RoundTrip(ctx, nil, opts...)
			So(rspBuf, ShouldBeNil)
			So(err, ShouldResemble, errs.NewFrameError(errs.RetClientNetErr, "fake error"))
		})
		rows := new(sql.Rows)
		rowsMock := gomonkey.ApplyMethod(reflect.TypeOf(rows), "Close",
			func() error {
				return nil
			},
		)
		defer rowsMock.Reset()

		Convey("QueryContext success", func() {
			queryContextMock := gomonkey.ApplyMethod(reflect.TypeOf(db), "QueryContext",
				func(*sql.DB, context.Context, string, ...interface{}) (*sql.Rows, error) {
					return rows, nil
				},
			)
			defer queryContextMock.Reset()

			request.next = fakeNextFunc

			rowsMock := gomonkey.ApplyMethod(reflect.TypeOf(rows), "Next",
				func() bool {
					return true
				},
			)
			defer rowsMock.Reset()
			rspBuf, err := ct.RoundTrip(ctx, nil, opts...)
			So(rspBuf, ShouldBeNil)
			So(err, ShouldNotBeNil)
		})
	})

	Convey(" Do Query Row", func() {
		request.Query = "select * from table"
		request.op = opQueryRow
		db := new(sql.DB)

		Convey("QueryRowContext failed", func() {
			row := new(sql.Row)
			queryContextMock := gomonkey.ApplyMethod(reflect.TypeOf(db), "QueryRowContext",
				func() *sql.Row {
					return row
				},
			)
			defer queryContextMock.Reset()

			rowMock := gomonkey.ApplyMethod(reflect.TypeOf(row), "Scan",
				func() error {
					return sql.ErrNoRows
				},
			)
			defer rowMock.Reset()

			rspBuf, err := ct.RoundTrip(ctx, nil, opts...)
			So(rspBuf, ShouldBeNil)
			So(err, ShouldEqual, ErrNoRows)
		})
	})

	Convey("Do QueryToStruct", func() {
		request.Query = "select * from table limit 1"
		request.op = opQueryToStruct
		db := new(sqlx.DB)

		row := new(sqlx.Row)
		ctrl := gomonkey.ApplyMethod(reflect.TypeOf(db), "QueryRowxContext",
			func() *sqlx.Row {
				return row
			},
		)
		defer ctrl.Reset()

		Convey("QueryToStruct failed", func() {
			rowMock := gomonkey.ApplyMethod(reflect.TypeOf(row), "StructScan",
				func() error {
					return sql.ErrConnDone
				},
			)
			defer rowMock.Reset()

			rspBuf, err := ct.RoundTrip(ctx, nil, opts...)
			So(rspBuf, ShouldBeNil)
			So(err, ShouldNotBeNil)
		})

		Convey("QueryToStruct success", func() {
			rowMock := gomonkey.ApplyMethod(reflect.TypeOf(row), "StructScan",
				func() error {
					return nil
				},
			)
			defer rowMock.Reset()
			_, err := ct.RoundTrip(ctx, nil, opts...)
			So(err, ShouldBeNil)
		})
	})

	Convey(" Do Invalid req.Op", func() {
		request.Query = "select * from table"
		request.op = 0

		rspBuf, err := ct.RoundTrip(ctx, nil, opts...)
		So(rspBuf, ShouldBeNil)
		So(err, ShouldNotBeNil)
	})
}

func roundTripGetTest(t *testing.T, ct *ClientTransport, opts ...transport.RoundTripOption) {
	ctx, msg := codec.WithNewMessage(context.Background())
	request := new(Request)
	msg.WithClientReqHead(request)

	response := new(Response)
	msg.WithClientRspHead(response)

	Convey("Do Get", func() {
		request.op = opGet
		request.Query = "SELECT * FROM user limit 1"
		db := new(sqlx.DB)
		gomonkey.ApplyFuncReturn(sqlx.NewDb, db)
		Convey("GetContext Fail With Err ", func() {
			gomonkey.ApplyMethodReturn(db, "GetContext", sql.ErrNoRows)
			rspBuf, err := ct.RoundTrip(ctx, nil, opts...)
			So(err, ShouldResemble, ErrNoRows)
			So(rspBuf, ShouldBeNil)
		})
		Convey("GetContext Success", func() {
			request.QueryToDest = &User{}
			selectMock := gomonkey.ApplyMethod(reflect.TypeOf(db), "GetContext",
				func(*sqlx.DB, context.Context, interface{}, string, ...interface{}) error {
					request.QueryToDest = &User{ID: 1, Name: "Haha", Age: 18}
					return nil
				},
			)
			defer selectMock.Reset()
			rspBuf, err := ct.RoundTrip(ctx, nil, opts...)
			So(err, ShouldBeNil)
			So(rspBuf, ShouldBeNil)
			So(request.QueryToDest, ShouldResemble, &User{ID: 1, Name: "Haha", Age: 18})
		})
	})
}

func roundTripSelectTest(t *testing.T, ct *ClientTransport, opts ...transport.RoundTripOption) {
	ctx, msg := codec.WithNewMessage(context.Background())
	request := new(Request)
	msg.WithClientReqHead(request)

	response := new(Response)
	msg.WithClientRspHead(response)

	Convey("Do Select", func() {
		request.op = opSelect
		request.Query = "SELECT * FROM user"
		db := new(sqlx.DB)
		gomonkey.ApplyFuncReturn(sqlx.NewDb, db)
		Convey("SelectContext Fail With ErrBadConn ", func() {
			gomonkey.ApplyMethodReturn(db, "SelectContext", fmt.Errorf("fake error"))
			rspBuf, err := ct.RoundTrip(ctx, nil, opts...)
			So(err, ShouldResemble, errs.NewFrameError(errs.RetClientNetErr, "fake error"))
			So(rspBuf, ShouldBeNil)
		})
		Convey("SelectContext Success", func() {
			selectMock := gomonkey.ApplyMethod(reflect.TypeOf(db), "SelectContext",
				func(*sqlx.DB, context.Context, interface{}, string, ...interface{}) error {
					request.QueryToDest = []*User{{1, "Haha", 18}, {2, "Foo", 15}}
					return nil
				},
			)
			defer selectMock.Reset()
			rspBuf, err := ct.RoundTrip(ctx, nil, opts...)
			So(err, ShouldBeNil)
			So(rspBuf, ShouldBeNil)
			So(request.QueryToDest, ShouldResemble,
				[]*User{{1, "Haha", 18}, {2, "Foo", 15}})
		})
	})
}

func roundTripNamedQuery(t *testing.T, ct *ClientTransport, opts ...transport.RoundTripOption) {
	ctx, msg := codec.WithNewMessage(context.Background())
	request := new(Request)
	msg.WithClientReqHead(request)

	response := new(Response)
	msg.WithClientRspHead(response)
	Convey("Do NamedQuery", func() {
		request.op = opNamedQuery
		request.Query = "SELECT * FROM user where age =:age"
		request.Args = []interface{}{User{Age: 18}}
		db := new(sqlx.DB)
		gomonkey.ApplyFuncReturn(sqlx.NewDb, db)
		Convey("NamedQueryContext Fail With Err ", func() {
			gomonkey.ApplyMethodReturn(db, "NamedQueryContext", fmt.Errorf("fake error"))
			rspBuf, err := ct.RoundTrip(ctx, nil, opts...)
			So(err, ShouldResemble, errs.NewFrameError(errs.RetClientNetErr, "fake error"))
			So(rspBuf, ShouldBeNil)
		})
		rows := new(sqlx.Rows)
		gomonkey.ApplyMethodReturn(rows, "Close", nil)
		Convey("NamedQueryContext Success", func() {
			gomonkey.ApplyMethodReturn(rows, "NamedQueryContext", rows, nil)
			rspBuf, err := ct.RoundTrip(ctx, nil, opts...)
			So(rspBuf, ShouldBeNil)
			So(err, ShouldNotBeNil)
		})
	})
}

// roundTripDoTransactionTest Transaction Test
func roundTripDoTransactionTest(t *testing.T, ct *ClientTransport, opts ...transport.RoundTripOption) {
	ctx, msg := codec.WithNewMessage(context.Background())
	request := new(Request)
	msg.WithClientReqHead(request)

	response := new(Response)
	msg.WithClientRspHead(response)
	ct.dbs = map[string]*sql.DB{
		dbDsn: new(sql.DB),
	}

	db := new(sql.DB)
	tx := new(sql.Tx)

	transMock := gomonkey.ApplyMethod(reflect.TypeOf(db), "BeginTx",
		func(*sql.DB, context.Context, *sql.TxOptions) (*sql.Tx, error) {
			return tx, nil
		},
	)
	defer transMock.Reset()

	txMock1 := gomonkey.ApplyMethod(reflect.TypeOf(tx), "Commit",
		func() error {
			return nil
		},
	)
	defer txMock1.Reset()
	txMock2 := gomonkey.ApplyMethod(reflect.TypeOf(tx), "Rollback",
		func() error {
			return nil
		},
	)
	defer txMock2.Reset()

	Convey("Do Transaction", func() {
		request.op = opTransaction
		request.tx = fakeTxFunc

		rspBuf, err := ct.RoundTrip(ctx, nil, opts...)
		So(rspBuf, ShouldBeNil)
		So(err, ShouldBeNil)
	})

	Convey("WrapNoRowsError after Transaction", func() {
		request.op = opTransaction
		request.tx = fakeTxNoRowsFunc

		rspBuf, err := ct.RoundTrip(ctx, nil, opts...)
		So(rspBuf, ShouldBeNil)
		So(err, ShouldEqual, ErrNoRows)
	})
}

// roundTripDoTransactionxTest sqlx transaction test.
func roundTripDoTransactionxTest(t *testing.T, ct *ClientTransport, opts ...transport.RoundTripOption) {
	ctx, msg := codec.WithNewMessage(context.Background())
	request := new(Request)
	msg.WithClientReqHead(request)

	response := new(Response)
	msg.WithClientRspHead(response)
	ct.dbs = map[string]*sql.DB{
		dbDsn: new(sql.DB),
	}

	dbx := new(sqlx.DB)
	txx := new(sqlx.Tx)

	transMock := gomonkey.ApplyMethod(reflect.TypeOf(dbx), "BeginTxx",
		func(*sqlx.DB, context.Context, *sql.TxOptions) (*sqlx.Tx, error) {
			return txx, nil
		},
	)
	defer transMock.Reset()

	tx := new(sql.Tx)
	txMock1 := gomonkey.ApplyMethod(reflect.TypeOf(tx), "Commit",
		func() error {
			return nil
		},
	)
	defer txMock1.Reset()
	txMock2 := gomonkey.ApplyMethod(reflect.TypeOf(tx), "Rollback",
		func() error {
			return nil
		},
	)
	defer txMock2.Reset()

	gomonkey.ApplyFuncReturn(sqlx.NewDb, dbx)
	Convey("Do Transactionx", func() {
		request.op = opTransactionx
		request.txx = fakeTxxFunc

		rspBuf, err := ct.RoundTrip(ctx, nil, opts...)
		So(rspBuf, ShouldBeNil)
		So(err, ShouldBeNil)
	})
	Convey("WrapNoRowsError after Transactionx", func() {
		request.op = opTransactionx
		request.txx = fakeTxxNoRowsFunc

		rspBuf, err := ct.RoundTrip(ctx, nil, opts...)
		So(rspBuf, ShouldBeNil)
		So(err, ShouldEqual, ErrNoRows)
	})
}

// fakeNextFunc fakenext function.
func fakeNextFunc(*sql.Rows) error {
	return sql.ErrNoRows
}

// fakeTxFunc Fake tx function.
func fakeTxFunc(*sql.Tx) error {
	return nil
}

// fakeTxxFunc Fake txx function.
func fakeTxxFunc(*sqlx.Tx) error {
	return nil
}

// fakeTxNoRowsFunc fakes a tx function that returns an error in sql.ErrNoRows.
func fakeTxNoRowsFunc(*sql.Tx) error {
	return sql.ErrNoRows
}

// fakeTxxNoRowsFunc fakes a tx function that will return sql.ErrNoRows.
func fakeTxxNoRowsFunc(*sqlx.Tx) error {
	return sql.ErrNoRows
}

func TestGetCfg(t *testing.T) {
	cases := []struct {
		Dsn     string
		WantErr error
	}{
		{"mysql://MOCK_SECRET_KEY_ID:MOCK_SECRET_KEY_VALUE@tcp(1.2.3.4:3306)/dbname?multiStatements=true", nil},
		{"mysql://MOCK_SECRET_KEY_ID:MOCK_SECRET_KEY_VALUE@fakeTcp(1.2.3.4:3306)/dbname?multiStatements=true",
			fmt.Errorf("invalid dsn: %s",
				"mysql://MOCK_SECRET_KEY_ID:MOCK_SECRET_KEY_VALUE@fakeTcp(1.2.3.4:3306)/dbname?multiStatements=true")},
		{"http://test.com", fmt.Errorf("invalid dsn: %s", "http://test.com")},
	}
	for _, c := range cases {
		t.Run(c.Dsn, func(t *testing.T) {
			_, err := getCfg(c.Dsn)
			assert.Equal(t, c.WantErr, err)
		})
	}
}

func TestCfgToEndpoint(t *testing.T) {
	const fakeMode mode = "fakeMode"
	domainCfg, _ := getCfg("mysql://MOCK_SECRET_KEY_ID:MOCK_SECRET_KEY_VALUE@tcp(domain.must.failed:3306)/dbname?multiStatements=true")
	ipCfg, _ := getCfg("mysql://MOCK_SECRET_KEY_ID:MOCK_SECRET_KEY_VALUE@tcp(1.2.3.4:3306)/dbname?multiStatements=true")
	cases := []struct {
		Name     string
		Cfg      *mysql.Config
		Mode     mode
		Expected string
		WantErr  error
	}{
		{"nil Config", nil, modeIP, "", fmt.Errorf("nil *mysql.Config")},
		{"unsupported mode", domainCfg, fakeMode, "", fmt.Errorf("unsupported mode: %s", fakeMode)},
		{
			"cannot resolve domain",
			domainCfg,
			modeIP,
			"",
			fmt.Errorf("cannot resolve domain or resolved too many IP: %s", "domain.must.failed"),
		},
		{
			"invalid mysql config dsn address",
			&mysql.Config{
				Addr: "1.2.3.4:3306:evil",
			},
			modeIP,
			"",
			fmt.Errorf("invalid mysql config dsn address: %s", "1.2.3.4:3306:evil"),
		},
		{"valid ip", ipCfg, modeIP, "1.2.3.4:3306", nil},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			ans, err := cfgToEndpoint(c.Cfg, c.Mode)
			assert.Equal(t, c.Expected, ans)
			assert.Equal(t, c.WantErr, err)
		})
	}
}

func TestPostProcessing(t *testing.T) {
	cases := []struct {
		Msg codec.Msg
		Dsn string
	}{
		{
			codec.Message(context.Background()),
			"mysql://MOCK_SECRET_KEY_ID:MOCK_SECRET_KEY_VALUE@tcp(1.2.3.4:3306)/dbname?multiStatements=true",
		},
	}
	for _, c := range cases {
		t.Run("", func(t *testing.T) {
			postProcessing(c.Msg, c.Dsn)
		})
	}
}

func TestMask(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{"url", "dsn://username:passwd@tcp(127.0.0.1:3306)/dbName?timeout=1s&parseTime=true&interpolateParams=true", "dsn://***...***s=true"},
		{"url with secret", "dsn://root:this-is-a-secret@tcp(1.2.3.4:3306)/dbname?multiStatements=true", "dsn://***...***s=true"},
		{"not url", "trpc.app.server.service", "trpc.a***...***ervice"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mask(tt.url); got != tt.want {
				t.Errorf("mask() = %v, want %v", got, tt.want)
			}
		})
	}
}
