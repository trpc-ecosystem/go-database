package mysql

import "database/sql"

// TxOption transaction options.
type TxOption func(*sql.TxOptions)

// WithTxIsolation setting the transaction isolation level.
func WithTxIsolation(i sql.IsolationLevel) TxOption {
	return func(o *sql.TxOptions) {
		o.Isolation = i
	}
}

// WithTxReadOnly setting up read-only transactions.
func WithTxReadOnly(readOnly bool) TxOption {
	return func(o *sql.TxOptions) {
		o.ReadOnly = readOnly
	}
}
