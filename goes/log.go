package goes

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"trpc.group/trpc-go/trpc-go/log"
)

// esLogger es logger
type esLogger struct {
	Enable             bool
	EnableRequestBody  bool
	EnableResponseBody bool
}

// LogRoundTrip prints request in log
func (e esLogger) LogRoundTrip(
	request *http.Request, response *http.Response, err error, _ time.Time, duration time.Duration) error {
	if !e.Enable {
		return nil
	}

	field := log.Field{Key: "goes_round_trip", Value: map[string]any{
		"duration": duration,
		"request":  string(copyRequestBody(request)),
		"response": string(copyResponseBody(response)),
	}}

	if err != nil {
		log.WithContext(request.Context(), field).Errorf("goes request: %v", err)
		return nil
	}

	log.WithContext(request.Context(), field).Debug("goes request")
	return nil
}

// RequestBodyEnabled returns whether to print request body
func (e esLogger) RequestBodyEnabled() bool {
	return e.Enable && e.EnableRequestBody
}

// ResponseBodyEnabled return whether to print response body
func (e esLogger) ResponseBodyEnabled() bool {
	return e.Enable && e.EnableResponseBody
}

// copyRequestBody copy request body
func copyRequestBody(req *http.Request) []byte {
	if req.Body == nil {
		return nil
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil
	}

	req.Body = io.NopCloser(bytes.NewReader(body))
	return body
}

// copyResponseBody copy response body
func copyResponseBody(resp *http.Response) []byte {
	if resp.Body == nil {
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	resp.Body = io.NopCloser(bytes.NewReader(body))
	return body
}
