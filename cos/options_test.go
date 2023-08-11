package cos

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewOptions tests 'NewOptons'.
func TestNewOptions(t *testing.T) {
	opts := NewOptions()
	assert.NotNil(t, opts)
}

// TestOptions_GetHeader tests 'GetHeader'.
func TestOptions_GetHeader(t *testing.T) {
	opt := NewOptions()

	opts := []Option{
		WithCustomHeader("Host", "myqcloud.com"),
	}
	for _, o := range opts {
		o(opt)
	}

	header := opt.GetHeader()
	assert.NotNil(t, header)
	assert.Equal(t, "myqcloud.com", header["Host"])
}

// TestOptions_GetParam tests 'GetParam'.
func TestOptions_GetParam(t *testing.T) {
	opt := NewOptions()

	opts := []Option{
		WithPrefix("my-prefix"),
		WithDelimiter(","),
		WithKeyMaker("obj_key"),
		WithMarker("marker_key"),
		WithMaxKeys("10"),
		WithVersionID("ver_id"),
		WithVersionIDMarker("mark_ver_id"),
		WithCustomParam("custom_key", "custom_val"),
	}

	for _, o := range opts {
		o(opt)
	}

	param := opt.GetParam()
	assert.NotNil(t, param)

	paramStr := opt.GetParamString()
	assert.Greater(t, len(paramStr), 0)
}
