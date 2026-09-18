package wallet

import (
	"context"
	"fmt"

	"financial-ledger/internal/ledger/domain"
)

type WalletBalanceProjector struct {
	wallets     WalletReader
	balances    WalletBalanceReader
	projections WalletBalanceProjectionRepository
}

func NewWalletBalanceProjector(
	wallets WalletReader,
	balances WalletBalanceReader,
	projections WalletBalanceProjectionRepository,
) WalletBalanceProjector {
	return WalletBalanceProjector{
		wallets:     wallets,
		balances:    balances,
		projections: projections,
	}
}

func (p WalletBalanceProjector) ProjectBalance(
	ctx context.Context,
	walletID domain.WalletID,
) (WalletBalance, error) {
	if walletID.IsZero() {
		return WalletBalance{}, domain.ErrInvalidID
	}

	foundWallet, err := p.wallets.GetByID(ctx, walletID)
	if err != nil {
		return WalletBalance{}, fmt.Errorf("projector get wallet %q: %w", walletID, err)
	}

	snapshot, err := p.balances.GetLedgerBalance(ctx, foundWallet.LedgerAccountID())
	if err != nil {
		return WalletBalance{}, fmt.Errorf("projector get ledger balance for %q: %w", walletID, err)
	}

	balance, err := CalculateWalletBalance(foundWallet, snapshot)
	if err != nil {
		return WalletBalance{}, fmt.Errorf("projector calculate balance for %q: %w", walletID, err)
	}

	if err := p.projections.SaveWalletBalance(ctx, balance); err != nil {
		return WalletBalance{}, fmt.Errorf("projector save balance in dynamodb for %q: %w", walletID, err)
	}

	return balance, nil
}
