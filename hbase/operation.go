package hbase

import (
	"context"
	"fmt"

	"github.com/tsuna/gohbase"
	"github.com/tsuna/gohbase/hrpc"
)

type clientImp struct {
	Client gohbase.Client
}

func (c *clientImp) doGet(ctx context.Context, req *Request, rsp *Response) error {
	getReq, err := hrpc.NewGetStr(ctx, req.Table, req.RowKey, hrpc.Families(req.Family))
	if err != nil {
		return fmt.Errorf("hrpc.NewGetStr err: %v", err)
	}

	getRsp, err := c.Client.Get(getReq)
	if err != nil {
		return fmt.Errorf("hbase.Get err: %v", err)
	}

	rsp.Result = getRsp
	return nil
}

func (c *clientImp) doPut(ctx context.Context, req *Request, rsp *Response) error {
	putReq, err := hrpc.NewPutStr(ctx, req.Table, req.RowKey, req.Value)
	if err != nil {
		return fmt.Errorf("hrpc.NewGetStr err: %v", err)
	}

	putRsp, err := c.Client.Put(putReq)
	if err != nil {
		return fmt.Errorf("hbase.Put err: %v", err)
	}

	rsp.Result = putRsp
	return nil
}

func (c *clientImp) doRawGet(ctx context.Context, req *Request, rsp *Response) error {
	result, err := c.Client.Get(req.Get)
	if err != nil {
		return fmt.Errorf("raw get err: %v", err)
	}
	rsp.Result = result
	return nil
}

func (c *clientImp) doRawPut(ctx context.Context, req *Request, rsp *Response) error {
	result, err := c.Client.Put(req.Put)
	if err != nil {
		return fmt.Errorf("raw put err: %v", err)
	}
	rsp.Result = result
	return nil
}

func (c *clientImp) doRawDel(ctx context.Context, req *Request, rsp *Response) error {
	result, err := c.Client.Delete(req.Del)
	if err != nil {
		return fmt.Errorf("raw delete err: %v", err)
	}
	rsp.Result = result
	return nil
}

func (c *clientImp) doRawScan(ctx context.Context, req *Request, rsp *Response) error {
	rsp.Scanner = c.Client.Scan(req.Scan)
	return nil
}

func (c *clientImp) doRawAppend(ctx context.Context, req *Request, rsp *Response) error {
	result, err := c.Client.Append(req.Append)
	if err != nil {
		return fmt.Errorf("raw append err: %v", err)
	}
	rsp.Result = result
	return nil
}

func (c *clientImp) doRawInc(ctx context.Context, req *Request, rsp *Response) error {
	result, err := c.Client.Increment(req.Inc)
	if err != nil {
		return fmt.Errorf("raw increment err: %v", err)
	}
	rsp.IncVal = result
	return nil
}

func (c *clientImp) doRawCheckAndPut(ctx context.Context, req *Request, rsp *Response) error {
	result, err := c.Client.CheckAndPut(req.Put, req.FamilyStr, req.QualifierStr, req.ExpectedValue)
	if err != nil {
		return fmt.Errorf("raw check and put err: %v", err)
	}
	rsp.CheckAndPutVal = result
	return nil
}

func (c *clientImp) doDel(ctx context.Context, req *Request, rsp *Response) error {
	delReq, err := hrpc.NewDelStr(ctx, req.Table, req.RowKey, req.Value)
	if err != nil {
		return fmt.Errorf("hrpc.NewGetStr err: %v", err)
	}

	delRsp, err := c.Client.Delete(delReq)
	if err != nil {
		return fmt.Errorf("del err: %v", err)
	}

	rsp.Result = delRsp
	return nil
}

func (c *clientImp) doRawClose(ctx context.Context, req *Request, rsp *Response) error {
	c.Client.Close()
	return nil
}
