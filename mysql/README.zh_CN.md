[English](README.md) | 中文

# tRPC-Go mysql 插件

[![Coverage](https://codecov.io/gh/trpc-ecosystem/go-database/branch/coverage/graph/badge.svg?flag=mysql&precision=2)](https://app.codecov.io/gh/trpc-ecosystem/go-database/tree/coverage/mysql)

## 封装标准库原生 sql

## Client 配置

```yaml
client: #客户端调用的后端配置
  service: #针对单个后端的配置
    - name: trpc.mysql.xxx.xxx
      target: dsn://${username}:${passwd}@tcp(${vip}:${port})/${db}?timeout=1s&parseTime=true&interpolateParams=true #mdb使用域名多实例需要加上 &interpolateParams=true
    - name: trpc.mysql.xxx.xxx
      #mysql+polaris表示target为uri，其中uri中的host会进行北极星解析，解析得到address(host:port)并替换polaris_name
      target: mysql+polaris://${username}:${passwd}@tcp(${polaris_name})/${db}?timeout=1s&parseTime=true&interpolateParams=true
```

## Unsafe 模式

日常使用中，我们偶尔会遇到 model struct 与查询的字段不匹配而报错的情况，比如：当执行的 sql 是 `select id, name, age from users limit 1;` ，而定义的 `struct` 只有 `id` 和 `name` 字段时，执行查询会报错 `missing destination age in *User`。

要解决上面说的问题，可以通过 `NewUnsafeClient()` 得到一个开启了 sqlx unsafe 模式的 Client 进行查询调用，在字段和结构体不匹配时就不会报错，只查满足条件的字段。详细请参考 [sqlx Safety 文档](https://jmoiron.github.io/sqlx/#safety)。

Unsafe 模式不作用于原生 `Exec` / `Query` / `QueryRow` / `Transaction`，它们没有将 model struct 与表数据映射的过程。Unsafe 模式对这几个方法也不会产生副作用。

> **`注意：Unsafe 模式可能会隐藏掉非预期的字段定义错误，使用时需谨慎。`**

## Usage
```go
package main

import (
	"context"
	"time"

	"trpc.group/trpc-go/trpc-database/mysql"
)

// QAModuleTag 对应数据库字段的结构体，结构体字段名自己定义，右边的tag为数据库里面的字段名
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
	proxy := mysql.NewClientProxy("trpc.mysql.xxx.xxx") // service name自己随便填，主要用于监控上报和寻找配置项，必须要跟client配置的name一致
	unsafeProxy := mysql.NewUnsafeClient("trpc.mysql.xxx.xxx")

	// 以下是通过原生 sql.DB 实现的方法
	// 插入数据，所有参数都使用 ? 占位符，避免 sql 注入攻击
	_, err = proxy.Exec(ctx, "INSERT INTO qa_module_tags (game_id, tag_name) VALUES (?, ?), (?, ?)", 1, "tag1", 2, "tag2")
	if err != nil {
		return err
	}

	// 更新数据
	_, err = proxy.Exec(ctx, "UPDATE qa_module_tags SET tag_name = ? WHERE game_id = ?", "tag11", 1)
	if err != nil {
		return err
	}

	// 读取单条数据（读取数据到 []field 结构中）
	var id int64
	var name string
	dest := []interface{}{&id, &name}
	// 也可以使用 struct 字段的形式赋值
	// instance := new(QAModuleTag)
	// dest := []interface{}{&instance.ID, &instance.TagName}
	err = proxy.QueryRow(ctx, dest, "SELECT id, tag_name FROM qa_module_tags LIMIT 1")
	if err != nil {
		// 判断是否查询到记录
		if mysql.IsNoRowsError(err) {
			return nil
		}
		return
	}
	// 使用查询到的值
	_, _ = id, name
	// 如果是使用 struct 字段的形式，则使用：
	// _, _ = instance.ID, instance.TagName

	// 使用 sql.Tx 事务
	// 定义事务执行函数 fn，当 fn 返回的 error 为 nil 时，事务自动提交，否则事务自动回滚。
	// 注意：fn 中的 db 操作需要使用 tx 执行，否则不是事务操作。
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
	// 以下是通过 sqlx 实现的方法
	// 读取单条数据（读取数据到 struct 结构中）
	tag := QAModuleTag{}
	err := proxy.QueryToStruct(ctx, &tag, "SELECT tag_name FROM qa_module_tags WHERE id = ?", 1)
	if err != nil {
		// 判断是否查询到记录
		if mysql.IsNoRowsError(err) {
			return nil
		}
		return
	}

	// 使用查询到的值
	println(tag.TagName)

	// 读取数据，select 字段尽量只 select 自己关心的字段，不要用 *，以下只是简单示例
	// 如果使用 *，可能会因为找不到部分结构体字段或部分字段类型不匹配（如 NULL）报错
	var tags []*QAModuleTag
	err := proxy.QueryToStructs(ctx, &tags, "SELECT * FROM qa_module_tags WHERE parent_id = 0")
	if err != nil {
		return err
	}

	// 如果 model 结构和查询的 DB 字段不匹配，比如 model 中出现了表中不存在的字段，又比如表中加字段了但 model 中未定义，这种情况查询时默认会报错。
	// 如果不希望报错，需要使用 NewUnsafeClient() 得到的 Client 进行操作，如：
	err = unsafeProxy.QueryToStructs(ctx, &tags, "SELECT * FROM qa_module_tags WHERE parent_id = 0")
	if err != nil {
        return err
    }

	// 通过参数查询单条数据，读取到 struct 或常规类型中 (可替代 QueryToStruct)
	// 如果给的 dest 是 struct，但是查询出来多条数据，则只会读取第一条数据
	tag := QAModuleTag{}
	err = proxy.Get(ctx, &tag, "SELECT * FROM qa_module_tags WHERE id = ? AND tag_name = ?", 10, "Foo")
	if err != nil {
		if mysql.IsNoRowsError(err) {
			return nil
		}
		return
	}

	// 可以用 Get 来进行 count 操作
	var c int
	err = db.Get(&c, "SELECT COUNT(*) FROM qa_module_tags WHERE id > ?", 10)
	if err != nil {
		return err
	}

	// 通过参数查询多条数据，读取到 struct 数组中 (可替代 QueryToStructs)
	tags := []QAModuleTag{}
	err = proxy.Select(ctx, &tags, "SELECT * FROM qa_module_tags WHERE id > ?", 99)
	if err != nil {
		return err
	}

	// 通过 struct 或 map 绑定 SQL 同名字段参数查询数据
	ql := "SELECT * from qa_module_tags WHERE id = :id AND tag_name = :tag_name"
	rows, err := proxy.NamedQuery(ctx, ql, QAModuleTag{id: 10, name :"Foo"})
	// rows, err := proxy.NamedQuery(ctx, ql, map[string]interface{}{"id": 10, "name": "Foo"})
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
		// 业务逻辑处理
	}

	// 使用字段映射插入数据
	_, err = proxy.NamedExec(ctx, "INSERT INTO qa_module_tags (game_id, tag_name) VALUES (:game_id, :tag_name)", &QAModuleTag{GameID: 1, TagName: "tagxxx"})
	if err != nil {
		return err
	}

	// 使用字段映射批量插入数据
	_, err = proxy.NamedExec(ctx, "INSERT INTO qa_module_tags (game_id, tag_name) VALUES (:game_id, :tag_name)", []QAModuleTag{{GameID: 1, TagName: "tagxxx"}, {GameID: 2, TagName: "tagyyy"}})
	if err != nil {
		return err
	}

	// 使用 sqlx.Tx 事务
	// 定义事务执行函数 fn，当 fn 返回的 error 为 nil 时，事务自动提交，否则事务自动回滚。
	// 注意：fn 中的 db 操作需要使用 tx 执行，否则不是事务操作。
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

## Plugin 配置

目前通过配置 `trpc_go.yaml` 文件进行默认配置加载，如下所示：

```yaml
plugins: # 插件配置
  database:
    mysql:
      max_idle: 20 # 最大空闲连接数
      max_open: 100 # 最大在线连接数
      max_lifetime: 180000 # 连接最大生命周期 (单位：毫秒)
```

## FAQ

1. MYSQL 错误信息：`Error 1243: Unknown prepared statement handler (1) given to mysqld_stmt_execute`

> 答：使用 dsn 连接 mysql server 增加连接参数 _&interpolateParams=true_ 可以解决问题，如：
>
> 出问题的 DSN：
```
"dsn://root:123456@tcp(127.0.0.1:3306)/databasesXXX?timeout=1s&parseTime=true"
```
> 解决方式：
```
"dsn://root:123456@tcp(127.0.0.1:3306)/databasesXXX?timeout=1s&parseTime=true&interpolateParams=true"
```
>
> `interpolateParams` 参数说明：当开启该参数时，则该库除了 BIG5, CP932, GB2312, GBK or SJIS 不能防注入外，其他都可以。具体参见：https://github.com/go-sql-driver/mysql#interpolateparams
>
> 第二种解决方案： 比如像gorm、xorm第三方库基本上都是在client端就把sql语句和占位符、参数构建成一个完整的句子后，再发送到mysql server处理，免去了在服务端进行处理。但是目前还没有找到对所有编码防注入的第三方库。目前go-driver-sql库已经完全够用。
>
> 具体原因: 当interpolateParams为false时,  mysql server会对所有的sql语句进行db.Prepare、db.Exec/db.Query两步骤处理。前者主要用于构建sql语法树，然后后者提交时只需要对占位符进行防注入处理以及参数补充。就可以构建成一个完整可执行的sql语句。而go-driver-sql库的Query/Exec本身是有Prepare处理的，但是mysql server有集群、主从模式读写分离。如果使用host去绑定多个实例，比如有A 、B两个mysql server实例，如果第一次请求db.Prepare到达了实例A，当第二个网络请求db.Exec/db.Query若到达了实例B，同时A的db.Prepare语句还没有同步到实例B，则实例B接收到db.Exec/db.Query请求时，认为还没有进行db.Prepare处理过，所以就会报上面的错误，如果时mysql集群、主从分离有VIP，则不会出现上述错误。

2. 服务偶现 `invalid connection` 错误

> 答：该插件对 golang 和 go-sql-driver 版本有依赖，golang>=1.10, 且 go-sql-driver>=1.5.0

3. Query 错误: `unsupported Scan, storing driver.Value type []uint8 into type *time.Time`

> 答：在连接 DNS 字符串参数加上 _parseTime=true_, 用于支持 time.Time 解析

4. Transaction/Transactionx 事务操作异常。比如事务显示回滚，而实际上对数据操作有部分成功

> 答：很有可能是传递 CRUD 的自定义事务闭包函数 fn 里面没有使用传递进来的 \*sql.Tx/\*sqlx.Tx 变量, 直接使用了外部的 client，导致 CRUD 操作并没有在事务中执行，所以某些操作执行成功没有发生实际的回滚现象。而错误显示回滚了, 因为 fn 闭包函数执行异常。

5. 在使用 `QueryToStruct` 时遇到 `Unknown column 'xxx' in 'field list` 或 `missing destination name ...`

> 答：这是因为 SQL 条件中 select 的字段和结构体不匹配。我们建议不要使用 `select *`，平时写业务时，保持 model 和 表字段的一致，通过 DB 类库的约束可以帮助我们写出更健壮的代码。
>
> 如果你有特殊场景，如动态拼接查询条件、反射动态 model 等诉求，可以参考本文档的 `Unsafe 模式` 章节。

## 参考资料

1. [issues:interpolateParams](https://github.com/go-sql-driver/mysql/issues/413)
2. [Mysql 读写分离+防止 sql 注入攻击「GO 源码剖析」](https://zhuanlan.zhihu.com/p/111682902)
3. [interpolateparams 参数说明](https://github.com/go-sql-driver/mysql#interpolateparams)
4. [Illustrated guide to SQLX](https://jmoiron.github.io/sqlx/)
