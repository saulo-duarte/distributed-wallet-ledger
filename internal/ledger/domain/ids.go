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

type WalletID string

func NewWalletID(id string) (WalletID, error) {
	cleanID := strings.TrimSpace(id)
	if cleanID == "" {
		return "", ErrInvalidID
	}

	return WalletID(cleanID), nil
}

func (id WalletID) String() string {
	return string(id)
}

func (id WalletID) IsZero() bool {
	return id == ""
}

type OwnerID string

func NewOwnerID(id string) (OwnerID, error) {
	cleanID := strings.TrimSpace(id)
	if cleanID == "" {
		return "", ErrInvalidID
	}

	return OwnerID(cleanID), nil
}

func (id OwnerID) String() string {
	return string(id)
}

func (id OwnerID) IsZero() bool {
	return id == ""
}

type HoldID string

func NewHoldID(id string) (HoldID, error) {
	cleanID := strings.TrimSpace(id)
	if cleanID == "" {
		return "", ErrInvalidID
	}

	return HoldID(cleanID), nil
}

func (id HoldID) String() string {
	return string(id)
}

func (id HoldID) IsZero() bool {
	return id == ""
}

type EventID string

func NewEventID(id string) (EventID, error) {
	cleanID := strings.TrimSpace(id)
	if cleanID == "" {
		return "", ErrInvalidID
	}
	return EventID(cleanID), nil
}

func (id EventID) String() string {
	return string(id)
}

func (id EventID) IsZero() bool {
	return id == ""
}
