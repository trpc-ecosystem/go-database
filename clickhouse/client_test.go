package clickhouse

import (
	"database/sql"
	"fmt"
	"reflect"
	"testing"
)

type customCopier struct {
	val interface{}
	err error
}

func (c *customCopier) Copy() (interface{}, error) {
	return c.val, c.err
}
func (c *customCopier) CopyTo(dst interface{}) error {
	return c.err
}

func TestRequest_Copy(t *testing.T) {
	type fields struct {
		Query              string
		Exec               string
		Args               []interface{}
		op                 int
		next               NextFunc
		tx                 TxFunc
		queryToStructsDest interface{}
		queryRowDest       []interface{}
	}
	tests := []struct {
		name    string
		fields  fields
		want    interface{}
		wantErr bool
	}{
		{
			name: "next err",
			fields: fields{
				next: NextFunc(
					func(rows *sql.Rows) error {
						return nil
					},
				),
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "tx err",
			fields: fields{
				tx: TxFunc(
					func(s *sql.Tx) error {
						return nil
					},
				),
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "queryToStructsDest copy err",
			fields: fields{
				queryToStructsDest: &customCopier{err: fmt.Errorf("custom")},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "succ",
			fields: fields{
				Query: "query",
			},
			want: &Request{
				Query: "query",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				r := &Request{
					Query:              tt.fields.Query,
					Exec:               tt.fields.Exec,
					Args:               tt.fields.Args,
					op:                 tt.fields.op,
					next:               tt.fields.next,
					tx:                 tt.fields.tx,
					queryToStructsDest: tt.fields.queryToStructsDest,
					queryRowDest:       tt.fields.queryRowDest,
				}
				got, err := r.Copy()
				if (err != nil) != tt.wantErr {
					t.Errorf("Copy() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("Copy() got = %v, want %v", got, tt.want)
				}
			},
		)
	}
}

func TestRequest_CopyTo(t *testing.T) {
	type fields struct {
		Query              string
		Exec               string
		Args               []interface{}
		op                 int
		next               NextFunc
		tx                 TxFunc
		queryToStructsDest interface{}
		queryRowDest       []interface{}
	}
	type args struct {
		dst interface{}
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "next err",
			fields: fields{
				next: NextFunc(
					func(rows *sql.Rows) error {
						return nil
					},
				),
			},
			args: args{
				dst: nil,
			},
			wantErr: true,
		},
		{
			name: "tx err",
			fields: fields{
				tx: TxFunc(
					func(s *sql.Tx) error {
						return nil
					},
				),
			},
			args: args{
				dst: nil,
			},
			wantErr: true,
		},
		{
			name:   "dst type err",
			fields: fields{},
			args: args{
				dst: nil,
			},
			wantErr: true,
		},
		{
			name: "queryToStructsDest ShallowCopy err",
			fields: fields{
				queryToStructsDest: &customCopier{err: fmt.Errorf("custom")},
			},
			args: args{
				dst: &Request{
					queryToStructsDest: 1,
				},
			},
			wantErr: true,
		},
		{
			name: "len err",
			fields: fields{
				queryRowDest: []interface{}{""},
			},
			args: args{
				dst: &Request{
					queryRowDest: []interface{}{},
				},
			},
			wantErr: true,
		},
		{
			name: "queryRowDest ShallowCopy err",
			fields: fields{
				queryRowDest: []interface{}{&customCopier{err: fmt.Errorf("custom")}},
			},
			args: args{
				dst: &Request{
					queryRowDest: []interface{}{1},
				},
			},
			wantErr: true,
		},
		{
			name: "succ",
			fields: fields{
				queryRowDest: []interface{}{&[]interface{}{1}},
			},
			args: args{
				dst: &Request{
					queryRowDest: []interface{}{&[]interface{}{0}},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				r := &Request{
					Query:              tt.fields.Query,
					Exec:               tt.fields.Exec,
					Args:               tt.fields.Args,
					op:                 tt.fields.op,
					next:               tt.fields.next,
					tx:                 tt.fields.tx,
					queryToStructsDest: tt.fields.queryToStructsDest,
					queryRowDest:       tt.fields.queryRowDest,
				}
				if err := r.CopyTo(tt.args.dst); (err != nil) != tt.wantErr {
					t.Errorf("CopyTo() error = %v, wantErr %v", err, tt.wantErr)
				}
			},
		)
	}
}
