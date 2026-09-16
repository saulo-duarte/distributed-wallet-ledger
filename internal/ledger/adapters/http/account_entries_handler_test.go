package httpadapter

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"financial-ledger/internal/ledger/application/account"
	"financial-ledger/internal/ledger/domain"
)

type fakeListAccountEntriesRepository struct {
	entries []account.AccountEntry
	limit   int
	cursor  *account.AccountEntriesCursor
}

func (f *fakeListAccountEntriesRepository) ListEntries(
	_ context.Context,
	_ domain.AccountID,
	limit int,
	cursor *account.AccountEntriesCursor,
) ([]account.AccountEntry, error) {
	f.limit = limit
	f.cursor = cursor
	return f.entries, nil
}

func TestAccountEntriesHandlerReturnsPaginatedEntries(t *testing.T) {
	currency, err := domain.NewCurrency("BRL")
	if err != nil {
		t.Fatal(err)
	}

	createdAt := time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	repository := &fakeListAccountEntriesRepository{
		entries: []account.AccountEntry{
			{
				PostingID:      domain.PostingID("posting-1"),
				JournalEntryID: domain.JournalEntryID("journal-1"),
				TransactionID:  domain.TransactionID("transaction-1"),
				Description:    "Newest entry",
				Currency:       currency,
				Direction:      domain.PostingDirectionDebit,
				AmountMinor:    10000,
				CreatedAt:      createdAt,
			},
			{
				PostingID:      domain.PostingID("posting-2"),
				JournalEntryID: domain.JournalEntryID("journal-2"),
				TransactionID:  domain.TransactionID("transaction-2"),
				Description:    "Older entry",
				Currency:       currency,
				Direction:      domain.PostingDirectionCredit,
				AmountMinor:    5000,
				CreatedAt:      createdAt.Add(-time.Minute),
			},
			{
				PostingID:      domain.PostingID("posting-3"),
				JournalEntryID: domain.JournalEntryID("journal-3"),
				TransactionID:  domain.TransactionID("transaction-3"),
				Description:    "Oldest entry",
				Currency:       currency,
				Direction:      domain.PostingDirectionDebit,
				AmountMinor:    2500,
				CreatedAt:      createdAt.Add(-2 * time.Minute),
			},
		},
	}
	useCase := account.NewListAccountEntriesUseCase(repository)
	handler := NewAccountEntriesHandler(useCase, nil)
	router := NewRouter(nil, func(context.Context) error { return nil }, nil, handler)

	request := httptest.NewRequest(
		http.MethodGet,
		"/accounts/account-1/entries?limit=2",
		nil,
	)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: got %d, body: %s", recorder.Code, recorder.Body.String())
	}

	var response accountEntriesResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(response.Items) != 2 {
		t.Fatalf("unexpected item count: got %d, want 2", len(response.Items))
	}
	if response.Items[0].Description != "Newest entry" {
		t.Fatalf("unexpected first item: %q", response.Items[0].Description)
	}
	if response.NextCursor == "" {
		t.Fatal("expected next cursor")
	}
	if repository.limit != 3 {
		t.Fatalf("unexpected repository limit: got %d, want 3", repository.limit)
	}
}

func TestAccountEntriesHandlerRejectsInvalidLimit(t *testing.T) {
	useCase := account.NewListAccountEntriesUseCase(
		&fakeListAccountEntriesRepository{},
	)
	handler := NewAccountEntriesHandler(useCase, nil)
	router := NewRouter(nil, func(context.Context) error { return nil }, nil, handler)

	request := httptest.NewRequest(
		http.MethodGet,
		"/accounts/account-1/entries?limit=101",
		nil,
	)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status code: got %d", recorder.Code)
	}
}

func TestAccountEntriesHandlerRejectsInvalidCursor(t *testing.T) {
	useCase := account.NewListAccountEntriesUseCase(
		&fakeListAccountEntriesRepository{},
	)
	handler := NewAccountEntriesHandler(useCase, nil)
	router := NewRouter(nil, func(context.Context) error { return nil }, nil, handler)

	request := httptest.NewRequest(
		http.MethodGet,
		"/accounts/account-1/entries?cursor=invalid",
		nil,
	)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status code: got %d", recorder.Code)
	}
}
