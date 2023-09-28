// Package clickhouse packages standard library clickhouse.
package clickhouse

import (
	"context"
	"reflect"
	"testing"

	"trpc.group/trpc-go/trpc-go/codec"
)

func TestClientCodec_Decode(t *testing.T) {
	type args struct {
		msg    codec.Msg
		buffer []byte
	}
	tests := []struct {
		name     string
		args     args
		wantBody []byte
		wantErr  bool
	}{
		{
			name:     "succ",
			args:     args{},
			wantBody: nil,
			wantErr:  false,
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				c := &ClientCodec{}
				gotBody, err := c.Decode(tt.args.msg, tt.args.buffer)
				if (err != nil) != tt.wantErr {
					t.Errorf("Decode() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if !reflect.DeepEqual(gotBody, tt.wantBody) {
					t.Errorf("Decode() gotBody = %v, want %v", gotBody, tt.wantBody)
				}
			},
		)
	}
}

func TestClientCodec_Encode(t *testing.T) {
	type args struct {
		msg  codec.Msg
		body []byte
	}
	tests := []struct {
		name       string
		args       args
		wantBuffer []byte
		wantErr    bool
	}{
		{
			name:       "succ",
			args:       args{msg: codec.Message(context.Background())},
			wantBuffer: nil,
			wantErr:    false,
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				c := &ClientCodec{}
				gotBuffer, err := c.Encode(tt.args.msg, tt.args.body)
				if (err != nil) != tt.wantErr {
					t.Errorf("Encode() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if !reflect.DeepEqual(gotBuffer, tt.wantBuffer) {
					t.Errorf("Encode() gotBuffer = %v, want %v", gotBuffer, tt.wantBuffer)
				}
			},
		)
	}
}
