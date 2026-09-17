package wallet

import (
	"context"
	"errors"

	"financial-ledger/internal/ledger/application/transaction"
	"financial-ledger/internal/ledger/domain"
)

type WithdrawWalletCommand struct {
	WalletID domain.WalletID

	ClearingAccountID domain.AccountID

	TransactionID  domain.TransactionID
	JournalEntryID domain.JournalEntryID

	WalletPostingID   domain.PostingID
	ClearingPostingID domain.PostingID

	AmountMinorUnits int64
	Description      string

	IdempotencyKey string
	RequestHash    string
}

type WithdrawWalletUseCase struct {
	wallets         WalletReader
	balances        WalletBalanceReader
	transactionPost TransactionPoster
}

func NewWithdrawWalletUseCase(
	wallets WalletReader,
	balances WalletBalanceReader,
	transactionPost TransactionPoster,
) WithdrawWalletUseCase {
	return WithdrawWalletUseCase{
		wallets:         wallets,
		balances:        balances,
		transactionPost: transactionPost,
	}
}

func (uc WithdrawWalletUseCase) Execute(
	ctx context.Context,
	command WithdrawWalletCommand,
) (domain.Transaction, error) {
	if command.WalletID.IsZero() {
		return domain.Transaction{}, domain.ErrInvalidID
	}

	if command.ClearingAccountID.IsZero() {
		return domain.Transaction{}, domain.ErrInvalidID
	}

	if command.TransactionID.IsZero() ||
		command.JournalEntryID.IsZero() ||
		command.WalletPostingID.IsZero() ||
		command.ClearingPostingID.IsZero() {
		return domain.Transaction{}, domain.ErrInvalidID
	}

	if command.AmountMinorUnits < 0 {
		return domain.Transaction{}, domain.ErrAmountMustNotBeNegative
	}

	if command.AmountMinorUnits == 0 {
		return domain.Transaction{}, domain.ErrAmountMustBePositive
	}

	wallet, err := uc.wallets.GetByID(ctx, command.WalletID)
	if err != nil {
		return domain.Transaction{}, err
	}

	if err := wallet.CanOperate(); err != nil {
		return domain.Transaction{}, err
	}

	if wallet.LedgerAccountID() == command.ClearingAccountID {
		return domain.Transaction{}, ErrWithdrawalSameAccount
	}

	snapshot, err := uc.balances.GetLedgerBalance(
		ctx,
		wallet.LedgerAccountID(),
	)
	if err != nil {
		return domain.Transaction{}, err
	}

	balance, err := CalculateWalletBalance(wallet, snapshot)
	if err != nil {
		return domain.Transaction{}, err
	}

	if balance.AvailableBalanceMinorUnits < command.AmountMinorUnits {
		return domain.Transaction{}, ErrInsufficientFunds
	}

	postedTransaction, err := uc.transactionPost.Execute(
		ctx,
		transaction.PostTransactionCommand{
			ID:             command.TransactionID,
			JournalEntryID: command.JournalEntryID,
			Description:    command.Description,
			CurrencyCode:   wallet.Currency().String(),
			IdempotencyKey: command.IdempotencyKey,
			RequestHash:    command.RequestHash,
			Postings: []transaction.PostingCommand{
				{
					ID:               command.WalletPostingID,
					AccountID:        wallet.LedgerAccountID(),
					Direction:        domain.PostingDirectionDebit,
					AmountMinorUnits: command.AmountMinorUnits,
				},
				{
					ID:               command.ClearingPostingID,
					AccountID:        command.ClearingAccountID,
					Direction:        domain.PostingDirectionCredit,
					AmountMinorUnits: command.AmountMinorUnits,
				},
			},
		},
	)
	if errors.Is(err, transaction.ErrInsufficientBalance) {
		return domain.Transaction{}, ErrInsufficientFunds
	}

	return postedTransaction, err
}
