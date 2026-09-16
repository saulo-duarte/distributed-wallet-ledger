package transaction

import (
	"context"
	"errors"
	"testing"

	"financial-ledger/internal/ledger/domain"
)

type fakeLedgerRepository struct {
	postCalled     bool
	posted         domain.Transaction
	idempotencyKey string
	requestHash    string
	err            error
}

func (f *fakeLedgerRepository) Post(
	_ context.Context,
	transaction domain.Transaction,
	idempotencyKey string,
	requestHash string,
) error {
	f.postCalled = true
	f.posted = transaction
	f.idempotencyKey = idempotencyKey
	f.requestHash = requestHash

	return f.err
}

func TestPostTransactionUseCaseCreatesBalancedTransaction(t *testing.T) {
	repository := &fakeLedgerRepository{}
	useCase := NewPostTransactionUseCase(repository)

	transactionID, err := domain.NewTransactionID("transaction-001")
	if err != nil {
		t.Fatal(err)
	}

	journalEntryID, err := domain.NewJournalEntryID("journal-entry-001")
	if err != nil {
		t.Fatal(err)
	}

	debitPostingID, err := domain.NewPostingID("posting-debit-001")
	if err != nil {
		t.Fatal(err)
	}

	creditPostingID, err := domain.NewPostingID("posting-credit-001")
	if err != nil {
		t.Fatal(err)
	}

	debitAccountID, err := domain.NewAccountID("account-debit-001")
	if err != nil {
		t.Fatal(err)
	}

	creditAccountID, err := domain.NewAccountID("account-credit-001")
	if err != nil {
		t.Fatal(err)
	}

	created, err := useCase.Execute(
		context.Background(),
		PostTransactionCommand{
			ID:             transactionID,
			JournalEntryID: journalEntryID,
			Description:    "Transfer between accounts",
			CurrencyCode:   "BRL",
			IdempotencyKey: "transfer-001",
			RequestHash:    "hash-001",
			Postings: []PostingCommand{
				{
					ID:               debitPostingID,
					AccountID:        debitAccountID,
					Direction:        domain.PostingDirectionDebit,
					AmountMinorUnits: 10000,
				},
				{
					ID:               creditPostingID,
					AccountID:        creditAccountID,
					Direction:        domain.PostingDirectionCredit,
					AmountMinorUnits: 10000,
				},
			},
		},
	)
	if err != nil {
		t.Fatalf("execute post transaction: %v", err)
	}

	if !repository.postCalled {
		t.Fatal("expected repository Post to be called")
	}

	if created.ID() != transactionID {
		t.Fatalf("unexpected transaction ID: got %q", created.ID())
	}

	if repository.posted.ID() != transactionID {
		t.Fatalf("unexpected persisted transaction ID: got %q", repository.posted.ID())
	}

	if repository.idempotencyKey != "transfer-001" {
		t.Fatalf("unexpected idempotency key: %q", repository.idempotencyKey)
	}

	if repository.requestHash != "hash-001" {
		t.Fatalf("unexpected request hash: %q", repository.requestHash)
	}

	balanced, err := created.JournalEntry().IsBalanced()
	if err != nil {
		t.Fatalf("check balance: %v", err)
	}

	if !balanced {
		t.Fatal("expected transaction to be balanced")
	}
}

func TestPostTransactionUseCaseRejectsUnbalancedTransaction(t *testing.T) {
	repository := &fakeLedgerRepository{}
	useCase := NewPostTransactionUseCase(repository)

	transactionID := domain.TransactionID("transaction-001")
	journalEntryID := domain.JournalEntryID("journal-entry-001")

	_, err := useCase.Execute(
		context.Background(),
		PostTransactionCommand{
			ID:             transactionID,
			JournalEntryID: journalEntryID,
			Description:    "Unbalanced transaction",
			CurrencyCode:   "BRL",
			IdempotencyKey: "transfer-001",
			RequestHash:    "hash-001",
			Postings: []PostingCommand{
				{
					ID:               domain.PostingID("posting-debit-001"),
					AccountID:        domain.AccountID("account-debit-001"),
					Direction:        domain.PostingDirectionDebit,
					AmountMinorUnits: 10000,
				},
				{
					ID:               domain.PostingID("posting-credit-001"),
					AccountID:        domain.AccountID("account-credit-001"),
					Direction:        domain.PostingDirectionCredit,
					AmountMinorUnits: 9000,
				},
			},
		},
	)
	if !errors.Is(err, domain.ErrUnbalancedJournalEntry) {
		t.Fatalf("expected unbalanced journal error, got %v", err)
	}

	if repository.postCalled {
		t.Fatal("repository must not be called for an invalid transaction")
	}
}

func TestPostTransactionUseCaseRejectsMissingIdempotencyKey(t *testing.T) {
	repository := &fakeLedgerRepository{}
	useCase := NewPostTransactionUseCase(repository)

	_, err := useCase.Execute(
		context.Background(),
		PostTransactionCommand{
			ID:             domain.TransactionID("transaction-001"),
			JournalEntryID: domain.JournalEntryID("journal-entry-001"),
			IdempotencyKey: " ",
			RequestHash:    "hash-001",
		},
	)
	if !errors.Is(err, ErrEmptyIdempotencyKey) {
		t.Fatalf("expected empty idempotency key error, got %v", err)
	}

	if repository.postCalled {
		t.Fatal("repository must not be called without idempotency key")
	}
}

func TestPostTransactionUseCasePropagatesRepositoryError(t *testing.T) {
	repositoryError := errors.New("repository unavailable")
	repository := &fakeLedgerRepository{err: repositoryError}
	useCase := NewPostTransactionUseCase(repository)

	_, err := useCase.Execute(
		context.Background(),
		validPostTransactionCommand(),
	)
	if !errors.Is(err, repositoryError) {
		t.Fatalf("expected repository error, got %v", err)
	}
}

func validPostTransactionCommand() PostTransactionCommand {
	return PostTransactionCommand{
		ID:             domain.TransactionID("transaction-001"),
		JournalEntryID: domain.JournalEntryID("journal-entry-001"),
		Description:    "Valid transaction",
		CurrencyCode:   "BRL",
		IdempotencyKey: "transfer-001",
		RequestHash:    "hash-001",
		Postings: []PostingCommand{
			{
				ID:               domain.PostingID("posting-debit-001"),
				AccountID:        domain.AccountID("account-debit-001"),
				Direction:        domain.PostingDirectionDebit,
				AmountMinorUnits: 10000,
			},
			{
				ID:               domain.PostingID("posting-credit-001"),
				AccountID:        domain.AccountID("account-credit-001"),
				Direction:        domain.PostingDirectionCredit,
				AmountMinorUnits: 10000,
			},
		},
	}
}
