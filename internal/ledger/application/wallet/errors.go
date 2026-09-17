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

	ErrTransferSameWallet = errors.New(
		"source and destination wallets must be different",
	)

	ErrTransferSameAccount = errors.New(
		"source and destination ledger accounts must be different",
	)
)
