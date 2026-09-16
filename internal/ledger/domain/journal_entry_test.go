package domain

import (
	"errors"
	"math"
	"testing"
)

func TestNewJournalEntry(t *testing.T) {
	t.Parallel()

	brl := fakeValidCurrency("BRL")
	usd := fakeValidCurrency("USD")
	validPostings := validJournalEntryPostings(t, brl)

	tests := []struct {
		name          string
		id            JournalEntryID
		transactionID TransactionID
		currency      Currency
		postings      []Posting
		wantErr       error
	}{
		{
			name:          "creates a balanced journal entry",
			id:            JournalEntryID("entry-001"),
			transactionID: TransactionID("transaction-001"),
			currency:      brl,
			postings:      validPostings,
		},
		{
			name:          "rejects an empty ID",
			id:            JournalEntryID(""),
			transactionID: TransactionID("transaction-001"),
			currency:      brl,
			postings:      validPostings,
			wantErr:       ErrInvalidID,
		},
		{
			name:          "rejects an empty transaction ID",
			id:            JournalEntryID("entry-001"),
			transactionID: TransactionID(""),
			currency:      brl,
			postings:      validPostings,
			wantErr:       ErrInvalidID,
		},
		{
			name:          "rejects an invalid currency",
			id:            JournalEntryID("entry-001"),
			transactionID: TransactionID("transaction-001"),
			currency:      Currency{code: "BR"},
			postings:      validPostings,
			wantErr:       ErrInvalidCurrency,
		},
		{
			name:          "rejects an entry without postings",
			id:            JournalEntryID("entry-001"),
			transactionID: TransactionID("transaction-001"),
			currency:      brl,
			postings:      nil,
			wantErr:       ErrJournalEntryWithoutPostings,
		},
		{
			name:          "rejects an entry without a debit",
			id:            JournalEntryID("entry-001"),
			transactionID: TransactionID("transaction-001"),
			currency:      brl,
			postings: []Posting{
				mustTestPosting(t, PostingID("posting-001"), AccountID("account-001"), PostingDirectionCredit, brl, 10000),
			},
			wantErr: ErrJournalEntryWithoutDebit,
		},
		{
			name:          "rejects an entry without a credit",
			id:            JournalEntryID("entry-001"),
			transactionID: TransactionID("transaction-001"),
			currency:      brl,
			postings: []Posting{
				mustTestPosting(t, PostingID("posting-001"), AccountID("account-001"), PostingDirectionDebit, brl, 10000),
			},
			wantErr: ErrJournalEntryWithoutCredit,
		},
		{
			name:          "rejects a posting currency mismatch",
			id:            JournalEntryID("entry-001"),
			transactionID: TransactionID("transaction-001"),
			currency:      brl,
			postings: []Posting{
				mustTestPosting(t, PostingID("posting-001"), AccountID("account-001"), PostingDirectionDebit, brl, 10000),
				mustTestPosting(t, PostingID("posting-002"), AccountID("account-002"), PostingDirectionCredit, usd, 10000),
			},
			wantErr: ErrCurrencyMismatch,
		},
		{
			name:          "rejects an unbalanced entry",
			id:            JournalEntryID("entry-001"),
			transactionID: TransactionID("transaction-001"),
			currency:      brl,
			postings: []Posting{
				mustTestPosting(t, PostingID("posting-001"), AccountID("account-001"), PostingDirectionDebit, brl, 10000),
				mustTestPosting(t, PostingID("posting-002"), AccountID("account-002"), PostingDirectionCredit, brl, 9000),
			},
			wantErr: ErrUnbalancedJournalEntry,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			entry, err := NewJournalEntry(
				tt.id,
				tt.transactionID,
				tt.currency,
				tt.postings,
			)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("NewJournalEntry() error = %v, want %v", err, tt.wantErr)
			}

			if tt.wantErr != nil {
				return
			}

			if entry.ID() != tt.id {
				t.Errorf("ID() = %q, want %q", entry.ID(), tt.id)
			}

			if entry.TransactionID() != tt.transactionID {
				t.Errorf("TransactionID() = %q, want %q", entry.TransactionID(), tt.transactionID)
			}

			if !entry.Currency().Equal(tt.currency) {
				t.Errorf("Currency() = %q, want %q", entry.Currency(), tt.currency)
			}
		})
	}
}

func TestJournalEntry_TotalDebitsAndCredits(t *testing.T) {
	t.Parallel()

	brl := fakeValidCurrency("BRL")
	postings := []Posting{
		mustTestPosting(t, PostingID("posting-001"), AccountID("account-001"), PostingDirectionDebit, brl, 10000),
		mustTestPosting(t, PostingID("posting-002"), AccountID("account-002"), PostingDirectionDebit, brl, 5000),
		mustTestPosting(t, PostingID("posting-003"), AccountID("account-003"), PostingDirectionCredit, brl, 15000),
	}

	entry := mustTestJournalEntry(t, JournalEntryID("entry-001"), TransactionID("transaction-001"), brl, postings)

	debits, err := entry.TotalDebits()
	if err != nil {
		t.Fatalf("TotalDebits() returned an unexpected error: %v", err)
	}

	if got := debits.AmountMinorUnits(); got != 15000 {
		t.Errorf("TotalDebits() = %d, want 15000", got)
	}

	credits, err := entry.TotalCredits()
	if err != nil {
		t.Fatalf("TotalCredits() returned an unexpected error: %v", err)
	}

	if got := credits.AmountMinorUnits(); got != 15000 {
		t.Errorf("TotalCredits() = %d, want 15000", got)
	}
}

func TestJournalEntry_IsBalanced(t *testing.T) {
	t.Parallel()

	brl := fakeValidCurrency("BRL")

	tests := []struct {
		name     string
		postings []Posting
		want     bool
	}{
		{
			name:     "balanced",
			postings: validJournalEntryPostings(t, brl),
			want:     true,
		},
		{
			name: "unbalanced",
			postings: []Posting{
				mustTestPosting(t, PostingID("posting-001"), AccountID("account-001"), PostingDirectionDebit, brl, 10000),
				mustTestPosting(t, PostingID("posting-002"), AccountID("account-002"), PostingDirectionCredit, brl, 9000),
			},
			want: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			entry := JournalEntry{
				id:            JournalEntryID("entry-001"),
				transactionID: TransactionID("transaction-001"),
				currency:      brl,
				postings:      tt.postings,
			}

			got, err := entry.IsBalanced()
			if err != nil {
				t.Fatalf("IsBalanced() returned an unexpected error: %v", err)
			}

			if got != tt.want {
				t.Errorf("IsBalanced() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewJournalEntry_RejectsAmountOverflow(t *testing.T) {
	t.Parallel()

	brl := fakeValidCurrency("BRL")
	postings := []Posting{
		mustTestPosting(t, PostingID("posting-001"), AccountID("account-001"), PostingDirectionDebit, brl, math.MaxInt64),
		mustTestPosting(t, PostingID("posting-002"), AccountID("account-002"), PostingDirectionDebit, brl, math.MaxInt64),
		mustTestPosting(t, PostingID("posting-003"), AccountID("account-003"), PostingDirectionCredit, brl, 1),
	}

	_, err := NewJournalEntry(
		JournalEntryID("entry-001"),
		TransactionID("transaction-001"),
		brl,
		postings,
	)

	if !errors.Is(err, ErrAmountOverflow) {
		t.Fatalf("NewJournalEntry() error = %v, want %v", err, ErrAmountOverflow)
	}
}

func TestJournalEntry_PostingsAreCopied(t *testing.T) {
	t.Parallel()

	brl := fakeValidCurrency("BRL")
	postings := validJournalEntryPostings(t, brl)
	entry := mustTestJournalEntry(t, JournalEntryID("entry-001"), TransactionID("transaction-001"), brl, postings)

	originalID := entry.Postings()[0].ID()

	postings[0] = Posting{}
	if got := entry.Postings()[0].ID(); got != originalID {
		t.Errorf("mutating the original slice changed the entry: ID() = %q, want %q", got, originalID)
	}

	returnedPostings := entry.Postings()
	returnedPostings[0] = Posting{}
	if got := entry.Postings()[0].ID(); got != originalID {
		t.Errorf("mutating the returned slice changed the entry: ID() = %q, want %q", got, originalID)
	}
}

func validJournalEntryPostings(t *testing.T, currency Currency) []Posting {
	t.Helper()

	return []Posting{
		mustTestPosting(t, PostingID("posting-001"), AccountID("account-001"), PostingDirectionDebit, currency, 10000),
		mustTestPosting(t, PostingID("posting-002"), AccountID("account-002"), PostingDirectionCredit, currency, 10000),
	}
}

func mustTestPosting(
	t *testing.T,
	id PostingID,
	accountID AccountID,
	direction PostingDirection,
	currency Currency,
	amount int64,
) Posting {
	t.Helper()

	money, err := NewMoney(currency, amount)
	if err != nil {
		t.Fatalf("failed to create test Money: %v", err)
	}

	posting, err := NewPosting(id, accountID, direction, money)
	if err != nil {
		t.Fatalf("failed to create test Posting: %v", err)
	}

	return posting
}

func mustTestJournalEntry(
	t *testing.T,
	id JournalEntryID,
	transactionID TransactionID,
	currency Currency,
	postings []Posting,
) JournalEntry {
	t.Helper()

	entry, err := NewJournalEntry(id, transactionID, currency, postings)
	if err != nil {
		t.Fatalf("failed to create test JournalEntry: %v", err)
	}

	return entry
}
