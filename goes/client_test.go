package goes

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/go-elasticsearch/esapi"
	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/transport"
)

const testServiceName = "trpc.app.service.es"

func initClientOption() {
	dbConfigMap = make(map[string]*ClientOption)
	dbConfigMap[testServiceName] = &ClientOption{
		Name: testServiceName,
	}
}

type httpRoundTripperMock struct {
}

func (h *httpRoundTripperMock) RoundTrip(request *http.Request) (*http.Response, error) {
	return &http.Response{Status: "200 OK",
		StatusCode: 200,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Request:    &http.Request{Method: "GET"},
		Header: http.Header{
			"Content-Length":    {"256"},
			"X-Elastic-Product": {"Elasticsearch"},
		},
		TransferEncoding: nil,
		Close:            true,
		ContentLength:    256,
		Body: io.NopCloser(bytes.NewBuffer([]byte(`{
  "name" : "user",
  "cluster_name" : "es-xxx",
  "cluster_uuid" : "xxx",
  "version" : {
    "number" : "7.10.1",
    "build_flavor" : "default",
    "build_type" : "tar",
    "build_hash" : "biubiubiu",
    "build_date" : "2021-12-22T12:45:12.223537200Z",
    "build_snapshot" : false,
    "lucene_version" : "8.7.0",
    "minimum_wire_compatibility_version" : "6.8.0",
    "minimum_index_compatibility_version" : "6.0.0-beta1"
  },
  "tagline" : "You Know, for Search"
}`)))}, nil
}

// TestUnitImpIndexRequest test IndexRequest
func TestUnitImpIndexRequest(t *testing.T) {
	transport.RegisterClientTransport(codecName, NewClientTransport(WithHTTPRoundTripper(&httpRoundTripperMock{})))
	initClientOption()

	dbConfigMap[testServiceName] = &ClientOption{
		Log: LogConfig{Enabled: true, RequestEnabled: true, ResponseEnabled: true},
	}

	// test ElasticClientV7
	proxy, err := NewElasticClientV7(testServiceName)
	if err != nil {
		t.Errorf("NewElasticClientV7() = %v, want %v", err, nil)
	}
	assert.Nil(t, err)

	document := struct {
		Name string `json:"name"`
	}{
		Name: "xiaohua",
	}

	documentBytes, err := json.Marshal(document)
	assert.Nil(t, err)

	req := esapi.IndexRequest{
		Index:      "test",
		DocumentID: strconv.Itoa(1),
		Body:       bytes.NewReader(documentBytes),
	}
	_, err = req.Do(trpc.BackgroundContext(), proxy)
	if err != nil {
		t.Errorf("req.Do() = %v, want %v", err, nil)
	}
	assert.Nil(t, err)

	// test ElasticClientV8
	proxyV8, err := NewElasticClientV8(testServiceName)
	if err != nil {
		log.Errorf("NewClientProxy failed, err=%v", err)
	}
	assert.Nil(t, err)

	req = esapi.IndexRequest{
		Index:      "test",
		DocumentID: strconv.Itoa(1),
		Body:       bytes.NewReader(documentBytes),
	}
	_, err = req.Do(trpc.BackgroundContext(), proxyV8)
	if err != nil {
		t.Errorf("req.Do() = %v, want %v", err, nil)
	}
	assert.Nil(t, err)

	typedV8, err := NewElasticTypedClientV8(testServiceName)
	if err != nil {
		log.Errorf("NewClientProxy failed, err=%v", err)
	}
	assert.Nil(t, err)

	_, err = typedV8.Index(testServiceName).Request(document).Do(trpc.BackgroundContext())
	if err != nil {
		t.Errorf("req.Do() = %v, want %v", err, nil)
	}
	assert.Nil(t, err)
}

// TestUnitImpSearchRequestFailed test TestUnitImpSearchRequestFailed
func TestUnitImpSearchRequestFailed(t *testing.T) {
	initClientOption()
	transport.RegisterClientTransport(codecName, NewClientTransport(WithHTTPRoundTripper(&httpRoundTripperMock{})))

	proxy, err := NewElasticClientV7(testServiceName)
	if err != nil {
		log.Errorf("NewClientProxy failed, err=%v", err)
	}
	assert.Nil(t, err)

	req := esapi.SearchRequest{
		Body: bytes.NewReader([]byte(`{"name":"xiaohua"}`)),
	}
	_, err = req.Do(trpc.BackgroundContext(), proxy)
	if err != nil {
		t.Errorf("req.Do() = %v, want %v", err, nil)
	}
	assert.Nil(t, err)
}

// TestUnitImpConfigEmpty test TestUnitImpConfigEmpty
func TestUnitImpConfigEmpty(t *testing.T) {
	transport.RegisterClientTransport(codecName, NewClientTransport(WithHTTPRoundTripper(&httpRoundTripperMock{})))
	dbConfigMap = make(map[string]*ClientOption)

	var err error

	_, err = NewElasticClientV7(testServiceName)
	assert.NotNil(t, err)

	_, err = NewElasticClientV8(testServiceName)
	assert.NotNil(t, err)

	_, err = NewElasticTypedClientV8(testServiceName)
	assert.NotNil(t, err)
}
