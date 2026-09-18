package wallet

import (
	"context"
	"errors"
	"testing"

	"financial-ledger/internal/ledger/domain"
)

func TestCalculateWalletBalanceReturnsCreditMinusDebit(t *testing.T) {
	foundWallet := newTestWallet(t, "wallet-001", "owner-001", "account-001")

	balance, err := CalculateWalletBalance(
		foundWallet,
		LedgerBalanceSnapshot{
			TotalDebits:           2500,
			TotalCredits:          10000,
			ActiveHoldsMinorUnits: 1500,
		},
	)
	if err != nil {
		t.Fatalf("calculate wallet balance: %v", err)
	}

	if balance.LedgerBalanceMinorUnits != 7500 {
		t.Fatalf(
			"unexpected ledger balance: got %d, want 7500",
			balance.LedgerBalanceMinorUnits,
		)
	}
	if balance.AvailableBalanceMinorUnits != 6000 {
		t.Fatalf(
			"unexpected available balance: got %d, want 6000",
			balance.AvailableBalanceMinorUnits,
		)
	}
}

func TestCalculateWalletBalanceRejectsNegativeSnapshotTotals(t *testing.T) {
	foundWallet := newTestWallet(t, "wallet-001", "owner-001", "account-001")

	_, err := CalculateWalletBalance(
		foundWallet,
		LedgerBalanceSnapshot{TotalDebits: -1},
	)
	if !errors.Is(err, ErrInvalidBalanceSnapshot) {
		t.Fatalf("expected invalid snapshot error, got %v", err)
	}
}

func TestGetWalletBalanceUseCaseUsesWalletLedgerAccount(t *testing.T) {
	foundWallet := newTestWallet(t, "wallet-001", "owner-001", "account-001")
	wallets := &fakeWalletReader{wallet: foundWallet}
	balances := &fakeWalletBalanceReader{
		snapshot: LedgerBalanceSnapshot{
			TotalDebits:  1000,
			TotalCredits: 5000,
		},
	}
	useCase := NewGetWalletBalanceUseCase(wallets, balances)

	balance, err := useCase.Execute(
		context.Background(),
		foundWallet.ID(),
	)
	if err != nil {
		t.Fatalf("get wallet balance: %v", err)
	}

	if balances.accountID != foundWallet.LedgerAccountID() {
		t.Fatalf(
			"unexpected account ID: got %q, want %q",
			balances.accountID,
			foundWallet.LedgerAccountID(),
		)
	}
	if balance.LedgerBalanceMinorUnits != 4000 {
		t.Fatalf(
			"unexpected balance: got %d, want 4000",
			balance.LedgerBalanceMinorUnits,
		)
	}
}

func TestGetWalletBalanceUseCasePropagatesWalletError(t *testing.T) {
	expectedErr := ErrWalletNotFound
	wallets := &fakeWalletReader{err: expectedErr}
	balances := &fakeWalletBalanceReader{}
	useCase := NewGetWalletBalanceUseCase(wallets, balances)

	_, err := useCase.Execute(
		context.Background(),
		domain.WalletID("wallet-001"),
	)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected wallet error, got %v", err)
	}
	if balances.calls != 0 {
		t.Fatal("balance reader should not be called when wallet lookup fails")
	}
}

func TestGetWalletBalanceUseCasePropagatesBalanceError(t *testing.T) {
	expectedErr := errors.New("balance unavailable")
	foundWallet := newTestWallet(t, "wallet-001", "owner-001", "account-001")
	wallets := &fakeWalletReader{wallet: foundWallet}
	balances := &fakeWalletBalanceReader{err: expectedErr}
	useCase := NewGetWalletBalanceUseCase(wallets, balances)

	_, err := useCase.Execute(
		context.Background(),
		foundWallet.ID(),
	)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected balance error, got %v", err)
	}
}

func TestGetWalletBalanceUseCaseReturnsFromDynamoDB(t *testing.T) {
	foundWallet := newTestWallet(t, "wallet-001", "owner-001", "account-001")
	wallets := &fakeWalletReader{wallet: foundWallet}
	balances := &fakeWalletBalanceReader{}
	projections := &fakeWalletBalanceProjectionRepository{
		savedBalance: WalletBalance{
			WalletID:                   foundWallet.ID(),
			Currency:                   foundWallet.Currency(),
			LedgerBalanceMinorUnits:    9900,
			AvailableBalanceMinorUnits: 9900,
		},
	}

	useCase := NewGetWalletBalanceUseCaseWithProjection(wallets, balances, projections, nil)

	balance, err := useCase.Execute(context.Background(), foundWallet.ID())
	if err != nil {
		t.Fatalf("get wallet balance: %v", err)
	}

	if balance.AvailableBalanceMinorUnits != 9900 {
		t.Fatalf("expected balance from dynamodb 9900, got %d", balance.AvailableBalanceMinorUnits)
	}
	if balances.calls != 0 {
		t.Fatalf("expected 0 calls to postgres balance reader, got %d", balances.calls)
	}
}

func TestGetWalletBalanceUseCaseFallbacksToPostgresWhenDynamoMisses(t *testing.T) {
	foundWallet := newTestWallet(t, "wallet-001", "owner-001", "account-001")
	wallets := &fakeWalletReader{wallet: foundWallet}
	balances := &fakeWalletBalanceReader{
		snapshot: LedgerBalanceSnapshot{
			TotalDebits:  1000,
			TotalCredits: 5000,
		},
	}
	projections := &fakeWalletBalanceProjectionRepository{
		err: ErrWalletNotFound,
	}

	useCase := NewGetWalletBalanceUseCaseWithProjection(wallets, balances, projections, nil)

	balance, err := useCase.Execute(context.Background(), foundWallet.ID())
	if err != nil {
		t.Fatalf("get wallet balance: %v", err)
	}

	if balance.AvailableBalanceMinorUnits != 4000 {
		t.Fatalf("expected balance from postgres 4000, got %d", balance.AvailableBalanceMinorUnits)
	}
	if balances.calls != 1 {
		t.Fatalf("expected 1 call to postgres balance reader, got %d", balances.calls)
	}
	if projections.calls != 1 {
		t.Fatalf("expected 1 call to save warm-up projection in dynamodb, got %d", projections.calls)
	}
}

type fakeWalletBalanceReader struct {
	snapshot  LedgerBalanceSnapshot
	accountID domain.AccountID
	err       error
	calls     int
}

func (f *fakeWalletBalanceReader) GetLedgerBalance(
	_ context.Context,
	accountID domain.AccountID,
) (LedgerBalanceSnapshot, error) {
	f.calls++
	f.accountID = accountID
	if f.err != nil {
		return LedgerBalanceSnapshot{}, f.err
	}

	return f.snapshot, nil
}
