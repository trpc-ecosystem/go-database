English | [中文](README.zh_CN.md)

### cos config

[![Coverage](https://codecov.io/gh/trpc-ecosystem/go-database/branch/coverage/graph/badge.svg?flag=cos&precision=2)](https://app.codecov.io/gh/trpc-ecosystem/go-database/tree/coverage/cos)

The COS client that adapts to the trpc-go framework. Service based on the trpc-go framework.

#### Tencent Cloud cos.Conf

Add it to the trpc-go.yaml configuration, where Region can refer to the [Tencent Cloud documentation](https://cloud.tencent.com/document/product/436/6224).
```yaml
# This is the configuration of trpc-client.
client:
  service:
    - name: "trpc.http.cos.Access"
      namespace: Production
      network: tcp
      protocol: http
      target: "dns://cos.ap-guangzhou.myqcloud.com" #is the server address.
      #target: "polaris://1084865:196608"
      timeout: 5000 #is the maximum processing time.
```
#### Tencent Cloud COS Client
Using the `NewStdClientProxy` method, return the Client defined by the Golang SDK encapsulated on GitHub provided by Tencent Cloud COS to implement access to all COS capabilities, and at the same time, use the Filter capability of the HTTP Transport in tRPC-Go to implement request QPS, time-consuming monitoring, and link reporting. 
For more usage of the official COS Go SDK, please refer to the official documentation: https://cloud.tencent.com/document/product/436/31215. 
The minimum configuration requires filling in SecretID, SecretKey, and BaseURL in cos.Conf.

```go
	cosClient := cos.NewStdClientProxy("http.cos", cos.Conf{
		SecretID:  "SecretID",
		SecretKey: "SecretKey",
		BaseURL:   "https://BucketName-1300000000.cos.ap-nanjing.myqcloud.com",
	})

	ctx := trpc.BackgroundContext()
    // Use the official COS Client to operate Object
	resp, err := cosClient.Object.Get(ctx, "object", nil)
	if err != nil {
		panic(err)
	}

    // Use the official COS Client to operate Object
    _, err = cosClient.Object.PutFromFile(ctx, "dst", "src", nil)
	if err != nil {
		panic(err)
	}
```
### cos operation

The case of operatioin

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


Upload file to COS
```
imgData := []byte("hello world")
cosProxy := cos.New("trpc.http.cos.Access", cos.Conf)
uri := "test.jpg"
url, err := cosProxy.PutObject(ctx, imgData, uri)
```
Download file from COS
```
cosProxy := cos.New("trpc.http.cos.Access", cos.Conf)
uri := "test.jpg"
data, err := cosProxy.GetObject(ctx, uri)
```
Delete file
```
cosProxy := cos.New("trpc.http.cos.Access", cos.Conf)
uri := "test.jpg"
err := cosProxy.DelObject(ctx, uri)
```
Batch delete files
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