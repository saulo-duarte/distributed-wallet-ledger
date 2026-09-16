package account

import (
	"context"
	"errors"
	"testing"
	"time"

	"financial-ledger/internal/ledger/domain"
)

type fakeAccountEntriesRepository struct {
	entries []AccountEntry
	err     error
	limit   int
	cursor  *AccountEntriesCursor
}

func (f *fakeAccountEntriesRepository) ListEntries(
	_ context.Context,
	_ domain.AccountID,
	limit int,
	cursor *AccountEntriesCursor,
) ([]AccountEntry, error) {
	f.limit = limit
	f.cursor = cursor
	if f.err != nil {
		return nil, f.err
	}
	return f.entries, nil
}

func TestListAccountEntriesUseCaseReturnsNextCursor(t *testing.T) {
	createdAt := time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	repository := &fakeAccountEntriesRepository{
		entries: []AccountEntry{
			{PostingID: domain.PostingID("posting-1"), CreatedAt: createdAt},
			{PostingID: domain.PostingID("posting-2"), CreatedAt: createdAt.Add(-time.Minute)},
			{PostingID: domain.PostingID("posting-3"), CreatedAt: createdAt.Add(-2 * time.Minute)},
		},
	}
	useCase := NewListAccountEntriesUseCase(repository)

	page, err := useCase.Execute(
		context.Background(),
		ListAccountEntriesCommand{
			AccountID: domain.AccountID("account-1"),
			Limit:     2,
		},
	)
	if err != nil {
		t.Fatalf("list account entries: %v", err)
	}

	if repository.limit != 3 {
		t.Fatalf("unexpected repository limit: got %d, want 3", repository.limit)
	}

	if len(page.Entries) != 2 {
		t.Fatalf("unexpected entry count: got %d, want 2", len(page.Entries))
	}

	if page.NextCursor == nil {
		t.Fatal("expected next cursor")
	}

	if page.NextCursor.PostingID != domain.PostingID("posting-2") {
		t.Fatalf("unexpected next cursor posting ID: %q", page.NextCursor.PostingID)
	}
}

func TestListAccountEntriesUseCaseUsesDefaultLimit(t *testing.T) {
	repository := &fakeAccountEntriesRepository{}
	useCase := NewListAccountEntriesUseCase(repository)

	_, err := useCase.Execute(
		context.Background(),
		ListAccountEntriesCommand{
			AccountID: domain.AccountID("account-1"),
		},
	)
	if err != nil {
		t.Fatalf("list account entries: %v", err)
	}

	if repository.limit != DefaultAccountEntriesPageSize+1 {
		t.Fatalf(
			"unexpected default repository limit: got %d, want %d",
			repository.limit,
			DefaultAccountEntriesPageSize+1,
		)
	}
}

func TestListAccountEntriesUseCaseRejectsInvalidPageSize(t *testing.T) {
	repository := &fakeAccountEntriesRepository{}
	useCase := NewListAccountEntriesUseCase(repository)

	_, err := useCase.Execute(
		context.Background(),
		ListAccountEntriesCommand{
			AccountID: domain.AccountID("account-1"),
			Limit:     MaxAccountEntriesPageSize + 1,
		},
	)
	if !errors.Is(err, ErrInvalidPageSize) {
		t.Fatalf("expected invalid page size error, got %v", err)
	}

	if repository.limit != 0 {
		t.Fatal("repository must not be called for invalid page size")
	}
}

func TestListAccountEntriesUseCasePropagatesRepositoryError(t *testing.T) {
	repositoryError := errors.New("database unavailable")
	repository := &fakeAccountEntriesRepository{err: repositoryError}
	useCase := NewListAccountEntriesUseCase(repository)

	_, err := useCase.Execute(
		context.Background(),
		ListAccountEntriesCommand{
			AccountID: domain.AccountID("account-1"),
		},
	)
	if !errors.Is(err, repositoryError) {
		t.Fatalf("expected repository error, got %v", err)
	}
}
