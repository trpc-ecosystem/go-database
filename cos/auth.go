package cos

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"fmt"
	"hash"
	"sort"
	"strings"
	"time"
)

const sha1SignAlgorithm = "sha1"
const privateHeaderPrefix = "x-cos-"

// NeedSignHeaders is the list of Headers that need to be verified.
var NeedSignHeaders = map[string]bool{
	"host":                           true,
	"range":                          true,
	"x-cos-acl":                      true,
	"x-cos-grant-read":               true,
	"x-cos-grant-write":              true,
	"x-cos-grant-full-control":       true,
	"response-content-type":          true,
	"response-content-language":      true,
	"response-expires":               true,
	"response-cache-control":         true,
	"response-content-disposition":   true,
	"response-content-encoding":      true,
	"cache-control":                  true,
	"content-disposition":            true,
	"content-encoding":               true,
	"content-type":                   true,
	"content-length":                 true,
	"content-md5":                    true,
	"transfer-encoding":              true,
	"versionid":                      true,
	"expect":                         true,
	"expires":                        true,
	"x-cos-content-sha1":             true,
	"x-cos-storage-class":            true,
	"if-match":                       true,
	"if-modified-since":              true,
	"if-none-match":                  true,
	"if-unmodified-since":            true,
	"origin":                         true,
	"access-control-request-method":  true,
	"access-control-request-headers": true,
	"x-cos-object-type":              true,
}

// SetNeedSignHeaders is not thread-safe, it can only be set during
// process initialization instead of Client initialization.
func SetNeedSignHeaders(key string, val bool) {
	NeedSignHeaders[key] = val
}

func isSignHeader(key string) bool {
	for k, v := range NeedSignHeaders {
		if key == k && v {
			return true
		}
	}
	return strings.HasPrefix(key, privateHeaderPrefix)
}

var ciParameters = map[string]bool{
	"imagemogr2/": true,
	"watermark/":  true,
	"imageview2/": true,
}

func isCIParameter(key string) bool {
	for k, v := range ciParameters {
		if strings.HasPrefix(key, k) && v {
			return true
		}
	}
	return false
}

type valuesSignMap map[string][]string

func (vs valuesSignMap) Add(key, value string) {
	key = strings.ToLower(key)
	vs[key] = append(vs[key], value)
}

func (vs valuesSignMap) Encode() string {
	var keys []string
	for k := range vs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var pairs []string
	for _, k := range keys {
		items := vs[k]
		sort.Strings(items)
		for _, val := range items {
			pairs = append(
				pairs,
				fmt.Sprintf("%s=%s", safeURLEncode(k), safeURLEncode(val)),
			)
		}
	}
	return strings.Join(pairs, "&")
}

func safeURLEncode(s string) string {
	s = EncodeURIComponent(s, false)
	s = strings.Replace(s, "!", "%21", -1)
	s = strings.Replace(s, "'", "%27", -1)
	s = strings.Replace(s, "(", "%28", -1)
	s = strings.Replace(s, ")", "%29", -1)
	s = strings.Replace(s, "*", "%2A", -1)
	return s
}

// AuthTime is used to generate q-sign-time and q-key-time related parameters required for signature generation.
type AuthTime struct {
	SignStartTime time.Time
	SignEndTime   time.Time
	KeyStartTime  time.Time
	KeyEndTime    time.Time
}

// signString returns q-sign-time string.
func (a *AuthTime) signString() string {
	return fmt.Sprintf("%d;%d", a.SignStartTime.Unix(), a.SignEndTime.Unix())
}

// keyString returns q-key-time string.
func (a *AuthTime) keyString() string {
	return fmt.Sprintf("%d;%d", a.KeyStartTime.Unix(), a.KeyEndTime.Unix())
}

// calHMACDigest calculates the HMAC sign.
func calHMACDigest(key, msg, signMethod string) []byte {
	var hashFunc func() hash.Hash
	switch signMethod {
	case "sha1":
		hashFunc = sha1.New
	default:
		hashFunc = sha1.New
	}
	h := hmac.New(hashFunc, []byte(key))
	h.Write([]byte(msg))
	return h.Sum(nil)
}

// calSignKey calculates sign key.
func calSignKey(secretKey, keyTime string) string {
	digest := calHMACDigest(secretKey, keyTime, sha1SignAlgorithm)
	return fmt.Sprintf("%x", digest)
}

// calStringToSign calculates StringToSign.
func calStringToSign(signAlgorithm, signTime, formatString string) string {
	h := sha1.New()
	h.Write([]byte(formatString))
	return fmt.Sprintf("%s\n%s\n%x\n", signAlgorithm, signTime, h.Sum(nil))
}

// calSignature calculates signature.
func calSignature(signKey, stringToSign string) string {
	digest := calHMACDigest(signKey, stringToSign, sha1SignAlgorithm)
	return fmt.Sprintf("%x", digest)
}

// genFormatParameters generates FormatParameters and SignedParameterList.
// instead of the url.Values{}
func genFormatParameters(parameters map[string]string) (formatParameters string, signedParameterList []string) {
	ps := valuesSignMap{}
	for key, value := range parameters {
		key = strings.ToLower(key)
		if !isCIParameter(key) {
			ps.Add(key, value)
			signedParameterList = append(signedParameterList, key)
		}
	}
	formatParameters = ps.Encode()
	sort.Strings(signedParameterList)
	return
}

// genFormatHeaders generates FormatHeaders and SignedHeaderList.
func genFormatHeaders(headers map[string]string) (formatHeaders string, signedHeaderList []string) {
	hs := valuesSignMap{}
	for key, value := range headers {
		key = strings.ToLower(key)
		if isSignHeader(key) {
			hs.Add(key, value)
			signedHeaderList = append(signedHeaderList, key)
		}
	}
	formatHeaders = hs.Encode()
	sort.Strings(signedHeaderList)
	return
}

// genFormatString generates FormatString.
func genFormatString(method string, uri string, formatParameters, formatHeaders string) string {
	formatMethod := strings.ToLower(method)
	formatURI := uri

	return fmt.Sprintf(
		"%s\n%s\n%s\n%s\n", formatMethod, formatURI,
		formatParameters, formatHeaders,
	)
}

// genAuthorization generates public cloud COS Authorization
func genAuthorization(
	secretID, signTime, keyTime, signature string, signedHeaderList, signedParameterList []string,
) string {
	return strings.Join(
		[]string{
			"q-sign-algorithm=" + sha1SignAlgorithm,
			"q-ak=" + secretID,
			"q-sign-time=" + signTime,
			"q-key-time=" + keyTime,
			"q-header-list=" + strings.Join(signedHeaderList, ";"),
			"q-url-param-list=" + strings.Join(signedParameterList, ";"),
			"q-signature=" + signature,
		}, "&",
	)
}

func (c *cosClient) GenAuthorization(
	ctx context.Context, method, uri string, params, headers map[string]string,
) (string, string) {
	return c.genPublicAuthorization(ctx, method, uri, params, headers)
}

// genPublicAuthorization generates the final Authorization string through a series of steps.
// Transplanted from COS SDK.
func (c *cosClient) genPublicAuthorization(
	ctx context.Context, method, uri string, params, headers map[string]string,
) (string, string) {
	uri = "/" + strings.TrimLeft(uri, "/")

	// Step 1. Gen SignKey
	keyTimeStart := time.Now()
	keyTimeEnd := keyTimeStart.Add(time.Duration(c.Expired) * time.Second)
	authTime := AuthTime{
		SignStartTime: keyTimeStart,
		SignEndTime:   keyTimeEnd,
		KeyStartTime:  keyTimeStart,
		KeyEndTime:    keyTimeEnd,
	}

	signTime := authTime.signString()
	keyTime := authTime.keyString()
	signKey := calSignKey(c.Config.SecretKey, keyTime)

	formatHeaders := *new(string)
	signedHeaderList := *new([]string)
	formatHeaders, signedHeaderList = genFormatHeaders(headers)
	formatParameters, signedParameterList := genFormatParameters(params)
	formatString := genFormatString(method, uri, formatParameters, formatHeaders)

	stringToSign := calStringToSign(sha1SignAlgorithm, keyTime, formatString)
	signature := calSignature(signKey, stringToSign)

	return genAuthorization(
		c.Config.SecretID, signTime, keyTime, signature, signedHeaderList,
		signedParameterList,
	), formatParameters
}
