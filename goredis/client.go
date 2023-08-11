// Package goredis extend github.com/go-redis/redis to add support for the trpc-go ecosystem.
package goredis

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"strings"
	"time"
	"unicode/utf8"

	redis "github.com/redis/go-redis/v9"
	"trpc.group/trpc-go/trpc-database/goredis/internal/joinfilters"
	"trpc.group/trpc-go/trpc-database/goredis/internal/options"
	trpc "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/naming/selector"
)

// ContextKeyType go redis context msg key type
type ContextKeyType string

const (
	ContextKeyName ContextKeyType = "trpc-go-redis" // context key

	RetRedisNil     = 30001 // redis field doesn't exist
	RetCASMismatch  = 30002 // cas confict
	RetRedisCmdFail = 30003 // redis cmd fail
	RetParamInvalid = 30004 // invalid parameter
	RetTypeMismatch = 30005 // type mismatch
	RetLockOccupied = 30006 // lock is already occupied
	RetLockExpired  = 30007 // lock has expired
	RetLockRobbed   = 30008 // lock has been seized
	RetKeyNotFound  = 30009 // key doesn't exist or expired, it will alert but RetRedisNil will not
	RetAddCronFail  = 30010 // fail to add the cron job
	RetInitFail     = 30011 // fail to initiate
	RetLockExtend   = 30012 // fail to renew the lock

	InfinityMin = "-inf" // negative infinity
	InfinityMax = "+inf" // positive infinity

	calleeApp        = "[redis]"              // redis callee app.
	appKey           = "overrideCalleeApp"    // metadata app key.
	serverKey        = "overrideCalleeServer" // metadata server key.
	CronSelectorName = "cron"                 // cron
)

func init() {
	// keep compatibility with trpc-go.
	if selector.Get(options.RedisSelectorName) == nil {
		selector.Register(options.RedisSelectorName, selector.NewIPSelector())
	}
	if selector.Get(options.RedissSelectorName) == nil {
		selector.Register(options.RedissSelectorName, selector.NewIPSelector())
	}
	if selector.Get(CronSelectorName) == nil {
		selector.Register(CronSelectorName, selector.NewIPSelector())
	}
}

var (
	ErrM007RedisNil = errs.New(RetRedisNil, redis.Nil.Error())   // redis.Nil error.
	ErrParamInvalid = errs.New(RetParamInvalid, "param invalid") // invalid parameter.
	ErrTypeMismatch = errs.New(RetTypeMismatch, "type mismatch") // type missmatch.
	ErrLockOccupied = errs.New(RetLockOccupied, "lock occupied") // lock is occupied.

	MaxRspLen    int64 = 100 // max response length.
	matricsTable       = map[int32]string{
		int32(errs.RetClientTimeout): context.DeadlineExceeded.Error(),
		RetCASMismatch:               "cas mismatch",
		RetLockExpired:               "lock expired",
		RetLockRobbed:                "lock robbed",
		RetKeyNotFound:               "key not found",
	}
)

// New creates a redis client.
var New = func(name string, opts ...client.Option) (
	redis.UniversalClient, error) {
	// parse configuration file.
	filters, err := joinfilters.New(name, opts...)
	if err != nil {
		return nil, errs.Wrapf(err, RetInitFail, "joinfilters.NewFilters fail %v", err)
	}
	trpcOptions, err := filters.LoadClientOptions()
	if err != nil {
		return nil, errs.Wrapf(err, RetInitFail, "filters.GetClientOptions fail %v", err)
	}
	option, err := options.New(trpcOptions)
	if err != nil {
		return nil, err
	}
	redisClient := redis.NewUniversalClient(option.RedisOption)
	redisHook, err := newHook(filters, option)
	if err != nil {
		return nil, err
	}
	redisClient.AddHook(redisHook)
	// check the connection health
	if _, err = redisClient.Ping(trpc.BackgroundContext()).Result(); err != nil {
		return nil, errs.Wrapf(err, RetInitFail, "New Ping fail %v", err)
	}
	return redisClient, nil
}

// Req trpc filter request.
type Req struct {
	Cmd string
}

// Rsp trpc filter response.
type Rsp struct {
	Cmd string
}

// Message redis context
type Message struct {
	EnableFilter bool            // enable filters or not
	Options      []client.Option // trpc client options for a single request
}

// WithMessage gets Message.
func WithMessage(ctx context.Context) (context.Context, *Message) {
	vi := ctx.Value(ContextKeyName)
	vm, _ := vi.(*Message)
	if vm != nil {
		return ctx, vm
	}
	msg := &Message{
		EnableFilter: true,
	}
	return context.WithValue(ctx, ContextKeyName, msg), msg
}

// hook redis hook.
type hook struct {
	filters    *joinfilters.Filters
	remoteAddr net.Addr
	options    *options.Options
	selector   selector.Selector
}

func newHook(f *joinfilters.Filters, o *options.Options) (*hook, error) {
	h := &hook{
		filters: f,
		options: o,
	}
	var err error
	addr := h.options.RedisOption.Addrs[0]
	if h.remoteAddr, err = net.ResolveTCPAddr("tcp", addr); err != nil {
		return nil, errs.Wrapf(err, RetInitFail, "net.ResolveTCPAddr fail %s %v", addr, err)
	}
	if h.options.QueryOption.IsProxy {
		if h.selector = selector.Get("polaris"); h.selector == nil {
			return nil, errs.Newf(RetInitFail, "selector polaris not found")
		}
	}
	return h, nil
}

// DialHook is triggered during the dial process.
func (h *hook) DialHook(next redis.DialHook) redis.DialHook {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		if !h.options.QueryOption.IsProxy {
			return next(ctx, network, addr)
		}
		// polaris proxy mode.
		start := time.Now()
		node, err := h.selector.Select(h.options.Service, selector.WithNamespace(h.options.Namespace))
		if err != nil {
			return nil, err
		}
		conn, err := next(ctx, network, node.Address)
		// convert error codes to trpc format for easy statistics.
		if err != nil {
			err = errs.Wrapf(err, errs.RetClientRouteErr, "parsePolarisDialer fail %v", err)
		}
		// report error shold be ignored since it should not affect bussiness result
		_ = h.selector.Report(node, time.Since(start), err)
		return conn, err
	}
}

// ProcessHook is triggered when sending single redis command.
func (h *hook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		// redis parameter
		ctx, redisMsg := WithMessage(ctx)
		if !redisMsg.EnableFilter {
			return next(ctx, cmd)
		}
		reqBody := nameKey(cmd)
		call := func(context.Context) (string, error) {
			nextErr := next(ctx, cmd)
			rspBody := fixedResponseLength(reqBody, cmd.String(), MaxRspLen)
			return rspBody, nextErr
		}
		req := &invokeReq{
			rpcName:      cmd.Name(),
			calleeMethod: "/redis/" + cmd.Name(),
			reqBody:      reqBody,
			call:         call,
		}
		return h.invoke(ctx, req, redisMsg.Options...)
	}
}

// ProcessPipelineHook is triggered when sending pipeline redis command.
func (h *hook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		ctx, redisMsg := WithMessage(ctx)
		if !redisMsg.EnableFilter {
			return next(ctx, cmds)
		}
		call := func(context.Context) (string, error) {
			nextErr := next(ctx, cmds)
			rspBody := pipelineRspBody(cmds)
			return rspBody, nextErr
		}
		req := &invokeReq{
			rpcName: "pipeline",
			call:    call,
		}
		req.calleeMethod, req.reqBody = pipelineReqBody(cmds)
		return h.invoke(ctx, req, redisMsg.Options...)
	}
}

type invokeReq struct {
	rpcName      string
	calleeMethod string
	reqBody      string
	call         func(context.Context) (string, error)
}

func (h *hook) invoke(ctx context.Context, req *invokeReq, opts ...client.Option) error {
	ctx = h.fillTRPCMessage(ctx, req.rpcName, req.calleeMethod)
	var (
		callErr error
		rReq    = &Req{Cmd: req.reqBody}
		rRsp    = &Rsp{}
	)
	trpcErr := h.filters.Invoke(ctx, rReq, rRsp, func(ctx context.Context, _, _ interface{}) error {
		rRsp.Cmd, callErr = req.call(ctx)
		return TRPCErr(callErr)
	}, opts...)
	if callErr != nil {
		return callErr
	}
	return trpcErr
}

func fixedResponseLength(reqCmd, rspCmd string, max int64) string {
	rsp := rspCmd
	if strings.Index(rsp, reqCmd) == 0 {
		rsp = rsp[len(reqCmd):]
	}
	if int64(len(rsp)) > max {
		rsp = rsp[:max]
	}
	if !utf8.ValidString(rsp) {
		rsp = base64.StdEncoding.EncodeToString([]byte(rsp))
	}
	if int64(len(rsp)) > max {
		rsp = rsp[:max]
	}
	return rsp
}

// nameKey simpify request.
func nameKey(cmd redis.Cmder) string {
	args := cmd.Args()
	if len(args) == 1 {
		return cmd.Name()
	}
	return fmt.Sprintf(`%s %s`, args[0], args[1])
}

// fillTRPCMessage fills trpc message.
func (h *hook) fillTRPCMessage(ctx context.Context, rpcName, calleeMethod string) context.Context {
	var trpcMsg codec.Msg
	ctx, trpcMsg = codec.WithCloneMessage(ctx)
	rpcName = strings.ReplaceAll(rpcName, "/", "_")
	trpcMsg.WithClientRPCName(fmt.Sprintf("/%s/%s", h.filters.Name, rpcName))
	trpcMsg.WithCalleeMethod(calleeMethod)
	trpcMsg.WithCalleeServiceName(h.filters.Name)
	withCommonMetaData(trpcMsg, calleeApp, h.remoteAddr.String())
	trpcMsg.WithRemoteAddr(h.remoteAddr)
	return ctx
}

// withCommonMetaData fills redis endpoint in meta data.
func withCommonMetaData(msg codec.Msg, calleeApp, calleeServer string) {
	meta := msg.CommonMeta()
	if meta == nil {
		meta = codec.CommonMeta{}
	}
	meta[appKey] = calleeApp
	meta[serverKey] = calleeServer
	msg.WithCommonMeta(meta)
}

// pipelineReqBody pipeline request.
func pipelineReqBody(commands []redis.Cmder) (string, string) {
	reqBody := bytes.NewBufferString("[")
	calleeMethod := bytes.NewBufferString("/redis/pipeline")
	cmdMap := make(map[string]bool)
	for i, cmd := range commands {
		if i != 0 {
			reqBody.WriteString(",")
		}
		reqBody.WriteString(nameKey(cmd))
		if _, ok := cmdMap[cmd.Name()]; !ok {
			cmdMap[cmd.Name()] = true
			calleeMethod.WriteString("/" + cmd.Name())
		}
	}
	reqBody.WriteString("]")
	return calleeMethod.String(), reqBody.String()
}

// pipelineRspBody pipeline response.
func pipelineRspBody(commands []redis.Cmder) string {
	rspBody := bytes.NewBufferString("[")
	for i, cmd := range commands {
		if i != 0 {
			rspBody.WriteString(",")
		}
		if cmd.Err() != nil {
			rspBody.WriteString(cmd.Err().Error())
		} else {
			rspBody.WriteString("nil")
		}
	}
	rspBody.WriteString("]")
	return rspBody.String()
}

// TRPCErr convert error to trpc format.
func TRPCErr(err error) error {
	switch err {
	case nil, redis.Nil:
		return nil
	default:
		if _, ok := err.(*errs.Error); ok {
			return err
		}
		msg := err.Error()
		for key, value := range matricsTable {
			if strings.Contains(msg, value) {
				return errs.New(int(key), err.Error())
			}
		}
		return errs.New(RetRedisCmdFail, msg)
	}
}
