[English](README.md) | 中文

### cos 配置

[![Go Reference](https://pkg.go.dev/badge/trpc.group/trpc-go/trpc-database/cos.svg)](https://pkg.go.dev/trpc.group/trpc-go/trpc-database/cos)
[![Go Report Card](https://goreportcard.com/badge/trpc.group/trpc-go/trpc-database/cos)](https://goreportcard.com/report/trpc.group/trpc-go/trpc-database/cos)
[![Tests](https://github.com/trpc-ecosystem/go-database/actions/workflows/cos.yml/badge.svg)](https://github.com/trpc-ecosystem/go-database/actions/workflows/cos.yml)
[![Coverage](https://codecov.io/gh/trpc-ecosystem/go-database/branch/coverage/graph/badge.svg?flag=cos&precision=2)](https://app.codecov.io/gh/trpc-ecosystem/go-database/tree/coverage/cos)

适配trpc-go框架的cos客户端。基于trpc-go框架的服务

#### 腾讯云cos.Conf

在配置trpc-go.yaml中加入,其中Region可参考[腾讯云文档](https://cloud.tencent.com/document/product/436/6224)
```yaml
# 注意，这里是trpc-client的配置
client:
  service:
    - name: "trpc.http.cos.Access"
      namespace: Production
      network: tcp
      protocol: http
      target: "dns://cos.ap-guangzhou.myqcloud.com" #请求服务地址
      #target: "polaris://1084865:196608"
      timeout: 5000 #请求最长处理时间
```
#### 腾讯云官网COS Client
使用NewStdClientProxy接口，返回腾讯云COS提供在 GitHub 封装的 Golang SDK 定义的Client，实现对所有COS能力的访问，同时能利用tRPC-Go中 HTTP Transport 的 Filter 能力，实现请求QPS、耗时监控与链路上报。
官网COS Go SDK更多用法参考官网文档：https://cloud.tencent.com/document/product/436/31215
最小化配置需要将SecretID、SecretKey与BaseURL填入cos.Conf中

```go
	cosClient := cos.NewStdClientProxy("http.cos", cos.Conf{
		SecretID:  "SecretID",
		SecretKey: "SecretKey",
		BaseURL:   "https://BucketName-1300000000.cos.ap-nanjing.myqcloud.com",
	})

	ctx := trpc.BackgroundContext()
	// 使用官网COS Client操作Object
	resp, err := cosClient.Object.Get(ctx, "object", nil)
	if err != nil {
		panic(err)
	}

    // 使用官网COS Client操作Object
    _, err = cosClient.Object.PutFromFile(ctx, "dst", "src", nil)
	if err != nil {
		panic(err)
	}
```

### cos 操作

操作用例

```golang
    proxy := cos.New("trpc.http.cos.Access", config)

    name := "test.txt"
	uri := "/" + name

	hasSpaceURI := "/has space.txt"

	subDirChinaURI := "/txt/汉字.txt"

	var (
		res interface{}
		err error
	)

	res, err = proxy.HeadBucket(ctx)
	log.Info("HeadBucket:", res, err)

	res, err = proxy.GetBucket(ctx, cos.WithPrefix(name))
	log.Info("GetBucket with prefix:", string(res.([]byte)), err)

	res, err = proxy.GetBucket(ctx, cos.WithCustomParam("prefix", name))
	log.Info("GetBucket custom prefix:", string(res.([]byte)), err)

	res, err = proxy.HeadObject(ctx, uri)
	log.Info("HeadObject:", res, err)

	res, err = proxy.HeadObject(ctx, hasSpaceURI)
	log.Info("HeadObject has space:", res, err)

	res, err = proxy.HeadObject(ctx, subDirChinaURI)
	log.Info("HeadObject subDir:", res, err)

	res, err = proxy.HeadObject(ctx, uri, cos.WithCustomHeader("If-None-Match", "tag"))
	log.Info("HeadObject:", res, err)

	err = proxy.DelObject(ctx, uri)
	log.Info(err)

	url, err := proxy.PutObject(trpc.BackgroundContext(), []byte(dat), uri)
	log.Info(url, err)

	dat, err := proxy.GetObject(ctx, uri)
	log.Info("GetObject:", string(dat), err)

	dat, err = proxy.GetObjectVersions(ctx, cos.WithPrefix(name))
	log.Info("GetObjectVersions:", string(dat), err)

```


上传文件到cos
```
imgData := []byte("hello world")
cosProxy := cos.New("trpc.http.cos.Access", cos.Conf)
uri := "test.jpg"
url, err := cosProxy.PutObject(ctx, imgData, uri)
```
从cos下载文件
```
cosProxy := cos.New("trpc.http.cos.Access", cos.Conf)
uri := "test.jpg"
data, err := cosProxy.GetObject(ctx, uri)
```
删除文件
```
cosProxy := cos.New("trpc.http.cos.Access", cos.Conf)
uri := "test.jpg"
err := cosProxy.DelObject(ctx, uri)
```
批量删除文件
```
cosProxy := cos.New("trpc.http.cos.Access", cos.Conf)
opt := &cos.ObjectDeleteMultiOptions{
    XMLName: xml.Name{},
    Quiet:   true,
    Objects: []cos.Object{
        {
            Key: "test1.txt",
        },
        {
            Key: "test2.txt",
        },
    },
}
res, err := cosProxy.DelMultipleObjects(ctx, opt)
```