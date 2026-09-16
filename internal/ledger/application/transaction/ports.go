package transaction

import (
	"context"
	"financial-ledger/internal/ledger/domain"
)

type LedgerRepository interface {
	Post(
		ctx context.Context,
		transaction domain.Transaction,
		idempotencyKey string,
		requestHash string,
	) error
}
