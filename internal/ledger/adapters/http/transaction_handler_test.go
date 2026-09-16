package httpadapter

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"financial-ledger/internal/ledger/application/transaction"
	"financial-ledger/internal/ledger/domain"
)

func TestTransactionHandlerPostReturnsCreated(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	repository := &handlerFakeLedgerRepository{}
	useCase := transaction.NewPostTransactionUseCase(repository)
	handler := NewTransactionHandler(
		useCase,
		transaction.GetTransactionUseCase{},
		logger,
	)

	body := `{
		"id": "transaction-001",
		"journal_entry_id": "journal-entry-001",
		"description": "Transfer between accounts",
		"currency": "BRL",
		"postings": [
			{
				"id": "posting-debit-001",
				"account_id": "account-debit-001",
				"direction": "debit",
				"amount_minor_units": 10000
			},
			{
				"id": "posting-credit-001",
				"account_id": "account-credit-001",
				"direction": "credit",
				"amount_minor_units": 10000
			}
		]
	}`

	request := httptest.NewRequest(
		http.MethodPost,
		"/transactions",
		bytes.NewBufferString(body),
	)
	request.Header.Set(idempotencyKeyHeader, "transfer-001")
	recorder := httptest.NewRecorder()

	handler.Post(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("unexpected status code: got %d, body: %s", recorder.Code, recorder.Body.String())
	}

	var response transactionResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response.ID != "transaction-001" {
		t.Fatalf("unexpected transaction ID: %q", response.ID)
	}
	if response.Status != "posted" {
		t.Fatalf("unexpected transaction status: %q", response.Status)
	}
	if repository.idempotencyKey != "transfer-001" {
		t.Fatalf("unexpected idempotency key: %q", repository.idempotencyKey)
	}
	if repository.requestHash == "" {
		t.Fatal("expected request hash")
	}
}

func TestTransactionHandlerPostRequiresIdempotencyKey(t *testing.T) {
	useCase := transaction.NewPostTransactionUseCase(
		&handlerFakeLedgerRepository{},
	)
	handler := NewTransactionHandler(
		useCase,
		transaction.GetTransactionUseCase{},
		nil,
	)

	request := httptest.NewRequest(
		http.MethodPost,
		"/transactions",
		bytes.NewBufferString(`{}`),
	)
	recorder := httptest.NewRecorder()

	handler.Post(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status code: got %d", recorder.Code)
	}
}

func TestTransactionHandlerPostRejectsUnbalancedTransaction(t *testing.T) {
	useCase := transaction.NewPostTransactionUseCase(
		&handlerFakeLedgerRepository{},
	)
	handler := NewTransactionHandler(
		useCase,
		transaction.GetTransactionUseCase{},
		nil,
	)

	body := `{
		"id": "transaction-001",
		"journal_entry_id": "journal-entry-001",
		"description": "Unbalanced transaction",
		"currency": "BRL",
		"postings": [
			{"id":"posting-1","account_id":"account-1","direction":"debit","amount_minor_units":10000},
			{"id":"posting-2","account_id":"account-2","direction":"credit","amount_minor_units":9000}
		]
	}`

	request := httptest.NewRequest(
		http.MethodPost,
		"/transactions",
		bytes.NewBufferString(body),
	)
	request.Header.Set(idempotencyKeyHeader, "transfer-001")
	recorder := httptest.NewRecorder()

	handler.Post(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status code: got %d, body: %s", recorder.Code, recorder.Body.String())
	}
}

func TestNewRouterRegistersTransactionEndpoint(t *testing.T) {
	useCase := transaction.NewPostTransactionUseCase(
		&handlerFakeLedgerRepository{},
	)
	handler := NewTransactionHandler(
		useCase,
		transaction.GetTransactionUseCase{},
		nil,
	)
	router := NewRouter(
		nil,
		func(context.Context) error { return nil },
		handler,
		nil,
	)

	body := `{
		"id": "transaction-001",
		"journal_entry_id": "journal-entry-001",
		"description": "Transfer",
		"currency": "BRL",
		"postings": [
			{"id":"posting-1","account_id":"account-1","direction":"debit","amount_minor_units":10000},
			{"id":"posting-2","account_id":"account-2","direction":"credit","amount_minor_units":10000}
		]
	}`

	request := httptest.NewRequest(
		http.MethodPost,
		"/transactions",
		bytes.NewBufferString(body),
	)
	request.Header.Set(idempotencyKeyHeader, "transfer-001")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("unexpected status code: got %d, body: %s", recorder.Code, recorder.Body.String())
	}
}

func TestTransactionHandlerGetReturnsTransactionDetails(t *testing.T) {
	createdAt := time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	postedAt := createdAt.Add(time.Second)
	reader := &fakeTransactionDetailsReader{
		details: transaction.TransactionDetails{
			ID:          domain.TransactionID("transaction-001"),
			Description: "Transfer",
			CreatedAt:   createdAt,
			JournalEntry: transaction.JournalEntryDetails{
				ID:       domain.JournalEntryID("journal-entry-001"),
				Currency: "BRL",
				PostedAt: postedAt,
				Postings: []transaction.PostingDetails{
					{
						ID:               domain.PostingID("posting-001"),
						AccountID:        domain.AccountID("account-001"),
						Direction:        domain.PostingDirectionDebit,
						AmountMinorUnits: 10000,
					},
				},
			},
		},
	}
	getUseCase := transaction.NewGetTransactionUseCase(reader)
	handler := NewTransactionHandler(
		transaction.PostTransactionUseCase{},
		getUseCase,
		nil,
	)
	router := NewRouter(nil, func(context.Context) error { return nil }, handler, nil)

	request := httptest.NewRequest(
		http.MethodGet,
		"/transactions/transaction-001",
		nil,
	)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: got %d, body: %s", recorder.Code, recorder.Body.String())
	}

	var response transactionDetailsResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response.ID != "transaction-001" {
		t.Fatalf("unexpected transaction ID: %q", response.ID)
	}
	if len(response.JournalEntry.Postings) != 1 {
		t.Fatalf("unexpected posting count: %d", len(response.JournalEntry.Postings))
	}
	if !response.CreatedAt.Equal(createdAt) {
		t.Fatalf("unexpected created_at: %v", response.CreatedAt)
	}
}

func TestTransactionHandlerGetReturnsNotFound(t *testing.T) {
	reader := &fakeTransactionDetailsReader{
		err: transaction.ErrTransactionNotFound,
	}
	getUseCase := transaction.NewGetTransactionUseCase(reader)
	handler := NewTransactionHandler(
		transaction.PostTransactionUseCase{},
		getUseCase,
		nil,
	)
	router := NewRouter(nil, func(context.Context) error { return nil }, handler, nil)

	request := httptest.NewRequest(
		http.MethodGet,
		"/transactions/transaction-404",
		nil,
	)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("unexpected status code: got %d", recorder.Code)
	}
}

type fakeTransactionDetailsReader struct {
	details transaction.TransactionDetails
	err     error
}

func (f *fakeTransactionDetailsReader) Get(
	context.Context,
	domain.TransactionID,
) (transaction.TransactionDetails, error) {
	return f.details, f.err
}

type handlerFakeLedgerRepository struct {
	idempotencyKey string
	requestHash    string
}

func (f *handlerFakeLedgerRepository) Post(
	_ context.Context,
	_ domain.Transaction,
	idempotencyKey string,
	requestHash string,
) error {
	f.idempotencyKey = idempotencyKey
	f.requestHash = requestHash
	return nil
}
