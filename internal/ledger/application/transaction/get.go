package transaction

import (
	"context"
	"errors"

	"financial-ledger/internal/ledger/domain"
)

var ErrTransactionNotFound = errors.New("transaction not found")

type GetTransactionUseCase struct {
	transactions TransactionReader
}

func NewGetTransactionUseCase(
	transactions TransactionReader,
) GetTransactionUseCase {
	return GetTransactionUseCase{
		transactions: transactions,
	}
}

func (uc GetTransactionUseCase) Execute(
	ctx context.Context,
	id domain.TransactionID,
) (TransactionDetails, error) {
	if id.IsZero() {
		return TransactionDetails{}, domain.ErrInvalidID
	}

	return uc.transactions.Get(ctx, id)
}
