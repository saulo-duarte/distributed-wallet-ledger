package wallet

import (
	"context"
	"financial-ledger/internal/ledger/application/transaction"
	"financial-ledger/internal/ledger/domain"
)

type WalletRepository interface {
	Create(
		ctx context.Context,
		wallet domain.Wallet,
	) error
}

type WalletReader interface {
	GetByID(
		ctx context.Context,
		walletID domain.WalletID,
	) (domain.Wallet, error)
	ListByOwnerID(
		ctx context.Context,
		ownerID domain.OwnerID,
	) ([]domain.Wallet, error)
}

type WalletBalanceReader interface {
	GetLedgerBalance(
		ctx context.Context,
		accountID domain.AccountID,
	) (LedgerBalanceSnapshot, error)
}

type TransactionPoster interface {
	Execute(
		ctx context.Context,
		command transaction.PostTransactionCommand,
	) (domain.Transaction, error)
}
