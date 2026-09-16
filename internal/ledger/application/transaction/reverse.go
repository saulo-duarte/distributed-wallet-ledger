package transaction

import (
	"context"
	"financial-ledger/internal/ledger/domain"
	"strings"
)

type TransactionAggregateReader interface {
	GetAggregate(
		ctx context.Context,
		id domain.TransactionID,
	) (domain.Transaction, error)
}

type ReverseTransactionCommand struct {
	OriginalTransactionID domain.TransactionID
	NewTransactionID      domain.TransactionID
	NewJournalEntryID     domain.JournalEntryID
	NewPostingIDs         []domain.PostingID
	Description           string
	IdempotencyKey        string
	RequestHash           string
}

type ReverseTransactionUseCase struct {
	transactions TransactionAggregateReader
	ledger       LedgerRepository
}

func NewReverseTransactionUseCase(
	transactions TransactionAggregateReader,
	ledger LedgerRepository,
) ReverseTransactionUseCase {
	return ReverseTransactionUseCase{
		transactions: transactions,
		ledger:       ledger,
	}
}

func (uc ReverseTransactionUseCase) Execute(
	ctx context.Context,
	command ReverseTransactionCommand,
) (domain.Transaction, error) {
	idempotencyKey := strings.TrimSpace(command.IdempotencyKey)
	if idempotencyKey == "" {
		return domain.Transaction{}, ErrEmptyIdempotencyKey
	}

	requestHash := strings.TrimSpace(command.RequestHash)
	if requestHash == "" {
		return domain.Transaction{}, ErrEmptyRequestHash
	}

	if command.OriginalTransactionID.IsZero() {
		return domain.Transaction{}, domain.ErrInvalidID
	}

	originalTransaction, err := uc.transactions.GetAggregate(
		ctx,
		command.OriginalTransactionID,
	)
	if err != nil {
		return domain.Transaction{}, err
	}

	reversalTransaction, err := originalTransaction.Reverse(
		command.NewTransactionID,
		command.NewJournalEntryID,
		command.NewPostingIDs,
		command.Description,
	)
	if err != nil {
		return domain.Transaction{}, err
	}

	if err := uc.ledger.Post(
		ctx,
		reversalTransaction,
		idempotencyKey,
		requestHash,
	); err != nil {
		return domain.Transaction{}, err
	}

	return reversalTransaction, nil
}
