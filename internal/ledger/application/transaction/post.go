package transaction

import (
	"context"
	"errors"
	"financial-ledger/internal/ledger/domain"
	"strings"
)

var (
	ErrEmptyIdempotencyKey = errors.New(
		"idempotency key cannot be empty",
	)

	ErrEmptyRequestHash = errors.New(
		"request hash cannot be empty",
	)
)

type PostingCommand struct {
	ID               domain.PostingID
	AccountID        domain.AccountID
	Direction        domain.PostingDirection
	AmountMinorUnits int64
}

type PostTransactionCommand struct {
	ID             domain.TransactionID
	JournalEntryID domain.JournalEntryID
	Description    string
	CurrencyCode   string
	IdempotencyKey string
	RequestHash    string
	Postings       []PostingCommand
}

type PostTransactionUseCase struct {
	ledger LedgerRepository
}

func NewPostTransactionUseCase(
	ledger LedgerRepository,
) PostTransactionUseCase {
	return PostTransactionUseCase{
		ledger: ledger,
	}
}

func (uc PostTransactionUseCase) Execute(
	ctx context.Context,
	command PostTransactionCommand,
) (domain.Transaction, error) {
	idempotencyKey := strings.TrimSpace(command.IdempotencyKey)
	if idempotencyKey == "" {
		return domain.Transaction{}, ErrEmptyIdempotencyKey
	}

	requestHash := strings.TrimSpace(command.RequestHash)
	if requestHash == "" {
		return domain.Transaction{}, ErrEmptyRequestHash
	}

	currency, err := domain.NewCurrency(command.CurrencyCode)
	if err != nil {
		return domain.Transaction{}, err
	}

	postings := make(
		[]domain.Posting,
		0,
		len(command.Postings),
	)

	for _, postingCommand := range command.Postings {
		amount, err := domain.NewMoney(
			currency,
			postingCommand.AmountMinorUnits,
		)
		if err != nil {
			return domain.Transaction{}, err
		}

		posting, err := domain.NewPosting(
			postingCommand.ID,
			postingCommand.AccountID,
			postingCommand.Direction,
			amount,
		)
		if err != nil {
			return domain.Transaction{}, err
		}

		postings = append(postings, posting)
	}

	journalEntry, err := domain.NewJournalEntry(
		command.JournalEntryID,
		command.ID,
		currency,
		postings,
	)

	if err != nil {
		return domain.Transaction{}, err
	}

	transaction, err := domain.NewTransaction(
		command.ID,
		command.Description,
		journalEntry,
	)
	if err != nil {
		return domain.Transaction{}, err
	}

	if err := uc.ledger.Post(
		ctx,
		transaction,
		idempotencyKey,
		requestHash,
	); err != nil {
		return domain.Transaction{}, err
	}

	return transaction, nil
}
