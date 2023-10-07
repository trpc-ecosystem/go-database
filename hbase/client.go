// Package hbase 封装标准库hbase
package hbase

import (
	"context"
	"errors"
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/tsuna/gohbase/hrpc"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
)

// ErrRowKeyNotExist RowKey不存在
var ErrRowKeyNotExist = errors.New("ErrRowKeyNotExist")

// Client HBase查询接口
type Client interface {
	Get(ctx context.Context, table, rowKey string, family map[string][]string) (*hrpc.Result, error)
	Put(ctx context.Context, table, rowKey string, value map[string]map[string][]byte) (*hrpc.Result, error)
	Del(ctx context.Context, table, rowKey string, value map[string]map[string][]byte) (*hrpc.Result, error)
	RawGet(ctx context.Context, get *hrpc.Get) (*hrpc.Result, error)
	RawPut(ctx context.Context, put *hrpc.Mutate) (*hrpc.Result, error)
	RawDel(ctx context.Context, del *hrpc.Mutate) (*hrpc.Result, error)
	RawScan(ctx context.Context, scan *hrpc.Scan) (hrpc.Scanner, error)
	RawAppend(ctx context.Context, a *hrpc.Mutate) (*hrpc.Result, error)
	RawInc(ctx context.Context, inc *hrpc.Mutate) (int64, error)
	RawCheckAndPut(ctx context.Context, put *hrpc.Mutate,
		family string, qualifier string, expectedValue []byte) (bool, error)
	RawClose(ctx context.Context) error
}

// hbaseCli hbase client
type hbaseCli struct {
	ServiceName string
	Client      client.Client
	opts        []client.Option
}

// NewClientProxy 新建一个hbase后端请求代理 必传参数 hbase服务名: trpc.hbase.xxx.xxx
var NewClientProxy = func(name string, opts ...client.Option) Client {
	c := &hbaseCli{
		ServiceName: name,
		Client:      client.DefaultClient,
	}

	c.opts = make([]client.Option, 0, len(opts)+2)
	c.opts = append(c.opts, opts...)
	c.opts = append(c.opts, client.WithProtocol("hbase"), client.WithDisableServiceRouter())

	logrus.SetOutput(defaultlogger)
	return c
}

// Request hbase request body
type Request struct {
	Command       string
	Table         string
	RowKey        string
	Family        map[string][]string
	Value         map[string]map[string][]byte
	Get           *hrpc.Get
	Put           *hrpc.Mutate
	Del           *hrpc.Mutate
	Scan          *hrpc.Scan
	Append        *hrpc.Mutate
	Inc           *hrpc.Mutate
	FamilyStr     string
	QualifierStr  string
	ExpectedValue []byte
}

// Response hbase response body
type Response struct {
	Result         *hrpc.Result
	Scanner        hrpc.Scanner
	IncVal         int64
	CheckAndPutVal bool
}

// Get 读取指定table，指定rowKey，如果指定filter，返回指定对应的列簇的列；否则返回所有列簇的列值
func (c *hbaseCli) Get(ctx context.Context, table, rowKey string,
	family map[string][]string) (*hrpc.Result, error) {
	hbaseReq := &Request{
		Command: get,
		Table:   table,
		RowKey:  rowKey,
		Family:  family,
	}
	hbaseRsp := &Response{}
	if err := c.invoke(ctx, hbaseReq, hbaseRsp); err != nil {
		return nil, err
	}
	return hbaseRsp.Result, nil
}

// ParseGetResult 读取hbase内容
func ParseGetResult(result *hrpc.Result, err error) (map[string]map[string]string, error) {
	if err != nil {
		return nil, err
	}

	if len(result.Cells) == 0 {
		return nil, ErrRowKeyNotExist
	}

	data := make(map[string]map[string]string)
	for _, cell := range result.Cells {
		if _, ok := data[string(cell.Family)]; !ok {
			data[string(cell.Family)] = make(map[string]string)
		}

		data[string(cell.Family)][string(cell.Qualifier)] = string(cell.Value)
	}

	return data, nil
}

// Put 向表table中的行键rowKey中写入value， value含义为 列族 -> [列 -> 值]
func (c *hbaseCli) Put(ctx context.Context, table, rowKey string,
	value map[string]map[string][]byte) (*hrpc.Result, error) {
	hbaseReq := &Request{
		Command: put,
		Table:   table,
		RowKey:  rowKey,
		Value:   value,
	}
	hbaseRsp := &Response{}
	if err := c.invoke(ctx, hbaseReq, hbaseRsp); err != nil {
		return nil, err
	}
	return hbaseRsp.Result, nil
}

// Del 删除表table中的行键rowKey中的value， value含义为 列族 -> [列 -> 值]
func (c *hbaseCli) Del(ctx context.Context, table, rowKey string,
	value map[string]map[string][]byte) (*hrpc.Result, error) {
	hbaseReq := &Request{
		Command: del,
		Table:   table,
		RowKey:  rowKey,
		Value:   value,
	}
	hbaseRsp := &Response{}
	if err := c.invoke(ctx, hbaseReq, hbaseRsp); err != nil {
		return nil, err
	}
	return hbaseRsp.Result, nil
}

// RawGet hbase 原生 Get api
func (c *hbaseCli) RawGet(ctx context.Context, get *hrpc.Get) (*hrpc.Result, error) {
	hbaseReq := &Request{
		Command: rawGet,
		Get:     get,
	}
	hbaseRsp := &Response{}
	if err := c.invoke(ctx, hbaseReq, hbaseRsp); err != nil {
		return nil, err
	}
	return hbaseRsp.Result, nil
}

// RawPut hbase 原生 Put api
func (c *hbaseCli) RawPut(ctx context.Context, put *hrpc.Mutate) (*hrpc.Result, error) {
	hbaseReq := &Request{
		Command: rawPut,
		Put:     put,
	}
	hbaseRsp := &Response{}
	if err := c.invoke(ctx, hbaseReq, hbaseRsp); err != nil {
		return nil, err
	}
	return hbaseRsp.Result, nil
}

// RawDel hbase 原生 del api
func (c *hbaseCli) RawDel(ctx context.Context, del *hrpc.Mutate) (*hrpc.Result, error) {
	hbaseReq := &Request{
		Command: rawDel,
		Del:     del,
	}
	hbaseRsp := &Response{}
	if err := c.invoke(ctx, hbaseReq, hbaseRsp); err != nil {
		return nil, err
	}
	return hbaseRsp.Result, nil
}

// RawScan hbase 原生 scan api
func (c *hbaseCli) RawScan(ctx context.Context, scan *hrpc.Scan) (hrpc.Scanner, error) {
	hbaseReq := &Request{
		Command: rawScan,
		Scan:    scan,
	}
	hbaseRsp := &Response{}
	if err := c.invoke(ctx, hbaseReq, hbaseRsp); err != nil {
		return nil, err
	}
	return hbaseRsp.Scanner, nil
}

// RawAppend hbase 原生 Append api
func (c *hbaseCli) RawAppend(ctx context.Context, a *hrpc.Mutate) (*hrpc.Result, error) {
	hbaseReq := &Request{
		Command: rawAppend,
		Append:  a,
	}
	hbaseRsp := &Response{}
	if err := c.invoke(ctx, hbaseReq, hbaseRsp); err != nil {
		return nil, err
	}
	return hbaseRsp.Result, nil
}

// RawInc hbase 原生 Increment api
func (c *hbaseCli) RawInc(ctx context.Context, inc *hrpc.Mutate) (int64, error) {
	hbaseReq := &Request{
		Command: rawInc,
		Inc:     inc,
	}
	hbaseRsp := &Response{}
	if err := c.invoke(ctx, hbaseReq, hbaseRsp); err != nil {
		return 0, err
	}
	return hbaseRsp.IncVal, nil
}

// RawCheckAndPut hbase 原生 CheckAndPut api
func (c *hbaseCli) RawCheckAndPut(ctx context.Context, put *hrpc.Mutate,
	family string, qualifier string, expectedValue []byte) (bool, error) {
	hbaseReq := &Request{
		Command:       rawCheckAndPut,
		Put:           put,
		FamilyStr:     family,
		QualifierStr:  qualifier,
		ExpectedValue: expectedValue,
	}
	hbaseRsp := &Response{}
	if err := c.invoke(ctx, hbaseReq, hbaseRsp); err != nil {
		return false, err
	}
	return hbaseRsp.CheckAndPutVal, nil
}

// RawClose hbase 原生 close api
func (c *hbaseCli) RawClose(ctx context.Context) error {
	hbaseReq := &Request{
		Command: rawClose,
	}
	hbaseRsp := &Response{}
	return c.invoke(ctx, hbaseReq, hbaseRsp)
}

func (c *hbaseCli) invoke(ctx context.Context,
	hbaseReq *Request, hbaseRsp *Response) error {
	ctx, msg := codec.WithCloneMessage(ctx)
	defer codec.PutBackMessage(msg)
	msg.WithClientRPCName(fmt.Sprintf("/%s/%v", c.ServiceName, hbaseReq.Command))
	msg.WithCalleeServiceName(c.ServiceName)
	msg.WithSerializationType(-1) //不序列化
	msg.WithCompressType(0)       //不压缩
	msg.WithClientReqHead(hbaseReq)
	msg.WithClientRspHead(hbaseRsp)

	return c.Client.Invoke(ctx, hbaseReq, hbaseRsp, c.opts...)
}

// hbase cmd
const (
	get            = "get"
	put            = "put"
	del            = "del"
	rawGet         = "rawGet"
	rawPut         = "rawPut"
	rawDel         = "rawDel"
	rawScan        = "rawScan"
	rawAppend      = "rawAppend"
	rawInc         = "rawInc"
	rawCheckAndPut = "rawCheckAndPut"
	rawClose       = "rawClose"
)
