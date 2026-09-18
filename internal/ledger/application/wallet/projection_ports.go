package wallet

import (
	"context"
	"financial-ledger/internal/ledger/domain"
)

type WalletBalanceProjectionRepository interface {
	SaveWalletBalance(ctx context.Context, balance WalletBalance) error
	GetWalletBalance(ctx context.Context, walletID domain.WalletID) (WalletBalance, error)
}

type WalletBalanceProjectorPort interface {
	ProjectBalance(ctx context.Context, walletID domain.WalletID) (WalletBalance, error)
}

