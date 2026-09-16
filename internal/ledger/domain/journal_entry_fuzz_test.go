package domain

import (
	"errors"
	"testing"
)

func FuzzNewJournalEntryBalance(f *testing.F) {
	f.Add("BRL", int64(10000), int64(10000))
	f.Add("BRL", int64(10000), int64(9000))
	f.Add("USD", int64(1), int64(1))
	f.Add("invalid", int64(100), int64(100))
	f.Add("BRL", int64(0), int64(100))

	f.Fuzz(func(t *testing.T, currencyCode string, debitAmount, creditAmount int64) {
		currency, currencyErr := NewCurrency(currencyCode)
		if currencyErr != nil || debitAmount <= 0 || creditAmount <= 0 {
			return
		}

		debit := mustFuzzPosting(t, PostingID("posting-debit"), AccountID("account-debit"), PostingDirectionDebit, currency, debitAmount)
		credit := mustFuzzPosting(t, PostingID("posting-credit"), AccountID("account-credit"), PostingDirectionCredit, currency, creditAmount)

		entry, err := NewJournalEntry(
			JournalEntryID("entry-001"),
			TransactionID("transaction-001"),
			currency,
			[]Posting{debit, credit},
		)

		if debitAmount != creditAmount {
			if !errors.Is(err, ErrUnbalancedJournalEntry) {
				t.Fatalf("NewJournalEntry() error = %v, want %v", err, ErrUnbalancedJournalEntry)
			}
			return
		}

		if err != nil {
			t.Fatalf("NewJournalEntry() returned an unexpected error: %v", err)
		}

		balanced, err := entry.IsBalanced()
		if err != nil {
			t.Fatalf("IsBalanced() returned an unexpected error: %v", err)
		}
		if !balanced {
			t.Fatal("a successfully created journal entry should be balanced")
		}
	})
}

func FuzzNewJournalEntryCurrencyConsistency(f *testing.F) {
	f.Add("BRL", "BRL", int64(10000))
	f.Add("BRL", "USD", int64(10000))
	f.Add("invalid", "BRL", int64(10000))
	f.Add("BRL", "BRL", int64(0))

	f.Fuzz(func(t *testing.T, entryCurrencyCode, postingCurrencyCode string, amount int64) {
		entryCurrency, entryCurrencyErr := NewCurrency(entryCurrencyCode)
		postingCurrency, postingCurrencyErr := NewCurrency(postingCurrencyCode)
		if entryCurrencyErr != nil || postingCurrencyErr != nil || amount <= 0 {
			return
		}

		debit := mustFuzzPosting(t, PostingID("posting-debit"), AccountID("account-debit"), PostingDirectionDebit, postingCurrency, amount)
		credit := mustFuzzPosting(t, PostingID("posting-credit"), AccountID("account-credit"), PostingDirectionCredit, entryCurrency, amount)

		_, err := NewJournalEntry(
			JournalEntryID("entry-001"),
			TransactionID("transaction-001"),
			entryCurrency,
			[]Posting{debit, credit},
		)

		if !entryCurrency.Equal(postingCurrency) {
			if !errors.Is(err, ErrCurrencyMismatch) {
				t.Fatalf("NewJournalEntry() error = %v, want %v", err, ErrCurrencyMismatch)
			}
			return
		}

		if err != nil {
			t.Fatalf("NewJournalEntry() returned an unexpected error: %v", err)
		}
	})
}

func mustFuzzPosting(
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
		t.Fatalf("failed to create fuzz Money: %v", err)
	}

	posting, err := NewPosting(id, accountID, direction, money)
	if err != nil {
		t.Fatalf("failed to create fuzz Posting: %v", err)
	}

	return posting
}
