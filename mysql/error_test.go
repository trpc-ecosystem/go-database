package mysql

import (
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"trpc.group/trpc-go/trpc-go/errs"
)

var (
	dupEntryError = errs.New(errcodeDupEntry, "dupEntryError")
	syntaxError   = errs.New(errcodeSyntax, "syntaxError")
	stdError      = errors.New("stdError")
)

func TestUnit_IsDupEntryError(t *testing.T) {
	type args struct {
		err error
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			"touch IsDupEntryError",
			args{dupEntryError},
			true,
		},
		{
			"not touch IsDupEntryError",
			args{stdError},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsDupEntryError(tt.args.err); got != tt.want {
				t.Errorf("IsDupEntryError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUnit_IsSyntaxError(t *testing.T) {
	type args struct {
		err error
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			"touch IsSyntaxError",
			args{syntaxError},
			true,
		},
		{
			"not touch IsSyntaxError",
			args{stdError},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsSyntaxError(tt.args.err); got != tt.want {
				t.Errorf("IsSyntaxError() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestUnit_IsNoRowsError Line error unit test.
func TestUnit_IsNoRowsError(t *testing.T) {
	type args struct {
		err error
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"other err", args{err: errors.New("err")}, false},
		{"no rows err", args{err: errs.New(errcodeNoRows, "no rows")}, true},
		{"no rows err from standard `sql` library", args{err: sql.ErrNoRows}, true},
		{
			"wrapped no rows err from standard `sql` library",
			args{err: fmt.Errorf("something went wrong %w", sql.ErrNoRows)},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNoRowsError(tt.args.err); got != tt.want {
				t.Errorf("IsNoRowsError() = %v, want %v", got, tt.want)
			}
		})
	}
}
