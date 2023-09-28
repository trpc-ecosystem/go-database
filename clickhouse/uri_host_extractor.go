// Package clickhouse encapsulates standard library clickhouse.
package clickhouse

import (
	"strings"
)

// URIHostExtractor extracts host from URI for ip resolution
// (such as host and then query ip from Polaris), and use it with ResolvableSelector.
type URIHostExtractor struct {
}

// Extract input parameter uri needs to be processed by itself
// to remove the connection string "://" and the part before it.
// If it is a V2 connection string, return one or more hosts, if it is a V1 connection string, return a host.
// (V2 connection string format starts with clickhouse://, V1 connection string format starts with tcp://)
func (e *URIHostExtractor) Extract(uri string) (int, int, error) {
	// clickhouse://username:password@host1:9000,host2:9000/database?dial_timeout=200ms&max_execution_time=60
	// clickhouse+polaris://service_name?username=*&password=*&database=*
	idx := strings.IndexByte(uri, '?')
	if idx > -1 {
		if uri[idx-1] == '/' {
			idx--
		}
		uri = uri[0:idx]
	}
	begin := 0
	if idx := strings.LastIndexByte(uri, '@'); idx > -1 {
		begin = idx + 1
		uri = uri[begin:]
	}
	length := len(uri)
	if idx := strings.IndexByte(uri, '/'); idx > -1 {
		length = idx
	}
	return begin, length, nil
}
