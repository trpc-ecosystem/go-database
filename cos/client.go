// Package cos is the client based on trpc-go
package cos

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/tencentyun/cos-go-sdk-v5"
	"github.com/tencentyun/cos-go-sdk-v5/debug"

	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/filter"

	thttp "trpc.group/trpc-go/trpc-go/http"
	trpcpb "trpc.group/trpc/trpc-protocol/pb/go/trpc"
)

//go:generate  mockgen -source=./client.go -destination=./mockcos/cos_mock.go -package=mockcos

const (
	// CosTempCredentialSessionTokenHeaderName header name for temp token
	CosTempCredentialSessionTokenHeaderName = "x-cos-security-token"
	// CosAuthorizationHeaderName auth header name
	CosAuthorizationHeaderName = "Authorization"

	// defaultURI default uri
	defaultURI = "/"
)

// InitiateMultipartUploadResult is the return value of the initialization of the multipart upload.
type InitiateMultipartUploadResult struct {
	Bucket   string `xml:"Bucket"`   // bucket name
	Key      string `xml:"Key"`      // key name
	UploadID string `xml:"UploadId"` // upload id
}

// CompleteMultipartUpload is the request of the completed multipart upload.
type CompleteMultipartUpload struct {
	UploadID string         `xml:"-"`    // upload id.
	Parts    []CompletePart `xml:"Part"` // the part informations.
}

// CompleteMultipartUploadResult is the result of the completed multipart upload.
type CompleteMultipartUploadResult struct {
	Location string `xml:"Location"` // the storage location
	Bucket   string `xml:"Bucket"`   // bucket name
	Key      string `xml:"Key"`      // the name of the key used during the upload
	ETag     string `xml:"ETag"`     // the ETag value of the object after the part are merged
}

// Part is the information of part upload.
type Part struct {
	PartNumber int `xml:"PartNumber"` // part number, ranging from 1 to 10000
}

// UploadPart is the part information used when uploading a part.
type UploadPart struct {
	Part // the public part information
	// UploadId is the ID used to identify this part upload,
	// which is obtained when the part upload was initialized
	// using the 'Initiate Multipart Upload' interface.
	UploadID    string `xml:"-"`
	Content     []byte `xml:"-"` // uploaded binary content, each block size is 1MB - 5GB, and the last block can be less than 1MB
	ContentType string `xml:"-"` // uploaded content type
}

// CompletePart is the part information used when completing the part upload.
type CompletePart struct {
	Part        // the public part information
	ETag string `xml:"ETag"` // the ETag value of the part returned after the part upload is complete
}

// TextWaterMaskParams are parameters for text watermark on an image,
// specific parameters refer to https://cloud.tencent.com/document/product/436/44888.
type TextWaterMaskParams struct {
	Text     string `json:"text"`     // watermark content, which needs to be URL-safe Base64 encoded
	Font     string `json:"font"`     // watermark font
	Fontsize string `json:"fontsize"` // font size of the watermark text, in points, default 13
	// font color, with a default of gray, needs to be set
	// to hexadecimal RGB format (for example, #FF0000), default #3D3D3D
	Fill     string `json:"fill"`
	Dissolve string `json:"dissolve"` // text transparency, with a value range of 1-100, default 90 (90% opacity)
	// text watermark position, nine-grid position (see the nine-grid orientation diagram), default SouthEast
	Gravity string `json:"gravity"`
	Dx      string `json:"dx"` // horizontal (x-axis) margin, in pixels, default 0
	Dy      string `json:"dy"` // vertical (y-axis) margin, in pixels,default 0
	// tiling watermark function, which can tile the text watermark to the entire image,
	// when the value is 1, it means that the tiling watermark function is enabled
	Batch string `json:"batch"`
	// when the batch value is 1, it takes effect,
	// the rotation angle setting of the text watermark has a value range of 0-360,default 0
	Degree string `json:"degree"`
	// text shadow effect, valid values are [0,100],
	// the default is 0, which means no shadow
	Shadow string `json:"shadow"`
}

// Client the interfaces of cos client.
type Client interface {
	HeadBucket(ctx context.Context, opts ...Option) (http.Header, error)
	GetObjectVersions(ctx context.Context, opts ...Option) ([]byte, error)
	GetBucket(ctx context.Context, opts ...Option) ([]byte, error)
	HeadObject(ctx context.Context, uri string, opts ...Option) (http.Header, error)
	GetObject(ctx context.Context, uri string, opts ...Option) ([]byte, error)
	GetObjectWithHead(ctx context.Context, uri string, headers map[string]string) ([]byte, error)
	PutObject(ctx context.Context, content []byte, uri string, opts ...Option) (url string, err error)
	PutObjectWithHead(ctx context.Context, content []byte, uri string,
		headers map[string]string) (url string, err error)
	PutObjectACLWithHead(ctx context.Context, uri string, headers map[string]string) (url string, err error)
	CopyObject(ctx context.Context, srcURI string, dstURI string, headers map[string]string) (url string, err error)
	DelObject(ctx context.Context, uri string, opts ...Option) error
	DelMultipleObjects(ctx context.Context, opt *ObjectDeleteMultiOptions) (*ObjectDeleteMultiResult, error)
	GenAuthorization(ctx context.Context, method, uri string, params, headers map[string]string) (string, string)
	GetPreSignedURL(ctx context.Context, name, method string, expired time.Duration) (string, error)
	// InitMultipartUpload initializes multipart upload, currently only supports Tencent Cloud COS.
	InitMultipartUpload(ctx context.Context, key string, opts ...Option) (*InitiateMultipartUploadResult, error)
	// UploadPart uploads by part, currently only supports Tencent Cloud COS, and returns the ETag of the uploaded part.
	UploadPart(ctx context.Context, key string, uploadPart UploadPart, opts ...Option) (string, error)
	// CompleteMultipartUpload completes multipart upload,only supports Tencent Cloud COS.
	CompleteMultipartUpload(ctx context.Context, key string, completeMultipartUpload CompleteMultipartUpload,
		opts ...Option) (*CompleteMultipartUploadResult, error)
	// GetImageWithTextWaterMask adds a text watermark when obtaining an object,
	// which only supports images and currently only supports Tencent Cloud COS.
	GetImageWithTextWaterMask(ctx context.Context, key string, waterMask TextWaterMaskParams,
		opts ...Option) ([]byte, error)
	SendRequest(ctx context.Context, method, uri string, headers map[string]string,
		data []byte) (http.Header, []byte, error)
}

// cosClient cos client
type cosClient struct {
	Config Conf

	ServiceName   string
	Host          string
	Authorization string
	ContentType   string
	Expired       int64
	opts          []client.Option // trpc client related options
}

// Owner is the Bucket or Object’s owner.
type Owner struct {
	UIN         string `xml:"uin,omitempty"`
	ID          string `xml:",omitempty"`
	DisplayName string `xml:",omitempty"`
}

// Object is the cos object.
type Object struct {
	Key          string `xml:",omitempty"`
	ETag         string `xml:",omitempty"`
	Size         int64  `xml:",omitempty"`
	PartNumber   int    `xml:",omitempty"`
	LastModified string `xml:",omitempty"`
	StorageClass string `xml:",omitempty"`
	Owner        *Owner `xml:",omitempty"`
	VersionId    string `xml:",omitempty"`
}

// BucketGetResult is the result of 'GetBucket'.
type BucketGetResult struct {
	XMLName        xml.Name `xml:"ListBucketResult"`
	Name           string
	Prefix         string `xml:"Prefix,omitempty"`
	Marker         string `xml:"Marker,omitempty"`
	NextMarker     string `xml:"NextMarker,omitempty"`
	Delimiter      string `xml:"Delimiter,omitempty"`
	MaxKeys        int
	IsTruncated    bool
	Contents       []Object `xml:"Contents,omitempty"`
	CommonPrefixes []string `xml:"CommonPrefixes>Prefix,omitempty"`
	EncodingType   string   `xml:"EncodingType,omitempty"`
}

func getCosHost(conf Conf) string {
	// default Tencent Cloud’s cos splicing rule
	prefix := "cos"
	if conf.Prefix != "" {
		prefix = conf.Prefix
	}

	host := fmt.Sprintf("%s-%s.%s.%s.%s", conf.Bucket, conf.AppID, prefix, conf.Region, conf.Domain)
	// if the region parameter contains the 'cos.' prefix and suffix
	if strings.Contains(conf.AppID, ".cos") || strings.Contains(conf.Region, "cos.") {
		host = fmt.Sprintf("%s-%s.%s.%s", conf.Bucket, conf.AppID, conf.Region, conf.Domain)
	}

	return host
}

// getBaseURL splices baseURL.
func getBaseURL(conf Conf) string {
	if conf.BaseURL != "" {
		return conf.BaseURL
	}
	if conf.Scheme != "" {
		return fmt.Sprintf("%s://%s", conf.Scheme, getCosHost(conf))
	}
	return fmt.Sprintf("https://%s", getCosHost(conf))
}

// NewStdClientProxy return a cos Client of cos-go-sdk-v5,
// which is convenient for compatible public cloud scenarios and has monitoring reporting.
var NewStdClientProxy = func(name string, conf Conf, opts ...client.Option) *cos.Client {
	baseURL := getBaseURL(conf)
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil
	}

	cli := thttp.NewStdHTTPClient(name, opts...)
	b := &cos.BaseURL{BucketURL: u, ServiceURL: u, BatchURL: u, CIURL: u}

	cosClient := cos.NewClient(
		b, &http.Client{
			Transport: &cos.AuthorizationTransport{
				SecretID:  conf.SecretID,
				SecretKey: conf.SecretKey,
				Transport: cli.Transport,
			},
		},
	)

	return cosClient
}

// NewClientProxy return a new cos client.
var NewClientProxy = func(name string, conf Conf, opts ...client.Option) Client {
	c := &cosClient{
		Config: conf,
	}
	c.ServiceName = name
	c.Host = getCosHost(conf)
	c.Authorization = ""
	c.Expired = 1800
	// provides a custom signature expiration time,
	// which can be used to control the URL validity period of non-public read resources.
	if conf.SignExpiration != 0 {
		c.Expired = conf.SignExpiration
	}
	c.ContentType = "application/xml"
	c.opts = opts
	return c
}

// New is compatible with past New methods
var New = NewClientProxy

// HeadBucket can confirm whether the Bucket exists and whether there is permission to access it by the request.
func (c *cosClient) HeadBucket(ctx context.Context, opts ...Option) (http.Header, error) {
	o := c.getOptions(ctx, http.MethodHead, defaultURI, opts...)

	header, _, err := c.SendRequest(ctx, http.MethodHead, defaultURI, o.GetHeader(), nil)
	if err != nil {
		return nil, err
	}
	return header, nil
}

// GetObjectVersions gets all versions of objects under the Bucket and supports filtering by prefix.
func (c *cosClient) GetObjectVersions(ctx context.Context, opts ...Option) ([]byte, error) {
	o := c.getOptions(ctx, http.MethodGet, defaultURI, opts...)

	uri := fmt.Sprintf("/?versions&%v", o.GetParamString())
	_, res, err := c.SendRequest(ctx, http.MethodGet, uri, o.GetHeader(), nil)

	return res, err
}

// GetBucket gets the bucket information.
func (c *cosClient) GetBucket(ctx context.Context, opts ...Option) ([]byte, error) {
	o := c.getOptions(ctx, http.MethodGet, defaultURI, opts...)

	uri := fmt.Sprintf("/?%v", o.GetParamString())
	_, body, err := c.SendRequest(ctx, http.MethodGet, uri, o.GetHeader(), nil)
	if err != nil {
		return nil, err
	}
	return body, nil
}

// HeadObject gets the metadata of the object.
func (c *cosClient) HeadObject(ctx context.Context, uri string, opts ...Option) (http.Header, error) {
	o := c.getOptions(ctx, http.MethodHead, uri, opts...)

	uri = c.getEncodeURI(uri)
	header, _, err := c.SendRequest(ctx, http.MethodHead, uri, o.GetHeader(), nil)
	if err != nil {
		return nil, err
	}
	return header, nil
}

// GetObject downloads files.
func (c *cosClient) GetObject(ctx context.Context, uri string, opts ...Option) ([]byte, error) {
	o := c.getOptions(ctx, http.MethodGet, uri, opts...)

	uri = c.getEncodeURI(uri)
	_, body, err := c.SendRequest(ctx, http.MethodGet, uri, o.GetHeader(), nil)
	if err != nil {
		return nil, err
	}
	return body, nil
}

// GetObjectWithHead downloads with special parameters, deprecated.
// You can directly use the 'GetObject' function and add the header Option,
// for example: cos.WithCustomHeader("If-None-Match", "tag").
func (c *cosClient) GetObjectWithHead(ctx context.Context, uri string, headers map[string]string) ([]byte, error) {
	if headers == nil {
		headers = map[string]string{}
	}
	headers["Host"] = c.Host

	params := map[string]string{}
	authHeader := map[string]string{
		"Host": c.Host,
	}
	auth, _ := c.GenAuthorization(ctx, http.MethodGet, uri, params, authHeader)
	headers[CosAuthorizationHeaderName] = auth
	// support for Tencent Cloud temporary keys.
	if c.Config.SessionToken != "" {
		headers[CosTempCredentialSessionTokenHeaderName] = c.Config.SessionToken
	}
	_, body, err := c.SendRequest(ctx, http.MethodGet, uri, headers, nil)
	if err != nil {
		return nil, err
	}
	return body, nil
}

// PutObject uploads files.
func (c *cosClient) PutObject(ctx context.Context, content []byte, uri string, opts ...Option) (url string, err error) {
	if len(content) == 0 {
		return "", fmt.Errorf("upload empty data. uri: %s", uri)
	}

	uri = "/" + strings.TrimLeft(uri, "/")

	o := NewOptions()
	o.GetHeader()["Host"] = c.Host
	o.GetHeader()["Content-Length"] = strconv.Itoa(len(content))
	for _, opt := range opts {
		opt(o)
	}
	c.addAuthHeader(ctx, http.MethodPut, uri, o.GetParam(), o.GetHeader())

	uri = c.getEncodeURI(uri)
	if _, _, err := c.SendRequest(ctx, http.MethodPut, uri, o.GetHeader(), content); err != nil {
		return "", err
	}
	url = c.Host + uri
	return url, nil
}

// PutObjectWithHead uploads files and sets metaData. If metaData is not set, headers can be filled with nil.
func (c *cosClient) PutObjectWithHead(ctx context.Context, content []byte, uri string,
	headers map[string]string) (url string, err error) {
	if len(content) == 0 {
		return "", fmt.Errorf("upload empty data. uri: %s", uri)
	}

	uri = "/" + strings.TrimLeft(uri, "/")

	params := map[string]string{}
	authHeader := map[string]string{}
	if headers == nil {
		headers = map[string]string{}
	}
	headers["Content-Length"] = strconv.Itoa(len(content))
	headers["Host"] = c.Host

	authHeader["Content-Length"] = strconv.Itoa(len(content))
	authHeader["Host"] = c.Host

	auth, _ := c.GenAuthorization(ctx, http.MethodPut, uri, params, authHeader)
	headers[CosAuthorizationHeaderName] = auth
	// support for Tencent Cloud temporary keys.
	if c.Config.SessionToken != "" {
		headers[CosTempCredentialSessionTokenHeaderName] = c.Config.SessionToken
	}

	uri = c.getEncodeURI(uri)
	if _, _, err := c.SendRequest(ctx, http.MethodPut, uri, headers, content); err != nil {
		return "", err
	}
	url = c.Host + uri
	return url, nil
}

// PutObjectACLWithHead sets ACL through request headers.
// For detailed documentation, please refer to: https://cloud.tencent.com/document/product/436/7748
func (c *cosClient) PutObjectACLWithHead(ctx context.Context, uri string,
	headers map[string]string) (url string, err error) {
	if len(headers) == 0 {
		return "", fmt.Errorf("put acl empty headers. uri: %s", uri)
	}

	uri = "/" + strings.TrimLeft(uri, "/")

	auth, _ := c.GenAuthorization(ctx, http.MethodPut, uri, map[string]string{}, map[string]string{})
	headers[CosAuthorizationHeaderName] = auth
	// support for Tencent Cloud temporary keys.
	if c.Config.SessionToken != "" {
		headers[CosTempCredentialSessionTokenHeaderName] = c.Config.SessionToken
	}

	// when calculating the signature, do not include parameters. when making the request, include them
	if _, _, err := c.SendRequest(ctx, http.MethodPut, uri+"?acl", headers, nil); err != nil {
		return "", err
	}
	return c.Host + uri, nil
}

// CopyObject Copy files, modify file's metadata, or do both at the same time.
// If metadata is not modified, fill in 'nil' for the headers.
// If metadata is modified, fill in the headers and overwrite it with COS.
// If only the metadata of the file is modified, fill in the same path for 'srcUri' and 'dstUri'.
func (c *cosClient) CopyObject(ctx context.Context, srcURI string, dstURI string,
	headers map[string]string) (url string, err error) {
	srcURI = "/" + strings.TrimLeft(srcURI, "/")
	dstURI = "/" + strings.TrimLeft(dstURI, "/")

	params := map[string]string{}
	authHeader := map[string]string{}
	if headers == nil {
		headers = map[string]string{}
	} else {
		headers["x-cos-metadata-directive"] = "Replaced"
	}
	headers["Host"] = c.Host
	headers["x-cos-copy-source"] = c.Host + srcURI

	authHeader["Host"] = c.Host

	auth, _ := c.GenAuthorization(ctx, http.MethodPut, dstURI, params, authHeader)
	headers[CosAuthorizationHeaderName] = auth
	// support for Tencent Cloud temporary keys.
	if c.Config.SessionToken != "" {
		headers[CosTempCredentialSessionTokenHeaderName] = c.Config.SessionToken
	}
	if _, _, err := c.SendRequest(ctx, http.MethodPut, dstURI, headers, nil); err != nil {
		return "", err
	}
	url = c.Host + dstURI
	return url, nil
}

// DelObject deletes object.
func (c *cosClient) DelObject(ctx context.Context, uri string, opts ...Option) error {
	o := c.getOptions(ctx, http.MethodDelete, uri, opts...)

	uri = c.getEncodeURI(uri)
	_, _, err := c.SendRequest(ctx, http.MethodDelete, uri, o.GetHeader(), nil)
	return err
}

// ObjectDeleteMultiOptions are the options of batch deletion.
type ObjectDeleteMultiOptions struct {
	XMLName xml.Name `xml:"Delete" header:"-"`
	Quiet   bool     `xml:"Quiet" header:"-"`
	Objects []Object `xml:"Object" header:"-"`
}

// ObjectDeleteMultiResult is the result of batch deletion.
type ObjectDeleteMultiResult struct {
	XMLName        xml.Name `xml:"DeleteResult"`
	DeletedObjects []Object `xml:"Deleted,omitempty"`
	Errors         []struct {
		Key       string `xml:",omitempty"`
		Code      string `xml:",omitempty"`
		Message   string `xml:",omitempty"`
		VersionId string `xml:",omitempty"`
	} `xml:"Error,omitempty"`
}

// DelMultipleObjects batch deletes files, supports up to one thousand records.
// Object only supports Key and VersionId properties.
// For the returned results, COS provides two result modes: Verbose and Quiet.
// Verbose mode returns the deletion result of each Object,Quiet mode only returns error information for Objects.
func (c *cosClient) DelMultipleObjects(ctx context.Context,
	opt *ObjectDeleteMultiOptions) (*ObjectDeleteMultiResult, error) {
	res := ObjectDeleteMultiResult{}

	o := c.getOptions(ctx, http.MethodPost, defaultURI)

	uri := "/?delete"

	// generates xml
	xmlData, err := xml.Marshal(opt)
	if err != nil {
		return nil, fmt.Errorf("xml marshal error: %v", err)
	}

	// calculates md5
	dataHash := md5.New()
	_, err = io.WriteString(dataHash, string(xmlData))
	if err != nil {
		return nil, fmt.Errorf("fail to write to md5 writer: %v", err)
	}

	headers := o.GetHeader()
	headers["Content-Length"] = strconv.Itoa(len(xmlData))
	headers["Content-Type"] = "application/xml"
	headers["Content-MD5"] = base64.StdEncoding.EncodeToString(dataHash.Sum(nil))

	_, result, err := c.SendRequest(ctx, http.MethodPost, uri, headers, xmlData)
	if err != nil {
		return nil, err
	}

	err = xml.Unmarshal(result, &res)
	if err != nil {
		return nil, fmt.Errorf("fail to xml Unmarshal: %v", err)
	}

	return &res, nil
}

// GetPreSignedURL gets a pre-signed URL,There is no corresponding API in
// the COS API documentation, so we use the SDK here.
// details refer to https://cloud.tencent.com/document/product/436/35059
func (c *cosClient) GetPreSignedURL(ctx context.Context, name string, method string,
	expired time.Duration) (string, error) {
	var purl *url.URL
	handleFunc := func(ctx context.Context, req interface{}, rsp interface{}) error {
		u, err := url.Parse(getBaseURL(c.Config))
		if err != nil {
			return err
		}
		cli := cos.NewClient(&cos.BaseURL{BucketURL: u}, &http.Client{
			Transport: &cos.AuthorizationTransport{
				SecretID:  c.Config.SecretID,
				SecretKey: c.Config.SecretKey,
				Expire:    expired,
				Transport: &debug.DebugRequestTransport{
					RequestHeader:  true,
					RequestBody:    true,
					ResponseHeader: true,
					ResponseBody:   true,
				},
			},
		})
		purl, err = cli.Object.GetPresignedURL(ctx, method, name, c.Config.SecretID, c.Config.SecretKey, expired, nil)
		return err
	}
	ctx, msg := codec.WithCloneMessage(ctx)
	defer codec.PutBackMessage(msg)
	msg.WithClientRPCName(fmt.Sprintf("/%s/%s", c.ServiceName, "GetPreSignedURL"))
	msg.WithCalleeServiceName(c.ServiceName)

	cliOpts := &client.Options{}
	for _, o := range c.opts {
		o(cliOpts)
	}
	if err := loadClientFilterConfig(cliOpts, c.ServiceName); err != nil {
		return "", err
	}
	if err := cliOpts.Filters.Filter(ctx, name, nil, handleFunc); err != nil {
		return "", err
	}
	return purl.String(), nil
}

// InitMultipartUpload initializes multipart upload.
func (c *cosClient) InitMultipartUpload(ctx context.Context, key string,
	opts ...Option) (*InitiateMultipartUploadResult, error) {
	o := c.getOptions(ctx, http.MethodPost, key, opts...)

	uri := c.getEncodeURI(key) + "?uploads"
	_, respBody, err := c.SendRequest(ctx, http.MethodPost, uri, o.GetHeader(), nil)
	if err != nil {
		return nil, err
	}

	// deserialize the return value.
	var result InitiateMultipartUploadResult
	if err := xml.Unmarshal(respBody, &result); err != nil {
		return nil, errs.Newf(errs.RetClientDecodeFail,
			"unmarshal response body error, response:%v, error:%+v", string(respBody), err)
	}
	return &result, nil
}

// UploadPart uploads by part,now only supports Tencent Cloud COS.
func (c *cosClient) UploadPart(ctx context.Context, key string, uploadPart UploadPart, opts ...Option) (string, error) {
	// there should be another check here: each chunk size is 1MB - 5GB,
	// and the last chunk can be less than 1MB.
	// because there was no check at the beginning,
	// it is not easy to handle compatibility when adding it later, so it is not added.
	// Tencent Cloud also intercepts this check and can locate the problem based on the error message.
	if err := validUploadPartParams(uploadPart); err != nil {
		return "", err
	}

	opts = append(opts, WithCustomHeader("Content-Type", uploadPart.ContentType),
		WithCustomHeader("Content-Length", strconv.Itoa(len(uploadPart.Content))))
	o := c.getOptions(ctx, http.MethodPut, key, opts...)

	uri := fmt.Sprintf("%s?partNumber=%d&uploadId=%s", c.getEncodeURI(key), uploadPart.PartNumber,
		uploadPart.UploadID)
	head, _, err := c.SendRequest(ctx, http.MethodPut, uri, o.GetHeader(), uploadPart.Content)
	if err != nil {
		return "", err
	}
	return head.Get("Etag"), nil
}

// validUploadPartParams checks parameters for multipart uploading.
func validUploadPartParams(uploadPart UploadPart) error {
	if uploadPart.UploadID == "" {
		return fmt.Errorf("uploadID is empty")
	}
	if uploadPart.PartNumber < 1 || uploadPart.PartNumber > 10000 {
		return fmt.Errorf("partNumber: %d param err, 1 <= partNumber <= 10000", uploadPart.PartNumber)
	}
	if uploadPart.ContentType == "" {
		return fmt.Errorf("contentType is empty")
	}
	return nil
}

// CompleteMultipartUpload completes multipart upload,only supports Tencent Cloud COS.
func (c *cosClient) CompleteMultipartUpload(ctx context.Context, key string,
	completeMultipartUpload CompleteMultipartUpload, opts ...Option) (*CompleteMultipartUploadResult, error) {
	bytes, err := xml.Marshal(completeMultipartUpload)
	if err != nil {
		return nil, errs.New(errs.RetClientEncodeFail, err.Error())
	}

	opts = append(opts, WithCustomHeader("Content-Type", "application/xml"),
		WithCustomHeader("Content-Length", strconv.Itoa(len(bytes))))

	o := c.getOptions(ctx, http.MethodPost, key, opts...)

	uri := fmt.Sprintf("%s?uploadId=%s", c.getEncodeURI(key), completeMultipartUpload.UploadID)
	_, respBody, err := c.SendRequest(ctx, http.MethodPost, uri, o.GetHeader(), bytes)
	if err != nil {
		return nil, err
	}

	var completeMultipartUploadResult CompleteMultipartUploadResult
	if err := xml.Unmarshal(respBody, &completeMultipartUploadResult); err != nil {
		return nil, errs.Newf(errs.RetClientDecodeFail,
			"unmarshal response body error, response:%v, error:%+v", string(respBody), err)
	}

	return &completeMultipartUploadResult, nil
}

// GetImageWithTextWaterMask adds a text watermark when obtaining an object,
// which only supports images and currently only supports Tencent Cloud COS.
func (c *cosClient) GetImageWithTextWaterMask(ctx context.Context, key string,
	waterMask TextWaterMaskParams, opts ...Option) ([]byte, error) {
	o := c.getOptions(ctx, http.MethodGet, key, opts...)

	uri := c.getEncodeURI(key)
	uri = addTextWaterMaskParams(uri, waterMask)
	_, body, err := c.SendRequest(ctx, http.MethodGet, uri, o.GetHeader(), nil)
	if err != nil {
		return nil, err
	}
	return body, nil
}

// addTextWaterMaskParams are the parameters for adding text watermark.
func addTextWaterMaskParams(uri string, waterMask TextWaterMaskParams) string {
	if waterMask.Text == "" {
		return uri
	}
	uri = fmt.Sprintf("%s?watermark/2/text/%s", uri, base64.URLEncoding.EncodeToString([]byte(waterMask.Text)))

	if waterMask.Font != "" {
		uri = fmt.Sprintf("%s/font/%s", uri, base64.URLEncoding.EncodeToString([]byte(waterMask.Font)))
	}

	if waterMask.Fontsize != "" {
		uri = fmt.Sprintf("%s/fontsize/%s", uri, waterMask.Fontsize)
	}

	if waterMask.Fill != "" {
		uri = fmt.Sprintf("%s/fill/%s", uri, base64.URLEncoding.EncodeToString([]byte(waterMask.Fill)))
	}

	if waterMask.Dissolve != "" {
		uri = fmt.Sprintf("%s/dissolve/%s", uri, []byte(waterMask.Dissolve))
	}

	if waterMask.Gravity != "" {
		uri = fmt.Sprintf("%s/gravity/%s", uri, []byte(waterMask.Gravity))
	}

	if waterMask.Dx != "" {
		uri = fmt.Sprintf("%s/dx/%s", uri, []byte(waterMask.Dx))
	}

	if waterMask.Dy != "" {
		uri = fmt.Sprintf("%s/dy/%s", uri, []byte(waterMask.Dy))
	}
	if waterMask.Batch != "" {
		uri = fmt.Sprintf("%s/batch/%s", uri, []byte(waterMask.Batch))
	}

	if waterMask.Degree != "" {
		uri = fmt.Sprintf("%s/degree/%s", uri, []byte(waterMask.Degree))
	}
	if waterMask.Shadow != "" {
		uri = fmt.Sprintf("%s/shadow/%s", uri, []byte(waterMask.Shadow))
	}
	return uri
}

// SendRequest sends request
func (c *cosClient) SendRequest(ctx context.Context,
	method, uri string, headers map[string]string, data []byte) (http.Header, []byte, error) {
	uri = "/" + strings.TrimLeft(uri, "/")

	ctx, msg := codec.WithCloneMessage(ctx)
	defer codec.PutBackMessage(msg)
	msg.WithClientRPCName(uri)
	msg.WithCalleeServiceName(c.ServiceName)
	msg.WithCalleeMethod(method)

	reqhead := &thttp.ClientReqHeader{
		Method: method,
		Host:   c.Host,
		Header: make(http.Header),
	}
	for k, v := range headers {
		reqhead.Header.Add(k, v)
	}

	rsphead := &thttp.ClientRspHeader{}
	reqbody := &codec.Body{
		Data: data,
	}
	rspbody := &codec.Body{}

	callopts := []client.Option{
		client.WithReqHead(reqhead),
		client.WithRspHead(rsphead),
		client.WithProtocol("coshttp"),
		// the current client only performs pass-through without serialization.
		client.WithCurrentSerializationType(codec.SerializationTypeNoop),
		// the current client only performs pass-through without serialization.
		client.WithSerializationType(codec.SerializationTypeNoop),
	}
	callopts = append(callopts, c.opts...)

	err := client.DefaultClient.Invoke(ctx, reqbody, rspbody, callopts...)
	if err != nil {
		if trpcErr, ok := err.(*errs.Error); ok {
			if rsphead.Response != nil && rsphead.Response.Header != nil {
				trpcErr.Msg += fmt.Sprintf(" requestID=%v", rsphead.Response.Header.Get("X-Cos-Request-Id"))
			}
			// adapt to error codes 204 and 206
			if trpcErr.Code != trpcpb.TrpcRetCode(http.StatusPartialContent) &&
				trpcErr.Code != trpcpb.TrpcRetCode(http.StatusNoContent) {
				return nil, nil, err
			}
		} else {
			return nil, nil, err
		}
	}

	// gets the response header.
	statusCode := rsphead.Response.StatusCode
	if statusCode == http.StatusOK {
		return rsphead.Response.Header, rspbody.Data, nil
	}
	if method == http.MethodDelete && statusCode == http.StatusNoContent {
		return rsphead.Response.Header, rspbody.Data, nil
	}
	err = fmt.Errorf("[%d] %s", statusCode, rsphead.Response.Status)
	return nil, nil, err
}

// addAuthHeader adds auth to the header.
func (c *cosClient) addAuthHeader(ctx context.Context, method, uri string, params, headers map[string]string) {
	auth, _ := c.GenAuthorization(ctx, method, uri, params, headers)

	headers[CosAuthorizationHeaderName] = auth
	// support for Tencent Cloud temporary keys.
	if c.Config.SessionToken != "" {
		headers[CosTempCredentialSessionTokenHeaderName] = c.Config.SessionToken
	}
}

// getEncodeURI gets the encoded URI.
// The URI for signature does not require URI encoding, but it needs to be encoded when sending.
// If there are special characters in the URI other than '/', URL encoding is required.
func (c *cosClient) getEncodeURI(uri string) string {
	return EncodeURIComponent("/"+strings.TrimLeft(uri, "/"), false, []byte("/"))
}

// getOptions returns an options of cos.
func (c *cosClient) getOptions(ctx context.Context, method, uri string, opts ...Option) *options {
	o := NewOptions()
	o.GetHeader()["Host"] = c.Host
	for _, opt := range opts {
		opt(o)
	}
	c.addAuthHeader(ctx, method, uri, o.GetParam(), o.GetHeader())
	return o
}

func loadClientFilterConfig(opts *client.Options, key string) error {
	if opts.DisableFilter {
		opts.Filters = filter.EmptyChain
		return nil
	}
	cfg, ok := client.DefaultClientConfig()[key]
	if !ok {
		return nil
	}
	for _, filterName := range cfg.Filter {
		f := filter.GetClient(filterName)
		if f == nil {
			return fmt.Errorf("client config: filter %s no registered", filterName)
		}
		opts.Filters = append(opts.Filters, f)
		opts.FilterNames = append(opts.FilterNames, filterName)
	}
	return nil
}
