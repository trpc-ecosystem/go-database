package hbase

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/tsuna/gohbase"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/naming/selector"
	"trpc.group/trpc-go/trpc-go/transport"

	dsn "trpc.group/trpc-go/trpc-selector-dsn"
)

func init() {
	selector.Register(pluginName, dsn.DefaultSelector)
	transport.RegisterClientTransport(pluginName, DefaultClientTransport)
}

// ClientTransport client端 hbase transport
type ClientTransport struct {
	lock    sync.RWMutex
	clients map[string]*clientImp
	opts    *transport.ClientTransportOption
}

// DefaultClientTransport 默认client hbase transport
var DefaultClientTransport = NewClientTransport()

// NewClientTransport 创建hbase transport
func NewClientTransport(opt ...transport.ClientTransportOption) transport.ClientTransport {
	opts := &transport.ClientTransportOptions{}

	// 将传入的func option写到opts字段中
	for _, o := range opt {
		o(opts)
	}

	return &ClientTransport{
		clients: map[string]*clientImp{},
	}
}

func (ct *ClientTransport) getHbaseClient(address string) (*clientImp, error) {
	ct.lock.RLock()
	c, ok := ct.clients[address]
	ct.lock.RUnlock()
	if ok {
		return c, nil
	}

	conf, err := parseAddress(address)
	if err != nil {
		return nil, err
	}

	opts := []gohbase.Option{
		gohbase.ZookeeperRoot(conf.ZookeeperRoot),
		gohbase.ZookeeperTimeout(time.Duration(conf.ZookeeperTimeout) * time.Millisecond),
		gohbase.RegionLookupTimeout(time.Duration(conf.RegionLookupTimeout) * time.Millisecond),
		gohbase.RegionReadTimeout(time.Duration(conf.RegionReadTimeout) * time.Millisecond),
	}

	if len(conf.EffectiveUser) != 0 {
		opts = append(opts, gohbase.EffectiveUser(conf.EffectiveUser))
	}

	hbaseClient := gohbase.NewClient(conf.Addr, opts...)
	ct.lock.Lock()
	defer ct.lock.Unlock()
	c = &clientImp{Client: hbaseClient}
	ct.clients[address] = c

	return c, nil
}

// RoundTrip 收发hbase包, 回包hbase response放到ctx里面，这里不需要返回rspBuf
func (ct *ClientTransport) RoundTrip(ctx context.Context, reqBuf []byte,
	callOpts ...transport.RoundTripOption) (rspBuf []byte, err error) {

	msg := codec.Message(ctx)
	req, ok := msg.ClientReqHead().(*Request)
	if !ok {
		return nil, errs.NewFrameError(errs.RetClientEncodeFail,
			"hbase client transport: ReqHead should be type of *hbase.Request")
	}

	rsp, ok := msg.ClientRspHead().(*Response)
	if !ok {
		return nil, errs.NewFrameError(errs.RetClientEncodeFail,
			"hbase client transport: RspHead should be type of *hbase.Response")
	}

	opts := &transport.RoundTripOptions{}
	for _, o := range callOpts {
		o(opts)
	}

	hbaseClient, err := ct.getHbaseClient(opts.Address)
	if err != nil {
		return nil, fmt.Errorf("getHbaseClient err: %v", err)
	}

	switch req.Command {
	case get:
		return nil, hbaseClient.doGet(ctx, req, rsp)
	case put:
		return nil, hbaseClient.doPut(ctx, req, rsp)
	case del:
		return nil, hbaseClient.doDel(ctx, req, rsp)
	case rawGet:
		return nil, hbaseClient.doRawGet(ctx, req, rsp)
	case rawPut:
		return nil, hbaseClient.doRawPut(ctx, req, rsp)
	case rawDel:
		return nil, hbaseClient.doRawDel(ctx, req, rsp)
	case rawScan:
		return nil, hbaseClient.doRawScan(ctx, req, rsp)
	case rawAppend:
		return nil, hbaseClient.doRawAppend(ctx, req, rsp)
	case rawInc:
		return nil, hbaseClient.doRawInc(ctx, req, rsp)
	case rawCheckAndPut:
		return nil, hbaseClient.doRawCheckAndPut(ctx, req, rsp)
	case rawClose:
		return nil, hbaseClient.doRawClose(ctx, req, rsp)
	default:
		return nil, fmt.Errorf("command[%v] not support", req.Command)
	}
}
