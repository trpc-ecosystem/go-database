package cos

import (
	"context"
	"testing"
	"time"

	monkey "github.com/cch123/supermonkey"
	"github.com/stretchr/testify/assert"
	"trpc.group/trpc-go/trpc-go/client"
)

func Test_genFormatHeaders(t *testing.T) {
	SetNeedSignHeaders("k1", true)
	SetNeedSignHeaders("k2", true)
	SetNeedSignHeaders("test", true)

	type args struct {
		headers map[string]string
	}
	tests := []struct {
		name                 string
		args                 args
		wantFormatHeaders    string
		wantSignedHeaderList []string
	}{
		{
			"case empty",
			args{
				headers: map[string]string{},
			},
			"",
			nil,
		},
		{
			"case Upper",
			args{
				headers: map[string]string{"Test": "tEst"},
			},
			"test=tEst",
			[]string{"test"},
		},
		{
			"case normal",
			args{
				headers: map[string]string{"k1": "v1", "k2": "v2"},
			},
			"k1=v1&k2=v2",
			[]string{"k1", "k2"},
		},
		{
			"case with '/'",
			args{
				headers: map[string]string{"content-type": "application/json"},
			},
			"content-type=application%2Fjson",
			[]string{"content-type"},
		},
		{
			"case with ' '",
			args{
				headers: map[string]string{
					"content-type": "application/json", "Content-Disposition": "attachment; filename=123.txt",
				},
			},
			"content-disposition=attachment%3B%20filename%3D123.txt&content-type=application%2Fjson",
			[]string{"content-disposition", "content-type"},
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				gotFormatHeaders, gotSignedHeaderList := genFormatHeaders(tt.args.headers)
				assert.Equalf(t, tt.wantFormatHeaders, gotFormatHeaders, "genFormatHeaders(%v)", tt.args.headers)
				assert.Equalf(t, tt.wantSignedHeaderList, gotSignedHeaderList, "genFormatHeaders(%v)", tt.args.headers)
			},
		)
	}
}

func Test_genFormatParameters(t *testing.T) {
	type args struct {
		parameters map[string]string
	}
	tests := []struct {
		name                    string
		args                    args
		wantFormatParameters    string
		wantSignedParameterList []string
	}{
		{
			"case empty",
			args{
				parameters: map[string]string{},
			},
			"",
			nil,
		},
		{
			"case Upper",
			args{
				parameters: map[string]string{"Test": "tEst"},
			},
			"test=tEst",
			[]string{"test"},
		},
		{
			"case mormal",
			args{
				parameters: map[string]string{"k1": "v1", "k2": "v2"},
			},
			"k1=v1&k2=v2",
			[]string{"k1", "k2"},
		},
		{
			"case with '/'",
			args{
				parameters: map[string]string{"k1": "Prefix/test", "k2": "v2"},
			},
			"k1=Prefix%2Ftest&k2=v2",
			[]string{"k1", "k2"},
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				gotFormatParameters, gotSignedParameterList := genFormatParameters(tt.args.parameters)
				assert.Equalf(
					t, tt.wantFormatParameters, gotFormatParameters, "genFormatParameters(%v)", tt.args.parameters,
				)
				assert.Equalf(
					t, tt.wantSignedParameterList, gotSignedParameterList, "genFormatParameters(%v)",
					tt.args.parameters,
				)
			},
		)
	}
}

// TestClient_GenAuthorization tests 'GenAuthorization'.
func TestClient_GenAuthorization(t *testing.T) {
	SetNeedSignHeaders("head_key", true)
	// mock
	monkey.Patch(
		time.Now, func() time.Time {
			t, _ := time.Parse("2006-01-02", "2021-05-11")
			return t
		},
	)
	// prepare data
	cli := NewClientProxy("cos", config, client.WithTarget(target))
	ctx := context.Background()
	method := "PUT"
	uri := "/test/li.txt"
	params := map[string]string{
		"param_key": "param_val",
	}
	headers := map[string]string{
		"head_key": "head_val",
	}
	expectedAuthVal := "q-sign-algorithm=sha1&q-ak=your_secretid&q-sign-time=1620691200;" +
		"1620693000&q-key-time=1620691200;1620693000&q-header-list=head_key&" +
		"q-url-param-list=param_key&q-signature=32db75df5c7332d5dc1abc3b4d483aa90723f3a4"
	// run func
	authVal, _ := cli.GenAuthorization(ctx, method, uri, params, headers)
	// assert
	assert.Equal(t, expectedAuthVal, authVal)
}
