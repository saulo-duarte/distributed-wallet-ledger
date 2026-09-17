//go:build integration

package postgres

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	db "financial-ledger/internal/ledger/adapters/postgres/generated"
	"financial-ledger/internal/ledger/application/wallet"
	"financial-ledger/internal/ledger/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestHoldRepositoryAuthorizesAndReplaysHold(t *testing.T) {
	pool := openIntegrationPool(t)
	queries := db.New(pool)

	walletAccount := newIntegrationAccountWithCode(
		t,
		"0198f3d0-7f0a-7b10-8def-123456789ab0",
		"integration-hold-wallet-account",
	)
	clearingAccount := newIntegrationAccountWithCode(
		t,
		"0198f3d0-7f0a-7b11-8def-123456789ab1",
		"integration-hold-clearing-account",
	)
	foundWallet := newIntegrationWallet(
		t,
		"0198f3d0-7f0a-7b12-8def-123456789ab2",
		"integration-hold-owner",
		walletAccount.ID().String(),
	)

	depositTransactionID := "0198f3d0-7f0a-7b13-8def-123456789ab3"
	holdID := "0198f3d0-7f0a-7b14-8def-123456789ab4"
	captureTransactionID := "0198f3d0-7f0a-7b15-8def-123456789ab5"

	cleanupLedgerTransaction(t, pool, depositTransactionID)
	cleanupLedgerTransaction(t, pool, captureTransactionID)
	cleanupWalletHold(t, pool, holdID)
	cleanupWallet(t, pool, foundWallet.ID().String())
	cleanupAccount(t, pool, walletAccount.Code())
	cleanupAccount(t, pool, clearingAccount.Code())
	t.Cleanup(func() {
		cleanupLedgerTransaction(t, pool, depositTransactionID)
		cleanupLedgerTransaction(t, pool, captureTransactionID)
		cleanupWalletHold(t, pool, holdID)
		cleanupWallet(t, pool, foundWallet.ID().String())
		cleanupAccount(t, pool, walletAccount.Code())
		cleanupAccount(t, pool, clearingAccount.Code())
	})

	accountRepository := NewAccountRepository(queries)
	walletRepository := NewWalletRepository(queries)
	transactionRepository := NewTransactionRepository(pool, queries)
	holdRepository := NewHoldRepository(pool, queries, transactionRepository)
	createHold := wallet.NewCreateHoldUseCase(walletRepository, holdRepository)

	if err := accountRepository.Create(context.Background(), walletAccount); err != nil {
		t.Fatalf("create wallet account: %v", err)
	}
	if err := accountRepository.Create(context.Background(), clearingAccount); err != nil {
		t.Fatalf("create clearing account: %v", err)
	}
	if err := walletRepository.Create(context.Background(), foundWallet); err != nil {
		t.Fatalf("create wallet: %v", err)
	}

	deposit := newIntegrationTransaction(
		t,
		depositTransactionID,
		clearingAccount.ID(),
		walletAccount.ID(),
	)
	if err := transactionRepository.Post(
		context.Background(),
		deposit,
		"integration-hold-deposit",
		"integration-hold-deposit-hash",
	); err != nil {
		t.Fatalf("post deposit: %v", err)
	}

	command := wallet.CreateHoldCommand{
		ID:               domain.HoldID(holdID),
		WalletID:         foundWallet.ID(),
		AmountMinorUnits: 3000,
		ExpiresAt:        time.Now().UTC().Add(time.Hour),
		IdempotencyKey:   "integration-hold-key",
		RequestHash:      "integration-hold-hash",
	}

	createdHold, err := createHold.Execute(context.Background(), command)
	if err != nil {
		t.Fatalf("create hold: %v", err)
	}
	if !createdHold.IsAuthorized() {
		t.Fatal("created hold should be authorized")
	}

	replayedHold, err := createHold.Execute(context.Background(), command)
	if err != nil {
		t.Fatalf("replay hold: %v", err)
	}
	if replayedHold.ID() != createdHold.ID() {
		t.Fatalf("replayed hold ID = %q, want %q", replayedHold.ID(), createdHold.ID())
	}

	snapshot, err := walletRepository.GetLedgerBalance(
		context.Background(),
		foundWallet.LedgerAccountID(),
	)
	if err != nil {
		t.Fatalf("get wallet balance: %v", err)
	}
	if snapshot.ActiveHoldsMinorUnits != 3000 {
		t.Fatalf(
			"active holds = %d, want 3000",
			snapshot.ActiveHoldsMinorUnits,
		)
	}

	balance, err := wallet.CalculateWalletBalance(foundWallet, snapshot)
	if err != nil {
		t.Fatalf("calculate wallet balance: %v", err)
	}
	if balance.LedgerBalanceMinorUnits != 10000 {
		t.Fatalf("ledger balance = %d, want 10000", balance.LedgerBalanceMinorUnits)
	}
	if balance.AvailableBalanceMinorUnits != 7000 {
		t.Fatalf("available balance = %d, want 7000", balance.AvailableBalanceMinorUnits)
	}

	conflictingCommand := command
	conflictingCommand.RequestHash = "different-hash"
	_, err = createHold.Execute(context.Background(), conflictingCommand)
	if !errors.Is(err, wallet.ErrHoldIdempotencyConflict) {
		t.Fatalf("conflicting replay error = %v, want idempotency conflict", err)
	}

	captureHold := wallet.NewCaptureHoldUseCase(
		walletRepository,
		holdRepository,
		holdRepository,
	)
	captureCommand := wallet.CaptureHoldCommand{
		HoldID:              createdHold.ID(),
		SettlementAccountID: clearingAccount.ID(),
		TransactionID:       domain.TransactionID(captureTransactionID),
		JournalEntryID:      domain.JournalEntryID("0198f3d0-7f0a-7b16-8def-123456789ab6"),
		WalletPostingID:     domain.PostingID("0198f3d0-7f0a-7b17-8def-123456789ab7"),
		SettlementPostingID: domain.PostingID("0198f3d0-7f0a-7b18-8def-123456789ab8"),
		Description:         "Capture integration hold",
		IdempotencyKey:      "integration-capture-key",
		RequestHash:         "integration-capture-hash",
	}

	captured, err := captureHold.Execute(context.Background(), captureCommand)
	if err != nil {
		t.Fatalf("capture hold: %v", err)
	}
	if !captured.Hold.IsCaptured() {
		t.Fatal("captured hold should have captured status")
	}

	replayedCapture, err := captureHold.Execute(
		context.Background(),
		captureCommand,
	)
	if err != nil {
		t.Fatalf("replay capture: %v", err)
	}
	if !replayedCapture.Hold.IsCaptured() {
		t.Fatal("replayed capture should remain captured")
	}

	snapshot, err = walletRepository.GetLedgerBalance(
		context.Background(),
		foundWallet.LedgerAccountID(),
	)
	if err != nil {
		t.Fatalf("get balance after capture: %v", err)
	}
	if snapshot.ActiveHoldsMinorUnits != 0 {
		t.Fatalf(
			"active holds after capture = %d, want 0",
			snapshot.ActiveHoldsMinorUnits,
		)
	}
	if snapshot.TotalDebits != 3000 {
		t.Fatalf("wallet debits after capture = %d, want 3000", snapshot.TotalDebits)
	}
}

func TestHoldRepositorySerializesConcurrentAuthorizations(t *testing.T) {
	pool := openIntegrationPool(t)
	queries := db.New(pool)

	walletAccount := newIntegrationAccountWithCode(
		t,
		"0198f3d0-7f0a-7b20-8def-123456789ab0",
		"integration-concurrent-hold-wallet",
	)
	clearingAccount := newIntegrationAccountWithCode(
		t,
		"0198f3d0-7f0a-7b21-8def-123456789ab1",
		"integration-concurrent-hold-clearing",
	)
	foundWallet := newIntegrationWallet(
		t,
		"0198f3d0-7f0a-7b22-8def-123456789ab2",
		"integration-concurrent-hold-owner",
		walletAccount.ID().String(),
	)

	depositTransactionID := "0198f3d0-7f0a-7b23-8def-123456789ab3"
	firstHoldID := "0198f3d0-7f0a-7b24-8def-123456789ab4"
	secondHoldID := "0198f3d0-7f0a-7b25-8def-123456789ab5"

	cleanupLedgerTransaction(t, pool, depositTransactionID)
	cleanupWalletHold(t, pool, firstHoldID)
	cleanupWalletHold(t, pool, secondHoldID)
	cleanupWallet(t, pool, foundWallet.ID().String())
	cleanupAccount(t, pool, walletAccount.Code())
	cleanupAccount(t, pool, clearingAccount.Code())
	t.Cleanup(func() {
		cleanupLedgerTransaction(t, pool, depositTransactionID)
		cleanupWalletHold(t, pool, firstHoldID)
		cleanupWalletHold(t, pool, secondHoldID)
		cleanupWallet(t, pool, foundWallet.ID().String())
		cleanupAccount(t, pool, walletAccount.Code())
		cleanupAccount(t, pool, clearingAccount.Code())
	})

	accountRepository := NewAccountRepository(queries)
	walletRepository := NewWalletRepository(queries)
	transactionRepository := NewTransactionRepository(pool, queries)
	holdRepository := NewHoldRepository(pool, queries, transactionRepository)
	createHold := wallet.NewCreateHoldUseCase(walletRepository, holdRepository)

	if err := accountRepository.Create(context.Background(), walletAccount); err != nil {
		t.Fatalf("create wallet account: %v", err)
	}
	if err := accountRepository.Create(context.Background(), clearingAccount); err != nil {
		t.Fatalf("create clearing account: %v", err)
	}
	if err := walletRepository.Create(context.Background(), foundWallet); err != nil {
		t.Fatalf("create wallet: %v", err)
	}

	deposit := newIntegrationTransaction(
		t,
		depositTransactionID,
		clearingAccount.ID(),
		walletAccount.ID(),
	)
	if err := transactionRepository.Post(
		context.Background(),
		deposit,
		"integration-concurrent-hold-deposit",
		"integration-concurrent-hold-deposit-hash",
	); err != nil {
		t.Fatalf("post deposit: %v", err)
	}

	commands := []wallet.CreateHoldCommand{
		{
			ID:               domain.HoldID(firstHoldID),
			WalletID:         foundWallet.ID(),
			AmountMinorUnits: 7000,
			ExpiresAt:        time.Now().UTC().Add(time.Hour),
			IdempotencyKey:   "integration-concurrent-hold-key-1",
			RequestHash:      "integration-concurrent-hold-hash-1",
		},
		{
			ID:               domain.HoldID(secondHoldID),
			WalletID:         foundWallet.ID(),
			AmountMinorUnits: 7000,
			ExpiresAt:        time.Now().UTC().Add(time.Hour),
			IdempotencyKey:   "integration-concurrent-hold-key-2",
			RequestHash:      "integration-concurrent-hold-hash-2",
		},
	}

	results := make(chan error, len(commands))
	var waitGroup sync.WaitGroup
	waitGroup.Add(len(commands))

	for _, command := range commands {
		command := command
		go func() {
			defer waitGroup.Done()
			_, err := createHold.Execute(context.Background(), command)
			results <- err
		}()
	}

	waitGroup.Wait()
	close(results)

	successes := 0
	insufficientFunds := 0
	for err := range results {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, wallet.ErrInsufficientFunds):
			insufficientFunds++
		default:
			t.Fatalf("unexpected concurrent hold error: %v", err)
		}
	}

	if successes != 1 || insufficientFunds != 1 {
		t.Fatalf(
			"concurrent hold results: successes=%d, insufficient_funds=%d; want 1/1",
			successes,
			insufficientFunds,
		)
	}
}

func cleanupWalletHold(t *testing.T, pool *pgxpool.Pool, holdID string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if _, err := pool.Exec(
		ctx,
		"DELETE FROM wallet_holds WHERE id = $1::uuid",
		holdID,
	); err != nil {
		t.Fatalf("cleanup wallet hold %q: %v", holdID, err)
	}
}
