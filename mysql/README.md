English | [中文](README.zh_CN.md)

# tRPC-Go mysql plugin

[![Go Reference](https://pkg.go.dev/badge/trpc.group/trpc-go/trpc-database/mysql.svg)](https://pkg.go.dev/trpc.group/trpc-go/trpc-database/mysql)
[![Go Report Card](https://goreportcard.com/badge/trpc.group/trpc-go/trpc-database/mysql)](https://goreportcard.com/report/trpc.group/trpc-go/trpc-database/mysql)
[![Tests](https://github.com/trpc-ecosystem/go-database/actions/workflows/mysql.yml/badge.svg)](https://github.com/trpc-ecosystem/go-database/actions/workflows/mysql.yml)
[![Coverage](https://codecov.io/gh/trpc-ecosystem/go-database/branch/coverage/graph/badge.svg?flag=mysql&precision=2)](https://app.codecov.io/gh/trpc-ecosystem/go-database/tree/coverage/mysql)

## Wrapping Standard Library Native Sql

## Client config

```yaml
client: #Back-end configuration for client calls.
  service: #Configuration for a single back-end.
    - name: trpc.mysql.xxx.xxx
      target: dsn://${username}:${passwd}@tcp(${vip}:${port})/${db}?timeout=1s&parseTime=true&interpolateParams=true #mdb needs to add &interpolateParams=true to use multiple instances of the domain.
    - name: trpc.mysql.xxx.xxx
      #mysql+polaris means target is uri, where host in uri will be parsed by Polaris to get address(host:port) and replace polaris_name.
      target: mysql+polaris://${username}:${passwd}@tcp(${polaris_name})/${db}?timeout=1s&parseTime=true&interpolateParams=true
```

## Unsafe Mode

For example, when the execution sql is `select id, name, age from users limit 1;` and the defined `struct` has only `id` and `name` fields, the execution query will report the error ` missing destination age in *User`.

To solve the problem mentioned above, you can get a Client with sqlx unsafe mode enabled through `NewUnsafeClient()` to make a query call, so that if the fields and structures do not match, no error will be reported and only the fields that meet the conditions will be checked. Please refer to the [sqlx Safety documentation](https://jmoiron.github.io/sqlx/#safety) for details.

Unsafe mode does not work with native `Exec` / `Query` / `QueryRow` / `Transaction`, which do not map model structs to table data, nor does Unsafe mode have side effects on these methods.

> **`Note: Unsafe mode may hide unintended field definition errors and should be used with caution.`**

## Usage
```go
package main

import (
	"context"
	"time"

	"trpc.group/trpc-go/trpc-database/mysql"
)

// The QAModuleTag corresponds to the structure of the database field, the field name of the structure is self-defined, the tag on the right is the name of the field inside the database.
type QAModuleTag struct {
	ID         int64     `db:"id"`
	GameID     int64     `db:"game_id"`
	TagName    string    `db:"tag_name"`
	Sequence   int16     `db:"sequence"`
	ParentID   int16     `db:"parent_id"`
	QaDuration int64     `db:"qa_duration"`
	Remark     string    `db:"remark"`
	IsDeleted  int16     `db:"is_deleted"`
	CreateTime time.Time `db:"create_time"`
	UpdateTime time.Time `db:"update_time"`
}

func (s *server) SayHello(ctx context.Context, req *pb.ReqBody, rsp *pb.RspBody) (err error) {
	proxy := mysql.NewClientProxy("trpc.mysql.xxx.xxx") // The service name is randomly filled in by yourself, mainly used for monitoring and reporting and finding configuration items, must be consistent with the name of the client configuration.
	unsafeProxy := mysql.NewUnsafeClient("trpc.mysql.xxx.xxx")

	// Here's how it's done with native sql.DB.
	// Insert data with all parameters using ? placeholder to avoid sql injection attacks.
	_, err = proxy.Exec(ctx, "INSERT INTO qa_module_tags (game_id, tag_name) VALUES (?, ?), (?, ?)", 1, "tag1", 2, "tag2")
	if err != nil {
		return err
	}

	// Update data
	_, err = proxy.Exec(ctx, "UPDATE qa_module_tags SET tag_name = ? WHERE game_id = ?", "tag11", 1)
	if err != nil {
		return err
	}

	// Read a single piece of data (read data into []field structure).
	var id int64
	var name string
	dest := []interface{}{&id, &name}
	// you can also assign values in the form of struct fields.
	// instance := new(QAModuleTag).
	// dest := []interface{}{&instance.ID, &instance.TagName}.
	err = proxy.QueryRow(ctx, dest, "SELECT id, tag_name FROM qa_module_tags LIMIT 1")
	if err != nil {
		// Determine if the record is queried.
		if mysql.IsNoRowsError(err) {
			return nil
		}
		return
	}
	// Use the queried value.
	_, _ = id, name
	// If the form of a struct field is used, then use:
	// _, _ = instance.ID, instance.TagName.

	// Use sql.Tx transactions.
	// Define the transaction execution function fn. When the error returned by fn is nil, the transaction is automatically committed, otherwise the transaction is automatically rolled back.
	// Note: db operations in fn need to be executed using tx, otherwise they are not transactions.
	fn := func(tx *sql.Tx) (err error) {
		ql := "INSERT INTO qa_module_tags (game_id, tag_name) VALUES (?, ?), (?, ?)"
		if _, err = tx.Exec(ql, 1, "tag1", 2, "tag2"); err != nil {
			return
		}
		ql = "UPDATE qa_module_tags SET tag_name = ? WHERE game_id = ?"
		if _, err = tx.Exec(ql, "tag11", 1); err != nil {
			return
		}
		return
	}
	if err = proxy.Transaction(ctx, fn); err != nil {
		return
	}

	// -------------------------------------------------------------------------
	// Here's how to do it via sqlx.
	// Read a single piece of data (read data into a struct).
	tag := QAModuleTag{}
	err := proxy.QueryToStruct(ctx, &tag, "SELECT tag_name FROM qa_module_tags WHERE id = ?", 1)
	if err != nil {
		// Determine if the record is queried.
		if mysql.IsNoRowsError(err) {
			return nil
		}
		return
	}

	// Use the queried value.
	println(tag.TagName)

	// Read the data, select the fields you care about as much as possible, do not use *, the following is just a simple example.
	// If you use *, you may get an error because some of the structure fields are not found or some of the field types do not match (e.g. NULL).
	var tags []*QAModuleTag
	err := proxy.QueryToStructs(ctx, &tags, "SELECT * FROM qa_module_tags WHERE parent_id = 0")
	if err != nil {
		return err
	}

	// If the model structure and the query DB fields do not match, for example, there are fields in the model that do not exist in the table, or fields are added to the table but not defined in the model, the query will report an error by default.
	// If you do not want to report an error, you need to use the Client obtained by NewUnsafeClient() to perform the operation, e.g:
	err = unsafeProxy.QueryToStructs(ctx, &tags, "SELECT * FROM qa_module_tags WHERE parent_id = 0")
	if err != nil {
        return err
    }

	// Query a single piece of data by parameter, read into struct or regular type (can replace QueryToStruct).
	// If the given dest is struct, but the query returns multiple data, only the first data will be read.
	tag := QAModuleTag{}
	err = proxy.Get(ctx, &tag, "SELECT * FROM qa_module_tags WHERE id = ? AND tag_name = ?", 10, "Foo")
	if err != nil {
		if mysql.IsNoRowsError(err) {
			return nil
		}
		return
	}

	// You can use Get for count operations.
	var c int
	err = db.Get(&c, "SELECT COUNT(*) FROM qa_module_tags WHERE id > ?", 10)
	if err != nil {
		return err
	}

	// Query multiple data by parameters, read into struct array (can be used instead of QueryToStructs).
	tags := []QAModuleTag{}
	err = proxy.Select(ctx, &tags, "SELECT * FROM qa_module_tags WHERE id > ?", 99)
	if err != nil {
		return err
	}

	// Query data by struct or map binding SQL with same name field parameters.
	ql := "SELECT * from qa_module_tags WHERE id = :id AND tag_name = :tag_name"
	rows, err := proxy.NamedQuery(ctx, ql, QAModuleTag{id: 10, name :"Foo"})
	// rows, err := proxy.NamedQuery(ctx, ql, map[string]interface{}{"id": 10, "name": "Foo"}).
	if err != nil {
		return nil
	}
	defer rows.Close()
	for rows.Next() {
		var tag QAModuleTag
		err = rows.StructScan(&tag)
		if err != nil {
			return err
		}
		// Business Logic Processing.
	}

	// Inserting data using field mapping.
	_, err = proxy.NamedExec(ctx, "INSERT INTO qa_module_tags (game_id, tag_name) VALUES (:game_id, :tag_name)", &QAModuleTag{GameID: 1, TagName: "tagxxx"})
	if err != nil {
		return err
	}

	// Bulk data insertion using field mapping.
	_, err = proxy.NamedExec(ctx, "INSERT INTO qa_module_tags (game_id, tag_name) VALUES (:game_id, :tag_name)", []QAModuleTag{{GameID: 1, TagName: "tagxxx"}, {GameID: 2, TagName: "tagyyy"}})
	if err != nil {
		return err
	}

	// Use sqlx.Tx transaction.
	// Define the transaction execution function fn. When the error returned by fn is nil, the transaction is automatically committed, otherwise the transaction is automatically rolled back.
	// Note: db operations in fn need to be executed using tx, otherwise they are not transactions.
	fn := func(tx *sqlx.Tx) (err error) {
		ql := "INSERT INTO qa_module_tags (game_id, tag_name) VALUES (:game_id, :tag_name)"
		if _, err = tx.NamedExec(ctx, ql, []QAModuleTag{{GameID: 1, TagName: "tagxxx"}, {GameID: 2, TagName: "tagyyy"}}); err != nil {
			return err
		}
		ql = "UPDATE qa_module_tags SET tag_name = ? WHERE game_id = ?"
		if _, err = tx.Exec(ql, "tag11", 1); err != nil {
			return
		}
		return
	}
	if err = proxy.Transactionx(ctx, fn); err != nil {
		return
	}

	return
}
```

## Plugin Config

The default configuration is currently loaded by configuring the `trpc_go.yaml` file, as follows:

```yaml
plugins: # Plugin Configuration.
  database:
    mysql:
      max_idle: 20 # Maximum number of idle connections.
      max_open: 100 # Maximum number of online connections.
      max_lifetime: 180000 # Maximum connection lifecycle (in milliseconds).
```

## FAQ

1. MYSQL error message:`Error 1243: Unknown prepared statement handler (1) given to mysqld_stmt_execute`

> A: Using dsn to connect to mysql server, add connection parameters _&interpolateParams=true_ can solve the problem, e.g:
>
> Wrong DSN:
```
"dsn://root:123456@tcp(127.0.0.1:3306)/databasesXXX?timeout=1s&parseTime=true"
```
> Solution:
```
"dsn://root:123456@tcp(127.0.0.1:3306)/databasesXXX?timeout=1s&parseTime=true&interpolateParams=true"
```
>
> `interpolateParams` Parameter Description: When this parameter is enabled, the library can be anti-injection except for BIG5, CP932, GB2312, GBK or SJIS. For details, see: https://github.com/go-sql-driver/mysql#interpolateparams.
>
> Second solution: For example, third-party libraries like gorm and xorm basically build the sql statement with placeholders and parameters into a complete sentence on the client side and then send it to mysql server for processing, eliminating the need to process it on the server side. However, no third-party library has been found for all coding anti-injection. Currently the go-driver-sql library is fully sufficient.
>
> The specific reason: When interpolateParams is false, mysql server will process all sql statements in two steps: db.Prepare, db.Exec/db.Query The former is mainly used to build the sql syntax tree, and then the latter is submitted with only anti-injection processing of placeholders and additional parameters. It can be built into a complete executable sql statement. The Query/Exec of the go-driver-sql library itself has Prepare processing, but mysql server has clustering, master-slave mode read/write separation. If you use host to bind multiple instances, for example, there are two mysql server instances A and B. If the first request db.Prepare reaches instance A, when the second network request db.Exec/db.Query reaches instance B, and at the same time A's db.Prepare statement has not been synchronized to instance B, then instance B receives the db. Query request, it thinks it has not been processed by db.Prepare, so it will report the above error.

2. Service Occasional `invalid connection` error

> A: The plugin has dependency on golang and go-sql-driver version, golang>=1.10, and go-sql-driver>=1.5.0.

3. Query error: `unsupported Scan, storing driver.Value type []uint8 into type *time.Time`.

> A: Add _parseTime=true_ to the connection DNS string parameter, to support time.

4. Transaction/Transactionx Transaction operation exception. For example, the transaction shows a rollback when in fact the operation on the data was partially successful.

> A: It is likely that the custom transaction closure function fn that passes CRUD does not use the \*sql.Tx/\*sqlx.Tx variables passed in, and directly uses the external client, resulting in CRUD operations are not executed in the transaction, so some operations are executed successfully without the actual rollback phenomenon. Instead, the error shows that it was rolled back because the fn closure function was executed with an exception.

5. When using `QueryToStruct` I encountered `unknown column 'xxx' in the 'field list' or `missing destination name... `.

> A: This is because the fields of select and the structure do not match in the SQL condition. We recommend not to use `select *`, but to keep the model and table fields consistent when writing business, and to help us write more robust code by using the constraints of DB library.
>
> If you have special scenarios, such as dynamic splicing of query conditions, reflection of dynamic models, etc., you can refer to the `Unsafe Patterns` section of this document.

## References

1. [issues:interpolateParams](https://github.com/go-sql-driver/mysql/issues/413).
2. [Mysql read-write separation + prevention of sql injection attacks "GO Source Code Analysis"](https://zhuanlan.zhihu.com/p/111682902).
3. [interpolateparams Parameter Description](https://github.com/go-sql-driver/mysql#interpolateparams).
4. [Illustrated guide to SQLX](https://jmoiron.github.io/sqlx/).
