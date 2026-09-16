package domain

import (
	"errors"
	"testing"
)

func TestNewTransaction(t *testing.T) {
	t.Parallel()

	brl := fakeValidCurrency("BRL")
	validEntry := mustTestJournalEntry(
		t,
		JournalEntryID("entry-001"),
		TransactionID("transaction-001"),
		brl,
		validJournalEntryPostings(t, brl),
	)

	tests := []struct {
		name        string
		id          TransactionID
		description string
		entry       JournalEntry
		wantErr     error
		wantDesc    string
	}{
		{
			name:        "creates a transaction and trims its description",
			id:          TransactionID("transaction-001"),
			description: "  transfer funds  ",
			entry:       validEntry,
			wantDesc:    "transfer funds",
		},
		{
			name:        "rejects an empty transaction ID",
			id:          TransactionID(""),
			description: "transfer funds",
			entry:       validEntry,
			wantErr:     ErrInvalidID,
		},
		{
			name:        "rejects an empty description",
			id:          TransactionID("transaction-001"),
			description: "   ",
			entry:       validEntry,
			wantErr:     ErrEmptyTransactionDescription,
		},
		{
			name:        "rejects a journal entry with a different transaction ID",
			id:          TransactionID("transaction-001"),
			description: "transfer funds",
			entry: mustTestJournalEntry(
				t,
				JournalEntryID("entry-002"),
				TransactionID("transaction-002"),
				brl,
				validJournalEntryPostings(t, brl),
			),
			wantErr: ErrTransactionJournalEntryMismatch,
		},
		{
			name:        "propagates an invalid journal entry",
			id:          TransactionID("transaction-001"),
			description: "transfer funds",
			entry:       JournalEntry{},
			wantErr:     ErrInvalidID,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			transaction, err := NewTransaction(tt.id, tt.description, tt.entry)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("NewTransaction() error = %v, want %v", err, tt.wantErr)
			}

			if tt.wantErr != nil {
				return
			}

			if transaction.ID() != tt.id {
				t.Errorf("ID() = %q, want %q", transaction.ID(), tt.id)
			}
			if transaction.Description() != tt.wantDesc {
				t.Errorf("Description() = %q, want %q", transaction.Description(), tt.wantDesc)
			}
			if transaction.IsReversal() {
				t.Error("new transaction should not be a reversal")
			}
			if transaction.ReversesTransactionID() != nil {
				t.Error("new transaction should not reference a reversed transaction")
			}
		})
	}
}

func TestTransaction_Validate(t *testing.T) {
	t.Parallel()

	brl := fakeValidCurrency("BRL")
	entry := mustTestJournalEntry(
		t,
		JournalEntryID("entry-001"),
		TransactionID("transaction-001"),
		brl,
		validJournalEntryPostings(t, brl),
	)

	zeroReversalID := TransactionID("")
	selfReversalID := TransactionID("transaction-001")

	tests := []struct {
		name        string
		transaction Transaction
		wantErr     error
	}{
		{
			name: "accepts a valid transaction",
			transaction: Transaction{
				id:           TransactionID("transaction-001"),
				description:  "transfer funds",
				journalEntry: entry,
			},
		},
		{
			name: "rejects a zero reversal ID",
			transaction: Transaction{
				id:                    TransactionID("transaction-001"),
				description:           "transfer funds",
				journalEntry:          entry,
				reversesTransactionID: &zeroReversalID,
			},
			wantErr: ErrInvalidID,
		},
		{
			name: "rejects a transaction reversing itself",
			transaction: Transaction{
				id:                    TransactionID("transaction-001"),
				description:           "transfer funds",
				journalEntry:          entry,
				reversesTransactionID: &selfReversalID,
			},
			wantErr: ErrReversalSameTransactionID,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.transaction.Validate()
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Validate() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestTransaction_Reverse(t *testing.T) {
	t.Parallel()

	brl := fakeValidCurrency("BRL")
	originalPostings := validJournalEntryPostings(t, brl)
	original := mustTestTransaction(t, "transfer funds", originalPostings)

	reversed, err := original.Reverse(
		TransactionID("transaction-002"),
		JournalEntryID("entry-002"),
		[]PostingID{PostingID("posting-003"), PostingID("posting-004")},
		"  reverse transfer  ",
	)
	if err != nil {
		t.Fatalf("Reverse() returned an unexpected error: %v", err)
	}

	if !reversed.IsReversal() {
		t.Error("reversed transaction should be marked as a reversal")
	}
	if reversed.ID() != TransactionID("transaction-002") {
		t.Errorf("ID() = %q, want transaction-002", reversed.ID())
	}
	if reversed.Description() != "reverse transfer" {
		t.Errorf("Description() = %q, want %q", reversed.Description(), "reverse transfer")
	}
	if reversed.JournalEntry().ID() != JournalEntryID("entry-002") {
		t.Errorf("JournalEntry().ID() = %q, want entry-002", reversed.JournalEntry().ID())
	}

	reversesID := reversed.ReversesTransactionID()
	if reversesID == nil || *reversesID != original.ID() {
		t.Fatalf("ReversesTransactionID() = %v, want %q", reversesID, original.ID())
	}

	reversedPostings := reversed.JournalEntry().Postings()
	if len(reversedPostings) != len(originalPostings) {
		t.Fatalf("reversed postings count = %d, want %d", len(reversedPostings), len(originalPostings))
	}

	wantPostingIDs := []PostingID{PostingID("posting-003"), PostingID("posting-004")}
	for index, posting := range originalPostings {
		reversedPosting := reversedPostings[index]
		if reversedPosting.ID() != wantPostingIDs[index] {
			t.Errorf("reversed posting %d ID = %q, want %q", index, reversedPosting.ID(), wantPostingIDs[index])
		}
		if reversedPosting.AccountID() != posting.AccountID() {
			t.Errorf("reversed posting %d account ID = %q, want %q", index, reversedPosting.AccountID(), posting.AccountID())
		}
		if reversedPosting.Direction() != posting.Direction().Opposite() {
			t.Errorf("reversed posting %d direction = %q, want %q", index, reversedPosting.Direction(), posting.Direction().Opposite())
		}
		if !reversedPosting.Amount().Equal(posting.Amount()) {
			t.Errorf("reversed posting %d amount changed", index)
		}
	}

	balanced, err := reversed.JournalEntry().IsBalanced()
	if err != nil {
		t.Fatalf("reversed JournalEntry().IsBalanced() returned an error: %v", err)
	}
	if !balanced {
		t.Error("reversed journal entry should remain balanced")
	}

	if original.IsReversal() {
		t.Error("reversing a transaction should not mutate the original")
	}
}

func TestTransaction_ReverseRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	brl := fakeValidCurrency("BRL")
	original := mustTestTransaction(t, "transfer funds", validJournalEntryPostings(t, brl))
	reversal := mustTestReversal(t, original)

	tests := []struct {
		name             string
		transaction      Transaction
		newTransactionID TransactionID
		newEntryID       JournalEntryID
		newPostingIDs    []PostingID
		description      string
		wantErr          error
	}{
		{
			name:             "rejects reversing a reversal",
			transaction:      reversal,
			newTransactionID: TransactionID("transaction-003"),
			newEntryID:       JournalEntryID("entry-003"),
			newPostingIDs:    []PostingID{PostingID("posting-005"), PostingID("posting-006")},
			description:      "second reversal",
			wantErr:          ErrCannotReverseReversal,
		},
		{
			name:             "rejects an empty new transaction ID",
			transaction:      original,
			newTransactionID: TransactionID(""),
			newEntryID:       JournalEntryID("entry-002"),
			newPostingIDs:    []PostingID{PostingID("posting-003"), PostingID("posting-004")},
			description:      "reversal",
			wantErr:          ErrInvalidID,
		},
		{
			name:             "rejects an empty new journal entry ID",
			transaction:      original,
			newTransactionID: TransactionID("transaction-002"),
			newEntryID:       JournalEntryID(""),
			newPostingIDs:    []PostingID{PostingID("posting-003"), PostingID("posting-004")},
			description:      "reversal",
			wantErr:          ErrInvalidID,
		},
		{
			name:             "rejects reusing the original transaction ID",
			transaction:      original,
			newTransactionID: original.ID(),
			newEntryID:       JournalEntryID("entry-002"),
			newPostingIDs:    []PostingID{PostingID("posting-003"), PostingID("posting-004")},
			description:      "reversal",
			wantErr:          ErrReversalSameTransactionID,
		},
		{
			name:             "rejects an incorrect posting ID count",
			transaction:      original,
			newTransactionID: TransactionID("transaction-002"),
			newEntryID:       JournalEntryID("entry-002"),
			newPostingIDs:    []PostingID{PostingID("posting-003")},
			description:      "reversal",
			wantErr:          ErrInvalidReversalPostingIDCount,
		},
		{
			name:             "rejects an empty posting ID",
			transaction:      original,
			newTransactionID: TransactionID("transaction-002"),
			newEntryID:       JournalEntryID("entry-002"),
			newPostingIDs:    []PostingID{PostingID("posting-003"), PostingID("")},
			description:      "reversal",
			wantErr:          ErrInvalidID,
		},
		{
			name:             "rejects an empty description",
			transaction:      original,
			newTransactionID: TransactionID("transaction-002"),
			newEntryID:       JournalEntryID("entry-002"),
			newPostingIDs:    []PostingID{PostingID("posting-003"), PostingID("posting-004")},
			description:      "   ",
			wantErr:          ErrEmptyTransactionDescription,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := tt.transaction.Reverse(
				tt.newTransactionID,
				tt.newEntryID,
				tt.newPostingIDs,
				tt.description,
			)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Reverse() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestTransaction_ReversesTransactionIDReturnsCopy(t *testing.T) {
	t.Parallel()

	brl := fakeValidCurrency("BRL")
	original := mustTestTransaction(t, "transfer funds", validJournalEntryPostings(t, brl))
	reversal := mustTestReversal(t, original)

	got := reversal.ReversesTransactionID()
	if got == nil {
		t.Fatal("ReversesTransactionID() returned nil")
	}

	*got = TransactionID("changed")
	returnedAgain := reversal.ReversesTransactionID()
	if returnedAgain == nil || *returnedAgain != original.ID() {
		t.Errorf("mutating returned ID changed the transaction: got %v, want %q", returnedAgain, original.ID())
	}
}

func mustTestTransaction(t *testing.T, description string, postings []Posting) Transaction {
	t.Helper()

	transaction, err := NewTransaction(
		TransactionID("transaction-001"),
		description,
		mustTestJournalEntry(
			t,
			JournalEntryID("entry-001"),
			TransactionID("transaction-001"),
			fakeValidCurrency("BRL"),
			postings,
		),
	)
	if err != nil {
		t.Fatalf("failed to create test transaction: %v", err)
	}

	return transaction
}

func mustTestReversal(t *testing.T, original Transaction) Transaction {
	t.Helper()

	reversal, err := original.Reverse(
		TransactionID("transaction-002"),
		JournalEntryID("entry-002"),
		[]PostingID{PostingID("posting-003"), PostingID("posting-004")},
		"reversal",
	)
	if err != nil {
		t.Fatalf("failed to create test reversal: %v", err)
	}

	return reversal
}
