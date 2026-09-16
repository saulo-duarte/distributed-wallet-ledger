package domain

import "strings"

type Transaction struct {
	id                    TransactionID
	description           string
	journalEntry          JournalEntry
	reversesTransactionID *TransactionID
}

func NewTransaction(
	id TransactionID,
	description string,
	journalEntry JournalEntry,
) (Transaction, error) {
	tx := Transaction{
		id:                    id,
		description:           strings.TrimSpace(description),
		journalEntry:          journalEntry,
		reversesTransactionID: nil,
	}

	if err := tx.Validate(); err != nil {
		return Transaction{}, err
	}

	return tx, nil
}

// ReconstituteTransaction rebuilds a transaction that was already persisted.
// Persistence adapters use this function to restore state without exposing
// the aggregate's internal fields outside the domain package.
func ReconstituteTransaction(
	id TransactionID,
	description string,
	journalEntry JournalEntry,
	reversesTransactionID *TransactionID,
) (Transaction, error) {
	var reversalIDCopy *TransactionID

	if reversesTransactionID != nil {
		copy := *reversesTransactionID
		reversalIDCopy = &copy
	}

	tx := Transaction{
		id:                    id,
		description:           strings.TrimSpace(description),
		journalEntry:          journalEntry,
		reversesTransactionID: reversalIDCopy,
	}

	if err := tx.Validate(); err != nil {
		return Transaction{}, err
	}

	return tx, nil
}

func (t Transaction) ID() TransactionID {
	return t.id
}

func (t Transaction) Description() string {
	return t.description
}

func (t Transaction) JournalEntry() JournalEntry {
	return t.journalEntry
}

func (t Transaction) ReversesTransactionID() *TransactionID {
	if t.reversesTransactionID == nil {
		return nil
	}

	idCopy := *t.reversesTransactionID
	return &idCopy
}

func (t Transaction) IsReversal() bool {
	return t.reversesTransactionID != nil
}

// Reverse creates a new transaction that compensates the original one.
// IDs are supplied by the caller so the domain does not generate identifiers
// or depend on a UUID implementation.
func (t Transaction) Reverse(
	newTransactionID TransactionID,
	newJournalEntryID JournalEntryID,
	newPostingIDs []PostingID,
	description string,
) (Transaction, error) {
	if err := t.Validate(); err != nil {
		return Transaction{}, err
	}

	if t.IsReversal() {
		return Transaction{}, ErrCannotReverseReversal
	}

	if newTransactionID.IsZero() || newJournalEntryID.IsZero() {
		return Transaction{}, ErrInvalidID
	}

	if newTransactionID == t.id {
		return Transaction{}, ErrReversalSameTransactionID
	}

	originalPostings := t.journalEntry.Postings()
	if len(newPostingIDs) != len(originalPostings) {
		return Transaction{}, ErrInvalidReversalPostingIDCount
	}

	reversedPostings := make([]Posting, 0, len(originalPostings))
	for index, posting := range originalPostings {
		postingID := newPostingIDs[index]
		if postingID.IsZero() {
			return Transaction{}, ErrInvalidID
		}

		reversedPosting, err := NewPosting(
			postingID,
			posting.AccountID(),
			posting.Direction().Opposite(),
			posting.Amount(),
		)
		if err != nil {
			return Transaction{}, err
		}

		reversedPostings = append(reversedPostings, reversedPosting)
	}

	newEntry, err := NewJournalEntry(
		newJournalEntryID,
		newTransactionID,
		t.journalEntry.Currency(),
		reversedPostings,
	)
	if err != nil {
		return Transaction{}, err
	}

	originalID := t.id
	reversedTransaction := Transaction{
		id:                    newTransactionID,
		description:           strings.TrimSpace(description),
		journalEntry:          newEntry,
		reversesTransactionID: &originalID,
	}

	if err := reversedTransaction.Validate(); err != nil {
		return Transaction{}, err
	}

	return reversedTransaction, nil
}

func (t Transaction) Validate() error {
	if t.id.IsZero() {
		return ErrInvalidID
	}

	if t.description == "" {
		return ErrEmptyTransactionDescription
	}

	if err := t.journalEntry.Validate(); err != nil {
		return err
	}

	if t.journalEntry.TransactionID() != t.id {
		return ErrTransactionJournalEntryMismatch
	}

	if t.reversesTransactionID != nil {
		if t.reversesTransactionID.IsZero() {
			return ErrInvalidID
		}

		if *t.reversesTransactionID == t.id {
			return ErrReversalSameTransactionID
		}
	}

	return nil
}
