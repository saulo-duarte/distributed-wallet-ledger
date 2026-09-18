package wallet

import (
	"context"
	"errors"
	"testing"

	"financial-ledger/internal/ledger/domain"
)

type fakeWalletBalanceProjectionRepository struct {
	savedBalance WalletBalance
	err          error
	calls        int
}

func (f *fakeWalletBalanceProjectionRepository) SaveWalletBalance(
	_ context.Context,
	balance WalletBalance,
) error {
	f.calls++
	f.savedBalance = balance
	return f.err
}

func (f *fakeWalletBalanceProjectionRepository) GetWalletBalance(
	_ context.Context,
	_ domain.WalletID,
) (WalletBalance, error) {
	return f.savedBalance, f.err
}

func TestWalletBalanceProjectorProjectsAndSavesBalance(t *testing.T) {
	w := newTestWallet(t, "wallet-001", "owner-001", "account-001")
	wallets := &fakeWalletReader{wallet: w}
	balances := &fakeWalletBalanceReader{
		snapshot: LedgerBalanceSnapshot{
			TotalDebits:  2000,
			TotalCredits: 10000,
		},
	}
	projections := &fakeWalletBalanceProjectionRepository{}

	projector := NewWalletBalanceProjector(wallets, balances, projections)

	projected, err := projector.ProjectBalance(context.Background(), w.ID())
	if err != nil {
		t.Fatalf("project balance: %v", err)
	}

	if projected.LedgerBalanceMinorUnits != 8000 {
		t.Fatalf("expected ledger balance 8000, got %d", projected.LedgerBalanceMinorUnits)
	}

	if projected.AvailableBalanceMinorUnits != 8000 {
		t.Fatalf("expected available balance 8000, got %d", projected.AvailableBalanceMinorUnits)
	}

	if projections.calls != 1 {
		t.Fatalf("expected 1 call to SaveWalletBalance, got %d", projections.calls)
	}

	if projections.savedBalance.LedgerBalanceMinorUnits != 8000 {
		t.Fatalf("expected saved balance 8000, got %d", projections.savedBalance.LedgerBalanceMinorUnits)
	}
}

func TestWalletBalanceProjectorRejectsZeroWalletID(t *testing.T) {
	projector := NewWalletBalanceProjector(nil, nil, nil)

	_, err := projector.ProjectBalance(context.Background(), "")
	if !errors.Is(err, domain.ErrInvalidID) {
		t.Fatalf("expected ErrInvalidID, got %v", err)
	}
}

func TestWalletBalanceProjectorPropagatesSaveError(t *testing.T) {
	w := newTestWallet(t, "wallet-001", "owner-001", "account-001")
	wallets := &fakeWalletReader{wallet: w}
	balances := &fakeWalletBalanceReader{
		snapshot: LedgerBalanceSnapshot{
			TotalDebits:  1000,
			TotalCredits: 5000,
		},
	}
	expectedErr := errors.New("dynamo failure")
	projections := &fakeWalletBalanceProjectionRepository{err: expectedErr}

	projector := NewWalletBalanceProjector(wallets, balances, projections)

	_, err := projector.ProjectBalance(context.Background(), w.ID())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected %v, got %v", expectedErr, err)
	}
}
