// package clickhouse_test 测试
package clickhouse_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"trpc.group/trpc-go/trpc-database/clickhouse"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/client/mockclient"
)

var (
	dsnV1 = "clickhouse://127.0.0.1:9000?username=default&password=&database=db_storage&debug=true"
	dsnV2 = "clickhouse://default:@127.0.0.1:9000/db_storage?debug=true"
)

// go test -run TestExec
func TestExec(t *testing.T) {
	proxy := clickhouse.NewClientProxy(
		"trpc.clickhouse.server.service", client.WithTarget(dsnV1), client.WithTimeout(3*time.Second),
	)
	res, err := proxy.Exec(
		context.Background(), `
		CREATE TABLE IF NOT EXISTS example (
			country_code FixedString(2),
			os_id        UInt8,
			browser_id   UInt8,
			categories   Array(Int16),
			action_day   Date,
			action_time  DateTime
		) engine=Memory
	`,
	)
	if err != nil {
		t.Logf("clickhouse exec fail:%+v", err)
		return
	}
	t.Logf("clickhouse exec success, result:%+v", res)
}

type TableRow struct {
	Country    string    `db:"country_code"`
	Os         uint8     `db:"os_id"`
	Browser    uint8     `db:"browser_id"`
	Categories []int16   `db:"categories"`
	ActionDay  time.Time `db:"action_day"`
	ActionTime time.Time `db:"action_time"`
}

// go test -run TestQuery
func TestQuery(t *testing.T) {
	var results []TableRow
	next := func(rows *sql.Rows) error { // The bottom layer of the framework automatically calls the next function for loop rows.
		var item TableRow
		err := rows.Scan(&item.Country, &item.Os, &item.Browser, &item.Categories, &item.ActionDay, &item.ActionTime)
		if err != nil {
			t.Logf("clickhouse scan fail:%+v", err)
			return err
		}
		results = append(results, item)
		return nil
	}
	proxy := clickhouse.NewClientProxy(
		"trpc.clickhouse.server.service", client.WithTarget(dsnV1), client.WithTimeout(3*time.Second),
	)
	err := proxy.Query(
		context.Background(), next,
		"SELECT country_code, os_id, browser_id, categories, action_day, action_time FROM example LIMIT 5",
	)
	if err != nil {
		t.Logf("clickhouse query fail:%+v", err)
		return
	}

	t.Logf("clickhouse query success, results: %+v", results)
}

// go test -run TestQueryRow
func TestQueryRow(t *testing.T) {
	var item TableRow
	result := []interface{}{&item.Country, &item.Os, &item.Browser, &item.Categories, &item.ActionDay, &item.ActionTime}
	proxy := clickhouse.NewClientProxy(
		"trpc.clickhouse.server.service", client.WithTarget(dsnV2), client.WithTimeout(3*time.Second),
	)
	err := proxy.QueryRow(
		context.Background(), result,
		"SELECT country_code, os_id, browser_id, categories, action_day, action_time FROM example LIMIT 1",
	)
	if err != nil {
		t.Logf("clickhouse query fail:%+v", err)
		return
	}
	t.Logf("clickhouse query success, item:%+v", item)
}

// go test -run TestQueryToStructs
func TestQueryToStructs(t *testing.T) {
	var results []*TableRow
	proxy := clickhouse.NewClientProxy(
		"trpc.clickhouse.server.service", client.WithTarget(dsnV2), client.WithTimeout(3*time.Second),
	)
	err := proxy.QueryToStructs(
		context.Background(), &results,
		"SELECT country_code, os_id, browser_id, categories, action_day, action_time FROM example LIMIT 5",
	)
	if err != nil {
		t.Logf("clickhouse query fail:%+v", err)
		return
	}
	t.Logf("clickhouse query success, count: %d", len(results))
	for _, val := range results {
		t.Logf("%+v", val)
	}
}

// go test -run TestTransaction
func TestTransaction(t *testing.T) {
	ctl := gomock.NewController(t)
	defer ctl.Finish()
	mockClient := mockclient.NewMockClient(ctl)
	client.DefaultClient = mockClient
	defer func() {
		client.DefaultClient = client.New()
	}()
	// 预期行为
	mockClient.EXPECT().Invoke(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	ctx := context.Background()
	tx := func(tx *sql.Tx) error {
		stmt, err := tx.PrepareContext(
			ctx,
			"INSERT INTO example (country_code, os_id, browser_id, categories, action_day, action_time) VALUES (?,?,?,?,?,?)",
		)
		if err != nil {
			return err
		}
		defer stmt.Close()

		for i := 0; i < 100; i++ {
			if _, err := stmt.ExecContext(
				ctx,
				"CN",
				uint8(10+i),
				uint8(100+i),
				[]int16{4, 5, 6},
				time.Now(),
				time.Now(),
			); err != nil {
				return err
			}
		}
		return nil
	}

	proxy := clickhouse.NewClientProxy(
		"trpc.clickhouse.server.service", client.WithTarget(dsnV2), client.WithTimeout(3*time.Second),
	)
	err := proxy.Transaction(ctx, tx)
	if err != nil {
		t.Logf("clickhouse transaction error(%v)", err)
		t.FailNow()
	}
	t.Logf("clickhouse transaction success")
}
