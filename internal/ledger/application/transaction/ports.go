package transaction

import (
	"context"
	"time"

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

type PostingDetails struct {
	ID               domain.PostingID
	AccountID        domain.AccountID
	Direction        domain.PostingDirection
	AmountMinorUnits int64
}

type JournalEntryDetails struct {
	ID       domain.JournalEntryID
	Currency string
	PostedAt time.Time
	Postings []PostingDetails
}

type TransactionDetails struct {
	ID           domain.TransactionID
	Description  string
	CreatedAt    time.Time
	JournalEntry JournalEntryDetails
}

type TransactionReader interface {
	Get(
		ctx context.Context,
		id domain.TransactionID,
	) (TransactionDetails, error)
}
