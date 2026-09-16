package httpadapter

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"financial-ledger/internal/ledger/application/transaction"
	"financial-ledger/internal/ledger/domain"
)

func TestReversalHandlerCreatesReversalThroughRouter(t *testing.T) {
	original := newHTTPTestTransaction(t)
	repository := &fakeHTTPReversalRepository{
		transaction: original,
	}
	useCase := transaction.NewReverseTransactionUseCase(
		repository,
		repository,
	)
	handler := NewReversalHandler(useCase, slog.Default())
	router := NewRouter(
		nil,
		func(context.Context) error { return nil },
		nil,
		nil,
		handler,
	)

	body := `{
		"id": "transaction-reversal-001",
		"journal_entry_id": "journal-reversal-001",
		"posting_ids": [
			"posting-reversal-001",
			"posting-reversal-002"
		],
		"description": "Reversal of original transaction"
	}`

	request := httptest.NewRequest(
		http.MethodPost,
		"/transactions/transaction-001/reversal",
		bytes.NewBufferString(body),
	)
	request.Header.Set(idempotencyKeyHeader, "reversal-001")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf(
			"unexpected status code: got %d, body: %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}

	var response reversalResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response.ID != "transaction-reversal-001" {
		t.Fatalf("unexpected reversal ID: %q", response.ID)
	}
	if response.ReversedTransactionID != original.ID().String() {
		t.Fatalf(
			"unexpected original transaction ID: %q",
			response.ReversedTransactionID,
		)
	}
	if repository.postedTransaction.ID().String() != response.ID {
		t.Fatal("expected reversal to be persisted")
	}
	if repository.postedTransaction.JournalEntry().Postings()[0].Direction() != domain.PostingDirectionCredit {
		t.Fatal("expected first reversal posting to be credit")
	}
}

type fakeHTTPReversalRepository struct {
	transaction       domain.Transaction
	postedTransaction domain.Transaction
}

func (f *fakeHTTPReversalRepository) GetAggregate(
	context.Context,
	domain.TransactionID,
) (domain.Transaction, error) {
	return f.transaction, nil
}

func (f *fakeHTTPReversalRepository) Post(
	_ context.Context,
	postedTransaction domain.Transaction,
	_ string,
	_ string,
) error {
	f.postedTransaction = postedTransaction
	return nil
}

func newHTTPTestTransaction(t *testing.T) domain.Transaction {
	t.Helper()

	currency, err := domain.NewCurrency("BRL")
	if err != nil {
		t.Fatal(err)
	}

	amount, err := domain.NewMoney(currency, 10000)
	if err != nil {
		t.Fatal(err)
	}

	debit, err := domain.NewPosting(
		"posting-debit-001",
		"account-debit-001",
		domain.PostingDirectionDebit,
		amount,
	)
	if err != nil {
		t.Fatal(err)
	}

	credit, err := domain.NewPosting(
		"posting-credit-001",
		"account-credit-001",
		domain.PostingDirectionCredit,
		amount,
	)
	if err != nil {
		t.Fatal(err)
	}

	journalEntry, err := domain.NewJournalEntry(
		"journal-entry-001",
		"transaction-001",
		currency,
		[]domain.Posting{debit, credit},
	)
	if err != nil {
		t.Fatal(err)
	}

	createdTransaction, err := domain.NewTransaction(
		"transaction-001",
		"Original transaction",
		journalEntry,
	)
	if err != nil {
		t.Fatal(err)
	}

	return createdTransaction
}
