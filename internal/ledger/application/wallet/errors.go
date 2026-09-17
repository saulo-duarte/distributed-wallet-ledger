package wallet

import "errors"

var (
	ErrEmptyIdempotencyKey = errors.New(
		"idempotency key cannot be empty",
	)

	ErrEmptyRequestHash = errors.New(
		"request hash cannot be empty",
	)

	ErrHoldNotFound = errors.New(
		"hold not found",
	)

	ErrHoldIdempotencyConflict = errors.New(
		"hold idempotency key was already used with another request",
	)

	ErrCaptureSameAccount = errors.New(
		"capture settlement account must differ from wallet account",
	)

	ErrWalletNotFound = errors.New("wallet not found")

	ErrInsufficientFunds = errors.New(
		"insufficient wallet funds",
	)

	ErrWithdrawalSameAccount = errors.New(
		"withdrawal clearing account must differ from wallet account",
	)

	ErrDepositSameAccount = errors.New(
		"deposit clearing account must differ from wallet account",
	)

	ErrTransferSameWallet = errors.New(
		"source and destination wallets must be different",
	)

	ErrTransferSameAccount = errors.New(
		"source and destination ledger accounts must be different",
	)
)
