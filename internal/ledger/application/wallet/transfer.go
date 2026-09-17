package wallet

import (
	"context"
	"errors"

	"financial-ledger/internal/ledger/application/transaction"
	"financial-ledger/internal/ledger/domain"
)

type TransferWalletCommand struct {
	SourceWalletID      domain.WalletID
	DestinationWalletID domain.WalletID

	TransactionID  domain.TransactionID
	JournalEntryID domain.JournalEntryID

	SourcePostingID      domain.PostingID
	DestinationPostingID domain.PostingID

	AmountMinorUnits int64
	Description      string

	IdempotencyKey string
	RequestHash    string
}

type TransferWalletUseCase struct {
	wallets         WalletReader
	balances        WalletBalanceReader
	transactionPost TransactionPoster
}

func NewTransferWalletUseCase(
	wallets WalletReader,
	balances WalletBalanceReader,
	transactionPost TransactionPoster,
) TransferWalletUseCase {
	return TransferWalletUseCase{
		wallets:         wallets,
		balances:        balances,
		transactionPost: transactionPost,
	}
}

func (uc TransferWalletUseCase) Execute(
	ctx context.Context,
	command TransferWalletCommand,
) (domain.Transaction, error) {
	if command.SourceWalletID.IsZero() ||
		command.DestinationWalletID.IsZero() {
		return domain.Transaction{}, domain.ErrInvalidID
	}

	if command.SourceWalletID == command.DestinationWalletID {
		return domain.Transaction{}, ErrTransferSameWallet
	}

	if command.TransactionID.IsZero() ||
		command.JournalEntryID.IsZero() ||
		command.SourcePostingID.IsZero() ||
		command.DestinationPostingID.IsZero() {
		return domain.Transaction{}, domain.ErrInvalidID
	}

	if command.AmountMinorUnits < 0 {
		return domain.Transaction{}, domain.ErrAmountMustNotBeNegative
	}

	if command.AmountMinorUnits == 0 {
		return domain.Transaction{}, domain.ErrAmountMustBePositive
	}

	sourceWallet, err := uc.wallets.GetByID(
		ctx,
		command.SourceWalletID,
	)
	if err != nil {
		return domain.Transaction{}, err
	}

	destinationWallet, err := uc.wallets.GetByID(
		ctx,
		command.DestinationWalletID,
	)
	if err != nil {
		return domain.Transaction{}, err
	}

	if err := sourceWallet.CanOperate(); err != nil {
		return domain.Transaction{}, err
	}

	if err := destinationWallet.CanOperate(); err != nil {
		return domain.Transaction{}, err
	}

	if !sourceWallet.Currency().Equal(destinationWallet.Currency()) {
		return domain.Transaction{}, domain.ErrCurrencyMismatch
	}

	if sourceWallet.LedgerAccountID() ==
		destinationWallet.LedgerAccountID() {
		return domain.Transaction{}, ErrTransferSameAccount
	}

	snapshot, err := uc.balances.GetLedgerBalance(
		ctx,
		sourceWallet.LedgerAccountID(),
	)
	if err != nil {
		return domain.Transaction{}, err
	}

	sourceBalance, err := CalculateWalletBalance(
		sourceWallet,
		snapshot,
	)

	if err != nil {
		return domain.Transaction{}, err
	}

	if sourceBalance.AvailableBalanceMinorUnits <
		command.AmountMinorUnits {
		return domain.Transaction{}, ErrInsufficientFunds
	}

	postedTransaction, err := uc.transactionPost.Execute(
		ctx,
		transaction.PostTransactionCommand{
			ID:             command.TransactionID,
			JournalEntryID: command.JournalEntryID,
			Description:    command.Description,
			CurrencyCode:   sourceWallet.Currency().String(),
			IdempotencyKey: command.IdempotencyKey,
			RequestHash:    command.RequestHash,
			Postings: []transaction.PostingCommand{
				{
					ID:               command.SourcePostingID,
					AccountID:        sourceWallet.LedgerAccountID(),
					Direction:        domain.PostingDirectionDebit,
					AmountMinorUnits: command.AmountMinorUnits,
				},
				{
					ID:               command.DestinationPostingID,
					AccountID:        destinationWallet.LedgerAccountID(),
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
