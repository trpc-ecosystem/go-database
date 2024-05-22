package mongodb

import (
	"context"
	"reflect"
	"testing"

	"trpc.group/trpc-go/trpc-go/codec"
)

func TestUnitClientCodec_Decode(t *testing.T) {
	type args struct {
		msg codec.Msg
		in1 []byte
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{"succ", args{}, nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ClientCodec{}
			got, err := c.Decode(tt.args.msg, tt.args.in1)
			if (err != nil) != tt.wantErr {
				t.Errorf("Decode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Decode() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUnitClientCodec_Encode(t *testing.T) {
	type args struct {
		msg codec.Msg
		in1 []byte
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{"succ", args{msg: codec.Message(context.Background())}, nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ClientCodec{}
			got, err := c.Encode(tt.args.msg, tt.args.in1)
			if (err != nil) != tt.wantErr {
				t.Errorf("Encode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Encode() got = %v, want %v", got, tt.want)
			}
		})
	}
}
