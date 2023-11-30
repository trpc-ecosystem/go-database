package clickhouse

import (
	"reflect"
	"testing"
	"time"
)

func Test_toV2DSN(t *testing.T) {
	type args struct {
		dsn string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "v1",
			args: args{dsn: "ip:port?username=user&password=pswd&database=db"},
			want: "user:pswd@ip:port/db",
		},
		{
			name: "v1./",
			args: args{dsn: "ip:port/?username=user&password=pswd&database=db"},
			want: "user:pswd@ip:port/db",
		},
		{
			name: "v1.empty",
			args: args{dsn: "ip:port"},
			want: "ip:port",
		},
		{
			name: "v1.err",
			args: args{dsn: "ip:port?t;="},
			want: "ip:port?t;=",
		},
		{
			name: "v1.timeout",
			args: args{dsn: "ip:port?username=user&password=pswd&database=db&read_timeout=10&write_timeout=10"},
			want: "user:pswd@ip:port/db?read_timeout=10s&write_timeout=10s",
		},
		{
			name: "v2",
			args: args{dsn: "user:pswd@ip:port/db?dial_timeout=200ms"},
			want: "user:pswd@ip:port/db?dial_timeout=200ms",
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				if got := toV2DSN(tt.args.dsn); got != tt.want {
					t.Errorf("toV2DSN() = %v, want %v", got, tt.want)
				}
			},
		)
	}
}

func Test_addSecUnitForTimeout(t *testing.T) {
	type args struct {
		v    string
		unit string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "empty",
			args: args{
				v:    "",
				unit: "s",
			},
			want: "",
		},
		{
			name: "10",
			args: args{
				v:    "10",
				unit: "s",
			},
			want: "10s",
		},
		{
			name: "10s",
			args: args{
				v:    "10s",
				unit: "s",
			},
			want: "10s",
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				if got := addSecUnitForTimeout(tt.args.v, tt.args.unit); got != tt.want {
					t.Errorf("addSecUnitForTimeout() = %v, want %v", got, tt.want)
				}
			},
		)
	}
}

func Test_parseOptionsFromDSN(t *testing.T) {
	type args struct {
		dsn string
	}
	tests := []struct {
		name  string
		args  args
		want  *options
		want1 string
	}{
		{
			name: "simple",
			args: args{
				dsn: "user:pswd@ip:port/db",
			},
			want:  nil,
			want1: "user:pswd@ip:port/db",
		},
		{
			name: "simple1",
			args: args{
				dsn: "user:pswd@ip:port/db?",
			},
			want:  nil,
			want1: "user:pswd@ip:port/db",
		},
		{
			name: "query err",
			args: args{
				dsn: "user:pswd@ip:port/db?t;=",
			},
			want:  nil,
			want1: "user:pswd@ip:port/db?t;=",
		},
		{
			name: "opt",
			args: args{
				dsn: "user:pswd@ip:port/db?max_idle=1&max_open=2&max_lifetime=3m",
			},
			want: &options{
				MaxIdle:     1,
				MaxOpen:     2,
				MaxLifetime: 3 * time.Minute,
			},
			want1: "user:pswd@ip:port/db",
		},
		{
			name: "opt1",
			args: args{
				dsn: "user:pswd@ip:port/db?dial_timeout=200ms&max_idle=1&max_open=2&max_lifetime=3m",
			},
			want: &options{
				MaxIdle:     1,
				MaxOpen:     2,
				MaxLifetime: 3 * time.Minute,
			},
			want1: "user:pswd@ip:port/db?dial_timeout=200ms",
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				got, got1 := parseOptionsFromDSN(tt.args.dsn)
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("parseOptionsFromDSN() got = %v, want %v", got, tt.want)
				}
				if got1 != tt.want1 {
					t.Errorf("parseOptionsFromDSN() got1 = %v, want %v", got1, tt.want1)
				}
			},
		)
	}
}
