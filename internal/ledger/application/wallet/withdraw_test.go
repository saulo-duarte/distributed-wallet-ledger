package wallet

import (
	"context"
	"errors"
	"testing"

	"financial-ledger/internal/ledger/application/transaction"
	"financial-ledger/internal/ledger/domain"
)

func TestWithdrawWalletUseCaseCreatesDebitAndCreditPostings(t *testing.T) {
	wallet := newTestWallet(t, "wallet-001", "owner-001", "wallet-account-001")

	wallets := &fakeWalletReader{wallet: wallet}
	balances := &fakeWalletBalanceReader{
		snapshot: LedgerBalanceSnapshot{
			TotalDebits:  2500,
			TotalCredits: 10000,
		},
	}
	poster := &fakeWithdrawTransactionPoster{}

	useCase := NewWithdrawWalletUseCase(wallets, balances, poster)
	command := newWithdrawCommand(wallet)
	command.AmountMinorUnits = 3000

	_, err := useCase.Execute(context.Background(), command)
	if err != nil {
		t.Fatalf("Execute() returned an unexpected error: %v", err)
	}

	if poster.calls != 1 {
		t.Fatalf("transaction poster calls = %d, want 1", poster.calls)
	}

	posted := poster.command
	if posted.ID != command.TransactionID {
		t.Fatalf("transaction ID = %q, want %q", posted.ID, command.TransactionID)
	}
	if posted.JournalEntryID != command.JournalEntryID {
		t.Fatalf("journal entry ID = %q, want %q", posted.JournalEntryID, command.JournalEntryID)
	}
	if posted.CurrencyCode != "BRL" {
		t.Fatalf("currency = %q, want BRL", posted.CurrencyCode)
	}
	if posted.IdempotencyKey != command.IdempotencyKey {
		t.Fatalf("idempotency key = %q, want %q", posted.IdempotencyKey, command.IdempotencyKey)
	}
	if posted.RequestHash != command.RequestHash {
		t.Fatalf("request hash = %q, want %q", posted.RequestHash, command.RequestHash)
	}

	if len(posted.Postings) != 2 {
		t.Fatalf("posting count = %d, want 2", len(posted.Postings))
	}

	walletPosting := posted.Postings[0]
	if walletPosting.ID != command.WalletPostingID {
		t.Fatalf("wallet posting ID = %q, want %q", walletPosting.ID, command.WalletPostingID)
	}
	if walletPosting.AccountID != wallet.LedgerAccountID() {
		t.Fatalf("wallet account ID = %q, want %q", walletPosting.AccountID, wallet.LedgerAccountID())
	}
	if walletPosting.Direction != domain.PostingDirectionDebit {
		t.Fatalf("wallet posting direction = %q, want debit", walletPosting.Direction)
	}
	if walletPosting.AmountMinorUnits != command.AmountMinorUnits {
		t.Fatalf("wallet posting amount = %d, want %d", walletPosting.AmountMinorUnits, command.AmountMinorUnits)
	}

	clearingPosting := posted.Postings[1]
	if clearingPosting.ID != command.ClearingPostingID {
		t.Fatalf("clearing posting ID = %q, want %q", clearingPosting.ID, command.ClearingPostingID)
	}
	if clearingPosting.AccountID != command.ClearingAccountID {
		t.Fatalf("clearing account ID = %q, want %q", clearingPosting.AccountID, command.ClearingAccountID)
	}
	if clearingPosting.Direction != domain.PostingDirectionCredit {
		t.Fatalf("clearing posting direction = %q, want credit", clearingPosting.Direction)
	}
	if clearingPosting.AmountMinorUnits != command.AmountMinorUnits {
		t.Fatalf("clearing posting amount = %d, want %d", clearingPosting.AmountMinorUnits, command.AmountMinorUnits)
	}
}

func TestWithdrawWalletUseCaseRejectsInsufficientFunds(t *testing.T) {
	wallet := newTestWallet(t, "wallet-001", "owner-001", "wallet-account-001")
	balances := &fakeWalletBalanceReader{
		snapshot: LedgerBalanceSnapshot{
			TotalCredits: 1000,
		},
	}
	poster := &fakeWithdrawTransactionPoster{}

	useCase := NewWithdrawWalletUseCase(
		&fakeWalletReader{wallet: wallet},
		balances,
		poster,
	)
	command := newWithdrawCommand(wallet)
	command.AmountMinorUnits = 1001

	_, err := useCase.Execute(context.Background(), command)
	if !errors.Is(err, ErrInsufficientFunds) {
		t.Fatalf("error = %v, want %v", err, ErrInsufficientFunds)
	}
	if poster.calls != 0 {
		t.Fatal("transaction must not be posted when funds are insufficient")
	}
}

func TestWithdrawWalletUseCasePropagatesWalletError(t *testing.T) {
	expectedErr := ErrWalletNotFound
	poster := &fakeWithdrawTransactionPoster{}

	useCase := NewWithdrawWalletUseCase(
		&fakeWalletReader{err: expectedErr},
		&fakeWalletBalanceReader{},
		poster,
	)

	_, err := useCase.Execute(
		context.Background(),
		newWithdrawCommandWithWalletID(domain.WalletID("wallet-001")),
	)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("error = %v, want %v", err, expectedErr)
	}
	if poster.calls != 0 {
		t.Fatal("transaction must not be posted when wallet lookup fails")
	}
}

func TestWithdrawWalletUseCasePropagatesBalanceError(t *testing.T) {
	wallet := newTestWallet(t, "wallet-001", "owner-001", "wallet-account-001")
	expectedErr := errors.New("balance unavailable")
	poster := &fakeWithdrawTransactionPoster{}

	useCase := NewWithdrawWalletUseCase(
		&fakeWalletReader{wallet: wallet},
		&fakeWalletBalanceReader{err: expectedErr},
		poster,
	)

	_, err := useCase.Execute(context.Background(), newWithdrawCommand(wallet))
	if !errors.Is(err, expectedErr) {
		t.Fatalf("error = %v, want %v", err, expectedErr)
	}
	if poster.calls != 0 {
		t.Fatal("transaction must not be posted when balance lookup fails")
	}
}

func TestWithdrawWalletUseCaseRejectsSuspendedWallet(t *testing.T) {
	wallet := newTestWallet(t, "wallet-001", "owner-001", "wallet-account-001")
	if err := wallet.Suspend(); err != nil {
		t.Fatal(err)
	}

	balances := &fakeWalletBalanceReader{}
	poster := &fakeWithdrawTransactionPoster{}
	useCase := NewWithdrawWalletUseCase(
		&fakeWalletReader{wallet: wallet},
		balances,
		poster,
	)

	_, err := useCase.Execute(context.Background(), newWithdrawCommand(wallet))
	if !errors.Is(err, domain.ErrWalletSuspended) {
		t.Fatalf("error = %v, want %v", err, domain.ErrWalletSuspended)
	}
	if balances.calls != 0 || poster.calls != 0 {
		t.Fatal("suspended wallet must not read balance or post transaction")
	}
}

func TestWithdrawWalletUseCaseRejectsClosedWallet(t *testing.T) {
	wallet := newTestWallet(t, "wallet-001", "owner-001", "wallet-account-001")
	if err := wallet.Close(); err != nil {
		t.Fatal(err)
	}

	balances := &fakeWalletBalanceReader{}
	poster := &fakeWithdrawTransactionPoster{}
	useCase := NewWithdrawWalletUseCase(
		&fakeWalletReader{wallet: wallet},
		balances,
		poster,
	)

	_, err := useCase.Execute(context.Background(), newWithdrawCommand(wallet))
	if !errors.Is(err, domain.ErrWalletClosed) {
		t.Fatalf("error = %v, want %v", err, domain.ErrWalletClosed)
	}
	if balances.calls != 0 || poster.calls != 0 {
		t.Fatal("closed wallet must not read balance or post transaction")
	}
}

func TestWithdrawWalletUseCaseRejectsInvalidAmount(t *testing.T) {
	wallet := newTestWallet(t, "wallet-001", "owner-001", "wallet-account-001")
	poster := &fakeWithdrawTransactionPoster{}
	useCase := NewWithdrawWalletUseCase(
		&fakeWalletReader{wallet: wallet},
		&fakeWalletBalanceReader{},
		poster,
	)

	tests := []struct {
		name   string
		amount int64
		want   error
	}{
		{
			name:   "zero",
			amount: 0,
			want:   domain.ErrAmountMustBePositive,
		},
		{
			name:   "negative",
			amount: -1,
			want:   domain.ErrAmountMustNotBeNegative,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			command := newWithdrawCommand(wallet)
			command.AmountMinorUnits = test.amount

			_, err := useCase.Execute(context.Background(), command)
			if !errors.Is(err, test.want) {
				t.Fatalf("error = %v, want %v", err, test.want)
			}
		})
	}

	if poster.calls != 0 {
		t.Fatal("invalid amount must not post a transaction")
	}
}

func TestWithdrawWalletUseCaseRejectsSameWalletAndClearingAccount(t *testing.T) {
	wallet := newTestWallet(t, "wallet-001", "owner-001", "wallet-account-001")
	command := newWithdrawCommand(wallet)
	command.ClearingAccountID = wallet.LedgerAccountID()

	poster := &fakeWithdrawTransactionPoster{}
	useCase := NewWithdrawWalletUseCase(
		&fakeWalletReader{wallet: wallet},
		&fakeWalletBalanceReader{},
		poster,
	)

	_, err := useCase.Execute(context.Background(), command)
	if !errors.Is(err, ErrWithdrawalSameAccount) {
		t.Fatalf("error = %v, want %v", err, ErrWithdrawalSameAccount)
	}
	if poster.calls != 0 {
		t.Fatal("same-account withdrawal must not be posted")
	}
}

func TestWithdrawWalletUseCasePropagatesTransactionError(t *testing.T) {
	wallet := newTestWallet(t, "wallet-001", "owner-001", "wallet-account-001")
	expectedErr := errors.New("transaction unavailable")
	poster := &fakeWithdrawTransactionPoster{err: expectedErr}

	useCase := NewWithdrawWalletUseCase(
		&fakeWalletReader{wallet: wallet},
		&fakeWalletBalanceReader{
			snapshot: LedgerBalanceSnapshot{TotalCredits: 10000},
		},
		poster,
	)

	_, err := useCase.Execute(context.Background(), newWithdrawCommand(wallet))
	if !errors.Is(err, expectedErr) {
		t.Fatalf("error = %v, want %v", err, expectedErr)
	}
}

func newWithdrawCommand(wallet domain.Wallet) WithdrawWalletCommand {
	return WithdrawWalletCommand{
		WalletID:          wallet.ID(),
		ClearingAccountID: domain.AccountID("clearing-account-001"),
		TransactionID:     domain.TransactionID("transaction-001"),
		JournalEntryID:    domain.JournalEntryID("journal-entry-001"),
		WalletPostingID:   domain.PostingID("posting-wallet-001"),
		ClearingPostingID: domain.PostingID("posting-clearing-001"),
		AmountMinorUnits:  1000,
		Description:       "Wallet withdrawal",
		IdempotencyKey:    "withdrawal-key-001",
		RequestHash:       "withdrawal-hash-001",
	}
}

func newWithdrawCommandWithWalletID(walletID domain.WalletID) WithdrawWalletCommand {
	return WithdrawWalletCommand{
		WalletID:          walletID,
		ClearingAccountID: domain.AccountID("clearing-account-001"),
		TransactionID:     domain.TransactionID("transaction-001"),
		JournalEntryID:    domain.JournalEntryID("journal-entry-001"),
		WalletPostingID:   domain.PostingID("posting-wallet-001"),
		ClearingPostingID: domain.PostingID("posting-clearing-001"),
		AmountMinorUnits:  1000,
		Description:       "Wallet withdrawal",
		IdempotencyKey:    "withdrawal-key-001",
		RequestHash:       "withdrawal-hash-001",
	}
}

type fakeWithdrawTransactionPoster struct {
	command transaction.PostTransactionCommand
	calls   int
	err     error
}

func (f *fakeWithdrawTransactionPoster) Execute(
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

type fakeWalletProjector struct {
	walletID domain.WalletID
	calls    int
	err      error
}

func (f *fakeWalletProjector) ProjectBalance(
	_ context.Context,
	walletID domain.WalletID,
) (WalletBalance, error) {
	f.calls++
	f.walletID = walletID
	return WalletBalance{}, f.err
}

func TestWithdrawWalletUseCaseInvokesProjectorOnSuccess(t *testing.T) {
	w := newTestWallet(t, "wallet-001", "owner-001", "wallet-account-001")
	wallets := &fakeWalletReader{wallet: w}
	balances := &fakeWalletBalanceReader{
		snapshot: LedgerBalanceSnapshot{
			TotalDebits:  0,
			TotalCredits: 10000,
		},
	}
	poster := &fakeWithdrawTransactionPoster{}
	projector := &fakeWalletProjector{}

	useCase := NewWithdrawWalletUseCaseWithProjector(wallets, balances, poster, projector)
	command := newWithdrawCommand(w)
	command.AmountMinorUnits = 1000

	_, err := useCase.Execute(context.Background(), command)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if projector.calls != 1 {
		t.Fatalf("projector calls = %d, want 1", projector.calls)
	}

	if projector.walletID != w.ID() {
		t.Fatalf("projector walletID = %q, want %q", projector.walletID, w.ID())
	}
}
