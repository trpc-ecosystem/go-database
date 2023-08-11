// Package main is the main package.
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"testing"
	"time"

	"trpc.group/trpc-go/trpc-database/mysql"
	"trpc.group/trpc-go/trpc-go/client"
)

// ClientProxy definition mysql.Client type.
type ClientProxy struct {
	mysql.Client
}

// Return data from app_pos_type_flowcontrol_20170312, after the result is put into data.
// Usage: Select(NewMysqlClient()), Reference Test_Select of query_demo_test.go.
// When unit testing：Select(mysql.Client of mock), Reference Test_Select_MockMysql of query_demo_test.go.
func (c ClientProxy) Select() (err error) {
	fmt.Printf("Select begin\n")
	var (
		engine string
		xa     string
	)
	// The next function is called by the framework's underlying automatic for loop rows.
	next := func(rows *sql.Rows) error {
		fmt.Printf("next begin\n")
		err := rows.Scan(&engine, &xa)
		if err != nil {
			fmt.Printf("Scan err:%v\n", err)
			return err
		}
		fmt.Printf("engine: %s, xa: %s\n", engine, xa)
		return nil
	}
	err = c.Query(context.Background(), next,
		"SELECT ENGINE, XA FROM ENGINES WHERE TRANSACTIONS='YES'")
	if err != nil {
		fmt.Printf("mysql.Query.err:%v\n", err)
	}
	fmt.Printf("Select end\n")
	return
}

// SelectRow execute query operations.
func (c ClientProxy) SelectRow(support string) (engine string, err error) {
	dest := []interface{}{&engine}
	ql := "SELECT ENGINE FROM ENGINES WHERE SUPPORT=?"
	err = c.QueryRow(context.Background(), dest, ql, support)
	return
}

// ExecTrans executing mysql transactions, Since the second table does not exist, Rollback.
func (c ClientProxy) ExecTrans() (err error) {
	// The framework automatically executes encapsulated fn transactions at the bottom of the framework.
	/*
	  BeginTx
	  exec fn // Note that the closure must use tx as a transaction join to operate the CRUD, otherwise you are not operating the CRUD in a transaction.
	  Commit Or Rollback.
	*/
	fn := func(tx *sql.Tx) (err error) {
		_, err = tx.Exec("UPDATE ENGINES SET XA='YES' WHERE ENGINE='InnoDB'")
		if err != nil {
			return
		}
		_, err = tx.Exec("UPDATE NOT_EXIST_TABLE SET XA='YES' WHERE ENGINE='InnoDB'")
		if err != nil {
			fmt.Printf("mysql error: %s\n", err.Error())
			return
		}
		return err
	}
	err = c.Transaction(context.Background(), fn)
	if err != nil {
		fmt.Printf("mysql.Transaction.err:%v\n", err)
	}
	return
}

const (
	Select      = "select"
	SelectRow   = "selectrow"
	Transaction = "transction"
)

func TestMain(_ *testing.T) {
	var (
		err    error
		engine string
		op     *string
	)
	op = flag.String("op", "select", "select, selectrow, transaction. default: select")
	flag.Parse()
	cli := NewMysqlClient()
	fmt.Printf("client op: %s\n", *op)
	switch *op {
	case Select:
		if err = cli.Select(); err != nil {
			fmt.Printf("mysql select failed. err: %s\n", err.Error())
		}
	case SelectRow:
		if engine, err = cli.SelectRow("xxx"); err != nil {
			fmt.Printf("mysql select row failed. err: %s\n", err.Error())
		}
		fmt.Printf("engine: %s\n", engine)
	case Transaction:
		if err = cli.ExecTrans(); err != nil {
			fmt.Printf("mysql exec transaction failed. err: %s\n", err.Error())
		}
	default:
		fmt.Println("This operation is not supported at this time")
	}
}

// NewMysqlClient Construct a mysql.Client for easy testing.
func NewMysqlClient() *ClientProxy {
	// Config https://github.com/go-sql-driver/mysql#examples.
	// The MOCK value is used here to indicate the user password example to avoid security
	target := "dsn://MOCK_SECRET_KEY_ID:MOCK_SECRET_KEY_VALUE@tcp(127.0.0.1:3306)/information_schema?timeout=10s"
	timeout := 10 * time.Second
	mysqlClient := mysql.NewClientProxy("trpc.mysql.examples.info",
		client.WithTarget(target), client.WithTimeout(timeout))
	return &ClientProxy{mysqlClient}
}
