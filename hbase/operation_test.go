package hbase

import (
	"context"
	"fmt"
	"testing"

	"github.com/agiledragon/gomonkey"
	"github.com/tsuna/gohbase/hrpc"

	. "github.com/smartystreets/goconvey/convey"
)

type mockClient struct{}

func (c *mockClient) Scan(s *hrpc.Scan) hrpc.Scanner {
	return nil
}
func (c *mockClient) Get(g *hrpc.Get) (*hrpc.Result, error) {
	return &hrpc.Result{}, nil
}
func (c *mockClient) Put(p *hrpc.Mutate) (*hrpc.Result, error) {
	return &hrpc.Result{}, nil
}
func (c *mockClient) Delete(d *hrpc.Mutate) (*hrpc.Result, error) {
	return &hrpc.Result{}, nil
}
func (c *mockClient) Append(a *hrpc.Mutate) (*hrpc.Result, error) {
	return &hrpc.Result{}, nil
}
func (c *mockClient) Increment(i *hrpc.Mutate) (int64, error) {
	return 0, nil
}
func (c *mockClient) CheckAndPut(p *hrpc.Mutate, family string, qualifier string, expectedValue []byte) (bool, error) {
	return false, nil
}

func (c *mockClient) Close() {}

func Test_clientImp_doGet(t *testing.T) {
	mockCli := &mockClient{}
	c := &clientImp{
		Client: mockCli,
	}

	Convey("clientImp.doGet", t, func() {
		mock := gomonkey.ApplyFunc(hrpc.NewGetStr, func(ctx context.Context, table, key string,
			options ...func(hrpc.Call) error) (*hrpc.Get, error) {
			if table == "" {
				return nil, fmt.Errorf("xxx")
			}
			return &hrpc.Get{}, nil
		})
		defer mock.Reset()
		Convey("get-ok", func() {
			So(c.doGet(context.Background(), &Request{Table: "table"}, &Response{}), ShouldBeNil)
		})

		Convey("get-err", func() {
			So(c.doGet(context.Background(), &Request{}, &Response{}), ShouldNotBeNil)
		})
	})
}

func Test_clientImp_doPut(t *testing.T) {
	mockCli := &mockClient{}
	c := &clientImp{
		Client: mockCli,
	}

	Convey("clientImp.doPut", t, func() {
		mock := gomonkey.ApplyFunc(hrpc.NewPutStr, func(ctx context.Context, table, key string,
			values map[string]map[string][]byte, options ...func(hrpc.Call) error) (*hrpc.Mutate, error) {
			if table == "" {
				return nil, fmt.Errorf("xxx")
			}
			return &hrpc.Mutate{}, nil
		})
		defer mock.Reset()

		Convey("put-ok", func() {
			So(c.doPut(context.Background(), &Request{Table: "table"}, &Response{}), ShouldBeNil)
		})
	})
}

func Test_clientImp_doRawGet(t *testing.T) {
	c := &clientImp{
		Client: &mockClient{},
	}
	Convey("clientImp.doRawGet", t, func() {
		ctx := context.Background()
		req := new(Request)
		rsp := new(Response)
		So(c.doRawGet(ctx, req, rsp), ShouldBeNil)
	})
}

func Test_clientImp_doRawPut(t *testing.T) {
	c := &clientImp{
		Client: &mockClient{},
	}
	Convey("clientImp.doRawPut", t, func() {
		ctx := context.Background()
		req := new(Request)
		rsp := new(Response)
		So(c.doRawPut(ctx, req, rsp), ShouldBeNil)
	})
}

func Test_clientImp_doRawDel(t *testing.T) {
	c := &clientImp{
		Client: &mockClient{},
	}
	Convey("clientImp.doRawDel", t, func() {
		ctx := context.Background()
		req := new(Request)
		rsp := new(Response)
		So(c.doRawDel(ctx, req, rsp), ShouldBeNil)
	})
}

func Test_clientImp_doRawScan(t *testing.T) {
	c := &clientImp{
		Client: &mockClient{},
	}
	Convey("clientImp.doRawScan", t, func() {
		ctx := context.Background()
		req := new(Request)
		rsp := new(Response)
		So(c.doRawScan(ctx, req, rsp), ShouldBeNil)
	})
}

func Test_clientImp_doRawAppend(t *testing.T) {
	c := &clientImp{
		Client: &mockClient{},
	}
	Convey("clientImp.doRawAppend", t, func() {
		ctx := context.Background()
		req := new(Request)
		rsp := new(Response)
		So(c.doRawAppend(ctx, req, rsp), ShouldBeNil)
	})
}

func Test_clientImp_doRawInc(t *testing.T) {
	c := &clientImp{
		Client: &mockClient{},
	}
	Convey("clientImp.doRawInc", t, func() {
		ctx := context.Background()
		req := new(Request)
		rsp := new(Response)
		So(c.doRawInc(ctx, req, rsp), ShouldBeNil)
	})
}

func Test_clientImp_doRawCheckAndPut(t *testing.T) {
	c := &clientImp{
		Client: &mockClient{},
	}
	Convey("clientImp.doRawCheckAndPut", t, func() {
		ctx := context.Background()
		req := new(Request)
		rsp := new(Response)
		So(c.doRawCheckAndPut(ctx, req, rsp), ShouldBeNil)
	})
}

func Test_clientImp_doDel(t *testing.T) {
	c := &clientImp{
		Client: &mockClient{},
	}

	Convey("clientImp.doDel", t, func() {
		ctx := context.Background()
		req := new(Request)
		rsp := new(Response)
		So(c.doDel(ctx, req, rsp), ShouldBeNil)
	})
}
