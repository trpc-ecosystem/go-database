package mysql

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/require"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
)

const (
	MySQLName = "trpc.mysql.test.db"
)

type mockClient struct {
	handle func(context.Context, interface{}, interface{}, ...client.Option) error
}

func (mc *mockClient) Invoke(ctx context.Context, reqbody interface{}, rspbody interface{}, opt ...client.Option) error {
	return mc.handle(ctx, reqbody, rspbody, opt...)
}

func TestUnit_NewClientProxy_P0(t *testing.T) {
	Convey("TestUnit_NewClientProxy_P0", t, func() {
		mysqlClient := NewClientProxy(MySQLName)
		rawClient, ok := mysqlClient.(*mysqlCli)
		So(ok, ShouldBeTrue)
		So(rawClient.client, ShouldResemble, client.DefaultClient)
		So(rawClient.serviceName, ShouldEqual, MySQLName)
		So(len(rawClient.opts), ShouldEqual, 2)
	})
}

func mockNextFunc(*sql.Rows) error {
	return nil
}

func TestUnit_mysqlCli_Query_P0(t *testing.T) {
	Convey("TestUnit_mysqlCli_Query_P0", t, func() {
		Convey("Invoke error", func() {
			client.DefaultClient = &mockClient{func(ctx context.Context, req interface{}, rsp interface{}, opts ...client.Option) error {
				return fmt.Errorf("fake error")
			}}
			defer func() {
				client.DefaultClient = client.New()
			}()
			mysqlClient := NewClientProxy(MySQLName)
			err := mysqlClient.Query(context.Background(), mockNextFunc, "select * from table limit 1")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldEqual, "fake error")
		})
		Convey("Invoke success", func() {
			client.DefaultClient = &mockClient{func(ctx context.Context, req interface{}, rsp interface{}, opts ...client.Option) error {
				return nil
			}}
			defer func() {
				client.DefaultClient = client.New()
			}()
			mysqlClient := NewClientProxy(MySQLName)
			err := mysqlClient.Query(context.Background(), mockNextFunc, "select * from table limit 1")
			So(err, ShouldBeNil)
		})
	})
}

// TestUnit_mysqlCli_QueryRow_P0 query row unit test.
func TestUnit_mysqlCli_QueryRow_P0(t *testing.T) {
	Convey("TestUnit_mysqlCli_QueryRow_P0", t, func() {
		Convey("Invoke error", func() {
			client.DefaultClient = &mockClient{func(ctx context.Context, req interface{}, rsp interface{}, opts ...client.Option) error {
				return fmt.Errorf("fake error")
			}}
			defer func() {
				client.DefaultClient = client.New()
			}()
			mysqlClient := NewClientProxy(MySQLName)
			err := mysqlClient.QueryRow(context.Background(), []interface{}{}, "select * from table limit 1")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldEqual, "fake error")
		})
		Convey("Invoke success", func() {
			client.DefaultClient = &mockClient{func(ctx context.Context, req interface{}, rsp interface{}, opts ...client.Option) error {
				return nil
			}}
			defer func() {
				client.DefaultClient = client.New()
			}()
			mysqlClient := NewClientProxy(MySQLName)
			err := mysqlClient.QueryRow(context.Background(), []interface{}{}, "select * from table limit 1")
			So(err, ShouldBeNil)
		})
	})
}

type mockStruct struct {
}

func TestUnit_mysqlCli_QueryToStruct_P0(t *testing.T) {
	Convey("QueryToStruct", t, func() {
		Convey("Invoke error", func() {
			client.DefaultClient = &mockClient{func(ctx context.Context, req interface{}, rsp interface{}, opts ...client.Option) error {
				return fmt.Errorf("fake error")
			}}
			defer func() {
				client.DefaultClient = client.New()
			}()
			mysqlCli := NewClientProxy(MySQLName)
			err := mysqlCli.QueryToStruct(context.Background(), new(mockStruct), "select * from table limit 1")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldEqual, "fake error")
		})
		Convey("Invoke Success", func() {
			client.DefaultClient = &mockClient{func(ctx context.Context, req interface{}, rsp interface{}, opts ...client.Option) error {
				return nil
			}}
			defer func() {
				client.DefaultClient = client.New()
			}()
			mysqlCli := NewClientProxy(MySQLName)
			err := mysqlCli.QueryToStructs(context.Background(), new(mockStruct), "select * from table limit 1")
			So(err, ShouldBeNil)
		})
	})
}

func TestUnit_mysqlCli_QueryToStructs_P0(t *testing.T) {
	Convey("TestUnit_mysqlCli_QueryToStructs_P0", t, func() {
		Convey("Invoke error", func() {
			client.DefaultClient = &mockClient{func(ctx context.Context, req interface{}, rsp interface{}, opts ...client.Option) error {
				return fmt.Errorf("fake error")
			}}
			defer func() {
				client.DefaultClient = client.New()
			}()
			mysqlClient := NewClientProxy(MySQLName)
			err := mysqlClient.QueryToStructs(context.Background(), new(mockStruct), "select * from table limit 1")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldEqual, "fake error")
		})
		Convey("Invoke Success", func() {
			client.DefaultClient = &mockClient{func(ctx context.Context, req interface{}, rsp interface{}, opts ...client.Option) error {
				return nil
			}}
			defer func() {
				client.DefaultClient = client.New()
			}()
			mysqlClient := NewClientProxy(MySQLName)
			err := mysqlClient.QueryToStructs(context.Background(), new(mockStruct), "select * from table limit 1")
			So(err, ShouldBeNil)
		})
	})
}

func TestUnit_mysqlCli_Exec_P0(t *testing.T) {
	Convey("TestUnit_mysqlCli_Exec_P0", t, func() {
		Convey("Invoke error", func() {
			client.DefaultClient = &mockClient{func(ctx context.Context, req interface{}, rsp interface{}, opts ...client.Option) error {
				return fmt.Errorf("fake error")
			}}
			defer func() {
				client.DefaultClient = client.New()
			}()
			mysqlClient := NewClientProxy(MySQLName)
			result, err := mysqlClient.Exec(context.Background(), "select * from table limit 1")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldEqual, "fake error")
			So(result, ShouldBeNil)
		})
		Convey("Invoke Success", func() {
			client.DefaultClient = &mockClient{func(ctx context.Context, reqbody interface{}, rspbody interface{}, opts ...client.Option) error {
				msg := codec.Message(ctx)
				rsp, ok := msg.ClientRspHead().(*Response)
				So(ok, ShouldBeTrue)
				rsp.Result = driver.RowsAffected(1)
				return nil
			}}
			defer func() {
				client.DefaultClient = client.New()
			}()
			mysqlClient := NewClientProxy(MySQLName)
			result, err := mysqlClient.Exec(context.Background(), "select * from table limit 1")
			So(err, ShouldBeNil)
			So(result, ShouldResemble, driver.RowsAffected(1))
		})
	})
}

func mockTx(*sql.Tx) error {
	return nil
}

func mockTxx(*sqlx.Tx) error {
	return nil
}

func TestUnit_mysqlCli_Transaction_P0(t *testing.T) {
	Convey("TestUnit_mysqlCli_Transaction_P0", t, func() {
		Convey("Invoke error", func() {
			client.DefaultClient = &mockClient{func(ctx context.Context, req interface{}, rsp interface{}, opts ...client.Option) error {
				return fmt.Errorf("fake error")
			}}
			defer func() {
				client.DefaultClient = client.New()
			}()
			mysqlClient := NewClientProxy(MySQLName)
			err := mysqlClient.Transaction(context.Background(), mockTx)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldEqual, "fake error")
		})
		Convey("Invoke Success", func() {
			client.DefaultClient = &mockClient{func(ctx context.Context, req interface{}, rsp interface{}, opts ...client.Option) error {
				return nil
			}}
			defer func() {
				client.DefaultClient = client.New()
			}()
			mysqlClient := NewClientProxy(MySQLName)
			err := mysqlClient.Transaction(context.Background(), mockTx)
			So(err, ShouldBeNil)
		})
	})
}

func TestUnit_mysqlCli_Transactionx_P0(t *testing.T) {
	Convey("TestUnit_mysqlCli_Transactionx_P0", t, func() {
		Convey("Invoke error", func() {
			client.DefaultClient = &mockClient{func(ctx context.Context, req interface{}, rsp interface{}, opts ...client.Option) error {
				return fmt.Errorf("fake error")
			}}
			defer func() {
				client.DefaultClient = client.New()
			}()
			mysqlClient := NewClientProxy(MySQLName)
			err := mysqlClient.Transactionx(context.Background(), mockTxx)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldEqual, "fake error")
		})
		Convey("Invoke Success", func() {
			client.DefaultClient = &mockClient{func(ctx context.Context, req interface{}, rsp interface{}, opts ...client.Option) error {
				return nil
			}}
			defer func() {
				client.DefaultClient = client.New()
			}()
			mysqlClient := NewClientProxy(MySQLName)
			err := mysqlClient.Transactionx(context.Background(), mockTxx)
			So(err, ShouldBeNil)
		})
	})
}

// User user info.
type User struct {
	ID   int64  `db:"id"`
	Name string `db:"name"`
	Age  int    `db:"age"`
}

// UserForUnsafeClient is used to test whether the unsafe mysql client behaves correctly.
// where the Age field is intentionally missing and an Addr is added.
type UserForUnsafeClient struct {
	ID   int64  `db:"id"`
	Name string `db:"name"`
	Addr string `db:"addr"`
}

func Test_mysqlCli_NamedExec(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	db := mustNewTestingDB()
	patches.ApplyMethod(reflect.TypeOf(new(ClientTransport)), "GetDB",
		func(_ *ClientTransport, dsn string) (*sql.DB, error) {
			return db, nil
		})

	c := NewClientProxy(MySQLName)
	// create test table.
	_, err := c.Exec(trpc.BackgroundContext(),
		`CREATE TABLE user (
			id integer NOT NULL PRIMARY KEY AUTOINCREMENT,
			name VARCHAR(64) NOT NULL DEFAULT '',
			age INTEGER NOT NULL DEFAULT '0'
		)`)
	require.NoError(t, err)

	tests := []struct {
		name  string
		query string
		args  interface{}
		want  []*User
	}{
		{
			name:  "test insert one row",
			query: "INSERT INTO user (name, age) VALUES (:name, :age)",
			args:  &User{Name: "Jobs", Age: 20},
			want:  []*User{{ID: 1, Name: "Jobs", Age: 20}},
		},
		{
			name:  "test batch insert with structs",
			query: "INSERT INTO user (name, age) VALUES (:name, :age)",
			args:  []User{{Name: "Alice", Age: 18}, {Name: "Foo", Age: 30}},
			want: []*User{
				{ID: 1, Name: "Jobs", Age: 20},
				{ID: 2, Name: "Alice", Age: 18},
				{ID: 3, Name: "Foo", Age: 30},
			},
		},
		{
			name:  "test batch insert with maps",
			query: "INSERT INTO user (id, name, age) VALUES (:id, :name, :age)",
			args: []map[string]interface{}{
				{"id": 10, "name": "Boo", "age": 10},
				{"id": 20, "name": "King", "age": 20},
			},
			want: []*User{
				{ID: 1, Name: "Jobs", Age: 20},
				{ID: 2, Name: "Alice", Age: 18},
				{ID: 3, Name: "Foo", Age: 30},
				{ID: 10, Name: "Boo", Age: 10},
				{ID: 20, Name: "King", Age: 20},
			},
		},
		{
			name:  "test insert one row without args",
			query: "INSERT INTO user (id, name, age) VALUES (100, 'Scott', '18')",
			args:  nil,
			want: []*User{
				{ID: 1, Name: "Jobs", Age: 20},
				{ID: 2, Name: "Alice", Age: 18},
				{ID: 3, Name: "Foo", Age: 30},
				{ID: 10, Name: "Boo", Age: 10},
				{ID: 20, Name: "King", Age: 20},
				{ID: 100, Name: "Scott", Age: 18},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := c.NamedExec(trpc.BackgroundContext(), tt.query, tt.args)
			require.NoError(t, err)

			if len(tt.want) > 0 {
				var users []*User
				err = c.QueryToStructs(trpc.BackgroundContext(), &users, "SELECT id, name, age FROM user ORDER BY id ASC")
				require.NoError(t, err)
				require.Equal(t, tt.want, users)
			}
		})
	}
}

func mustNewTestingDB() *sql.DB {
	db, err := sql.Open("sqlite3", "file::memory:")
	if err != nil {
		panic(err)
	}
	return db
}

func prepareTestData(t *testing.T, c Client) *gomonkey.Patches {
	patches := gomonkey.NewPatches()
	db := mustNewTestingDB()
	patches.ApplyMethod(reflect.TypeOf(new(ClientTransport)), "GetDB",
		func(_ *ClientTransport, dsn string) (*sql.DB, error) {
			return db, nil
		},
	)

	// create test table
	_, err := c.Exec(trpc.BackgroundContext(),
		`CREATE TABLE user (
			id integer NOT NULL PRIMARY KEY AUTOINCREMENT,
			name VARCHAR(64) NOT NULL DEFAULT '',
			age INTEGER NOT NULL DEFAULT '0'
		)`)
	require.NoError(t, err)

	// init data
	_, err = c.Exec(trpc.BackgroundContext(),
		"INSERT INTO user (id, name, age) VALUES "+
			"(1, 'Jobs', 15),"+
			"(2,'Alice',16),"+
			"(3,'Foo', 17),"+
			"(4,'Boo',18),"+
			"(10,'King',18),"+
			"(100, 'Haha', 20)", nil)

	require.NoError(t, err)
	return patches
}

func Test_mysqlCli_Get(t *testing.T) {
	c := NewClientProxy(MySQLName)
	patches := prepareTestData(t, c)
	defer patches.Reset()
	tests := []struct {
		name  string
		query string
		args  []interface{}
		want  interface{}
		err   error
	}{
		{
			name:  "get one row without args",
			query: "SELECT * FROM user where id = 1",
			args:  nil,
			want:  User{ID: 1, Name: "Jobs", Age: 15},
		},
		{
			name:  "get one row with args",
			query: "SELECT * FROM user where id = ?",
			args:  []interface{}{3},
			want:  User{ID: 3, Name: "Foo", Age: 17},
		},
		{
			name:  "get one row with many args",
			query: "SELECT * FROM user where id = ? AND name = ?",
			args:  []interface{}{4, "Boo"},
			want:  User{ID: 4, Name: "Boo", Age: 18},
		},
		{
			name:  "get one row without record",
			query: "SELECT * FROM user where id = ? AND name= ?",
			args:  []interface{}{5, "Boo"},
			want:  User{},
			err:   ErrNoRows,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var u User
			err := c.Get(trpc.BackgroundContext(), &u, tt.query, tt.args...)
			if tt.err == nil {
				require.NoError(t, err)
			} else {
				fmt.Println(err)
				require.Equal(t, tt.err, err)
			}
			require.Equal(t, tt.want, u)
		})
	}

}

func Test_mysqlCli_Select(t *testing.T) {
	c := NewClientProxy(MySQLName)
	patches := prepareTestData(t, c)
	defer patches.Reset()
	tests := []struct {
		name  string
		query string
		args  []interface{}
		want  interface{}
	}{
		{
			name:  "select rows without args",
			query: "SELECT * FROM user",
			args:  nil,
			want: []*User{
				{1, "Jobs", 15},
				{2, "Alice", 16},
				{3, "Foo", 17},
				{4, "Boo", 18},
				{10, "King", 18},
				{100, "Haha", 20},
			},
		},
		{
			name:  "select rows with args",
			query: "SELECT * FROM user WHERE id < ?",
			args:  []interface{}{4},
			want: []*User{
				{1, "Jobs", 15},
				{2, "Alice", 16},
				{3, "Foo", 17},
			},
		},
		{
			name:  "select rows with many args",
			query: "SELECT * FROM user WHERE name = ? OR age = ?",
			args:  []interface{}{"Haha", 18},
			want: []*User{
				{4, "Boo", 18},
				{10, "King", 18},
				{100, "Haha", 20},
			},
		},
		{
			name:  "select rows without record",
			query: "SELECT * FROM user WHERE id > ? ",
			args:  []interface{}{100},
			want:  []*User(nil),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var u []*User
			err := c.Select(trpc.BackgroundContext(), &u, tt.query, tt.args...)
			require.NoError(t, err)
			require.Equal(t, tt.want, u)
		})
	}

}

func Test_unsafeClient_Select(t *testing.T) {
	c := NewUnsafeClient(MySQLName)
	patches := prepareTestData(t, c)
	defer patches.Reset()
	tests := []struct {
		name  string
		query string
		args  []interface{}
		want  interface{}
	}{
		{
			name:  "select rows without args",
			query: "SELECT * FROM user",
			args:  nil,
			want: []*UserForUnsafeClient{
				{ID: 1, Name: "Jobs"},
				{ID: 2, Name: "Alice"},
				{ID: 3, Name: "Foo"},
				{ID: 4, Name: "Boo"},
				{ID: 10, Name: "King"},
				{ID: 100, Name: "Haha"},
			},
		},
		{
			name:  "select rows with args",
			query: "SELECT * FROM user WHERE id < ?",
			args:  []interface{}{4},
			want: []*UserForUnsafeClient{
				{ID: 1, Name: "Jobs"},
				{ID: 2, Name: "Alice"},
				{ID: 3, Name: "Foo"},
			},
		},
		{
			name:  "select rows with many args",
			query: "SELECT * FROM user WHERE name = ? OR age = ?",
			args:  []interface{}{"Haha", 18},
			want: []*UserForUnsafeClient{
				{ID: 4, Name: "Boo"},
				{ID: 10, Name: "King"},
				{ID: 100, Name: "Haha"},
			},
		},
		{
			name:  "select rows without record",
			query: "SELECT * FROM user WHERE id > ? ",
			args:  []interface{}{100},
			want:  []*UserForUnsafeClient(nil),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var u []*UserForUnsafeClient
			err := c.Select(trpc.BackgroundContext(), &u, tt.query, tt.args...)
			require.NoError(t, err)
			require.Equal(t, tt.want, u)
		})
	}

}

func Test_mysqlCli_NamedQuery(t *testing.T) {
	c := NewClientProxy(MySQLName)
	patches := prepareTestData(t, c)
	defer patches.Reset()

	tests := []struct {
		name  string
		query string
		args  interface{}
		want  []*User
	}{
		{
			name:  "test query with struct",
			query: "SELECT * FROM user WHERE age = :age",
			args:  User{Age: 18},
			want: []*User{
				{4, "Boo", 18},
				{10, "King", 18},
			},
		},
		{
			name:  "test query with map",
			query: "SELECT * FROM user WHERE id = :id AND name = :name",
			args:  map[string]interface{}{"id": 100, "name": "Haha"},
			want: []*User{
				{100, "Haha", 20},
			},
		},
		{
			name:  "test query without args",
			query: "SELECT * FROM user",
			args:  nil,
			want: []*User{
				{1, "Jobs", 15},
				{2, "Alice", 16},
				{3, "Foo", 17},
				{4, "Boo", 18},
				{10, "King", 18},
				{100, "Haha", 20},
			},
		},
		{
			name:  "test query without record",
			query: "SELECT * FROM user WHERE id = :id",
			args:  User{ID: 111},
			want:  []*User(nil),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var us []*User
			rows, err := c.NamedQuery(trpc.BackgroundContext(), tt.query, tt.args)
			require.NoError(t, err)
			defer rows.Close()
			for rows.Next() {
				var u User
				err = rows.StructScan(&u)
				require.NoError(t, err)
				us = append(us, &u)
			}
			require.Equal(t, tt.want, us)
		})
	}

}

func TestRequest_Copy(t *testing.T) {
	type T struct{ I int }

	structDest := T{}
	// rowDest points to an application structure and must be of type pointer.
	rowDest := []interface{}{&T{}, &[]T{{}}}
	r := Request{
		Query:              "query",
		Exec:               "exec",
		Args:               []interface{}{1, "a", true},
		op:                 9,
		txOpts:             &sql.TxOptions{Isolation: sql.LevelReadUncommitted, ReadOnly: true},
		QueryToStructsDest: &structDest,
		QueryRowDest:       rowDest,
	}

	r.next = func(*sql.Rows) error { return nil }
	v, err := r.Copy()
	require.NotNil(t, err, "request with non nil next closure is not copiable")

	r.next = nil
	r.tx = func(*sql.Tx) error { return nil }
	v, err = r.Copy()
	require.NotNil(t, err, "requst with non nil tx closure is not copiable")

	r.tx = nil
	v, err = r.Copy()
	require.Nil(t, err)
	rr := v.(*Request)
	require.Equal(t, &r, rr)

	structDest.I = 1
	require.NotEqual(t, r.QueryToStructsDest, rr.QueryToStructsDest)

	(*rowDest[1].(*[]T))[0].I = 1
	require.NotEqual(t, r.QueryRowDest, rr.QueryRowDest)
}

func TestRequest_CopyTo(t *testing.T) {
	type T struct{ I int }

	rStructDest := T{}
	rRowDest := []interface{}{&T{}, &[]T{{}}}
	r := Request{
		Query:              "query",
		QueryToStructsDest: &rStructDest,
		QueryRowDest:       rRowDest,
	}

	rrStructDest := T{}
	rr := Request{
		QueryToStructsDest: &rrStructDest,
		QueryRowDest:       []interface{}{},
	}

	require.NotNil(t, rr.CopyTo(r), "copyTo dst must be a pointer")
	require.NotNil(t, rr.CopyTo(&r), "length of QueryRowDest must match")

	rr.QueryToStructsDest = &T{I: 1}
	rr.QueryRowDest = []interface{}{&T{I: 1}, &[]T{}}
	require.Nil(t, rr.CopyTo(&r))
	require.Equal(t, rr.Query, r.Query)
	require.Equal(t, rr.QueryToStructsDest, r.QueryToStructsDest)
	require.Equal(t, rr.QueryRowDest, r.QueryRowDest)
	require.Equal(t, 1, rStructDest.I)
	require.Equal(t, 1, rRowDest[0].(*T).I)
}
