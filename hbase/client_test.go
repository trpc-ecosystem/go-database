package hbase

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/tsuna/gohbase/hrpc"
	"trpc.group/trpc-go/trpc-go/client/mockclient"
)

func TestParseGetResult(t *testing.T) {
	Convey("ParseGetResult", t, func() {
		Convey("ok", func() {
			result := &hrpc.Result{
				Cells: []*hrpc.Cell{
					{
						Row:       []byte("rowKey1"),
						Family:    []byte("colFamily1"),
						Qualifier: []byte("col1"),
						Value:     []byte("val1"),
					},
					{
						Row:       []byte("rowKey1"),
						Family:    []byte("colFamily1"),
						Qualifier: []byte("col2"),
						Value:     []byte("val2"),
					},
					{
						Row:       []byte("rowKey1"),
						Family:    []byte("colFamily2"),
						Qualifier: []byte("col2"),
						Value:     []byte("val2"),
					},
				},
				Stale:   false,
				Partial: false,
				Exists:  nil,
			}
			rsp, err := ParseGetResult(result, nil)
			So(err, ShouldBeNil)
			So(rsp, ShouldResemble, map[string]map[string]string{
				"colFamily1": {"col1": "val1", "col2": "val2"},
				"colFamily2": {"col2": "val2"},
			})
		})

	})
}

func Test_hbaseCli(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockCli := mockclient.NewMockClient(ctrl)
	c := &hbaseCli{
		Client: mockCli,
	}

	mockCli.EXPECT().Invoke(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(
		func(ctx, reqbody, rspbody interface{}, opt ...interface{}) error {
			rsp, _ := rspbody.(*Response)
			rsp.Result = &hrpc.Result{
				Cells: []*hrpc.Cell{{
					Row: []byte("row"),
				}},
			}
			return nil
		})

	Convey("hbaseCli.Get", t, func() {
		rsp, err := c.Get(context.Background(), "table", "rowKey", nil)
		So(err, ShouldBeNil)
		So(rsp, ShouldResemble, &hrpc.Result{
			Cells: []*hrpc.Cell{{
				Row: []byte("row"),
			}},
		})
	})

	Convey("hbaseCli.Put", t, func() {
		rsp, err := c.Put(context.Background(), "table", "rowKey", nil)
		So(err, ShouldBeNil)
		So(rsp, ShouldResemble, &hrpc.Result{
			Cells: []*hrpc.Cell{{
				Row: []byte("row"),
			}},
		})
	})

	Convey("hbaseCli.Del", t, func() {
		rsp, err := c.Del(context.Background(), "table", "rowKey ", map[string]map[string][]byte{
			"r": {
				"col": nil,
			},
		})
		So(err, ShouldBeNil)
		So(rsp, ShouldResemble, &hrpc.Result{
			Cells: []*hrpc.Cell{{
				Row: []byte("row"),
			}},
		})
	})

	Convey("hbaseCli.RawGet", t, func() {
		rsp, err := c.RawGet(context.Background(), nil)
		So(err, ShouldBeNil)
		So(rsp, ShouldResemble, &hrpc.Result{
			Cells: []*hrpc.Cell{{
				Row: []byte("row"),
			}},
		})
	})

	Convey("hbaseCli.RawPut", t, func() {
		rsp, err := c.RawPut(context.Background(), nil)
		So(err, ShouldBeNil)
		So(rsp, ShouldResemble, &hrpc.Result{
			Cells: []*hrpc.Cell{{
				Row: []byte("row"),
			}},
		})
	})
	Convey("hbaseCli.RawDel", t, func() {
		rsp, err := c.RawDel(context.Background(), nil)
		So(err, ShouldBeNil)
		So(rsp, ShouldResemble, &hrpc.Result{
			Cells: []*hrpc.Cell{{
				Row: []byte("row"),
			}},
		})
	})

	Convey("hbaseCli.RawAppend", t, func() {
		rsp, err := c.RawAppend(context.Background(), nil)
		So(err, ShouldBeNil)
		So(rsp, ShouldResemble, &hrpc.Result{
			Cells: []*hrpc.Cell{{
				Row: []byte("row"),
			}},
		})
	})

	Convey("hbaseCli.RawDel", t, func() {
		rsp, err := c.RawDel(context.Background(), nil)
		So(err, ShouldBeNil)
		So(rsp, ShouldResemble, &hrpc.Result{
			Cells: []*hrpc.Cell{{
				Row: []byte("row"),
			}},
		})
	})

	Convey("hbaseCli.RawScan", t, func() {
		rsp, err := c.RawScan(context.Background(), nil)
		So(err, ShouldBeNil)
		So(rsp, ShouldBeNil)
	})

	Convey("hbaseCli.RawCheckAndPut", t, func() {
		rsp, err := c.RawCheckAndPut(context.Background(), nil, "table", "qualifier", []byte("1"))
		So(err, ShouldBeNil)
		So(rsp, ShouldBeZeroValue)
	})

	Convey("hbaseCli.RawClose", t, func() {
		So(c.RawClose(context.Background()), ShouldBeNil)
	})
}
