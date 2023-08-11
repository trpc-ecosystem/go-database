package redcas

import (
	"errors"
	"reflect"
	"testing"

	gomonkey "github.com/agiledragon/gomonkey/v2"
	redis "github.com/redis/go-redis/v9"
)

func TestGetCmd_Val(t *testing.T) {
	type fields struct {
		StringCmd *redis.StringCmd
	}
	tests := []struct {
		name   string
		fields fields
		want   string
		want1  int64
	}{
		{
			name: "ok",
			fields: fields{
				StringCmd: &redis.StringCmd{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &GetCmd{
				StringCmd: tt.fields.StringCmd,
			}
			got, got1 := cmd.Val()
			if got != tt.want {
				t.Errorf("Val() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("Val() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestMGetCmd_Unmarshal(t *testing.T) {
	cmd := &MGetCmd{SliceCmd: &redis.SliceCmd{}}
	t.Run("err != nil", func(t *testing.T) {
		cmd.SetErr(errors.New("test"))
		_, _ = cmd.Unmarshal(0, nil)
	})
	t.Run("i >= int64(len(results))", func(t *testing.T) {
		cmd.SetErr(nil)
		_, _ = cmd.Unmarshal(0, nil)
	})
	t.Run("interfaceResult == nil", func(t *testing.T) {
		patches := gomonkey.ApplyMethodSeq(reflect.TypeOf(cmd.SliceCmd), "Result", []gomonkey.OutputCell{
			{Values: gomonkey.Params{[]interface{}{nil}, nil}}})
		defer patches.Reset()
		cmd.SetErr(nil)
		_, _ = cmd.Unmarshal(0, nil)
	})
	t.Run("stringResult", func(t *testing.T) {
		patches := gomonkey.ApplyMethodSeq(reflect.TypeOf(cmd.SliceCmd), "Result", []gomonkey.OutputCell{
			{Values: gomonkey.Params{[]interface{}{1}, nil}}})
		defer patches.Reset()
		cmd.SetErr(nil)
		_, _ = cmd.Unmarshal(0, nil)
	})
	t.Run("Decode", func(t *testing.T) {
		patches := gomonkey.ApplyMethodSeq(reflect.TypeOf(cmd.SliceCmd), "Result", []gomonkey.OutputCell{
			{Values: gomonkey.Params{[]interface{}{"123"}, nil}}})
		defer patches.Reset()
		cmd.SetErr(nil)
		_, _ = cmd.Unmarshal(0, nil)
	})
}
