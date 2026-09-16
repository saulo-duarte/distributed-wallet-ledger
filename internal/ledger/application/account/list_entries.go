package account

import (
	"context"
	"errors"

	"financial-ledger/internal/ledger/domain"
)

const (
	DefaultAccountEntriesPageSize = 20
	MaxAccountEntriesPageSize     = 100
)

var ErrInvalidPageSize = errors.New("invalid page size")

type ListAccountEntriesCommand struct {
	AccountID domain.AccountID
	Limit     int
	Cursor    *AccountEntriesCursor
}

type AccountEntriesPage struct {
	Entries    []AccountEntry
	NextCursor *AccountEntriesCursor
}

type ListAccountEntriesUseCase struct {
	entries AccountEntriesRepository
}

func NewListAccountEntriesUseCase(
	entries AccountEntriesRepository,
) ListAccountEntriesUseCase {
	return ListAccountEntriesUseCase{
		entries: entries,
	}
}

func (uc ListAccountEntriesUseCase) Execute(
	ctx context.Context,
	command ListAccountEntriesCommand,
) (AccountEntriesPage, error) {
	if command.AccountID.IsZero() {
		return AccountEntriesPage{}, domain.ErrInvalidID
	}

	limit := command.Limit
	if limit == 0 {
		limit = DefaultAccountEntriesPageSize
	}

	if limit < 1 || limit > MaxAccountEntriesPageSize {
		return AccountEntriesPage{}, ErrInvalidPageSize
	}

	entries, err := uc.entries.ListEntries(
		ctx,
		command.AccountID,
		limit+1,
		command.Cursor,
	)
	if err != nil {
		return AccountEntriesPage{}, err
	}

	page := AccountEntriesPage{
		Entries: entries,
	}

	if len(entries) <= limit {
		return page, nil
	}

	page.Entries = entries[:limit]
	lastEntry := page.Entries[len(page.Entries)-1]
	page.NextCursor = &AccountEntriesCursor{
		CreatedAt: lastEntry.CreatedAt,
		PostingID: lastEntry.PostingID,
	}

	return page, nil
}
