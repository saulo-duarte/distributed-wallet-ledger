package wallet

import (
	"context"
	"errors"
	"testing"

	"financial-ledger/internal/ledger/domain"
)

func TestGetWalletUseCaseReturnsWallet(t *testing.T) {
	wallet := newTestWallet(t, "wallet-001", "owner-001", "account-001")
	repository := &fakeWalletReader{
		wallet: wallet,
	}
	useCase := NewGetWalletUseCase(repository)

	result, err := useCase.Execute(
		context.Background(),
		wallet.ID(),
	)
	if err != nil {
		t.Fatalf("get wallet: %v", err)
	}

	if result.ID() != wallet.ID() {
		t.Fatalf("unexpected wallet ID: got %q, want %q", result.ID(), wallet.ID())
	}
	if repository.getCalls != 1 {
		t.Fatalf("GetByID() calls = %d, want 1", repository.getCalls)
	}
}

func TestGetWalletUseCaseRejectsEmptyID(t *testing.T) {
	repository := &fakeWalletReader{}
	useCase := NewGetWalletUseCase(repository)

	_, err := useCase.Execute(context.Background(), domain.WalletID(""))
	if !errors.Is(err, domain.ErrInvalidID) {
		t.Fatalf("expected invalid ID error, got %v", err)
	}
	if repository.getCalls != 0 {
		t.Fatal("repository should not be called for an empty ID")
	}
}

func TestGetWalletUseCasePropagatesRepositoryError(t *testing.T) {
	expectedErr := ErrWalletNotFound
	repository := &fakeWalletReader{err: expectedErr}
	useCase := NewGetWalletUseCase(repository)

	_, err := useCase.Execute(
		context.Background(),
		domain.WalletID("wallet-001"),
	)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected repository error, got %v", err)
	}
}

func TestListWalletsByOwnerUseCaseReturnsWallets(t *testing.T) {
	first := newTestWallet(t, "wallet-001", "owner-001", "account-001")
	second := newTestWallet(t, "wallet-002", "owner-001", "account-002")
	repository := &fakeWalletReader{
		wallets: []domain.Wallet{first, second},
	}
	useCase := NewListWalletsByOwnerUseCase(repository)

	result, err := useCase.Execute(
		context.Background(),
		domain.OwnerID("owner-001"),
	)
	if err != nil {
		t.Fatalf("list wallets: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("unexpected wallet count: got %d, want 2", len(result))
	}
	if repository.listCalls != 1 {
		t.Fatalf("ListByOwnerID() calls = %d, want 1", repository.listCalls)
	}
}

func TestListWalletsByOwnerUseCaseRejectsEmptyOwnerID(t *testing.T) {
	repository := &fakeWalletReader{}
	useCase := NewListWalletsByOwnerUseCase(repository)

	_, err := useCase.Execute(context.Background(), domain.OwnerID(""))
	if !errors.Is(err, domain.ErrInvalidID) {
		t.Fatalf("expected invalid ID error, got %v", err)
	}
	if repository.listCalls != 0 {
		t.Fatal("repository should not be called for an empty owner ID")
	}
}

func TestListWalletsByOwnerUseCasePropagatesRepositoryError(t *testing.T) {
	expectedErr := errors.New("database unavailable")
	repository := &fakeWalletReader{err: expectedErr}
	useCase := NewListWalletsByOwnerUseCase(repository)

	_, err := useCase.Execute(
		context.Background(),
		domain.OwnerID("owner-001"),
	)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected repository error, got %v", err)
	}
}

type fakeWalletReader struct {
	wallets   []domain.Wallet
	wallet    domain.Wallet
	err       error
	getCalls  int
	listCalls int
}

func (f *fakeWalletReader) GetByID(
	_ context.Context,
	_ domain.WalletID,
) (domain.Wallet, error) {
	f.getCalls++
	if f.err != nil {
		return domain.Wallet{}, f.err
	}

	return f.wallet, nil
}

func (f *fakeWalletReader) ListByOwnerID(
	_ context.Context,
	_ domain.OwnerID,
) ([]domain.Wallet, error) {
	f.listCalls++
	if f.err != nil {
		return nil, f.err
	}

	return f.wallets, nil
}

func newTestWallet(
	t *testing.T,
	id string,
	ownerID string,
	ledgerAccountID string,
) domain.Wallet {
	t.Helper()

	currency, err := domain.NewCurrency("BRL")
	if err != nil {
		t.Fatal(err)
	}

	wallet, err := domain.NewWallet(
		domain.WalletID(id),
		domain.OwnerID(ownerID),
		domain.AccountID(ledgerAccountID),
		currency,
	)
	if err != nil {
		t.Fatal(err)
	}

	return wallet
}
