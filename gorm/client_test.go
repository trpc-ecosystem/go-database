package gorm

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/client/mockclient"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/transport"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
)

const (
	SQLName = "trpc.mysql.test.db"
)

func TestUnit_NewConnPool_P0(t *testing.T) {
	Convey("TestUnit_NewConnPool_P0", t, func() {
		connPool := NewConnPool(SQLName)
		rawClient, ok := connPool.(*Client)
		So(ok, ShouldBeTrue)
		So(rawClient.Client, ShouldResemble, client.DefaultClient)
		So(rawClient.ServiceName, ShouldEqual, SQLName)
		So(len(rawClient.opts), ShouldEqual, 3)
	})
}

func setMockError(mockClient *mockclient.MockClient) {
	mockClient.EXPECT().
		Invoke(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(fmt.Errorf("fake error"))
}

func TestUnit_gormCli_PrepareContext_P0(t *testing.T) {
	Convey("TestUnit_gormCli_PrepareContext_P0", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		mockClient := mockclient.NewMockClient(mockCtrl)

		gormClient := NewConnPool(SQLName)
		gormClient.(*Client).Client = mockClient

		Convey("Invoke error", func() {
			setMockError(mockClient)

			_, err := gormClient.PrepareContext(context.Background(), "select * from table limit 1")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldEqual, "fake error")
		})
		Convey("Invoke success", func() {
			var mockSQLStmt = &sql.Stmt{}
			mockClient.EXPECT().
				Invoke(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Do(func(ctx context.Context, reqbody interface{}, rspbody interface{}, opt ...client.Option) error {
					msg := codec.Message(ctx)
					rsp, ok := msg.ClientRspHead().(*Response)
					So(ok, ShouldBeTrue)
					rsp.Stmt = mockSQLStmt
					return nil
				})

			result, err := gormClient.PrepareContext(context.Background(), "select * from table limit 1")
			So(err, ShouldBeNil)
			So(result, ShouldEqual, mockSQLStmt)
		})
	})
}

func TestUnit_gormCli_ExecContext_P0(t *testing.T) {
	Convey("TestUnit_gormCli_ExecContext_P0", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		mockClient := mockclient.NewMockClient(mockCtrl)

		gormClient := NewConnPool(SQLName)
		gormClient.(*Client).Client = mockClient

		Convey("Invoke error", func() {
			mockClient.EXPECT().
				Invoke(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(fmt.Errorf("fake error"))

			_, err := gormClient.ExecContext(context.Background(), "select * from table limit 1")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldEqual, "fake error")
		})
		Convey("Invoke success", func() {
			mockClient.EXPECT().
				Invoke(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Do(func(ctx context.Context, reqbody interface{}, rspbody interface{}, opt ...client.Option) error {
					msg := codec.Message(ctx)
					rsp, ok := msg.ClientRspHead().(*Response)
					So(ok, ShouldBeTrue)
					rsp.Result = driver.RowsAffected(1)
					return nil
				})

			result, err := gormClient.ExecContext(context.Background(), "select * from table limit 1")
			So(err, ShouldBeNil)
			So(result, ShouldResemble, driver.RowsAffected(1))
		})
	})
}

func TestUnit_gormCli_QueryContext_P0(t *testing.T) {
	Convey("TestUnit_gormCli_ExecContext_P0", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		mockClient := mockclient.NewMockClient(mockCtrl)

		gormClient := NewConnPool(SQLName)
		gormClient.(*Client).Client = mockClient

		Convey("Invoke error", func() {
			setMockError(mockClient)

			_, err := gormClient.QueryContext(context.Background(), "select * from table limit 1")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldEqual, "fake error")
		})
		Convey("Invoke success", func() {
			var mockSQLRows = &sql.Rows{}
			mockClient.EXPECT().
				Invoke(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Do(func(ctx context.Context, reqbody interface{}, rspbody interface{}, opt ...client.Option) error {
					msg := codec.Message(ctx)
					rsp, ok := msg.ClientRspHead().(*Response)
					So(ok, ShouldBeTrue)
					rsp.Rows = mockSQLRows
					return nil
				})

			result, err := gormClient.QueryContext(context.Background(), "select * from table limit 1")
			So(err, ShouldBeNil)
			So(result, ShouldEqual, mockSQLRows)
		})
	})
}

func TestUnit_gormCli_BeginTx_P0(t *testing.T) {
	Convey("TestUnit_gormCli_BeginTx_P0", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		mockClient := mockclient.NewMockClient(mockCtrl)

		gormClient := NewConnPool(SQLName)
		gormClient.(*Client).Client = mockClient

		Convey("Invoke error", func() {
			setMockError(mockClient)

			_, err := gormClient.BeginTx(context.Background(), &sql.TxOptions{})
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldEqual, "fake error")
		})
		Convey("Invoke success", func() {
			var mockSQLTx = &sql.Tx{}
			mockClient.EXPECT().
				Invoke(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Do(func(ctx context.Context, reqbody interface{}, rspbody interface{}, opt ...client.Option) error {
					msg := codec.Message(ctx)
					rsp, ok := msg.ClientRspHead().(*Response)
					So(ok, ShouldBeTrue)
					rsp.Tx = mockSQLTx
					return nil
				})

			result, err := gormClient.BeginTx(context.Background(), &sql.TxOptions{})
			So(err, ShouldBeNil)
			txc, ok := result.(*TxClient)
			So(ok, ShouldBeTrue)
			So(txc.Tx, ShouldEqual, mockSQLTx)
		})
	})
}

func TestUnit_gormCli_GetDB_P0(t *testing.T) {
	Convey("TestUnit_gormCli_GetDB_P0", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		mockClient := mockclient.NewMockClient(mockCtrl)

		gormClient := NewConnPool(SQLName)
		gormClient.(*Client).Client = mockClient

		Convey("Invoke error", func() {
			setMockError(mockClient)

			_, err := gormClient.GetDBConn()
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldEqual, "fake error")
		})
		Convey("Invoke success", func() {
			var mockDB = &sql.DB{}
			mockClient.EXPECT().
				Invoke(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Do(func(ctx context.Context, reqbody interface{}, rspbody interface{}, opt ...client.Option) error {
					msg := codec.Message(ctx)
					rsp, ok := msg.ClientRspHead().(*Response)
					So(ok, ShouldBeTrue)
					rsp.DB = mockDB
					return nil
				})

			result, err := gormClient.GetDBConn()
			So(err, ShouldBeNil)
			So(result, ShouldEqual, mockDB)
		})
	})
}

func TestUnit_gormTxCli_Commit_P0(t *testing.T) {
	Convey("TestUnit_gormTxCli_Commit_P0", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		mockClient := mockclient.NewMockClient(mockCtrl)

		gormClient := NewConnPool(SQLName).(*Client)
		gormClient.Client = mockClient

		gormTxClient := &TxClient{
			Client: gormClient,
			Tx:     &sql.Tx{},
		}

		Convey("Invoke error", func() {
			setMockError(mockClient)

			err := gormTxClient.Commit()
			So(err.Error(), ShouldEqual, "fake error")
		})
		Convey("Invoke success", func() {
			mockClient.EXPECT().
				Invoke(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(nil)

			err := gormTxClient.Commit()
			So(err, ShouldBeNil)
		})
	})
}

func TestUnit_gormTxCli_Rollback_P0(t *testing.T) {
	Convey("TestUnit_gormTxCli_Rollback_P0", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		mockClient := mockclient.NewMockClient(mockCtrl)

		gormClient := NewConnPool(SQLName).(*Client)
		gormClient.Client = mockClient

		gormTxClient := &TxClient{
			Client: gormClient,
			Tx:     &sql.Tx{},
		}

		Convey("Invoke error", func() {
			setMockError(mockClient)

			err := gormTxClient.Rollback()
			So(err.Error(), ShouldEqual, "fake error")
		})
		Convey("Invoke success", func() {
			mockClient.EXPECT().
				Invoke(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(nil)

			err := gormTxClient.Rollback()
			So(err, ShouldBeNil)
		})
	})
}

func TestUnit_gormCli_Ping_P0(t *testing.T) {
	Convey("TestUnit_gormCli_Ping_P0", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		mockClient := mockclient.NewMockClient(mockCtrl)

		gormClient := NewConnPool(SQLName)
		gormClient.(*Client).Client = mockClient

		Convey("Invoke error", func() {
			setMockError(mockClient)

			err := gormClient.Ping()
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldEqual, "fake error")
		})
		Convey("Invoke success", func() {
			mockClient.EXPECT().
				Invoke(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(nil)

			err := gormClient.Ping()
			So(err, ShouldBeNil)
		})
	})
}

func TestUnit_gormCli_QueryRowContext_P0(t *testing.T) {
	// Invalid DSN address.
	Convey("TestUnit_gormCli_QueryRowContext_P0", t, func() {
		gormClient := NewConnPool(SQLName, client.WithTarget("dns://xxxx"))
		Convey("Invoke error", func() {
			err := gormClient.QueryRowContext(context.Background(), "id=?", "1").Scan()
			So(err, ShouldNotBeNil)
		})
	})
}

func TestUnit_CompleteCall(t *testing.T) {
	// Mock the database directly to create a complete call chain.
	Convey("TestUnit_CompleteCall", t, func() {
		// The structure to be inserted into the database.
		type Blog struct {
			ID      uint
			Title   string
			Content string
			Tags    string
		}
		blog := &Blog{
			Title:   "post",
			Content: "hello",
			Tags:    "1",
		}
		ct := &ClientTransport{
			SQLDB: make(map[string]*sql.DB),
			DefaultPoolConfig: PoolConfig{
				MaxIdle:     10,
				MaxOpen:     10000,
				MaxLifetime: 3 * time.Minute,
			},
		}
		transport.RegisterClientTransport("gorm", ct)
		dsn := "xxxx:xxxxx@tcp(0.0.0:0:8888)/xxx?timeout=1s&parseTime=true&loc=Local"
		sqlName := "a.b.c.d"
		mockDB, mock, _ := sqlmock.New()
		defer mockDB.Close()
		mock.ExpectQuery("SELECT VERSION()").WithArgs().WillReturnRows(sqlmock.NewRows([]string{"xxx"}).AddRow("xxx"))

		const sqlInsert = "INSERT INTO `blogs`"
		mock.ExpectBegin() // begin transaction
		mock.ExpectExec(sqlInsert).
			WithArgs(blog.Title, blog.Content, blog.Tags).
			WillReturnResult(sqlmock.NewResult(2, 1))
		mock.ExpectCommit() // commit transaction
		// The second transaction.
		mock.ExpectBegin() // begin transaction
		mock.ExpectExec(sqlInsert).
			WithArgs(blog.Title, blog.Content, blog.Tags).
			WillReturnResult(sqlmock.NewResult(3, 1))
		mock.ExpectCommit() // commit transaction

		ct.SQLDB[dsn] = mockDB
		client := NewConnPool(sqlName, client.WithTarget("dsn://"+dsn))
		gormDB, err := gorm.Open(mysql.New(mysql.Config{Conn: client}))
		So(err, ShouldBeNil)
		So(gormDB, ShouldNotBeNil)

		err = gormDB.Save(blog).Error
		So(err, ShouldBeNil)
		So(blog.ID, ShouldEqual, 2)

		// Reset.
		blog.ID = 0
		tx := gormDB.Begin(&sql.TxOptions{Isolation: sql.LevelReadCommitted})
		tx.Create(blog)
		err = tx.Commit().Error
		So(err, ShouldBeNil)
		So(blog.ID, ShouldEqual, 3)

		d, err := gormDB.DB()
		So(err, ShouldBeNil)
		So(d, ShouldEqual, mockDB)

	})
}

func TestUnit_NewClientProxy(t *testing.T) {
	Convey("TestUnit_NewClientProxy", t, func() {
		ct := &ClientTransport{
			SQLDB: make(map[string]*sql.DB),
			DefaultPoolConfig: PoolConfig{
				MaxIdle:     10,
				MaxOpen:     10000,
				MaxLifetime: 3 * time.Minute,
			},
		}
		transport.RegisterClientTransport("gorm", ct)
		mockDB, mock, _ := sqlmock.New()
		defer mockDB.Close()

		mock.ExpectQuery("SELECT VERSION()").WithArgs().WillReturnRows(sqlmock.NewRows([]string{"xxx"}).AddRow("xxx"))
		mock.ExpectQuery("SELECT VERSION()").WillReturnRows(sqlmock.NewRows([]string{"xxx"}).AddRow("xxx"))
		dsn := "xxxx:xxxxx@tcp(0.0.0:0:8888)/xxx?timeout=1s&parseTime=true&loc=Local"
		ct.SQLDB[dsn] = mockDB

		db, err := NewClientProxy("trpc.postgres.test.db", client.WithTarget("dsn://"+dsn))

		So(err, ShouldBeNil)
		So(db, ShouldNotBeNil)

		db, err = NewClientProxy("db", client.WithTarget("dsn://"+dsn))

		So(err, ShouldBeNil)
		So(db, ShouldNotBeNil)

		db, err = NewClientProxy("trpc.mysql.test.db", client.WithTarget("dsn://"+dsn))

		So(err, ShouldBeNil)
		So(db, ShouldNotBeNil)

	})
}

// testSerializer simulates the schema.serializer structure in gorm.
type testSerializer struct {
	Field      *testField
	fieldValue interface{}
}

type testSchema struct {
	Field *testField
}

type testField struct {
	Schema *testSchema
}

func (ts testSerializer) Value() (driver.Value, error) {
	result, err := json.Marshal(ts.fieldValue)
	return string(result), err
}

// Test_stackoverflow simulates a stack overflow.
func Test_stackoverflow(t *testing.T) {
	Convey("Test_stackoverflow", t, func() {
		schema := &testSchema{}
		field := &testField{
			Schema: schema,
		}
		schema.Field = field
		serializer := &testSerializer{
			Field: field,
		}
		req1 := &Request{
			Op:   OpExecContext,
			Args: []interface{}{serializer},
		}
		// This line causes a stack overflow, which is then passed on to OpenTelemetry.
		// In an actual program, this line would be passed to a JSON marshal in OpenTelemetry.
		// _, _ = jsoniter.Marshal(req1)
		// The official json.marshal has a limit of 1000 levels of recursion and will not overflow,
		// but it still fails to marshal in this case.
		_, err := json.Marshal(req1)
		t.Log(err)
		So(err, ShouldNotBeNil)
	})
}

type testValuer struct {
	value string
}

func (tv testValuer) Value() (driver.Value, error) {
	return tv.value, fmt.Errorf("test error")
}

func TestUnit_handleReqArgs(t *testing.T) {
	Convey("Test_handleReqArgs", t, func() {
		req1 := &Request{
			Op: OpExecContext,
			Args: []interface{}{1, "test1", sql.NullString{
				String: "test1",
				Valid:  false,
			}},
		}
		err := handleReqArgs(req1)
		So(err, ShouldBeNil)
		req2 := &Request{
			Op:   OpExecContext,
			Args: []interface{}{2, "test2", testValuer{}},
		}
		err = handleReqArgs(req2)
		So(err, ShouldNotBeNil)
	})
}
