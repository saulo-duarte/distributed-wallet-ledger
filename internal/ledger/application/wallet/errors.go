package wallet

import "errors"

var (
	ErrWalletNotFound = errors.New("wallet not found")

	ErrInsufficientFunds = errors.New(
		"insufficient wallet funds",
	)

	ErrWithdrawalSameAccount = errors.New(
		"withdrawal clearing account must differ from wallet account",
	)
)
