package wallet

import (
	"context"
	"log/slog"
	"time"

	"financial-ledger/internal/ledger/domain"
)

type GetWalletBalanceUseCase struct {
	wallets     WalletReader
	balances    WalletBalanceReader
	projections WalletBalanceProjectionRepository
	logger      *slog.Logger
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

func NewGetWalletBalanceUseCaseWithProjection(
	wallets WalletReader,
	balances WalletBalanceReader,
	projections WalletBalanceProjectionRepository,
	logger *slog.Logger,
) GetWalletBalanceUseCase {
	return GetWalletBalanceUseCase{
		wallets:     wallets,
		balances:    balances,
		projections: projections,
		logger:      logger,
	}
}

func (uc GetWalletBalanceUseCase) Execute(
	ctx context.Context,
	walletID domain.WalletID,
) (WalletBalance, error) {
	if walletID.IsZero() {
		return WalletBalance{}, domain.ErrInvalidID
	}

	if uc.projections != nil {
		readCtx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
		balance, err := uc.projections.GetWalletBalance(readCtx, walletID)
		cancel()
		if err == nil {
			if uc.logger != nil {
				uc.logger.DebugContext(
					ctx,
					"wallet_balance_source",
					slog.String("source", "dynamodb"),
					slog.String("wallet_id", walletID.String()),
					slog.Int64("available_balance", balance.AvailableBalanceMinorUnits),
				)
			}
			return balance, nil
		}
	}

	return uc.ExecuteFromLedger(ctx, walletID)
}

// ExecuteFromLedger returns an up-to-date balance calculated from the ledger
// source of truth. It is appropriate immediately after a write when an
// eventually consistent projection may still be catching up.
func (uc GetWalletBalanceUseCase) ExecuteFromLedger(
	ctx context.Context,
	walletID domain.WalletID,
) (WalletBalance, error) {
	if walletID.IsZero() {
		return WalletBalance{}, domain.ErrInvalidID
	}

	if uc.logger != nil {
		uc.logger.DebugContext(
			ctx,
			"wallet_balance_source",
			slog.String("source", "postgresql"),
			slog.String("wallet_id", walletID.String()),
		)
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

	balance, err := CalculateWalletBalance(foundWallet, snapshot)
	if err != nil {
		return WalletBalance{}, err
	}

	if uc.projections != nil {
		_ = uc.projections.SaveWalletBalance(ctx, balance)
	}

	return balance, nil
}
