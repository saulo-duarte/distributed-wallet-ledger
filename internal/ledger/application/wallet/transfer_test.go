package wallet

import (
	"context"
	"errors"
	"testing"

	"financial-ledger/internal/ledger/application/transaction"
	"financial-ledger/internal/ledger/domain"
)

func TestTransferWalletUseCaseCreatesDebitAndCreditPostings(t *testing.T) {
	source := newTestWallet(t, "wallet-source", "owner-001", "account-source")
	destination := newTestWallet(t, "wallet-destination", "owner-002", "account-destination")

	wallets := &fakeTransferWalletReader{
		wallets: map[domain.WalletID]domain.Wallet{
			source.ID():      source,
			destination.ID(): destination,
		},
	}
	balances := &fakeWalletBalanceReader{
		snapshot: LedgerBalanceSnapshot{
			TotalCredits: 10000,
		},
	}
	poster := &fakeTransferTransactionPoster{}
	useCase := NewTransferWalletUseCase(wallets, balances, poster)

	command := newTransferCommand(source, destination)
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
	if posted.CurrencyCode != "BRL" {
		t.Fatalf("currency = %q, want BRL", posted.CurrencyCode)
	}
	if len(posted.Postings) != 2 {
		t.Fatalf("posting count = %d, want 2", len(posted.Postings))
	}

	if posted.Postings[0].AccountID != source.LedgerAccountID() {
		t.Fatalf("source account = %q, want %q", posted.Postings[0].AccountID, source.LedgerAccountID())
	}
	if posted.Postings[0].Direction != domain.PostingDirectionDebit {
		t.Fatalf("source direction = %q, want debit", posted.Postings[0].Direction)
	}
	if posted.Postings[1].AccountID != destination.LedgerAccountID() {
		t.Fatalf("destination account = %q, want %q", posted.Postings[1].AccountID, destination.LedgerAccountID())
	}
	if posted.Postings[1].Direction != domain.PostingDirectionCredit {
		t.Fatalf("destination direction = %q, want credit", posted.Postings[1].Direction)
	}
	if posted.Postings[0].AmountMinorUnits != 3000 ||
		posted.Postings[1].AmountMinorUnits != 3000 {
		t.Fatal("source and destination postings must have the transfer amount")
	}
}

func TestTransferWalletUseCaseRejectsInsufficientFunds(t *testing.T) {
	source := newTestWallet(t, "wallet-source", "owner-001", "account-source")
	destination := newTestWallet(t, "wallet-destination", "owner-002", "account-destination")
	poster := &fakeTransferTransactionPoster{}
	useCase := NewTransferWalletUseCase(
		&fakeTransferWalletReader{
			wallets: map[domain.WalletID]domain.Wallet{
				source.ID():      source,
				destination.ID(): destination,
			},
		},
		&fakeWalletBalanceReader{
			snapshot: LedgerBalanceSnapshot{TotalCredits: 1000},
		},
		poster,
	)
	command := newTransferCommand(source, destination)
	command.AmountMinorUnits = 1001

	_, err := useCase.Execute(context.Background(), command)
	if !errors.Is(err, ErrInsufficientFunds) {
		t.Fatalf("error = %v, want %v", err, ErrInsufficientFunds)
	}
	if poster.calls != 0 {
		t.Fatal("transaction must not be posted with insufficient funds")
	}
}

func TestTransferWalletUseCaseRejectsDifferentCurrencies(t *testing.T) {
	source := newTestWallet(t, "wallet-source", "owner-001", "account-source")
	destination := newTestWalletWithCurrency(
		t,
		"wallet-destination",
		"owner-002",
		"account-destination",
		"USD",
	)
	poster := &fakeTransferTransactionPoster{}
	useCase := NewTransferWalletUseCase(
		&fakeTransferWalletReader{
			wallets: map[domain.WalletID]domain.Wallet{
				source.ID():      source,
				destination.ID(): destination,
			},
		},
		&fakeWalletBalanceReader{},
		poster,
	)

	_, err := useCase.Execute(
		context.Background(),
		newTransferCommand(source, destination),
	)
	if !errors.Is(err, domain.ErrCurrencyMismatch) {
		t.Fatalf("error = %v, want %v", err, domain.ErrCurrencyMismatch)
	}
	if poster.calls != 0 {
		t.Fatal("transaction must not be posted for different currencies")
	}
}

func TestTransferWalletUseCasePropagatesWalletError(t *testing.T) {
	source := newTestWallet(t, "wallet-source", "owner-001", "account-source")
	destination := newTestWallet(t, "wallet-destination", "owner-002", "account-destination")
	expectedErr := ErrWalletNotFound
	poster := &fakeTransferTransactionPoster{}
	useCase := NewTransferWalletUseCase(
		&fakeTransferWalletReader{
			wallets: map[domain.WalletID]domain.Wallet{
				source.ID():      source,
				destination.ID(): destination,
			},
			err: expectedErr,
		},
		&fakeWalletBalanceReader{},
		poster,
	)

	_, err := useCase.Execute(
		context.Background(),
		newTransferCommand(source, destination),
	)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("error = %v, want %v", err, expectedErr)
	}
	if poster.calls != 0 {
		t.Fatal("transaction must not be posted when destination lookup fails")
	}
}

func TestTransferWalletUseCaseRejectsSameWallet(t *testing.T) {
	source := newTestWallet(t, "wallet-source", "owner-001", "account-source")
	poster := &fakeTransferTransactionPoster{}
	useCase := NewTransferWalletUseCase(
		&fakeTransferWalletReader{
			wallets: map[domain.WalletID]domain.Wallet{
				source.ID(): source,
			},
		},
		&fakeWalletBalanceReader{},
		poster,
	)
	command := newTransferCommand(source, source)

	_, err := useCase.Execute(context.Background(), command)
	if !errors.Is(err, ErrTransferSameWallet) {
		t.Fatalf("error = %v, want %v", err, ErrTransferSameWallet)
	}
	if poster.calls != 0 {
		t.Fatal("same-wallet transfer must not be posted")
	}
}

func newTransferCommand(
	source domain.Wallet,
	destination domain.Wallet,
) TransferWalletCommand {
	return TransferWalletCommand{
		SourceWalletID:       source.ID(),
		DestinationWalletID:  destination.ID(),
		TransactionID:        domain.TransactionID("transaction-transfer-001"),
		JournalEntryID:       domain.JournalEntryID("journal-transfer-001"),
		SourcePostingID:      domain.PostingID("posting-source-001"),
		DestinationPostingID: domain.PostingID("posting-destination-001"),
		AmountMinorUnits:     1000,
		Description:          "Wallet transfer",
		IdempotencyKey:       "transfer-key-001",
		RequestHash:          "transfer-hash-001",
	}
}

func newTestWalletWithCurrency(
	t *testing.T,
	id string,
	ownerID string,
	ledgerAccountID string,
	currencyCode string,
) domain.Wallet {
	t.Helper()

	currency, err := domain.NewCurrency(currencyCode)
	if err != nil {
		t.Fatal(err)
	}

	foundWallet, err := domain.NewWallet(
		domain.WalletID(id),
		domain.OwnerID(ownerID),
		domain.AccountID(ledgerAccountID),
		currency,
	)
	if err != nil {
		t.Fatal(err)
	}

	return foundWallet
}

type fakeTransferWalletReader struct {
	wallets map[domain.WalletID]domain.Wallet
	err     error
}

func (f *fakeTransferWalletReader) GetByID(
	_ context.Context,
	walletID domain.WalletID,
) (domain.Wallet, error) {
	if f.err != nil {
		return domain.Wallet{}, f.err
	}

	foundWallet, ok := f.wallets[walletID]
	if !ok {
		return domain.Wallet{}, ErrWalletNotFound
	}

	return foundWallet, nil
}

func (f *fakeTransferWalletReader) ListByOwnerID(
	_ context.Context,
	_ domain.OwnerID,
) ([]domain.Wallet, error) {
	return nil, nil
}

type fakeTransferTransactionPoster struct {
	command transaction.PostTransactionCommand
	calls   int
	err     error
}

func (f *fakeTransferTransactionPoster) Execute(
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
