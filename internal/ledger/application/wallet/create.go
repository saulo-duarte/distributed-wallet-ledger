package wallet

import (
	"context"
	"financial-ledger/internal/ledger/domain"
)

type CreateWalletCommand struct {
	ID              domain.WalletID
	OwnerID         domain.OwnerID
	LedgerAccountID domain.AccountID
	CurrencyCode    string
}

type CreateWalletUseCase struct {
	wallets WalletRepository
}

func NewCreateWalletUseCase(
	wallets WalletRepository,
) CreateWalletUseCase {
	return CreateWalletUseCase{
		wallets: wallets,
	}
}

func (uc CreateWalletUseCase) Execute(
	ctx context.Context,
	command CreateWalletCommand,
) (domain.Wallet, error) {
	currency, err := domain.NewCurrency(command.CurrencyCode)
	if err != nil {
		return domain.Wallet{}, err
	}

	wallet, err := domain.NewWallet(
		command.ID,
		command.OwnerID,
		command.LedgerAccountID,
		currency,
	)

	if err != nil {
		return domain.Wallet{}, err
	}

	if err := uc.wallets.Create(ctx, wallet); err != nil {
		return domain.Wallet{}, err
	}

	return wallet, nil

}
