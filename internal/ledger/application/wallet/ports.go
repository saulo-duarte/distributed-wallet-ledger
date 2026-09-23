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

type HoldRepository interface {
	Authorize(
		ctx context.Context,
		hold domain.Hold,
		idempotencyKey string,
		requestHash string,
	) (domain.Hold, error)
	GetByID(
		ctx context.Context,
		holdID domain.HoldID,
	) (domain.Hold, error)
	UpdateStatus(
		ctx context.Context,
		holdID domain.HoldID,
		status domain.HoldStatus,
	) error
	ListExpiredHolds(
		ctx context.Context,
		limit int32,
	) ([]domain.HoldID, error)
}

type TransactionPoster interface {
	Execute(
		ctx context.Context,
		command transaction.PostTransactionCommand,
	) (domain.Transaction, error)
}

type HoldCaptureRepository interface {
	Capture(
		ctx context.Context,
		holdID domain.HoldID,
		postedTransaction domain.Transaction,
		idempotencyKey string,
		requestHash string,
	) error
}
