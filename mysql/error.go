// Package mysql Built-in error checking function to distinguish errors that occur during mysql execution
package mysql

import (
	"database/sql"
	"errors"

	"trpc.group/trpc-go/trpc-go/errs"
)

const (
	errcodeDupEntry = 1062
	errcodeSyntax   = 1064

	errcodeNoRows = 10000
)

var (
	// ErrNoRows equals to database/sql.ErrNoRows.
	ErrNoRows = errs.New(errcodeNoRows, "trpc-mysql: no rows in result set")
)

// IsDupEntryError Whether there are duplicate field values.
func IsDupEntryError(err error) bool {
	e, ok := err.(*errs.Error)
	if !ok {
		return false
	}

	return e.Code == errcodeDupEntry
}

// IsSyntaxError Whether there is a sql syntax error.
func IsSyntaxError(err error) bool {
	e, ok := err.(*errs.Error)
	if !ok {
		return false
	}

	return e.Code == errcodeSyntax
}

// IsNoRowsError Determine if it is an ErrNoRows type error.
func IsNoRowsError(err error) bool {
	if e, ok := err.(*errs.Error); ok {
		return e.Code == errcodeNoRows
	}

	return errors.Is(err, sql.ErrNoRows)
}
