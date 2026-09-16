package transaction

import (
	"context"
	"errors"
	"testing"

	"financial-ledger/internal/ledger/domain"
)

func TestReverseTransactionUseCaseCreatesCompensatingTransaction(t *testing.T) {
	original := validTransactionForReverseTest(t)
	reader := &fakeAggregateReader{
		transaction: original,
	}
	ledger := &fakeReverseLedgerRepository{}

	useCase := NewReverseTransactionUseCase(reader, ledger)

	reversed, err := useCase.Execute(
		context.Background(),
		ReverseTransactionCommand{
			OriginalTransactionID: original.ID(),
			NewTransactionID:      domain.TransactionID("transaction-reversal-001"),
			NewJournalEntryID:     domain.JournalEntryID("journal-reversal-001"),
			NewPostingIDs: []domain.PostingID{
				"posting-reversal-debit-001",
				"posting-reversal-credit-001",
			},
			Description:    "Reversal of original transaction",
			IdempotencyKey: "reversal-001",
			RequestHash:    "hash-001",
		},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if reversed.ID() != "transaction-reversal-001" {
		t.Fatalf("unexpected transaction ID: %s", reversed.ID())
	}

	if !reversed.IsReversal() {
		t.Fatal("expected transaction to be marked as reversal")
	}

	if reversed.ReversesTransactionID() == nil {
		t.Fatal("expected original transaction ID")
	}

	if *reversed.ReversesTransactionID() != original.ID() {
		t.Fatalf(
			"unexpected original transaction ID: %s",
			*reversed.ReversesTransactionID(),
		)
	}

	postings := reversed.JournalEntry().Postings()

	if postings[0].Direction() != domain.PostingDirectionCredit {
		t.Fatalf("expected first posting to be credit")
	}

	if postings[1].Direction() != domain.PostingDirectionDebit {
		t.Fatalf("expected second posting to be debit")
	}

	if ledger.postedTransaction.ID() != reversed.ID() {
		t.Fatal("expected reversal to be persisted")
	}

	if ledger.idempotencyKey != "reversal-001" {
		t.Fatalf("unexpected idempotency key: %s", ledger.idempotencyKey)
	}

	if ledger.requestHash != "hash-001" {
		t.Fatalf("unexpected request hash: %s", ledger.requestHash)
	}
}

func TestReverseTransactionUseCaseRequiresIdempotencyKey(t *testing.T) {
	useCase := NewReverseTransactionUseCase(
		&fakeAggregateReader{},
		&fakeReverseLedgerRepository{},
	)

	_, err := useCase.Execute(
		context.Background(),
		ReverseTransactionCommand{
			OriginalTransactionID: "transaction-001",
			NewTransactionID:      "transaction-reversal-001",
			NewJournalEntryID:     "journal-reversal-001",
			NewPostingIDs: []domain.PostingID{
				"posting-001",
				"posting-002",
			},
			RequestHash: "hash-001",
		},
	)
	if !errors.Is(err, ErrEmptyIdempotencyKey) {
		t.Fatalf("expected ErrEmptyIdempotencyKey, got %v", err)
	}
}

func TestReverseTransactionUseCaseRequiresRequestHash(t *testing.T) {
	useCase := NewReverseTransactionUseCase(
		&fakeAggregateReader{},
		&fakeReverseLedgerRepository{},
	)

	_, err := useCase.Execute(
		context.Background(),
		ReverseTransactionCommand{
			OriginalTransactionID: "transaction-001",
			NewTransactionID:      "transaction-reversal-001",
			NewJournalEntryID:     "journal-reversal-001",
			NewPostingIDs: []domain.PostingID{
				"posting-001",
				"posting-002",
			},
			IdempotencyKey: "reversal-001",
		},
	)
	if !errors.Is(err, ErrEmptyRequestHash) {
		t.Fatalf("expected ErrEmptyRequestHash, got %v", err)
	}
}

func TestReverseTransactionUseCasePropagatesOriginalTransactionError(t *testing.T) {
	expectedErr := errors.New("database unavailable")

	useCase := NewReverseTransactionUseCase(
		&fakeAggregateReader{
			err: expectedErr,
		},
		&fakeReverseLedgerRepository{},
	)

	_, err := useCase.Execute(
		context.Background(),
		ReverseTransactionCommand{
			OriginalTransactionID: "transaction-001",
			NewTransactionID:      "transaction-reversal-001",
			NewJournalEntryID:     "journal-reversal-001",
			NewPostingIDs: []domain.PostingID{
				"posting-001",
				"posting-002",
			},
			IdempotencyKey: "reversal-001",
			RequestHash:    "hash-001",
		},
	)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected original error, got %v", err)
	}
}

func TestReverseTransactionUseCaseRejectsReversalOfReversal(t *testing.T) {
	original := validTransactionForReverseTest(t)

	reversal, err := original.Reverse(
		"transaction-reversal-001",
		"journal-reversal-001",
		[]domain.PostingID{
			"posting-reversal-001",
			"posting-reversal-002",
		},
		"Original reversal",
	)
	if err != nil {
		t.Fatalf("failed to create test reversal: %v", err)
	}

	useCase := NewReverseTransactionUseCase(
		&fakeAggregateReader{
			transaction: reversal,
		},
		&fakeReverseLedgerRepository{},
	)

	_, err = useCase.Execute(
		context.Background(),
		ReverseTransactionCommand{
			OriginalTransactionID: reversal.ID(),
			NewTransactionID:      "transaction-reversal-002",
			NewJournalEntryID:     "journal-reversal-002",
			NewPostingIDs: []domain.PostingID{
				"posting-reversal-003",
				"posting-reversal-004",
			},
			Description:    "Invalid second reversal",
			IdempotencyKey: "reversal-002",
			RequestHash:    "hash-002",
		},
	)
	if !errors.Is(err, domain.ErrCannotReverseReversal) {
		t.Fatalf(
			"expected ErrCannotReverseReversal, got %v",
			err,
		)
	}
}

func TestReverseTransactionUseCasePropagatesPostError(t *testing.T) {
	expectedErr := errors.New("database unavailable")
	original := validTransactionForReverseTest(t)

	useCase := NewReverseTransactionUseCase(
		&fakeAggregateReader{
			transaction: original,
		},
		&fakeReverseLedgerRepository{
			err: expectedErr,
		},
	)

	_, err := useCase.Execute(
		context.Background(),
		ReverseTransactionCommand{
			OriginalTransactionID: original.ID(),
			NewTransactionID:      "transaction-reversal-001",
			NewJournalEntryID:     "journal-reversal-001",
			NewPostingIDs: []domain.PostingID{
				"posting-reversal-001",
				"posting-reversal-002",
			},
			Description:    "Reversal",
			IdempotencyKey: "reversal-001",
			RequestHash:    "hash-001",
		},
	)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected persistence error, got %v", err)
	}
}

type fakeAggregateReader struct {
	transaction domain.Transaction
	err         error
}

func (f *fakeAggregateReader) GetAggregate(
	context.Context,
	domain.TransactionID,
) (domain.Transaction, error) {
	return f.transaction, f.err
}

type fakeReverseLedgerRepository struct {
	postedTransaction domain.Transaction
	idempotencyKey    string
	requestHash       string
	err               error
}

func (f *fakeReverseLedgerRepository) Post(
	_ context.Context,
	postedTransaction domain.Transaction,
	idempotencyKey string,
	requestHash string,
) error {
	if f.err != nil {
		return f.err
	}

	f.postedTransaction = postedTransaction
	f.idempotencyKey = idempotencyKey
	f.requestHash = requestHash

	return nil
}

func validTransactionForReverseTest(
	t *testing.T,
) domain.Transaction {
	t.Helper()

	currency, err := domain.NewCurrency("BRL")
	if err != nil {
		t.Fatalf("create currency: %v", err)
	}

	amount, err := domain.NewMoney(currency, 10000)
	if err != nil {
		t.Fatalf("create money: %v", err)
	}

	debit, err := domain.NewPosting(
		"posting-debit-001",
		"account-debit-001",
		domain.PostingDirectionDebit,
		amount,
	)
	if err != nil {
		t.Fatalf("create debit posting: %v", err)
	}

	credit, err := domain.NewPosting(
		"posting-credit-001",
		"account-credit-001",
		domain.PostingDirectionCredit,
		amount,
	)
	if err != nil {
		t.Fatalf("create credit posting: %v", err)
	}

	entry, err := domain.NewJournalEntry(
		"journal-entry-001",
		"transaction-001",
		currency,
		[]domain.Posting{debit, credit},
	)
	if err != nil {
		t.Fatalf("create journal entry: %v", err)
	}

	transaction, err := domain.NewTransaction(
		"transaction-001",
		"Original transaction",
		entry,
	)
	if err != nil {
		t.Fatalf("create transaction: %v", err)
	}

	return transaction
}
