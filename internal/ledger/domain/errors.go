package domain

import "errors"

var (
	// Shared value errors.
	ErrInvalidID               = errors.New("invalid id")
	ErrInvalidCurrency         = errors.New("invalid currency")
	ErrCurrencyMismatch        = errors.New("currency mismatch")
	ErrAmountMustNotBeNegative = errors.New("amount must not be negative")
	ErrAmountMustBePositive    = errors.New("amount must be positive")
	ErrAmountOverflow          = errors.New("amount overflow")

	// Account errors.
	ErrInvalidAccount       = errors.New("invalid account")
	ErrInvalidAccountStatus = errors.New("invalid account status")
	ErrEmptyAccountCode     = errors.New("account code cannot be empty")
	ErrEmptyAccountName     = errors.New("account name cannot be empty")
	ErrAccountClosed        = errors.New("account is closed")
	ErrAccountAlreadyClosed = errors.New("account is already closed")

	// Posting errors.
	ErrInvalidPostingDirection = errors.New("invalid posting direction")

	// Journal entry errors.
	ErrJournalEntryWithoutPostings = errors.New("journal entry must contain at least one posting")
	ErrJournalEntryWithoutDebit    = errors.New("journal entry must contain at least one debit posting")
	ErrJournalEntryWithoutCredit   = errors.New("journal entry must contain at least one credit posting")
	ErrUnbalancedJournalEntry      = errors.New("journal entry is unbalanced: total debits must equal total credits")

	// Transaction and reversal errors.
	ErrEmptyTransactionDescription     = errors.New("transaction description cannot be empty")
	ErrTransactionJournalEntryMismatch = errors.New("journal entry does not belong to transaction")
	ErrCannotReverseReversal           = errors.New("a reversal transaction cannot be reversed")
	ErrReversalSameTransactionID       = errors.New("reversal must use a different transaction id")
	ErrInvalidReversalPostingIDCount   = errors.New("reversal must provide one posting id per original posting")

	// Wallet errors
	ErrInvalidWalletStatus    = errors.New("invalid wallet status")
	ErrEmptyWalletOwnerID     = errors.New("wallet owner ID cannot be empty")
	ErrWalletSuspended        = errors.New("wallet is suspended")
	ErrWalletClosed           = errors.New("wallet is closed")
	ErrWalletAlreadySuspended = errors.New("wallet is already suspended")
	ErrWalletAlreadyClosed    = errors.New("wallet is already closed")
)
