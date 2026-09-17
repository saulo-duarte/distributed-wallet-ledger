//go:build integration

package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	db "financial-ledger/internal/ledger/adapters/postgres/generated"
	"financial-ledger/internal/ledger/application/wallet"
	"financial-ledger/internal/ledger/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestWalletRepositoryCreatePersistsWallet(t *testing.T) {
	pool := openIntegrationPool(t)
	queries := db.New(pool)
	repository := NewWalletRepository(queries)
	accountRepository := NewAccountRepository(queries)

	account := newIntegrationAccountWithCode(
		t,
		"0198f3b2-7f0a-7b10-8def-123456789ab0",
		"integration-wallet-account",
	)
	wallet := newIntegrationWallet(
		t,
		"0198f3b2-7f0a-7b11-8def-123456789ab1",
		"owner-wallet-001",
		account.ID().String(),
	)

	cleanupWallet(t, pool, wallet.ID().String())
	cleanupAccount(t, pool, account.Code())
	t.Cleanup(func() {
		cleanupWallet(t, pool, wallet.ID().String())
		cleanupAccount(t, pool, account.Code())
	})

	if err := accountRepository.Create(context.Background(), account); err != nil {
		t.Fatalf("create ledger account: %v", err)
	}

	if err := repository.Create(context.Background(), wallet); err != nil {
		t.Fatalf("create wallet: %v", err)
	}

	walletID, err := walletIDToUUID(wallet.ID())
	if err != nil {
		t.Fatalf("convert wallet ID: %v", err)
	}

	persisted, err := queries.GetWalletByID(context.Background(), walletID)
	if err != nil {
		t.Fatalf("get persisted wallet: %v", err)
	}

	if persisted.ID.String() != wallet.ID().String() {
		t.Fatalf("unexpected persisted ID: got %q, want %q", persisted.ID.String(), wallet.ID())
	}
	if persisted.OwnerID != wallet.OwnerID().String() {
		t.Fatalf("unexpected owner ID: got %q, want %q", persisted.OwnerID, wallet.OwnerID())
	}
	if persisted.LedgerAccountID.String() != wallet.LedgerAccountID().String() {
		t.Fatalf(
			"unexpected ledger account ID: got %q, want %q",
			persisted.LedgerAccountID.String(),
			wallet.LedgerAccountID(),
		)
	}
	if persisted.Currency != wallet.Currency().String() {
		t.Fatalf("unexpected currency: got %q, want %q", persisted.Currency, wallet.Currency())
	}
	if persisted.Status != string(domain.WalletStatusOpen) {
		t.Fatalf("unexpected status: got %q, want %q", persisted.Status, domain.WalletStatusOpen)
	}
}

func TestWalletRepositoryCreateRejectsDuplicateLedgerAccount(t *testing.T) {
	pool := openIntegrationPool(t)
	queries := db.New(pool)
	repository := NewWalletRepository(queries)
	accountRepository := NewAccountRepository(queries)

	account := newIntegrationAccountWithCode(
		t,
		"0198f3b2-7f0a-7b12-8def-123456789ab2",
		"integration-wallet-duplicate-account",
	)
	first := newIntegrationWallet(
		t,
		"0198f3b2-7f0a-7b13-8def-123456789ab3",
		"owner-wallet-002",
		account.ID().String(),
	)
	second := newIntegrationWallet(
		t,
		"0198f3b2-7f0a-7b14-8def-123456789ab4",
		"owner-wallet-003",
		account.ID().String(),
	)

	cleanupWallet(t, pool, first.ID().String())
	cleanupWallet(t, pool, second.ID().String())
	cleanupAccount(t, pool, account.Code())
	t.Cleanup(func() {
		cleanupWallet(t, pool, first.ID().String())
		cleanupWallet(t, pool, second.ID().String())
		cleanupAccount(t, pool, account.Code())
	})

	if err := accountRepository.Create(context.Background(), account); err != nil {
		t.Fatalf("create ledger account: %v", err)
	}
	if err := repository.Create(context.Background(), first); err != nil {
		t.Fatalf("create first wallet: %v", err)
	}

	if err := repository.Create(context.Background(), second); err == nil {
		t.Fatal("expected duplicate ledger account error")
	}
}

func TestWalletRepositoryCreateRejectsInvalidUUID(t *testing.T) {
	pool := openIntegrationPool(t)
	queries := db.New(pool)
	repository := NewWalletRepository(queries)

	wallet := newIntegrationWallet(
		t,
		"wallet-invalid-uuid",
		"owner-wallet-invalid",
		"0198f3b2-7f0a-7b15-8def-123456789ab5",
	)

	if err := repository.Create(context.Background(), wallet); err == nil {
		t.Fatal("expected invalid UUID error")
	}
}

func TestWalletRepositoryCreatePropagatesCanceledContext(t *testing.T) {
	pool := openIntegrationPool(t)
	queries := db.New(pool)
	repository := NewWalletRepository(queries)

	wallet := newIntegrationWallet(
		t,
		"0198f3b2-7f0a-7b16-8def-123456789ab6",
		"owner-wallet-canceled",
		"0198f3b2-7f0a-7b17-8def-123456789ab7",
	)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := repository.Create(ctx, wallet); err == nil {
		t.Fatal("expected canceled context error")
	}
}

func TestWalletRepositoryGetByIDAndListByOwner(t *testing.T) {
	pool := openIntegrationPool(t)
	queries := db.New(pool)
	repository := NewWalletRepository(queries)
	accountRepository := NewAccountRepository(queries)

	firstAccount := newIntegrationAccountWithCode(
		t,
		"0198f3b2-7f0a-7b18-8def-123456789ab8",
		"integration-wallet-read-first-account",
	)
	secondAccount := newIntegrationAccountWithCode(
		t,
		"0198f3b2-7f0a-7b19-8def-123456789ab9",
		"integration-wallet-read-second-account",
	)
	firstWallet := newIntegrationWallet(
		t,
		"0198f3b2-7f0a-7b1a-8def-123456789aba",
		"owner-wallet-read",
		firstAccount.ID().String(),
	)
	secondWallet := newIntegrationWallet(
		t,
		"0198f3b2-7f0a-7b1b-8def-123456789abb",
		"owner-wallet-read",
		secondAccount.ID().String(),
	)

	cleanupWallet(t, pool, firstWallet.ID().String())
	cleanupWallet(t, pool, secondWallet.ID().String())
	cleanupAccount(t, pool, firstAccount.Code())
	cleanupAccount(t, pool, secondAccount.Code())
	t.Cleanup(func() {
		cleanupWallet(t, pool, firstWallet.ID().String())
		cleanupWallet(t, pool, secondWallet.ID().String())
		cleanupAccount(t, pool, firstAccount.Code())
		cleanupAccount(t, pool, secondAccount.Code())
	})

	if err := accountRepository.Create(context.Background(), firstAccount); err != nil {
		t.Fatalf("create first ledger account: %v", err)
	}
	if err := accountRepository.Create(context.Background(), secondAccount); err != nil {
		t.Fatalf("create second ledger account: %v", err)
	}
	if err := repository.Create(context.Background(), firstWallet); err != nil {
		t.Fatalf("create first wallet: %v", err)
	}
	time.Sleep(5 * time.Millisecond)
	if err := repository.Create(context.Background(), secondWallet); err != nil {
		t.Fatalf("create second wallet: %v", err)
	}

	found, err := repository.GetByID(
		context.Background(),
		firstWallet.ID(),
	)
	if err != nil {
		t.Fatalf("get wallet by ID: %v", err)
	}
	if found.ID() != firstWallet.ID() {
		t.Fatalf("unexpected wallet ID: got %q, want %q", found.ID(), firstWallet.ID())
	}
	if found.OwnerID() != firstWallet.OwnerID() {
		t.Fatalf("unexpected owner ID: got %q, want %q", found.OwnerID(), firstWallet.OwnerID())
	}
	if !found.IsOpen() {
		t.Fatal("expected persisted wallet to be open")
	}

	wallets, err := repository.ListByOwnerID(
		context.Background(),
		domain.OwnerID("owner-wallet-read"),
	)
	if err != nil {
		t.Fatalf("list wallets by owner: %v", err)
	}
	if len(wallets) != 2 {
		t.Fatalf("unexpected wallet count: got %d, want 2", len(wallets))
	}
	if wallets[0].ID() != secondWallet.ID() {
		t.Fatalf(
			"unexpected newest wallet: got %q, want %q",
			wallets[0].ID(),
			secondWallet.ID(),
		)
	}
}

func TestWalletRepositoryGetByIDReturnsNotFound(t *testing.T) {
	pool := openIntegrationPool(t)
	queries := db.New(pool)
	repository := NewWalletRepository(queries)

	walletID, err := domain.NewWalletID(
		"0198f3b2-7f0a-7b1c-8def-123456789abc",
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = repository.GetByID(context.Background(), walletID)
	if !errors.Is(err, wallet.ErrWalletNotFound) {
		t.Fatalf("expected wallet not found error, got %v", err)
	}
}

func newIntegrationWallet(
	t *testing.T,
	id string,
	ownerID string,
	ledgerAccountID string,
) domain.Wallet {
	t.Helper()

	walletID, err := domain.NewWalletID(id)
	if err != nil {
		t.Fatal(err)
	}

	domainOwnerID, err := domain.NewOwnerID(ownerID)
	if err != nil {
		t.Fatal(err)
	}

	accountID, err := domain.NewAccountID(ledgerAccountID)
	if err != nil {
		t.Fatal(err)
	}

	currency, err := domain.NewCurrency("BRL")
	if err != nil {
		t.Fatal(err)
	}

	wallet, err := domain.NewWallet(
		walletID,
		domainOwnerID,
		accountID,
		currency,
	)
	if err != nil {
		t.Fatalf("create domain wallet: %v", err)
	}

	return wallet
}

func cleanupWallet(t *testing.T, pool *pgxpool.Pool, walletID string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if _, err := pool.Exec(ctx, "DELETE FROM wallets WHERE id = $1::uuid", walletID); err != nil {
		t.Fatalf("cleanup wallet %q: %v", walletID, err)
	}
}
