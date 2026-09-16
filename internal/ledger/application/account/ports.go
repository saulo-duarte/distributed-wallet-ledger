package account

import (
	"context"
	"time"

	"financial-ledger/internal/ledger/domain"
)

type AccountRepository interface {
	Create(ctx context.Context, account domain.Account) error
}

type AccountEntry struct {
	PostingID      domain.PostingID
	JournalEntryID domain.JournalEntryID
	TransactionID  domain.TransactionID
	Description    string
	Currency       domain.Currency
	Direction      domain.PostingDirection
	AmountMinor    int64
	CreatedAt      time.Time
}

type AccountEntriesCursor struct {
	CreatedAt time.Time
	PostingID domain.PostingID
}

type AccountEntriesRepository interface {
	ListEntries(
		ctx context.Context,
		accountID domain.AccountID,
		limit int,
		cursor *AccountEntriesCursor,
	) ([]AccountEntry, error)
}
