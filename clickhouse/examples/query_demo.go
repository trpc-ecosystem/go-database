// Package main is the main package.
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"time"

	"trpc.group/trpc-go/trpc-database/clickhouse"
	"trpc.group/trpc-go/trpc-go/client"
)

// ClientProxy defines the clickhouse.Client type.
type ClientProxy struct {
	clickhouse.Client
}

// Select retrieves data from app_pos_type_flowcontrol_20170312, puts the result in data and returns.
// How to use: Select(NewClickHouseClient()), refer to Test_Select of query_demo_test.go.
// When writing a single test: Select(clickhouse.Client),
// refer to Test_Select_MockClickHouse of query_demo_test.go.
func (c ClientProxy) Select() (err error) {
	fmt.Printf("Select begin\n")
	var (
		engine string
		xa     string
	)
	// The bottom layer of the framework automatically calls the next function for loop rows.
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
		fmt.Printf("clickhouse.Query.err:%v\n", err)
	}
	fmt.Printf("Select end\n")
	return
}

// SelectRow performs query operations.
func (c ClientProxy) SelectRow(support string) (engine string, err error) {
	dest := []interface{}{&engine}
	ql := "SELECT ENGINE FROM ENGINES WHERE SUPPORT=?"
	err = c.QueryRow(context.Background(), dest, ql, support)
	return
}

// ExecTrans executes the clickhouse transaction,
// because the second table does not exist, Rollback.
func (c ClientProxy) ExecTrans() (err error) {
	// The bottom layer of the framework automatically executes the encapsulated fn transaction operations.
	/*
	  BeginTx
	  exec fn // Note that tx must be used as a transaction connection to operate CRUD in the closure,
	          // otherwise you do not operate CRUD in the transaction.
	  Commit Or Rollback
	*/
	fn := func(tx *sql.Tx) (err error) {
		_, err = tx.Exec("UPDATE ENGINES SET XA='YES' WHERE ENGINE='InnoDB'")
		if err != nil {
			return
		}
		_, err = tx.Exec("UPDATE NOT_EXIST_TABLE SET XA='YES' WHERE ENGINE='InnoDB'")
		if err != nil {
			fmt.Printf("clickhouse error: %s\n", err.Error())
			return
		}
		return err
	}
	err = c.Transaction(context.Background(), fn)
	if err != nil {
		fmt.Printf("clickhouse.Transaction.err:%v\n", err)
	}
	return
}

const (
	selectCMD      = "select"
	selectRowCMD   = "selectrow"
	transactionCMD = "transction"
)

func init() {
}

// demo verification.
func main() {
	var (
		err    error
		engine string
		op     *string
	)
	op = flag.String("op", "select", "select, selectrow, transaction. default: select")
	flag.Parse()
	cli := NewClickHouseClient()
	fmt.Printf("client op: %s\n", *op)
	switch *op {
	case selectCMD:
		if err = cli.Select(); err != nil {
			fmt.Printf("clickhouse select failed. err: %s\n", err.Error())
		}
	case selectRowCMD:
		if engine, err = cli.SelectRow("xxx"); err != nil {
			fmt.Printf("clickhouse select row failed. err: %s\n", err.Error())
		}
		fmt.Printf("engine: %s\n", engine)
	case transactionCMD:
		if err = cli.ExecTrans(); err != nil {
			fmt.Printf("clickhouse exec transaction failed. err: %s\n", err.Error())
		}
	default:
		fmt.Println("暂不支持该操作")
	}
}

// NewClickHouseClient constructs a clickhouse.Client for easy testing.
func NewClickHouseClient() *ClientProxy {
	target := "dsn://MOCK_SECRET_KEY_ID:MOCK_SECRET_KEY_VALUE@tcp(127.0.0.1:3306)/information_schema?timeout=10s"
	timeout := 10 * time.Second
	clickhouseClient := clickhouse.NewClientProxy("trpc.clickhouse.examples.info",
		client.WithTarget(target), client.WithTimeout(timeout))
	return &ClientProxy{clickhouseClient}
}
