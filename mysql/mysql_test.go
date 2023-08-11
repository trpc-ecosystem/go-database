package mysql_test

import (
	"context"
	"database/sql"
	"flag"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"

	. "github.com/smartystreets/goconvey/convey"

	"trpc.group/trpc-go/trpc-database/mysql"
	"trpc.group/trpc-go/trpc-go/client"
)

var ctx = context.Background()
var target = flag.String("target", "dsn", "mysql server target dsn address")
var uin = flag.Uint64("uin", 181431178, "uin")
var timeout = flag.Duration("timeout", time.Second, "timeout")

type mockClient struct {
	handle func(context.Context, interface{}, interface{}, ...client.Option) error
}

func (mc *mockClient) Invoke(ctx context.Context, reqbody interface{}, rspbody interface{}, opt ...client.Option) error {
	return mc.handle(ctx, reqbody, rspbody, opt...)
}

func TestMysqlQuery(t *testing.T) {

	flag.Parse()

	var count uint64

	next := func(rows *sql.Rows) error { // The framework underlying automatic for loop rows call the next function.

		err := rows.Scan(&count)
		if err != nil {
			t.Logf("mysql scan fail:%+v", err)
			return err
		}
		return nil
	}

	proxy := mysql.NewClientProxy("trpc.mysql.server.service", client.WithTarget(*target), client.WithTimeout(*timeout))
	err := proxy.Query(ctx, next, "select count(*) from t_accuse_record_0 where from_uin = ?", *uin)
	if err != nil {
		t.Logf("mysql query fail:%+v", err)
		return
	}

	t.Logf("mysql query success, count:%d", count)
}

// Staff Employee Information Definition
type Staff struct {
	Name  string `db:"name"`
	Phone string `db:"phone"`
	Age   int    `db:"age"`
}

func TestMysqlQueryToStructs(t *testing.T) {

	flag.Parse()

	var results []*Staff

	proxy := mysql.NewClientProxy("trpc.mysql.server.service", client.WithTarget(*target), client.WithTimeout(*timeout))
	err := proxy.QueryToStructs(ctx, &results, "select name,phone,age from staff_msg where age > 20")
	if err != nil {
		t.Logf("mysql query fail:%+v", err)
		return
	}

	t.Logf("mysql query success, results:%+v", results)
}

func TestMysqlExec(t *testing.T) {

	flag.Parse()

	proxy := mysql.NewClientProxy("trpc.mysql.server.service", client.WithTarget(*target), client.WithTimeout(*timeout))

	res, err := proxy.Exec(ctx, "insert into t_accuse_record_0(from_uin, timestamp) values (?, ?)", *uin, time.Now().Unix())
	if err != nil {
		t.Logf("mysql exec fail:%+v", err)
		return
	}

	t.Logf("mysql exec success, result:%s", res)
}

type user struct {
	ID   int64
	Name string
}

func mysqlTx(tx *sql.Tx) error {
	if _, err := tx.Exec("replace into users (id,name) values (?,?)", 1, "www"); err != nil {
		return err
	}
	var users []user
	rows, err := tx.Query("select id,name from users")
	if err != nil {
		return err
	}
	var id int64
	var name string
	for rows.Next() {
		if err = rows.Scan(&id, &name); err != nil {
			return err
		}
		users = append(users, user{ID: id, Name: name})
	}
	if err = rows.Err(); err != nil {
		return err
	}
	if len(users) > 0 {
		_, err := tx.Exec("update users set name=? where id=?", time.Now().Unix(), users[0].ID)
		if err != nil {
			return err
		}
	}
	return nil
}

// go test -run TestTransaction -target "dsn://root:@tcp(127.0.0.1:3306)/trpc_database_test?timeout=1s".
func TestTransaction(t *testing.T) {
	flag.Parse()
	Convey("mysql transaction nil", t, func() {
		client.DefaultClient = &mockClient{func(ctx context.Context, req interface{}, rsp interface{}, opts ...client.Option) error {
			return nil
		}}
		defer func() {
			client.DefaultClient = client.New()
		}()
		proxy := mysql.NewClientProxy("trpc.mysql.server.service", client.WithTarget(*target), client.WithTimeout(*timeout))
		err := proxy.Transaction(ctx, mysqlTx)
		if err != nil {
			t.Logf("mysql transaction error(%v)", err)
			t.FailNow()
		}
	})
}

// go test -run TestTransactionx -target "dsn://root:@tcp(127.0.0.1:3306)/trpc_database_test?timeout=1s".
func TestTransactionx(t *testing.T) {
	flag.Parse()
	Convey("mysql transactionx nil", t, func() {
		client.DefaultClient = &mockClient{func(ctx context.Context, req interface{}, rsp interface{}, opts ...client.Option) error {
			return nil
		}}
		defer func() {
			client.DefaultClient = client.New()
		}()
		proxy := mysql.NewClientProxy("trpc.mysql.server.service", client.WithTarget(*target), client.WithTimeout(*timeout))
		err := proxy.Transactionx(ctx, func(tx *sqlx.Tx) error {
			return mysqlTx(tx.Tx)
		})
		if err != nil {
			t.Logf("mysql transactionx error(%v)", err)
			t.FailNow()
		}
	})
}
