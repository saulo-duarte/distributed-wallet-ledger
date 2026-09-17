package wallet

import (
	"context"
	"errors"
	"testing"

	"financial-ledger/internal/ledger/application/transaction"
	"financial-ledger/internal/ledger/domain"
)

func TestDepositWalletUseCaseCreatesClearingDebitAndWalletCredit(t *testing.T) {
	foundWallet := newTestWallet(t, "wallet-001", "owner-001", "wallet-account-001")
	poster := &fakeDepositTransactionPoster{}
	useCase := NewDepositWalletUseCase(
		&fakeWalletReader{wallet: foundWallet},
		poster,
	)
	command := newDepositCommand(foundWallet)
	command.AmountMinorUnits = 3000

	_, err := useCase.Execute(context.Background(), command)
	if err != nil {
		t.Fatalf("Execute() returned an unexpected error: %v", err)
	}

	if poster.calls != 1 {
		t.Fatalf("transaction poster calls = %d, want 1", poster.calls)
	}
	if len(poster.command.Postings) != 2 {
		t.Fatalf("posting count = %d, want 2", len(poster.command.Postings))
	}

	clearingPosting := poster.command.Postings[0]
	if clearingPosting.AccountID != command.ClearingAccountID {
		t.Fatalf("clearing account = %q, want %q", clearingPosting.AccountID, command.ClearingAccountID)
	}
	if clearingPosting.Direction != domain.PostingDirectionDebit {
		t.Fatalf("clearing direction = %q, want debit", clearingPosting.Direction)
	}

	walletPosting := poster.command.Postings[1]
	if walletPosting.AccountID != foundWallet.LedgerAccountID() {
		t.Fatalf("wallet account = %q, want %q", walletPosting.AccountID, foundWallet.LedgerAccountID())
	}
	if walletPosting.Direction != domain.PostingDirectionCredit {
		t.Fatalf("wallet direction = %q, want credit", walletPosting.Direction)
	}
	if walletPosting.AmountMinorUnits != 3000 {
		t.Fatalf("wallet amount = %d, want 3000", walletPosting.AmountMinorUnits)
	}
}

func TestDepositWalletUseCaseRejectsInvalidAmount(t *testing.T) {
	foundWallet := newTestWallet(t, "wallet-001", "owner-001", "wallet-account-001")
	poster := &fakeDepositTransactionPoster{}
	useCase := NewDepositWalletUseCase(
		&fakeWalletReader{wallet: foundWallet},
		poster,
	)

	for _, test := range []struct {
		name   string
		amount int64
		want   error
	}{
		{name: "zero", amount: 0, want: domain.ErrAmountMustBePositive},
		{name: "negative", amount: -1, want: domain.ErrAmountMustNotBeNegative},
	} {
		t.Run(test.name, func(t *testing.T) {
			command := newDepositCommand(foundWallet)
			command.AmountMinorUnits = test.amount

			_, err := useCase.Execute(context.Background(), command)
			if !errors.Is(err, test.want) {
				t.Fatalf("error = %v, want %v", err, test.want)
			}
		})
	}

	if poster.calls != 0 {
		t.Fatal("invalid deposit must not be posted")
	}
}

func TestDepositWalletUseCaseRejectsSameAccount(t *testing.T) {
	foundWallet := newTestWallet(t, "wallet-001", "owner-001", "wallet-account-001")
	poster := &fakeDepositTransactionPoster{}
	useCase := NewDepositWalletUseCase(
		&fakeWalletReader{wallet: foundWallet},
		poster,
	)
	command := newDepositCommand(foundWallet)
	command.ClearingAccountID = foundWallet.LedgerAccountID()

	_, err := useCase.Execute(context.Background(), command)
	if !errors.Is(err, ErrDepositSameAccount) {
		t.Fatalf("error = %v, want %v", err, ErrDepositSameAccount)
	}
	if poster.calls != 0 {
		t.Fatal("same-account deposit must not be posted")
	}
}

func TestDepositWalletUseCasePropagatesWalletError(t *testing.T) {
	expectedErr := ErrWalletNotFound
	poster := &fakeDepositTransactionPoster{}
	useCase := NewDepositWalletUseCase(
		&fakeWalletReader{err: expectedErr},
		poster,
	)

	_, err := useCase.Execute(
		context.Background(),
		newDepositCommandWithWalletID(domain.WalletID("wallet-001")),
	)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("error = %v, want %v", err, expectedErr)
	}
	if poster.calls != 0 {
		t.Fatal("transaction must not be posted when wallet lookup fails")
	}
}

func newDepositCommand(foundWallet domain.Wallet) DepositWalletCommand {
	return DepositWalletCommand{
		WalletID:          foundWallet.ID(),
		ClearingAccountID: domain.AccountID("clearing-account-001"),
		TransactionID:     domain.TransactionID("transaction-deposit-001"),
		JournalEntryID:    domain.JournalEntryID("journal-deposit-001"),
		ClearingPostingID: domain.PostingID("posting-clearing-001"),
		WalletPostingID:   domain.PostingID("posting-wallet-001"),
		AmountMinorUnits:  1000,
		Description:       "Wallet deposit",
		IdempotencyKey:    "deposit-key-001",
		RequestHash:       "deposit-hash-001",
	}
}

func newDepositCommandWithWalletID(walletID domain.WalletID) DepositWalletCommand {
	command := newDepositCommand(domain.Wallet{})
	command.WalletID = walletID
	return command
}

type fakeDepositTransactionPoster struct {
	command transaction.PostTransactionCommand
	calls   int
	err     error
}

func (f *fakeDepositTransactionPoster) Execute(
	_ context.Context,
	command transaction.PostTransactionCommand,
) (domain.Transaction, error) {
	f.calls++
	f.command = command

	if f.err != nil {
		return domain.Transaction{}, f.err
	}

	return domain.Transaction{}, nil
}
