package wallet

import (
	"context"
	"errors"
	"testing"

	"financial-ledger/internal/ledger/domain"
)

func TestCreateWalletUseCaseCreatesWallet(t *testing.T) {
	t.Parallel()

	repository := &fakeWalletRepository{}
	useCase := NewCreateWalletUseCase(repository)

	createdWallet, err := useCase.Execute(
		context.Background(),
		CreateWalletCommand{
			ID:              domain.WalletID("wallet-001"),
			OwnerID:         domain.OwnerID("owner-001"),
			LedgerAccountID: domain.AccountID("account-001"),
			CurrencyCode:    "BRL",
		},
	)
	if err != nil {
		t.Fatalf("Execute() returned an unexpected error: %v", err)
	}

	if createdWallet.ID() != domain.WalletID("wallet-001") {
		t.Fatalf(
			"ID() = %q, want %q",
			createdWallet.ID(),
			"wallet-001",
		)
	}

	if createdWallet.OwnerID() != domain.OwnerID("owner-001") {
		t.Fatalf(
			"OwnerID() = %q, want %q",
			createdWallet.OwnerID(),
			"owner-001",
		)
	}

	if createdWallet.LedgerAccountID() != domain.AccountID("account-001") {
		t.Fatalf(
			"LedgerAccountID() = %q, want %q",
			createdWallet.LedgerAccountID(),
			"account-001",
		)
	}

	if createdWallet.Currency().String() != "BRL" {
		t.Fatalf(
			"Currency() = %q, want BRL",
			createdWallet.Currency(),
		)
	}

	if !createdWallet.IsOpen() {
		t.Fatal("new wallet should be open")
	}

	if repository.createCalls != 1 {
		t.Fatalf(
			"repository Create() calls = %d, want 1",
			repository.createCalls,
		)
	}
}

func TestCreateWalletUseCaseRejectsInvalidCurrency(t *testing.T) {
	t.Parallel()

	repository := &fakeWalletRepository{}
	useCase := NewCreateWalletUseCase(repository)

	_, err := useCase.Execute(
		context.Background(),
		CreateWalletCommand{
			ID:              domain.WalletID("wallet-001"),
			OwnerID:         domain.OwnerID("owner-001"),
			LedgerAccountID: domain.AccountID("account-001"),
			CurrencyCode:    "BR",
		},
	)
	if !errors.Is(err, domain.ErrInvalidCurrency) {
		t.Fatalf(
			"Execute() error = %v, want %v",
			err,
			domain.ErrInvalidCurrency,
		)
	}

	if repository.createCalls != 0 {
		t.Fatal("repository should not be called for invalid currency")
	}
}

func TestCreateWalletUseCaseRejectsInvalidWallet(t *testing.T) {
	t.Parallel()

	repository := &fakeWalletRepository{}
	useCase := NewCreateWalletUseCase(repository)

	_, err := useCase.Execute(
		context.Background(),
		CreateWalletCommand{
			ID:              domain.WalletID(""),
			OwnerID:         domain.OwnerID("owner-001"),
			LedgerAccountID: domain.AccountID("account-001"),
			CurrencyCode:    "BRL",
		},
	)
	if !errors.Is(err, domain.ErrInvalidID) {
		t.Fatalf(
			"Execute() error = %v, want %v",
			err,
			domain.ErrInvalidID,
		)
	}

	if repository.createCalls != 0 {
		t.Fatal("repository should not be called for invalid wallet")
	}
}

func TestCreateWalletUseCasePropagatesRepositoryError(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("database unavailable")

	repository := &fakeWalletRepository{
		err: expectedErr,
	}
	useCase := NewCreateWalletUseCase(repository)

	_, err := useCase.Execute(
		context.Background(),
		CreateWalletCommand{
			ID:              domain.WalletID("wallet-001"),
			OwnerID:         domain.OwnerID("owner-001"),
			LedgerAccountID: domain.AccountID("account-001"),
			CurrencyCode:    "BRL",
		},
	)
	if !errors.Is(err, expectedErr) {
		t.Fatalf(
			"Execute() error = %v, want %v",
			err,
			expectedErr,
		)
	}
}

type fakeWalletRepository struct {
	wallet      domain.Wallet
	createCalls int
	err         error
}

func (f *fakeWalletRepository) Create(
	_ context.Context,
	wallet domain.Wallet,
) error {
	f.createCalls++

	if f.err != nil {
		return f.err
	}

	f.wallet = wallet
	return nil
}
