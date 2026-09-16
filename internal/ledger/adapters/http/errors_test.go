package httpadapter

import (
	"errors"
	"net/http"
	"testing"

	"financial-ledger/internal/ledger/application/transaction"
	"financial-ledger/internal/ledger/domain"
)

func TestMapApplicationError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		statusCode int
		code       string
	}{
		{
			name:       "invalid domain id",
			err:        domain.ErrInvalidID,
			statusCode: http.StatusBadRequest,
			code:       "invalid_id",
		},
		{
			name:       "transaction not found",
			err:        transaction.ErrTransactionNotFound,
			statusCode: http.StatusNotFound,
			code:       "transaction_not_found",
		},
		{
			name:       "unexpected error",
			err:        errors.New("database unavailable"),
			statusCode: http.StatusInternalServerError,
			code:       "internal_error",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mapping := mapApplicationError(test.err)

			if mapping.statusCode != test.statusCode {
				t.Fatalf(
					"unexpected status code: got %d, want %d",
					mapping.statusCode,
					test.statusCode,
				)
			}

			if mapping.code != test.code {
				t.Fatalf(
					"unexpected error code: got %q, want %q",
					mapping.code,
					test.code,
				)
			}
		})
	}
}
