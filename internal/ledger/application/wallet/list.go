package wallet

import (
	"context"

	"financial-ledger/internal/ledger/domain"
)

type ListWalletsByOwnerUseCase struct {
	wallets WalletReader
}

func NewListWalletsByOwnerUseCase(
	wallets WalletReader,
) ListWalletsByOwnerUseCase {
	return ListWalletsByOwnerUseCase{
		wallets: wallets,
	}
}

func (uc ListWalletsByOwnerUseCase) Execute(
	ctx context.Context,
	ownerID domain.OwnerID,
) ([]domain.Wallet, error) {
	if ownerID.IsZero() {
		return nil, domain.ErrInvalidID
	}

	return uc.wallets.ListByOwnerID(ctx, ownerID)
}
