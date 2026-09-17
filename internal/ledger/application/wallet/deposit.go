package wallet

import (
	"context"
	"errors"

	"financial-ledger/internal/ledger/application/transaction"
	"financial-ledger/internal/ledger/domain"
)

type DepositWalletCommand struct {
	WalletID domain.WalletID

	ClearingAccountID domain.AccountID

	TransactionID  domain.TransactionID
	JournalEntryID domain.JournalEntryID

	ClearingPostingID domain.PostingID
	WalletPostingID   domain.PostingID

	AmountMinorUnits int64
	Description      string

	IdempotencyKey string
	RequestHash    string
}

type DepositWalletUseCase struct {
	wallets         WalletReader
	transactionPost TransactionPoster
}

func NewDepositWalletUseCase(
	wallets WalletReader,
	transactionPost TransactionPoster,
) DepositWalletUseCase {
	return DepositWalletUseCase{
		wallets:         wallets,
		transactionPost: transactionPost,
	}
}

func (uc DepositWalletUseCase) Execute(
	ctx context.Context,
	command DepositWalletCommand,
) (domain.Transaction, error) {
	if command.WalletID.IsZero() ||
		command.ClearingAccountID.IsZero() {
		return domain.Transaction{}, domain.ErrInvalidID
	}

	if command.TransactionID.IsZero() ||
		command.JournalEntryID.IsZero() ||
		command.ClearingPostingID.IsZero() ||
		command.WalletPostingID.IsZero() {
		return domain.Transaction{}, domain.ErrInvalidID
	}

	if command.AmountMinorUnits < 0 {
		return domain.Transaction{}, domain.ErrAmountMustNotBeNegative
	}

	if command.AmountMinorUnits == 0 {
		return domain.Transaction{}, domain.ErrAmountMustBePositive
	}

	foundWallet, err := uc.wallets.GetByID(ctx, command.WalletID)
	if err != nil {
		return domain.Transaction{}, err
	}

	if err := foundWallet.CanOperate(); err != nil {
		return domain.Transaction{}, err
	}

	if foundWallet.LedgerAccountID() == command.ClearingAccountID {
		return domain.Transaction{}, ErrDepositSameAccount
	}

	postedTransaction, err := uc.transactionPost.Execute(
		ctx,
		transaction.PostTransactionCommand{
			ID:             command.TransactionID,
			JournalEntryID: command.JournalEntryID,
			Description:    command.Description,
			CurrencyCode:   foundWallet.Currency().String(),
			IdempotencyKey: command.IdempotencyKey,
			RequestHash:    command.RequestHash,
			Postings: []transaction.PostingCommand{
				{
					ID:               command.ClearingPostingID,
					AccountID:        command.ClearingAccountID,
					Direction:        domain.PostingDirectionDebit,
					AmountMinorUnits: command.AmountMinorUnits,
				},
				{
					ID:               command.WalletPostingID,
					AccountID:        foundWallet.LedgerAccountID(),
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
