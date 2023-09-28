// Package clickhouse encapsulates standard library clickhouse.
package clickhouse

import "testing"

func TestURIHostExtractor_Extract(t *testing.T) {
	type args struct {
		uri string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name:    "V1_IP_Port",
			args:    args{uri: "ip:port?username=*&password=*&database=*"},
			want:    "ip:port",
			wantErr: false,
		},
		{
			name:    "V1_DSN_北极星",
			args:    args{uri: "polaris_name?username=*&password=*&database=*"},
			want:    "polaris_name",
			wantErr: false,
		},
		{
			name:    "V1_IP_Port_带斜杠",
			args:    args{uri: "ip:port/?username=*&password=*&database=*"},
			want:    "ip:port",
			wantErr: false,
		},
		{
			name:    "V2_IP_Port",
			args:    args{uri: "username:password@ip:port/database?dial_timeout=200ms"},
			want:    "ip:port",
			wantErr: false,
		},
		{
			name:    "V2_北极星",
			args:    args{uri: "username:password@polaris_name/database?dial_timeout=200ms"},
			want:    "polaris_name",
			wantErr: false,
		},
		{
			name:    "V2_多主机_取多个",
			args:    args{uri: "username:password@localhost:9000,host2:9100/database?dial_timeout=200ms&max_execution_time=60"},
			want:    "localhost:9000,host2:9100",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				e := &URIHostExtractor{}
				start, length, err := e.Extract(tt.args.uri)
				if (err != nil) != tt.wantErr {
					t.Errorf("Extract() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				res := tt.args.uri[start : start+length]
				if res != tt.want {
					t.Errorf("Extract() result = %v, want %v", res, tt.want)
				}
			},
		)
	}
}
