package cos

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	"github.com/tencentyun/cos-go-sdk-v5"
	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
	thttp "trpc.group/trpc-go/trpc-go/http"
)

var config = Conf{
	AppID:          "your_appid",
	SecretID:       "your_secretid",
	SecretKey:      "your_secretkey",
	Region:         "njc",
	Domain:         "vod.tencent-cloud.com",
	Bucket:         "test",
	SessionToken:   "your_session_token",
	SignExpiration: 1800,
} // please fill in your own test cos.

var oldconfig = Conf{
	AppID:          "your_appid",
	SecretID:       "your_secretid",
	SecretKey:      "your_secretkey",
	Region:         "cos.njc",
	Domain:         "vod.tencent-cloud.com",
	Bucket:         "test",
	SessionToken:   "your_session_token",
	SignExpiration: 1800,
}

const target = "ip://127.0.0.1:80" // change to your own test cos address

/*
// TestClientHost unit test host parameter
func TestClientHost(t *testing.T) {
	c := NewClientProxy("cos", config, client.WithTarget(target))
	Host := "test-your_appid.cos.njc.vod.tencent-cloud.com"
	assert.Equal(t, Host, c.Host)
	config.Region = "cos.njc"
	c = New("cos", config, client.WithTarget(target))
	assert.Equal(t, Host, c.Host)
	config.Region = "njc"
}
*/

func doTestPutHeadGetDelObject(t *testing.T, c Client) {
	ctx := context.Background()
	uri := "/test.txt"
	// put
	data := []byte("testok")
	url, err := c.PutObject(ctx, data, uri)
	if err != nil {
		t.Logf("put object failed, error: %s", err)
	}
	t.Logf("put object url: %s", url)
	// head
	header, err := c.HeadObject(ctx, uri)
	if err != nil {
		t.Logf("head object failed, error: %s", err)
	}
	t.Logf("head object result: %v", header)
	// get
	obj, err := c.GetObject(ctx, uri)
	if err != nil {
		t.Logf("get object failed, error: %s", err)
		return
	}
	t.Logf("get object size: %d", len(obj))
	// del
	err = c.DelObject(ctx, uri)
	if err != nil {
		t.Logf("del object failed, error: %s", err)
	}
}

// TestPutHeadGetDelObject tests the put/head/get/del methods of the cos client.
func TestPutHeadGetDelObject(t *testing.T) {
	patch := gomonkey.ApplyFunc((*cosClient).SendRequest, func(c *cosClient, ctx context.Context,
		method, uri string, headers map[string]string, data []byte) (http.Header, []byte, error) {
		rspHeader := http.Header{}
		for k, v := range headers {
			rspHeader.Add(k, v)
		}

		return rspHeader, data, nil
	})
	defer patch.Reset()

	// test the historical configuration of Tencent Cloud COS
	c := NewClientProxy("cos", oldconfig, client.WithTarget(target))
	doTestPutHeadGetDelObject(t, c)

	// test Tecent Cloud cos
	c = NewClientProxy("cos", config, client.WithTarget(target))
	doTestPutHeadGetDelObject(t, c)
}

// TestMultipartUpload tests multipart upload.
func TestMultipartUpload(t *testing.T) {
	patch := gomonkey.ApplyFunc((*cosClient).SendRequest, func(c *cosClient, ctx context.Context,
		method, uri string, headers map[string]string, data []byte) (http.Header, []byte, error) {
		rspHeader := http.Header{}
		var rspData []byte
		var err error
		switch method {
		case http.MethodPost:
			if strings.Contains(uri, "uploadId") {
				rspData = []byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n" +
					"<CompleteMultipartUploadResult xmlns=\"http://www.qcloud.com/document/product/436/7751\">\n" +
					"<Location>http://examplebucket-1250000000.cos.ap-beijing.myqcloud.com/exampleobject</Location>\n" +
					"<Bucket>examplebucket-1250000000</Bucket>\n<Key>exampleobject</Key>\n" +
					"<ETag>&quot;aa259a62513358f69e98e72e59856d88-3&quot;</ETag>\n</CompleteMultipartUploadResult>")
			} else {
				rspData = []byte("<InitiateMultipartUploadResult>\n" +
					"<Bucket>examplebucket-1250000000</Bucket>\n" +
					"<Key>exampleobject</Key>\n" +
					"<UploadId>1585130821cbb7df1d11846c073ad648e8f33b087cec2381df437acdc833cf654cc6361</UploadId>\n" +
					"</InitiateMultipartUploadResult>")
			}
		case http.MethodPut:
			var part UploadPart
			errUnmarshal := json.Unmarshal(data, &part)
			if errUnmarshal != nil {
				return rspHeader, rspData, err
			}
			rspHeader.Add("Etag", fmt.Sprintf("%d", part.PartNumber))
		default:
			return rspHeader, rspData, err
		}

		return rspHeader, rspData, err
	})
	defer patch.Reset()

	key := "/test/bigFileName"

	c := NewClientProxy("cos", config, client.WithTarget(target))

	// first, initialize the part upload and upload the file with a fixed key
	initUploadRsp, err := c.InitMultipartUpload(context.TODO(), key)
	assert.Nil(t, err)

	parts := make([]CompletePart, 0)
	// split the file into fixed-size chunks and call the upload chunk interface after splitting
	for i := 1; i <= 3; i++ {
		etag, errPart := c.UploadPart(context.TODO(), key, UploadPart{
			Part: Part{
				PartNumber: i,
			},
			UploadID:    initUploadRsp.UploadID,
			Content:     make([]byte, 1024*1024*5), // simulate 5M data
			ContentType: "application/pdf",
		})
		assert.Nil(t, errPart)

		// assemble the final parameters
		parts = append(parts, CompletePart{
			Part: Part{
				PartNumber: i,
			},
			ETag: etag,
		})
	}

	// finally, complete the part upload and test the results of the part upload.
	_, err = c.CompleteMultipartUpload(context.TODO(), key, CompleteMultipartUpload{
		UploadID: initUploadRsp.UploadID,
		Parts:    parts,
	})
	assert.Nil(t, err)
}

// TestGetImageWithTextWaterMask tests 'GetImageWithTextWaterMask'.
func TestGetImageWithTextWaterMask(t *testing.T) {
	key := "/0/20210923/09/8c1e3eb014722c311f07b4f8b166bf84.jpg"
	c := NewClientProxy("cos", config, client.WithTarget(target))
	patch := gomonkey.ApplyFunc((*cosClient).SendRequest, func(c *cosClient, ctx context.Context,
		method, uri string, headers map[string]string, data []byte) (http.Header, []byte, error) {
		rspHeader := http.Header{}
		for k, v := range headers {
			rspHeader.Add(k, v)
		}
		data = []byte("data")
		return rspHeader, data, nil
	})
	defer patch.Reset()

	data, err := c.GetImageWithTextWaterMask(context.TODO(), key, TextWaterMaskParams{
		Text: "测试水印",
	})
	assert.NotNil(t, data)
	assert.Nil(t, err)
}

// TestPutHeadGetDelObjectFailed tests the put/head/get/del methods of the cos client when they fail.
func TestPutHeadGetDelObjectFailed(t *testing.T) {
	patch := gomonkey.ApplyFunc((*cosClient).SendRequest, func(c *cosClient, ctx context.Context,
		method, uri string, headers map[string]string, data []byte) (http.Header, []byte, error) {
		rspHeader := http.Header{}
		for k, v := range headers {
			rspHeader.Add(k, v)
		}
		return rspHeader, nil, errors.New("send failed")
	})
	defer patch.Reset()

	// test Tencent Cloud cos
	c := NewClientProxy("cos", config, client.WithTarget(target))
	doTestPutHeadGetDelObject(t, c)
}

// TestDelMultipleObjects tests 'DelMultipleObjects'.
func TestDelMultipleObjects(t *testing.T) {
	patch := gomonkey.ApplyFunc((*cosClient).SendRequest, func(c *cosClient, ctx context.Context,
		method, uri string, headers map[string]string, data []byte) (http.Header, []byte, error) {
		rspData := []byte("<DeleteResult><Deleted><Key>sample1.txt</Key></Deleted><Error><Key>sample2.txt</Key><Code>-46628</Code><Message>file not exist</Message></Error></DeleteResult>")
		rspHeader := http.Header{}
		rspHeader.Add("Content-Length", strconv.Itoa(len(rspData)))
		rspHeader.Add("Content-Type", "application/xml")
		return rspHeader, rspData, nil
	})
	defer patch.Reset()

	// test Tencent cloud COS
	c := NewClientProxy("cos", config, client.WithTarget(target))

	ctx := context.Background()

	opt := &ObjectDeleteMultiOptions{
		XMLName: xml.Name{},
		Quiet:   true,
		Objects: []Object{
			{
				Key: "sample1.txt",
			},
			{
				Key: "sample2.txt",
			},
		},
	}

	res, err := c.DelMultipleObjects(ctx, opt)
	if err != nil {
		t.Logf("delete multiple objects failed, error: %s", err)
	}
	t.Logf("delete multiple objects result: %+v", res)
}

// TestHeadBucket tests 'HeadBucket', needs to fill in the real COS configuration.
func TestHeadBucket(t *testing.T) {
	ctx := context.Background()
	c := NewClientProxy("cos", config, client.WithTarget(target))
	header, err := c.HeadBucket(ctx)
	if err != nil {
		t.Logf("head bucket failed, error: %s", err)
	}
	t.Logf("head bucket result: %v", header)
}

// TestGetBucket tests 'GetBucket', needs to fill in the real COS configuration.
func TestGetBucket(t *testing.T) {
	ctx := context.Background()
	c := NewClientProxy("cos", config, client.WithTarget(target))
	if result, err := c.GetBucket(ctx); err != nil {
		t.Logf("head bucket failed, error: %s", err)
	} else {
		t.Logf("get bucket result: %v", string(result))
	}
}

// TestGetObjectWithHead tests 'GetObjectWithHead', needs to fill in the real COS configuration.
func TestGetObjectWithHead(t *testing.T) {
	ctx := context.Background()
	rangeHeader := map[string]string{
		"Range": fmt.Sprintf("bytes=%d-%d", 1, 2),
	}
	c := NewClientProxy("cos", config, client.WithTarget(target))
	if result, err := c.GetObjectWithHead(ctx, "test.txt", rangeHeader); err != nil {
		t.Logf("get object with head failed, error: %s", err)
	} else {
		t.Logf("get object with head result: %v", string(result))
	}
}

// TestClient_PutObjectWithHead tests 'PutObjectWithHead'
func TestClient_PutObjectWithHead(t *testing.T) {
	patch := gomonkey.ApplyFunc((*cosClient).SendRequest, func(c *cosClient, ctx context.Context,
		method, uri string, headers map[string]string, data []byte) (http.Header, []byte, error) {
		rspHeader := http.Header{}
		var rspData []byte
		var err error
		switch method {
		case http.MethodGet:
			switch uri {
			case "/Test1.txt":
				rspData = []byte("testok")
			case "/test2.txt":
				rspData = []byte("TestOkTestOk")
			case "/test3.txt":
				rspData = []byte("Xddd中国Ssss")
			default:
				rspData = data
			}
		case http.MethodHead:
			switch uri {
			case "/Test1.txt":
				rspHeader = nil
			case "/test2.txt":
				rspHeader.Add("X-COS-META-SXP", "Just a test")
			case "/test3.txt":
				rspHeader.Add("X-COS-META-SXP", "Xddd中国Ssss")
			default:
				rspHeader = nil
			}
		default:
			return rspHeader, rspData, err
		}

		return rspHeader, rspData, err
	})
	defer patch.Reset()

	ctx := context.Background()
	c := NewClientProxy("cos", config, client.WithTarget(target))

	testData := []struct {
		caseName   string
		headerName string
		header     map[string]string
		uri        string
		fileData   []byte
	}{
		{
			caseName:   "HeaderNil",
			headerName: "X-COS-META-SXP",
			header:     nil, // the case that header is empty
			uri:        "/Test1.txt",
			fileData:   []byte("testok"),
		}, {
			caseName:   "HeaderNotEmpty",
			headerName: "X-COS-META-SXP",
			header: map[string]string{
				"X-COS-META-SXP": "Just a test",
			}, // the case that header is not empty
			uri:      "/test2.txt",
			fileData: []byte("TestOkTestOk"),
		}, {
			caseName:   "headerNotEmptyAndEscape",
			headerName: "X-COS-META-SXP",
			header: map[string]string{
				"X-COS-META-SXP": "Xddd中国Ssss",
			}, // the case that header is not empty and there are escape characters
			uri:      "/test3.txt",
			fileData: []byte("Xddd中国Ssss"),
		},
	}

	for _, data := range testData {
		t.Run(data.caseName, func(t *testing.T) {
			// put
			retUrl, err := c.PutObjectWithHead(ctx, data.fileData, data.uri, data.header)
			if err != nil {
				t.Errorf("put object with HEAD failed, error: %s", err)
			}
			t.Logf("put object url: %s", retUrl)

			// head
			header, err := c.HeadObject(ctx, data.uri)
			if err != nil {
				t.Errorf("head object failed, error: %s", err)
			}
			if data.header != nil && header.Get(data.headerName) != data.header[data.headerName] {
				t.Errorf("file's metadata is error")
			}

			// get
			obj, err := c.GetObject(ctx, data.uri)
			if err != nil {
				t.Errorf("get object failed, error: %s", err)
			}
			if string(data.fileData) != string(obj) {
				t.Errorf("file data is not equal")
			}

			t.Logf("name: %s, case done", data.caseName)
		})
	}
}

// TestClient_PutObjectACLWithHead tests 'PutObjectACLWithHead'.
func TestClient_PutObjectACLWithHead(t *testing.T) {
	patch := gomonkey.ApplyFunc((*cosClient).SendRequest, func(c *cosClient, ctx context.Context,
		method, uri string, headers map[string]string, data []byte) (http.Header, []byte, error) {
		rspHeader := http.Header{}
		for k, v := range headers {
			rspHeader.Add(k, v)
		}

		return rspHeader, data, nil
	})
	defer patch.Reset()

	ctx := context.Background()
	uri := "/test.txt"

	tmpCfg := config
	tmpCfg.SessionToken = "sessionToken"
	c := NewClientProxy("cos", tmpCfg, client.WithTarget(target))

	headers := map[string]string{}
	url, err := c.PutObjectACLWithHead(ctx, uri, headers)
	if err == nil {
		t.Errorf("headers empty. put object acl should error. error:%v", err)
	}
	t.Logf("put object acl url: %s", url)

	headers["x-cos-acl"] = "private"
	url, err = c.PutObjectACLWithHead(ctx, uri, headers)
	if err != nil {
		t.Errorf("put object failed, error: %v", err)
	}
	t.Logf("put object acl url: %s", url)
}

// TestClient_CopyObject tests 'CopyObject'.
func TestClient_CopyObject(t *testing.T) {
	patch := gomonkey.ApplyFunc((*cosClient).SendRequest, func(c *cosClient, ctx context.Context,
		method, uri string, headers map[string]string, data []byte) (http.Header, []byte, error) {
		rspHeader := http.Header{}
		var rspData []byte
		var err error

		switch method {
		case http.MethodGet:
			switch uri {
			case "/Test-dst1.txt":
				rspData = []byte("src data")
			case "/Test-src.txt":
				rspData = []byte("src data")
			case "/Test-dst3.txt":
				rspData = []byte("src data")
			default:
				rspData = data
			}
		case http.MethodHead:
			switch uri {
			case "/Test-dst1.txt":
				rspHeader = nil
			case "/Test-src.txt":
				rspHeader.Add("X-COS-META-DST", "Test:src-->src")
			case "/Test-dst3.txt":
				rspHeader.Add("X-COS-META-DST", "Test:src-->dst with header")
			default:
				rspHeader = nil
			}
		default:
			return rspHeader, rspData, err
		}

		return rspHeader, rspData, err
	})
	defer patch.Reset()

	ctx := context.Background()

	c := NewClientProxy("cos", config, client.WithTarget(target))

	srcData := struct {
		headerName string
		header     map[string]string
		srcUri     string
		fileData   []byte
	}{
		headerName: "X-COS-META-SRC",
		header: map[string]string{
			"X-COS-META-SRC": "Test:src header",
		},
		srcUri:   "/Test-src.txt",
		fileData: []byte("src data"),
	}

	testData := []struct {
		caseName   string
		headerName string
		header     map[string]string
		dstUri     string
	}{
		{
			caseName:   "JustCopy",
			headerName: "X-COS-META-DST",
			header:     nil,
			dstUri:     "/Test-dst1.txt",
		},
		{
			caseName:   "JustChangeHeader",
			headerName: "X-COS-META-DST",
			header: map[string]string{
				"X-COS-META-DST": "Test:src-->src",
			},
			dstUri: srcData.srcUri,
		},
		{
			caseName:   "CopyWithChangeHeader",
			headerName: "X-COS-META-DST",
			header: map[string]string{
				"X-COS-META-DST": "Test:src-->dst with header",
			},
			dstUri: "/Test-dst3.txt",
		},
	}

	// Put the Src file data
	retUrl, err := c.PutObjectWithHead(ctx, srcData.fileData, srcData.srcUri, srcData.header)
	if err != nil {
		t.Errorf("put object with HEAD failed, error: %s", err)
	}
	t.Logf("put object url: %s", retUrl)

	for _, data := range testData {
		t.Run(data.caseName, func(t *testing.T) {
			// Copy object
			retUrl, err = c.CopyObject(ctx, srcData.srcUri, data.dstUri, data.header)
			if err != nil {
				t.Errorf("copy object, error: %s", err)
			}
			t.Logf("copy object url: %s", retUrl)

			// head
			header, err := c.HeadObject(ctx, data.dstUri)
			if err != nil {
				t.Errorf("head object failed, error: %s", err)
			}
			if data.header != nil && header.Get(data.headerName) != data.header[data.headerName] {
				t.Errorf("file's metadata is error")
			}

			// get
			obj, err := c.GetObject(ctx, data.dstUri)
			if err != nil {
				t.Errorf("get object failed, error: %s", err)
			}
			if string(srcData.fileData) != string(obj) {
				t.Errorf("file data is not equal")
			}

			t.Logf("name: %s, case done", data.caseName)
		})
	}
}

// TestClient_GetObjectVersions tests 'GetObjectVersions'.
func TestClient_GetObjectVersions(t *testing.T) {
	patches := gomonkey.ApplyPrivateMethod(reflect.TypeOf(client.DefaultClient), "Invoke",
		func(_ *client.Client, ctx context.Context, _, rsp interface{}, opt ...client.Option) error {

			rsp.(*codec.Body).Data = []byte("ok")

			// Mock the setting of the 'rspHead' in the Option of request params.
			opts := &client.Options{}
			for _, o := range opt {
				o(opts)
			}

			msg := trpc.Message(ctx)
			msg.WithClientRspHead(opts.RspHead)

			response := &http.Response{
				Status:        "200 OK",
				StatusCode:    200,
				Proto:         "HTTP/1.1",
				Body:          ioutil.NopCloser(bytes.NewBufferString("ok")),
				ContentLength: int64(len("ok")),
				Header:        make(http.Header, 0),
			}

			// set the response structure to return the header
			msg.ClientRspHead().(*thttp.ClientRspHeader).Response = response

			return nil
		})
	defer patches.Reset()

	cli := NewClientProxy("cos", config, client.WithTarget(target))

	res, err := cli.GetObjectVersions(trpc.BackgroundContext(), WithPrefix("prefix"))
	assert.Nil(t, err)
	assert.Equal(t, []byte("ok"), res)
}

func TestClient_GetPreSignedURL(t *testing.T) {
	cli := NewClientProxy("cos", config, client.WithTarget(target))
	ctx := context.Background()

	_, err := cli.GetPreSignedURL(ctx, "li.txt", http.MethodPut, time.Hour)
	assert.Nil(t, err)

	// Parse URL failed
	config.Region = "ap-beijing .com"
	cli = NewClientProxy("cos", config, client.WithTarget(target))
	_, err = cli.GetPreSignedURL(ctx, "li.txt", http.MethodPut, time.Hour)
	assert.NotNil(t, err)
}

func TestSort(t *testing.T) {
	paramsKeys := []string{"aaa", "aba", "abb", "abc"}
	sort.Sort(sort.StringSlice(paramsKeys))

	var paramList string
	var formatParams string
	for _, v := range paramsKeys {
		paramList += v + ";"
		formatParams += strings.ToLower(url.QueryEscape(v)) + "=" + strings.ToLower(url.QueryEscape(v)) + "&"
	}
}

// Test_addTextWaterMaskParams test 'addTextWaterMaskParams'.
func Test_addTextWaterMaskParams(t *testing.T) {
	type args struct {
		url       string
		waterMask TextWaterMaskParams
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "test add water mask, task is empty",
			args: args{
				url:       "cos",
				waterMask: TextWaterMaskParams{},
			},
			want: "cos",
		},
		{
			name: " full param",
			args: args{
				url: "cos",
				waterMask: TextWaterMaskParams{
					Text:     "test",
					Font:     "tahoma.ttf",
					Fontsize: "12",
					Fill:     "#3D3D3D",
					Dissolve: "90",
					Gravity:  "SouthEast",
					Dx:       "5",
					Dy:       "5",
					Batch:    "1",
					Degree:   "60",
					Shadow:   "50",
				},
			},
			want: "cos?watermark/2/text/dGVzdA==/font/dGFob21hLnR0Zg==/fontsize/12/fill/IzNEM0QzRA==/dissolve/90" +
				"/gravity/SouthEast/dx/5/dy/5/batch/1/degree/60/shadow/50",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := addTextWaterMaskParams(tt.args.url, tt.args.waterMask); got != tt.want {
				t.Errorf("addTextWaterMaskParams() = %v, want %v", got, tt.want)
			}
		})
	}
}

func testMethod(t *testing.T, r *http.Request, want string) {
	if got := r.Method; got != want {
		t.Errorf("Request method: %v, want %v", got, want)
	}
}

func TestPublicServiceService_Get(t *testing.T) {
	var (
		// mux is the HTTP request multiplexer used with the test server.
		mux *http.ServeMux
		// client is the COS client being tested.
		cli *cos.Client
		// server is a test HTTP server used to provide mock API responses.
		server *httptest.Server
	)

	// test server
	mux = http.NewServeMux()
	server = httptest.NewServer(mux)

	var stdConfig = Conf{
		SecretID:  "your_secretid",
		SecretKey: "your_secretkey",
		BaseURL:   server.URL,
	}
	cli = NewStdClientProxy("cos", stdConfig)
	defer func() {
		server.Close()
	}()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		fmt.Fprint(w, `<ListAllMyBucketsResult>
	<Owner>
		<ID>xbaccxx</ID>
		<DisplayName>100000760461</DisplayName>
	</Owner>
	<Buckets>
		<Bucket>
			<Name>huadong-1253846586</Name>
			<Location>ap-shanghai</Location>
			<CreationDate>2017-06-16T13:08:28Z</CreationDate>
		</Bucket>
		<Bucket>
			<Name>huanan-1253846586</Name>
			<Location>ap-guangzhou</Location>
			<CreationDate>2017-06-10T09:00:07Z</CreationDate>
		</Bucket>
	</Buckets>
</ListAllMyBucketsResult>`)
	})

	ref, _, err := cli.Service.Get(trpc.BackgroundContext())
	if err != nil {
		t.Fatalf("Service.Get returned error: %v", err)
	}

	want := &cos.ServiceGetResult{
		XMLName: xml.Name{Local: "ListAllMyBucketsResult"},
		Owner: &cos.Owner{
			ID:          "xbaccxx",
			DisplayName: "100000760461",
		},
		Buckets: []cos.Bucket{
			{
				Name:         "huadong-1253846586",
				Region:       "ap-shanghai",
				CreationDate: "2017-06-16T13:08:28Z",
			},
			{
				Name:         "huanan-1253846586",
				Region:       "ap-guangzhou",
				CreationDate: "2017-06-10T09:00:07Z",
			},
		},
	}

	if !reflect.DeepEqual(ref, want) {
		t.Errorf("Service.Get returned %+v, want %+v", ref, want)
	}
}

// TestUnitGetBaseURL test 'GetBaseURL'.
func TestUnitGetBaseURL(t *testing.T) {
	type args struct {
		conf Conf
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"hasBaseURL", args{conf: Conf{BaseURL: "https://BucketName-1300000000.cos.ap-nanjing.myqcloud.com"}},
			"https://BucketName-1300000000.cos.ap-nanjing.myqcloud.com"},
		{"defaultScheme", args{conf: Conf{
			AppID:          "your_appid",
			SecretID:       "your_secretid",
			SecretKey:      "your_secretkey",
			Region:         "njc",
			Domain:         "vod.tencent-cloud.com",
			Bucket:         "test",
			SessionToken:   "your_session_token",
			SignExpiration: 1800,
		}},
			"https://test-your_appid.cos.njc.vod.tencent-cloud.com"},
		{"httpScheme", args{conf: Conf{
			AppID:          "your_appid",
			SecretID:       "your_secretid",
			SecretKey:      "your_secretkey",
			Region:         "njc",
			Domain:         "vod.tencent-cloud.com",
			Bucket:         "test",
			SessionToken:   "your_session_token",
			SignExpiration: 1800,
			Scheme:         "http",
		}},
			"http://test-your_appid.cos.njc.vod.tencent-cloud.com"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, getBaseURL(tt.args.conf))
		})
	}
}
