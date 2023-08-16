package gorm

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"testing"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-sql-driver/mysql"
	. "github.com/smartystreets/goconvey/convey"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/transport"
)

var dsn = "user:password@tcp(localhost:5555)/dbname"
var clickhouseDsn = "dsn://tcp://localhost:9000/dbname"

func TestUnit_NewClientTransport_P0(t *testing.T) {
	Convey("TestUnit_NewClientTransport_P0", t, func() {
		ctt := NewClientTransport()

		So(ctt.opener, ShouldEqual, sql.Open)
		So(ctt.opts, ShouldResemble, &transport.ClientTransportOptions{})
		So(ctt.SQLDB, ShouldBeEmpty)
		So(ctt.DefaultPoolConfig, ShouldResemble, PoolConfig{
			MaxIdle:     10,
			MaxOpen:     10000,
			MaxLifetime: 3 * time.Minute,
		})
	})
}

func TestUnit_ClientTransport_GetDB_P0(t *testing.T) {
	Convey("TestUnit_ClientTransport_GetDB_P0", t, func() {
		ct := new(ClientTransport)
		ct.SQLDB = make(map[string]*sql.DB)
		ct.DefaultPoolConfig = PoolConfig{
			MaxIdle:     10,
			MaxLifetime: 3 * time.Minute,
			MaxOpen:     1000,
		}
		ct.PoolConfigs = map[string]PoolConfig{
			"trpc.mysql.test.db": {
				MaxIdle:     5,
				MaxOpen:     10,
				MaxLifetime: 10 * time.Minute,
			},
		}
		Convey("DSN Already Exists", func() {
			ct.SQLDB[dsn] = new(sql.DB)
			db, err := ct.GetDB("", dsn)
			So(db, ShouldNotBeNil)
			So(err, ShouldBeNil)
		})
		Convey("Open Fail", func() {
			ct.opener = func(driverName, dataSourceName string) (*sql.DB, error) {
				return nil, fmt.Errorf("fake error")
			}
			db, err := ct.GetDB("", dsn)
			So(db, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldEqual, "fake error")
		})
		Convey("GetDB success", func() {
			ct.opener = func(driverName, dataSourceName string) (*sql.DB, error) {
				return new(sql.DB), nil
			}
			db, err := ct.GetDB("", dsn)
			So(db, ShouldNotBeNil)
			So(err, ShouldBeNil)
		})
		Convey("GetDB with service config", func() {
			dsn1 := "user:password@tcp(localhost:5555)/dbname1"
			ct.opener = func(driverName, dataSourceName string) (*sql.DB, error) {
				return new(sql.DB), nil
			}

			db, err := ct.GetDB("trpc.mysql.test.db", dsn1)
			So(db, ShouldNotBeNil)
			So(err, ShouldBeNil)
		})
	})
}

func TestUnit_CT_GormDbInit_P0(t *testing.T) {
	Convey("TestUnit_CT_GormDbInit_P0", t, func() {
		ct := new(ClientTransport)
		ct.SQLDB = make(map[string]*sql.DB)
		ct.DefaultPoolConfig = PoolConfig{
			MaxIdle:     10,
			MaxLifetime: 3 * time.Minute,
			MaxOpen:     1000,
		}
		ct.opener = sql.Open
		ct.PoolConfigs = map[string]PoolConfig{
			"trpc.clickhouse.test.db": {
				MaxIdle:     5,
				MaxOpen:     10,
				MaxLifetime: 10 * time.Minute,
			},
		}
		Convey("Init Clickhouse gorm DB", func() {
			db, err := ct.GetDB("trpc.clickhouse.test.db", clickhouseDsn)
			So(db, ShouldNotBeNil)
			So(err, ShouldBeNil)
		})
		Convey("Init mysql gorm DB", func() {
			db, err := ct.GetDB("trpc.mysql.test.db", dsn)
			So(db, ShouldNotBeNil)
			So(err, ShouldBeNil)
		})
		Convey("Init postgresql gorm DB", func() {
			db, err := ct.GetDB("trpc.postgres.test.db", dsn)
			So(db, ShouldNotBeNil)
			So(err, ShouldBeNil)
		})

		// Perform default initialization logic for MySQL.
		Convey("Init default gorm DB", func() {
			db, err := ct.GetDB("db", dsn)
			So(db, ShouldNotBeNil)
			So(err, ShouldBeNil)
		})
	})
}

// TestUnit_CT_RoundTrip_P0 ClientTransport.RoundTrip test case.
func TestUnit_CT_RoundTrip_P0(t *testing.T) {
	Convey("TestUnit_CT_RoundTrip_P0", t, func() {
		ctx, msg := codec.WithNewMessage(context.Background())
		opts := []transport.RoundTripOption{
			transport.WithDialAddress(dsn),
		}
		reqBuf := make([]byte, 0)
		ct := new(ClientTransport)
		// without Request in ReqHead
		Convey("ClientReqHead() not *Request", func() {
			rspBuf, err := ct.RoundTrip(ctx, reqBuf, opts...)
			So(rspBuf, ShouldBeNil)
			So(err, ShouldResemble, errs.NewFrameError(errs.RetClientEncodeFail,
				"sql client transport: ReqHead should be type of *gormCli.Request"))
		})

		// add Request in ReqHead
		request := new(Request)
		msg.WithClientReqHead(request)

		// without Response in ReqHead
		Convey("msg.ClientRspHead() not *Response", func() {
			rspBuf, err := ct.RoundTrip(ctx, reqBuf, opts...)
			So(rspBuf, ShouldBeNil)
			So(err, ShouldResemble, errs.NewFrameError(errs.RetClientEncodeFail,
				"sql client transport: RspHead should be type of *gormCli.Response"))
		})

		// add Response in ReqHead
		response := new(Response)
		msg.WithClientRspHead(response)

		Convey("GetDB Fail", func() {
			fakeErr := fmt.Errorf("fake error")
			ct.opener = func(driverName, dataSourceName string) (*sql.DB, error) {
				return nil, fakeErr
			}

			rspBuf, err := ct.RoundTrip(ctx, reqBuf, opts...)
			So(rspBuf, ShouldBeNil)
			So(err.Error(), ShouldContainSubstring, fakeErr.Error())
		})
	})
}

func getFakeErr(mockErr error) error {
	switch sqlErr := mockErr.(type) {
	case *mysql.MySQLError:
		return errs.Wrap(sqlErr, int(sqlErr.Number), sqlErr.Message)
	case *clickhouse.Exception:
		return errs.Wrap(sqlErr, int(sqlErr.Code), sqlErr.Message)
	case *errs.Error, nil:
	default:
		return errs.Wrap(mockErr, errs.RetUnknown, mockErr.Error())
	}
	return mockErr
}

func prepareContextConvey(ctx context.Context, ct *ClientTransport, request *Request, reqBuf []byte,
	fakeErr error, mock sqlmock.Sqlmock, opts []transport.RoundTripOption) {
	Convey("Do PrepareContext", func() {
		request.Op = OpPrepareContext

		Convey("PrepareContext Fail", func() {
			mock.ExpectPrepare(".*").WillReturnError(fakeErr)
			rspBuf, err := ct.RoundTrip(ctx, reqBuf, opts...)
			So(rspBuf, ShouldBeNil)
			So(err, ShouldResemble, getFakeErr(fakeErr))
		})
		Convey("PrepareContext Success", func() {
			mock.ExpectPrepare(".*").WillReturnError(nil)
			rspBuf, err := ct.RoundTrip(ctx, reqBuf, opts...)
			So(rspBuf, ShouldBeNil)
			So(err, ShouldBeNil)
		})
	})
}

func execContextConvey(ctx context.Context, ct *ClientTransport, request *Request, response *Response, reqBuf []byte,
	fakeErr error, mock sqlmock.Sqlmock, opts []transport.RoundTripOption) {
	Convey("Do ExecContext", func() {
		request.Op = OpExecContext

		Convey("ExecContext Fail", func() {
			mock.ExpectExec(".*").WillReturnError(fakeErr)
			rspBuf, err := ct.RoundTrip(ctx, reqBuf, opts...)
			So(rspBuf, ShouldBeNil)
			So(err, ShouldResemble, getFakeErr(fakeErr))
		})
		Convey("ExecContext Success", func() {
			mock.ExpectExec(".*").WillReturnResult(driver.RowsAffected(1))

			rspBuf, err := ct.RoundTrip(ctx, reqBuf, opts...)
			So(rspBuf, ShouldBeNil)
			So(err, ShouldBeNil)

			affected, err := response.Result.RowsAffected()
			So(affected, ShouldEqual, 1)
			So(err, ShouldBeNil)
		})
	})
}

func queryContextConvey(ctx context.Context, ct *ClientTransport, request *Request, response *Response, reqBuf []byte,
	fakeErr error, mock sqlmock.Sqlmock, opts []transport.RoundTripOption) {
	Convey("Do QueryContext", func() {
		request.Op = OpQueryContext

		Convey("QueryContext Fail", func() {
			mock.ExpectQuery(".*").WillReturnError(fakeErr)
			rspBuf, err := ct.RoundTrip(ctx, reqBuf, opts...)
			So(rspBuf, ShouldBeNil)
			So(err, ShouldResemble, getFakeErr(fakeErr))
		})
		Convey("QueryContext Success", func() {
			mock.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"column_1"}).AddRow("value_1"))

			rspBuf, err := ct.RoundTrip(ctx, reqBuf, opts...)
			So(rspBuf, ShouldBeNil)
			So(err, ShouldBeNil)

			columns, err := response.Rows.Columns()
			So(columns, ShouldResemble, []string{"column_1"})
			So(err, ShouldBeNil)

			var values []string
			for response.Rows.Next() {
				var value string
				err = response.Rows.Scan(&value)
				So(err, ShouldBeNil)
				values = append(values, value)
			}
			So(values, ShouldResemble, []string{"value_1"})
		})
	})
}

func queryRowContextConvey(ctx context.Context, ct *ClientTransport, request *Request, response *Response,
	reqBuf []byte, fakeErr error, mock sqlmock.Sqlmock, opts []transport.RoundTripOption) {
	Convey("Do QueryRowContext", func() {
		request.Op = OpQueryRowContext

		// Will not fail.
		Convey("QueryRowContext Success", func() {
			mock.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"column_1"}).AddRow("value_1"))

			rspBuf, err := ct.RoundTrip(ctx, reqBuf, opts...)
			So(rspBuf, ShouldBeNil)
			So(err, ShouldBeNil)

			var value string
			err = response.Row.Scan(&value)
			So(err, ShouldBeNil)
			So(value, ShouldEqual, "value_1")
		})
	})
}

// TestUnit_CT_RoundTrip_P0_2 ClientTransport.RoundTrip test case the second part.
func TestUnit_CT_RoundTrip_P0_2(t *testing.T) {
	Convey("TestUnit_CT_RoundTrip_P0_2", t, func() {
		ctx, msg := codec.WithNewMessage(context.Background())
		opts := []transport.RoundTripOption{
			transport.WithDialAddress(dsn),
		}
		reqBuf := make([]byte, 0)
		ct := new(ClientTransport)

		db, mock, err := sqlmock.New()
		So(err, ShouldBeNil)

		// add SQLDB in ClientTransport
		ct.SQLDB = map[string]*sql.DB{
			dsn: db,
		}

		// add Request in ReqHead
		request := new(Request)
		msg.WithClientReqHead(request)

		// add Response in ReqHead
		response := new(Response)
		msg.WithClientRspHead(response)

		fakeErr := &mysql.MySQLError{
			Number:  1,
			Message: "fake error",
		}

		prepareContextConvey(ctx, ct, request, reqBuf, fakeErr, mock, opts)

		execContextConvey(ctx, ct, request, response, reqBuf, fakeErr, mock, opts)

		queryContextConvey(ctx, ct, request, response, reqBuf, fakeErr, mock, opts)

		queryRowContextConvey(ctx, ct, request, response, reqBuf, fakeErr, mock, opts)

		Convey("Do GetDB", func() {
			request.Op = OpGetDB
			mockDB := new(sql.DB)
			ct.SQLDB[dsn] = mockDB
			// Will not fail.
			Convey("GetDB Success", func() {
				rspBuf, err := ct.RoundTrip(ctx, reqBuf, opts...)
				So(rspBuf, ShouldBeNil)
				So(err, ShouldBeNil)
				So(response.DB, ShouldResemble, mockDB)
			})
		})
	})
}

func TestUnit_CT_RoundTrip_TX_P0(t *testing.T) {
	Convey("TestUnit_CT_RoundTrip_TX_P0", t, func() {
		ctx, msg := codec.WithNewMessage(context.Background())
		opts := []transport.RoundTripOption{
			transport.WithDialAddress(dsn),
		}
		reqBuf := make([]byte, 0)
		ct := new(ClientTransport)

		db, mock, err := sqlmock.New()
		So(err, ShouldBeNil)

		mock.ExpectBegin().WillReturnError(nil)
		tx, err := db.Begin()
		So(err, ShouldBeNil)

		// add Request in ReqHead
		request := new(Request)
		request.Tx = tx

		msg.WithClientReqHead(request)

		// add Response in ReqHead
		response := new(Response)
		msg.WithClientRspHead(response)

		fakeErr := &clickhouse.Exception{
			Code:    1,
			Message: "fake error",
		}

		prepareContextConvey(ctx, ct, request, reqBuf, fakeErr, mock, opts)

		execContextConvey(ctx, ct, request, response, reqBuf, fakeErr, mock, opts)

		queryContextConvey(ctx, ct, request, response, reqBuf, fakeErr, mock, opts)

		queryRowContextConvey(ctx, ct, request, response, reqBuf, fakeErr, mock, opts)
	})
}

// TestUnit_CT_RoundTrip_P1
func TestUnit_CT_RoundTrip_P1(t *testing.T) {
	Convey("TestUnit_CT_RoundTrip_P1", t, func() {
		ctx, msg := codec.WithNewMessage(context.Background())
		opts := []transport.RoundTripOption{
			transport.WithDialAddress(dsn),
		}
		reqBuf := make([]byte, 0)
		ct := new(ClientTransport)

		// add Request in ReqHead
		request := new(Request)
		msg.WithClientReqHead(request)

		// add Response in ReqHead
		response := new(Response)
		msg.WithClientRspHead(response)

		db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
		So(err, ShouldBeNil)

		// add SQLDB in ClientTransport
		ct.SQLDB = map[string]*sql.DB{
			dsn: db,
		}

		fakeErr := &mysql.MySQLError{
			Number:  1,
			Message: "fake error",
		}

		Convey("Do BeginTx", func() {
			request.Op = OpBeginTx

			Convey("BeginTx Fail", func() {
				mock.ExpectBegin().WillReturnError(fakeErr)
				rspBuf, err := ct.RoundTrip(ctx, reqBuf, opts...)
				So(rspBuf, ShouldBeNil)
				So(err, ShouldResemble, errs.Wrap(fakeErr, int(fakeErr.Number), fakeErr.Message))
			})
			Convey("BeginTx Success", func() {
				mock.ExpectBegin().WillReturnError(nil)

				rspBuf, err := ct.RoundTrip(ctx, reqBuf, opts...)
				So(rspBuf, ShouldBeNil)
				So(err, ShouldBeNil)
				So(response.Tx, ShouldNotBeNil)
			})
		})

		Convey("Do Ping", func() {
			request.Op = OpPing

			Convey("Ping Fail", func() {
				mock.ExpectPing().WillReturnError(fakeErr)
				rspBuf, err := ct.RoundTrip(ctx, reqBuf, opts...)
				So(rspBuf, ShouldBeNil)
				So(err, ShouldResemble, errs.Wrap(fakeErr, int(fakeErr.Number), fakeErr.Message))
			})
			Convey("Ping Success", func() {
				mock.ExpectPing().WillReturnError(nil)
				rspBuf, err := ct.RoundTrip(ctx, reqBuf, opts...)
				So(rspBuf, ShouldBeNil)
				So(err, ShouldBeNil)
			})
		})

		Convey("Default Operation", func() {
			request.Op = 0
			rspBuf, err := ct.RoundTrip(ctx, reqBuf, opts...)
			So(rspBuf, ShouldBeNil)
			So(err, ShouldResemble, errs.NewFrameError(errs.RetServerNoFunc, "Illegal Method"))
		})
	})
}

// TestUnit_CT_RoundTrip_TX_P1
func TestUnit_CT_RoundTrip_TX_P1(t *testing.T) {
	Convey("TestUnit_CT_RoundTrip_TX_P1", t, func() {
		ctx, msg := codec.WithNewMessage(context.Background())
		opts := []transport.RoundTripOption{
			transport.WithDialAddress(dsn),
		}
		reqBuf := make([]byte, 0)
		ct := new(ClientTransport)

		db, mock, err := sqlmock.New()
		So(err, ShouldBeNil)

		mock.ExpectBegin().WillReturnError(nil)
		tx, err := db.Begin()
		So(err, ShouldBeNil)

		// add Request in ReqHead
		request := new(Request)
		request.Tx = tx

		msg.WithClientReqHead(request)

		// add Response in ReqHead
		response := new(Response)
		msg.WithClientRspHead(response)

		fakeErr := &mysql.MySQLError{
			Number:  1,
			Message: "fake error",
		}

		Convey("Do Commit", func() {
			request.Op = OpCommit

			Convey("Commit Fail", func() {
				mock.ExpectCommit().WillReturnError(fakeErr)
				rspBuf, err := ct.RoundTrip(ctx, reqBuf, opts...)
				So(rspBuf, ShouldBeNil)
				So(err, ShouldResemble, errs.Wrap(fakeErr, int(fakeErr.Number), fakeErr.Message))
			})
			Convey("Commit Success", func() {
				mock.ExpectCommit().WillReturnError(nil)
				rspBuf, err := ct.RoundTrip(ctx, reqBuf, opts...)
				So(rspBuf, ShouldBeNil)
				So(err, ShouldBeNil)
			})
		})

		Convey("Do Rollback", func() {
			request.Op = OpRollback

			Convey("Rollback Fail", func() {
				mock.ExpectRollback().WillReturnError(fakeErr)
				rspBuf, err := ct.RoundTrip(ctx, reqBuf, opts...)
				So(rspBuf, ShouldBeNil)
				So(err, ShouldResemble, errs.Wrap(fakeErr, int(fakeErr.Number), fakeErr.Message))
			})
			Convey("Rollback Success", func() {
				mock.ExpectRollback().WillReturnError(nil)
				rspBuf, err := ct.RoundTrip(ctx, reqBuf, opts...)
				So(rspBuf, ShouldBeNil)
				So(err, ShouldBeNil)
			})
		})

		Convey("Default Operation", func() {
			request.Op = 0
			rspBuf, err := ct.RoundTrip(ctx, reqBuf, opts...)
			So(rspBuf, ShouldBeNil)
			So(err, ShouldResemble, errs.NewFrameError(errs.RetServerNoFunc, "Illegal Method"))
		})
	})
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
