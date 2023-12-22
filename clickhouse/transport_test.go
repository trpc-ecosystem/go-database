package clickhouse

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/agiledragon/gomonkey/v2"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/transport"
)

var tx = func(tx *sql.Tx) error {
	return nil
}

var errCommitFail = errors.New("commit test fail")
var txFail = func(tx *sql.Tx) error {
	return &clickhouse.Exception{Code: 1, Message: "fail"}
}

var next = func(rows *sql.Rows) error { // The bottom layer of the framework automatically
	// calls the next function for loop rows.
	return nil
}

func TestClientTransport_RoundTrip(t *testing.T) {
	db, mock, err := sqlmock.New()
	patches := gomonkey.ApplyFuncSeq(
		sql.Open, []gomonkey.OutputCell{
			{Values: gomonkey.Params{nil, fmt.Errorf("db err")}},
			{Values: gomonkey.Params{db, err}, Times: 8},
		},
	)
	defer patches.Reset()
	mock.MatchExpectationsInOrder(true)

	// case: empty rsp
	ctxWrongRsp, msg4 := codec.WithNewMessage(context.Background())
	msg4.WithClientReqHead(&Request{})

	// case: undefined
	ctxWithUndefined, msgun := codec.WithNewMessage(context.Background())
	sqlUndefined := "SELECT 1"
	msgun.WithClientReqHead(
		&Request{
			Query: sqlUndefined,
			next:  next,
		},
	)
	msgun.WithClientRspHead(&Response{})

	// case: query success
	ctxWithQuery, msg2 := codec.WithNewMessage(context.Background())
	sqlQuery := "SELECT country_code, os_id, browser_id, categories, action_day, action_time FROM example LIMIT 5"
	msg2.WithClientReqHead(
		&Request{
			op:    opQuery,
			Query: sqlQuery,
			next:  next,
		},
	)
	msg2.WithClientRspHead(&Response{})

	mock.ExpectQuery(sqlQuery).WillReturnRows(&sqlmock.Rows{}, &sqlmock.Rows{})

	// case: exec success
	ctxWithExec, msg3 := codec.WithNewMessage(context.Background())
	sqlExec := "UPDATE example SET country_code = 1"
	msg3.WithClientReqHead(
		&Request{
			op:   opExec,
			Exec: sqlExec,
		},
	)
	msg3.WithClientRspHead(&Response{})

	mock.ExpectExec(sqlExec).WillReturnResult(sqlmock.NewResult(1, 1)).WillReturnError(nil)

	tp := NewClientTransport().(*ClientTransport)
	type fields struct {
		opts         *transport.ClientTransportOptions
		clickhouseDB map[string]*sql.DB
	}
	type args struct {
		ctx      context.Context
		reqBuf   []byte
		callOpts []transport.RoundTripOption
	}
	tests := []struct {
		name       string
		fields     fields
		args       args
		wantRspBuf []byte
		wantErr    bool
	}{
		{
			name:    "test_ctx_wrong_req",
			fields:  fields{opts: nil, clickhouseDB: nil},
			args:    args{ctx: context.Background()},
			wantErr: true,
		},
		{
			name:    "test_ctx_wrong_rsp",
			fields:  fields{opts: nil, clickhouseDB: nil},
			args:    args{ctx: ctxWrongRsp},
			wantErr: true,
		},
		{
			name:   "test_ctx_db_err",
			fields: fields{opts: nil, clickhouseDB: nil},
			args: args{
				ctx: ctxWithUndefined,
				callOpts: []transport.RoundTripOption{
					transport.WithDialAddress("test_address"),
				},
			},
			wantErr: true,
		},
		{
			name:   "test_undefined",
			fields: fields{opts: nil, clickhouseDB: nil},
			args: args{
				ctx: ctxWithUndefined,
				callOpts: []transport.RoundTripOption{
					transport.WithDialAddress("test_address"),
				},
			},
			wantErr: true,
		},
		{
			name:   "test_query",
			fields: fields{opts: nil, clickhouseDB: nil},
			args: args{
				ctx: ctxWithQuery,
				callOpts: []transport.RoundTripOption{
					transport.WithDialAddress("test_address"),
				},
			},
			wantErr: false,
		},
		{
			name:   "test_exec",
			fields: fields{opts: nil, clickhouseDB: nil},
			args: args{
				ctx: ctxWithExec,
				callOpts: []transport.RoundTripOption{
					transport.WithDialAddress("test_address"),
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				ct := tp
				_, err := ct.RoundTrip(tt.args.ctx, tt.args.reqBuf, tt.args.callOpts...)
				if (err != nil) != tt.wantErr {
					t.Errorf("RoundTrip() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
			},
		)
	}
}

func TestClientTransport_RoundTripTx(t *testing.T) {
	db, mock, err := sqlmock.New()
	patches := gomonkey.ApplyFuncSeq(
		sql.Open, []gomonkey.OutputCell{
			{Values: gomonkey.Params{db, err}, Times: 8},
		},
	)
	defer patches.Reset()
	mock.MatchExpectationsInOrder(true)

	// case: tx success
	ctxWithTx, msg := codec.WithNewMessage(context.Background())
	msg.WithClientReqHead(
		&Request{
			op: opTransaction,
			tx: tx,
		},
	)
	msg.WithClientRspHead(&Response{})

	mock.ExpectBegin().WillReturnError(nil)
	mock.ExpectCommit().WillReturnError(nil)

	// case: tx fail
	ctxWithTxFail, msg := codec.WithNewMessage(context.Background())
	msg.WithClientReqHead(
		&Request{
			op: opTransaction,
			tx: txFail,
		},
	)
	msg.WithClientRspHead(&Response{})

	mock.ExpectBegin().WillReturnError(fmt.Errorf("begin err"))

	mock.ExpectBegin().WillReturnError(nil)
	mock.ExpectRollback().WillReturnError(nil)

	// case: tx rollback fail
	mock.ExpectBegin().WillReturnError(nil)
	mock.ExpectRollback().WillReturnError(fmt.Errorf("rollback err"))

	// case: commit fail
	ctxWithCommitFail, msg := codec.WithNewMessage(context.Background())
	msg.WithClientReqHead(
		&Request{
			op: opTransaction,
			tx: tx,
		},
	)
	msg.WithClientRspHead(&Response{})

	mock.ExpectBegin().WillReturnError(nil)
	mock.ExpectCommit().WillReturnError(errCommitFail)

	tp := NewClientTransport().(*ClientTransport)
	type fields struct {
		opts         *transport.ClientTransportOptions
		clickhouseDB map[string]*sql.DB
	}
	type args struct {
		ctx      context.Context
		reqBuf   []byte
		callOpts []transport.RoundTripOption
	}
	tests := []struct {
		name       string
		fields     fields
		args       args
		wantRspBuf []byte
		wantErr    bool
	}{
		{
			name:   "test_tx",
			fields: fields{opts: nil, clickhouseDB: nil},
			args: args{
				ctx: ctxWithTx,
				callOpts: []transport.RoundTripOption{
					transport.WithDialAddress("test_address"),
				},
			},
			wantErr: false,
		},
		{
			name:   "test_tx_begin_erorr",
			fields: fields{opts: nil, clickhouseDB: nil},
			args: args{
				ctx: ctxWithTxFail,
				callOpts: []transport.RoundTripOption{
					transport.WithDialAddress("test_address"),
				},
			},
			wantErr: true,
		},
		{
			name:   "test_tx_erorr",
			fields: fields{opts: nil, clickhouseDB: nil},
			args: args{
				ctx: ctxWithTxFail,
				callOpts: []transport.RoundTripOption{
					transport.WithDialAddress("test_address"),
				},
			},
			wantErr: true,
		},
		{
			name:   "test_tx_roll_erorr",
			fields: fields{opts: nil, clickhouseDB: nil},
			args: args{
				ctx: ctxWithTxFail,
				callOpts: []transport.RoundTripOption{
					transport.WithDialAddress("test_address"),
				},
			},
			wantErr: true,
		},
		{
			name:   "test_commit_erorr",
			fields: fields{opts: nil, clickhouseDB: nil},
			args: args{
				ctx: ctxWithCommitFail,
				callOpts: []transport.RoundTripOption{
					transport.WithDialAddress("test_address"),
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				ct := tp
				_, err := ct.RoundTrip(tt.args.ctx, tt.args.reqBuf, tt.args.callOpts...)
				if (err != nil) != tt.wantErr {
					t.Errorf("RoundTrip() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
			},
		)
	}
}

func TestClientTransport_GetDB(t *testing.T) {
	db, _, err := sqlmock.New()
	pathes := gomonkey.ApplyFunc(
		sql.Open, func(driverName, dataSourceName string) (*sql.DB, error) {
			return db, err
		},
	)
	defer pathes.Reset()
	type args struct {
		dsn string
	}
	tests := []struct {
		name    string
		args    args
		want    *sql.DB
		wantErr bool
	}{
		{
			name: "test_begin",
			args: args{
				dsn: "common",
			},
			want:    db,
			wantErr: false,
		},
		{
			name: "test_cache",
			args: args{
				dsn: "common",
			},
			want:    db,
			wantErr: false,
		},
	}
	tp := NewClientTransport().(*ClientTransport)
	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				ct := tp
				got, err := ct.GetDB(tt.args.dsn)
				if (err != nil) != tt.wantErr {
					t.Errorf("GetDB() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("GetDB() got = %v, want %v", got, tt.want)
				}
			},
		)
	}
}

func TestClientTransport_GetDBFail(t *testing.T) {
	pathes := gomonkey.ApplyFunc(
		sql.Open, func(driverName, dataSourceName string) (*sql.DB, error) {
			return nil, errors.New("sql open fail")
		},
	)
	defer pathes.Reset()

	type args struct {
		dsn string
	}
	tests := []struct {
		name    string
		args    args
		want    *sql.DB
		wantErr bool
	}{
		{
			name: "test_fail",
			args: args{
				dsn: "common",
			},
			want:    nil,
			wantErr: true,
		},
	}
	tp := NewClientTransport().(*ClientTransport)
	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				ct := tp
				got, err := ct.GetDB(tt.args.dsn)
				if (err != nil) != tt.wantErr {
					t.Errorf("GetDB() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("GetDB() got = %v, want %v", got, tt.want)
				}
			},
		)
	}
}

func TestClientTransport_setOptions(t *testing.T) {
	type fields struct {
		options options
	}
	type args struct {
		opt *options
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *ClientTransport
	}{
		{
			name: "nil",
			fields: fields{
				options: options{
					MaxIdle:     1,
					MaxOpen:     2,
					MaxLifetime: 3 * time.Second,
				},
			},
			args: args{
				opt: nil,
			},
			want: &ClientTransport{
				options: options{
					MaxIdle:     1,
					MaxOpen:     2,
					MaxLifetime: 3 * time.Second,
				},
			},
		},
		{
			name: "opt",
			fields: fields{
				options: options{
					MaxIdle:     1,
					MaxOpen:     2,
					MaxLifetime: 3 * time.Second,
				},
			},
			args: args{
				opt: &options{
					MaxIdle:     10,
					MaxOpen:     100,
					MaxLifetime: 30 * time.Second,
				},
			},
			want: &ClientTransport{
				options: options{
					MaxIdle:     10,
					MaxOpen:     100,
					MaxLifetime: 30 * time.Second,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				ct := &ClientTransport{
					options: tt.fields.options,
				}
				ct.setOptions(tt.args.opt)
				if !reflect.DeepEqual(ct, tt.want) {
					t.Errorf("setOptions() got = %v, want %v", ct, tt.want)
				}
			},
		)
	}
}

func Test_withRemoteAddr(t *testing.T) {
	type args struct {
		msg codec.Msg
		dsn string
	}

	msgWithNil := codec.Message(context.Background())
	addr, _ := net.ResolveTCPAddr("tcp", "xxxx")
	msgWithNil.WithRemoteAddr(addr)

	msgWithAddr := codec.Message(context.Background())
	addr, _ = net.ResolveTCPAddr("tcp", "127.0.0.1:9001")
	msgWithAddr.WithRemoteAddr(addr)

	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "v1",
			args: args{
				msg: codec.Message(context.Background()),
				dsn: "127.0.0.1:9000?username=*&password=*&database=*",
			},
			want: "127.0.0.1:9000",
		},
		{
			name: "v2",
			args: args{
				msg: codec.Message(context.Background()),
				dsn: "user:pswd@127.0.0.1:9000/db?dial_timeout=200ms",
			},
			want: "127.0.0.1:9000",
		},
		{
			name: "v2 multi host",
			args: args{
				msg: codec.Message(context.Background()),
				dsn: "user:pswd@127.0.0.1:9000,127.0.0.1:9001/db?dial_timeout=200ms",
			},
			want: "127.0.0.1:9000",
		},
		{
			name: "already set nil",
			args: args{
				msg: msgWithNil,
				dsn: "user:pswd@127.0.0.1:9000/db?dial_timeout=200ms",
			},
			want: "127.0.0.1:9000",
		},
		{
			name: "already set ok",
			args: args{
				msg: msgWithAddr,
				dsn: "user:pswd@127.0.0.1:9000/db?dial_timeout=200ms",
			},
			want: "127.0.0.1:9001",
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				fmt.Printf("before: %#v\n", tt.args.msg.RemoteAddr())
				withRemoteAddr(tt.args.msg, tt.args.dsn)
				fmt.Printf("after: %#v\n", tt.args.msg.RemoteAddr())
				if tt.args.msg.RemoteAddr().String() != tt.want {
					t.Errorf("withRemoteAddr() got = %v, want %v", tt.args.msg.RemoteAddr().String(), tt.want)
				}
			},
		)
	}
}

func Test_withCommonMeta(t *testing.T) {
	type args struct {
		msg codec.Msg
		dsn string
	}

	msgWithNil := codec.Message(context.Background())
	addr, _ := net.ResolveTCPAddr("tcp", "xxxx")
	msgWithNil.WithRemoteAddr(addr)

	msgWithAddr := codec.Message(context.Background())
	addr, _ = net.ResolveTCPAddr("tcp", "127.0.0.1:9001")
	msgWithAddr.WithRemoteAddr(addr)

	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "v1",
			args: args{
				msg: codec.Message(context.Background()),
				dsn: "127.0.0.1:9000?username=*&password=*&database=*",
			},
			want: calleeApp + "." + "127.0.0.1:9000",
		},
		{
			name: "v2",
			args: args{
				msg: codec.Message(context.Background()),
				dsn: "user:pswd@127.0.0.1:9000/db?dial_timeout=200ms",
			},
			want: calleeApp + "." + "127.0.0.1:9000",
		},
		{
			name: "v2 multi host",
			args: args{
				msg: codec.Message(context.Background()),
				dsn: "user:pswd@127.0.0.1:9000,127.0.0.1:9001/db?dial_timeout=200ms",
			},
			want: calleeApp + "." + "127.0.0.1:9000",
		},
		{
			name: "already set nil",
			args: args{
				msg: msgWithNil,
				dsn: "user:pswd@127.0.0.1:9000/db?dial_timeout=200ms",
			},
			want: calleeApp + "." + "127.0.0.1:9000",
		},
		{
			name: "already set ok",
			args: args{
				msg: msgWithAddr,
				dsn: "user:pswd@127.0.0.1:9000/db?dial_timeout=200ms",
			},
			want: calleeApp + "." + "127.0.0.1:9001",
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				// Need to rely on setting the ip address first.
				withRemoteAddr(tt.args.msg, tt.args.dsn)
				withCommonMeta(tt.args.msg, filledCommonMeta(tt.args.msg, calleeApp, tt.args.msg.RemoteAddr().String()))
				meta := tt.args.msg.CommonMeta()
				res := meta["overrideCalleeApp"].(string) + "." + meta["overrideCalleeServer"].(string)
				if res != tt.want {
					t.Errorf("withCommonMeta() got = %v, want %v", res, tt.want)
				}
			},
		)
	}
}

type customAddr struct{}

func (c *customAddr) Network() string {
	return ""
}

func (c *customAddr) String() string {
	return ""
}

func Test_validNetAddr(t *testing.T) {
	type args struct {
		addr net.Addr
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "nil",
			args: args{
				addr: nil,
			},
			want: false,
		},
		{
			name: "net.TCPAddr",
			args: args{
				addr: (*net.TCPAddr)(nil),
			},
			want: false,
		},
		{
			name: "net.UDPAddr",
			args: args{
				addr: (*net.UDPAddr)(nil),
			},
			want: false,
		},
		{
			name: "net.UnixAddr",
			args: args{
				addr: (*net.UnixAddr)(nil),
			},
			want: false,
		},
		{
			name: "net.IPAddr",
			args: args{
				addr: (*net.IPAddr)(nil),
			},
			want: false,
		},
		{
			name: "net.IPNet",
			args: args{
				addr: (*net.IPNet)(nil),
			},
			want: false,
		},
		{
			name: "customAddr",
			args: args{
				addr: (*customAddr)(nil),
			},
			want: false,
		},
		{
			name: "net.TCPAddr ok",
			args: args{
				addr: &net.TCPAddr{
					IP:   net.ParseIP("127.0.0.1"),
					Port: 9000,
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				if got := validNetAddr(tt.args.addr); got != tt.want {
					t.Errorf("validNetAddr() = %v, want %v", got, tt.want)
				}
			},
		)
	}
}

func Benchmark_validNetAddr(b *testing.B) {
	inputs := []net.Addr{
		nil, (*net.TCPAddr)(nil), (*net.UDPAddr)(nil), (*net.UnixAddr)(nil), (*net.IPAddr)(nil), &net.TCPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 9000,
		}, &net.UDPAddr{
			IP:   net.ParseIP("127.0.0.2"),
			Port: 9000,
		}, &net.UnixAddr{
			Name: "sock",
			Net:  "unix",
		},
		&net.IPAddr{
			IP: net.ParseIP("127.0.0.3"),
		},
	}
	l := len(inputs)
	for n := 0; n < b.N; n++ {
		validNetAddr(inputs[n%l])
	}
}
