package transaction

import (
	"context"
	"errors"
	"testing"
	"time"

	"financial-ledger/internal/ledger/domain"
)

type fakeTransactionReader struct {
	details TransactionDetails
	err     error
	id      domain.TransactionID
}

func (f *fakeTransactionReader) Get(
	_ context.Context,
	id domain.TransactionID,
) (TransactionDetails, error) {
	f.id = id
	return f.details, f.err
}

func TestGetTransactionUseCaseReturnsDetails(t *testing.T) {
	createdAt := time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	repository := &fakeTransactionReader{
		details: TransactionDetails{
			ID:          domain.TransactionID("transaction-001"),
			Description: "Transfer",
			CreatedAt:   createdAt,
		},
	}
	useCase := NewGetTransactionUseCase(repository)

	details, err := useCase.Execute(
		context.Background(),
		domain.TransactionID("transaction-001"),
	)
	if err != nil {
		t.Fatalf("get transaction: %v", err)
	}

	if repository.id != domain.TransactionID("transaction-001") {
		t.Fatalf("unexpected transaction ID: %q", repository.id)
	}
	if !details.CreatedAt.Equal(createdAt) {
		t.Fatalf("unexpected creation date: %v", details.CreatedAt)
	}
}

func TestGetTransactionUseCaseRejectsEmptyID(t *testing.T) {
	repository := &fakeTransactionReader{}
	useCase := NewGetTransactionUseCase(repository)

	_, err := useCase.Execute(context.Background(), "")
	if !errors.Is(err, domain.ErrInvalidID) {
		t.Fatalf("expected invalid ID error, got %v", err)
	}

	if repository.id != "" {
		t.Fatal("repository must not be called with an empty ID")
	}
}

func TestGetTransactionUseCasePropagatesNotFound(t *testing.T) {
	repository := &fakeTransactionReader{err: ErrTransactionNotFound}
	useCase := NewGetTransactionUseCase(repository)

	_, err := useCase.Execute(
		context.Background(),
		domain.TransactionID("transaction-001"),
	)
	if !errors.Is(err, ErrTransactionNotFound) {
		t.Fatalf("expected not found error, got %v", err)
	}
}
