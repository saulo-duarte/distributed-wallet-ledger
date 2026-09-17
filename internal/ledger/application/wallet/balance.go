package wallet

import (
	"errors"
	"financial-ledger/internal/ledger/domain"
)

var ErrInvalidBalanceSnapshot = errors.New(
	"invalid ledger balance snapshot",
)

type LedgerBalanceSnapshot struct {
	TotalDebits           int64
	TotalCredits          int64
	ActiveHoldsMinorUnits int64
}

type WalletBalance struct {
	WalletID                   domain.WalletID
	Currency                   domain.Currency
	LedgerBalanceMinorUnits    int64
	AvailableBalanceMinorUnits int64
}

func CalculateWalletBalance(
	wallet domain.Wallet,
	snapshot LedgerBalanceSnapshot,
) (WalletBalance, error) {
	if err := wallet.Validate(); err != nil {
		return WalletBalance{}, err
	}

	if snapshot.TotalDebits < 0 ||
		snapshot.TotalCredits < 0 ||
		snapshot.ActiveHoldsMinorUnits < 0 {
		return WalletBalance{}, ErrInvalidBalanceSnapshot
	}

	balance := snapshot.TotalCredits - snapshot.TotalDebits
	availableBalance := balance - snapshot.ActiveHoldsMinorUnits

	return WalletBalance{
		WalletID:                   wallet.ID(),
		Currency:                   wallet.Currency(),
		LedgerBalanceMinorUnits:    balance,
		AvailableBalanceMinorUnits: availableBalance,
	}, nil
}
