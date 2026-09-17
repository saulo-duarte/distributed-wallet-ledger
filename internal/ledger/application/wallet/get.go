package wallet

import (
	"context"

	"financial-ledger/internal/ledger/domain"
)

type GetWalletUseCase struct {
	wallets WalletReader
}

func NewGetWalletUseCase(wallets WalletReader) GetWalletUseCase {
	return GetWalletUseCase{
		wallets: wallets,
	}
}

func (uc GetWalletUseCase) Execute(
	ctx context.Context,
	walletID domain.WalletID,
) (domain.Wallet, error) {
	if walletID.IsZero() {
		return domain.Wallet{}, domain.ErrInvalidID
	}

	return uc.wallets.GetByID(ctx, walletID)
}
