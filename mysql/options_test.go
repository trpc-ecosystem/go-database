package mysql

import (
	"database/sql"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
)

// TestUnit_Options_P0 TxOption unit test.
func TestUnit_Options_P0(t *testing.T) {
	Convey("TestUnit_Options_P0", t, func() {
		txOpts := &sql.TxOptions{}

		opts := []TxOption{
			WithTxIsolation(1),
			WithTxReadOnly(true),
		}

		for _, o := range opts {
			o(txOpts)
		}

		assert.Equal(t, txOpts.ReadOnly, true)
		assert.Equal(t, txOpts.Isolation, sql.IsolationLevel(1))
	})
}
