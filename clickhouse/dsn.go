/**
 * @author aceyugong <aceyugong@tencent.com>
 * @date 2022/8/19
 */

// Package clickhouse encapsulates standard library clickhouse
package clickhouse

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// toV2DSN convert v1 DSN to v2 version.
// ref: v1: https://github.com/ClickHouse/clickhouse-go/tree/v1#dsn
// ref: v2: https://github.com/ClickHouse/clickhouse-go#dsn
func toV2DSN(dsn string) string {
	// username:password@host1:9000,host2:9000/database?dial_timeout=200ms&max_execution_time=60
	// host1:9000?username=*&password=*&database=*
	host, query := dsn, ""
	idx := strings.IndexByte(dsn, '?')
	if idx > -1 {
		query = dsn[idx+1:]
		if dsn[idx-1] == '/' {
			idx--
		}
		host = dsn[0:idx]
	}
	if idx := strings.LastIndexByte(host, '@'); idx > -1 {
		return dsn
	}
	if query == "" {
		return dsn
	}
	q, err := url.ParseQuery(query)
	if err != nil {
		return dsn
	}
	user, pswd, dbname := q.Get("username"), q.Get("password"), q.Get("database")
	q.Del("username")
	q.Del("password")
	q.Del("database")
	for _, k := range []string{"read_timeout", "write_timeout"} {
		if v := q.Get(k); v != "" {
			q.Set(k, addSecUnitForTimeout(v, "s"))
		}
	}
	remain := q.Encode()
	if remain != "" {
		remain = "?" + remain
	}
	return fmt.Sprintf("%s:%s@%s/%s%s", user, pswd, host, dbname, remain)
}

// addSecUnitForTimeout adds seconds unit.
func addSecUnitForTimeout(v, unit string) string {
	if v == "" {
		return ""
	}
	c := v[len(v)-1]
	if '0' <= c && c <= '9' {
		return v + unit
	}
	return v
}

// parseOptionsFromDSN parses configuration parameters from dsn.
func parseOptionsFromDSN(dsn string) (*options, string) {
	idx := strings.IndexByte(dsn, '?')
	if idx == -1 {
		return nil, dsn
	}
	host := dsn[0:idx]
	query := dsn[idx+1:]
	if query == "" {
		return nil, host
	}
	q, err := url.ParseQuery(query)
	if err != nil {
		return nil, dsn
	}
	opt := &options{}
	if v := q.Get("max_idle"); v != "" {
		q.Del("max_idle")
		opt.MaxIdle, _ = strconv.Atoi(v)
	}
	if v := q.Get("max_open"); v != "" {
		q.Del("max_open")
		opt.MaxOpen, _ = strconv.Atoi(v)
	}
	if v := q.Get("max_lifetime"); v != "" {
		q.Del("max_lifetime")
		opt.MaxLifetime, _ = time.ParseDuration(v)
	}
	remain := q.Encode()
	if remain != "" {
		remain = "?" + remain
	}
	return opt, host + remain
}
