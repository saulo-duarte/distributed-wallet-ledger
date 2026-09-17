package wallet

import (
	"context"

	"financial-ledger/internal/ledger/domain"
)

type GetWalletBalanceUseCase struct {
	wallets  WalletReader
	balances WalletBalanceReader
}

func NewGetWalletBalanceUseCase(
	wallets WalletReader,
	balances WalletBalanceReader,
) GetWalletBalanceUseCase {
	return GetWalletBalanceUseCase{
		wallets:  wallets,
		balances: balances,
	}
}

func (uc GetWalletBalanceUseCase) Execute(
	ctx context.Context,
	walletID domain.WalletID,
) (WalletBalance, error) {
	if walletID.IsZero() {
		return WalletBalance{}, domain.ErrInvalidID
	}

	foundWallet, err := uc.wallets.GetByID(ctx, walletID)
	if err != nil {
		return WalletBalance{}, err
	}

	snapshot, err := uc.balances.GetLedgerBalance(
		ctx,
		foundWallet.LedgerAccountID(),
	)
	if err != nil {
		return WalletBalance{}, err
	}

	return CalculateWalletBalance(foundWallet, snapshot)
}
