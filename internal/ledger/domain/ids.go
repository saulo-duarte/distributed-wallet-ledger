package domain

import (
	"strings"
)

type AccountID string

func NewAccountID(id string) (AccountID, error) {
	cleanID := strings.TrimSpace(id)
	if cleanID == "" {
		return "", ErrInvalidID
	}
	return AccountID(cleanID), nil
}

func (id AccountID) String() string {
	return string(id)
}

func (id AccountID) IsZero() bool {
	return id == ""
}

type TransactionID string

func NewTransactionID(id string) (TransactionID, error) {
	cleanID := strings.TrimSpace(id)
	if cleanID == "" {
		return "", ErrInvalidID
	}
	return TransactionID(cleanID), nil
}

func (id TransactionID) String() string {
	return string(id)
}

func (id TransactionID) IsZero() bool {
	return id == ""
}

type JournalEntryID string

func NewJournalEntryID(id string) (JournalEntryID, error) {
	cleanID := strings.TrimSpace(id)
	if cleanID == "" {
		return "", ErrInvalidID
	}
	return JournalEntryID(cleanID), nil
}

func (id JournalEntryID) String() string {
	return string(id)
}

func (id JournalEntryID) IsZero() bool {
	return id == ""
}

type PostingID string

func NewPostingID(id string) (PostingID, error) {
	cleanID := strings.TrimSpace(id)
	if cleanID == "" {
		return "", ErrInvalidID
	}
	return PostingID(cleanID), nil
}

func (id PostingID) String() string {
	return string(id)
}

func (id PostingID) IsZero() bool {
	return id == ""
}
