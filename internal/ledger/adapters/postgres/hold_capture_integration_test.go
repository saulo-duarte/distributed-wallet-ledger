//go:build integration

package postgres

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	db "financial-ledger/internal/ledger/adapters/postgres/generated"
	"financial-ledger/internal/ledger/application/transaction"
	"financial-ledger/internal/ledger/application/wallet"
	"financial-ledger/internal/ledger/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestCaptureHoldSerializesConcurrentCaptures(t *testing.T) {
	fixture := newHoldCaptureFixture(t, "0198f3e4")

	firstTransactionID := "0198f3e5-7f0a-7b20-8def-123456789ab0"
	secondTransactionID := "0198f3e6-7f0a-7b21-8def-123456789ab1"
	t.Cleanup(func() {
		cleanupLedgerTransaction(t, fixture.pool, firstTransactionID)
		cleanupLedgerTransaction(t, fixture.pool, secondTransactionID)
	})
	commands := []wallet.CaptureHoldCommand{
		fixture.captureCommand(firstTransactionID, "capture-key-1", "capture-hash-1"),
		fixture.captureCommand(secondTransactionID, "capture-key-2", "capture-hash-2"),
	}

	results := make(chan error, len(commands))
	var waitGroup sync.WaitGroup
	waitGroup.Add(len(commands))

	for _, command := range commands {
		command := command
		go func() {
			defer waitGroup.Done()
			_, err := fixture.captureHold.Execute(context.Background(), command)
			results <- err
		}()
	}

	waitGroup.Wait()
	close(results)

	successes := 0
	notAuthorized := 0
	for err := range results {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, domain.ErrHoldNotAuthorized):
			notAuthorized++
		default:
			t.Fatalf("unexpected concurrent capture error: %v", err)
		}
	}

	if successes != 1 || notAuthorized != 1 {
		t.Fatalf(
			"capture results: successes=%d, not_authorized=%d; want 1/1",
			successes,
			notAuthorized,
		)
	}

	var transactionCount int
	if err := fixture.pool.QueryRow(
		context.Background(),
		`SELECT COUNT(*)
         FROM transactions
         WHERE id IN ($1::uuid, $2::uuid)`,
		firstTransactionID,
		secondTransactionID,
	).Scan(&transactionCount); err != nil {
		t.Fatalf("count concurrent captures: %v", err)
	}
	if transactionCount != 1 {
		t.Fatalf("capture transaction count = %d, want 1", transactionCount)
	}
}

func TestCaptureHoldRollsBackLedgerAndHoldOnPostingFailure(t *testing.T) {
	fixture := newHoldCaptureFixture(t, "0198f3e7")
	captureTransactionID := "0198f3e8-7f0a-7b22-8def-123456789ab2"
	t.Cleanup(func() {
		cleanupLedgerTransaction(t, fixture.pool, captureTransactionID)
	})

	command := fixture.captureCommand(
		captureTransactionID,
		"capture-rollback-key",
		"capture-rollback-hash",
	)
	command.SettlementAccountID = domain.AccountID(
		"0198f3e9-7f0a-7b23-8def-123456789ab3",
	)

	_, err := fixture.captureHold.Execute(context.Background(), command)
	if err == nil {
		t.Fatal("expected capture to fail for an unknown settlement account")
	}

	foundHold, err := fixture.holdRepository.GetByID(
		context.Background(),
		fixture.hold.ID(),
	)
	if err != nil {
		t.Fatalf("get hold after failed capture: %v", err)
	}
	if !foundHold.IsAuthorized() {
		t.Fatal("failed capture must leave hold authorized")
	}

	snapshot, err := fixture.walletRepository.GetLedgerBalance(
		context.Background(),
		fixture.wallet.LedgerAccountID(),
	)
	if err != nil {
		t.Fatalf("get balance after failed capture: %v", err)
	}
	if snapshot.TotalDebits != 0 {
		t.Fatalf("wallet debits after failed capture = %d, want 0", snapshot.TotalDebits)
	}
	if snapshot.ActiveHoldsMinorUnits != fixture.hold.Amount().AmountMinorUnits() {
		t.Fatalf(
			"active holds after failed capture = %d, want %d",
			snapshot.ActiveHoldsMinorUnits,
			fixture.hold.Amount().AmountMinorUnits(),
		)
	}

	var transactionCount int
	if err := fixture.pool.QueryRow(
		context.Background(),
		"SELECT COUNT(*) FROM transactions WHERE id = $1::uuid",
		captureTransactionID,
	).Scan(&transactionCount); err != nil {
		t.Fatalf("count failed capture transaction: %v", err)
	}
	if transactionCount != 0 {
		t.Fatalf("failed capture transaction count = %d, want 0", transactionCount)
	}
}

func TestCaptureHoldRejectsIdempotencyConflict(t *testing.T) {
	fixture := newHoldCaptureFixture(t, "0198f3ea")
	captureTransactionID := "0198f3eb-7f0a-7b24-8def-123456789ab4"
	t.Cleanup(func() {
		cleanupLedgerTransaction(t, fixture.pool, captureTransactionID)
	})
	command := fixture.captureCommand(
		captureTransactionID,
		"capture-conflict-key",
		"capture-conflict-hash",
	)

	if _, err := fixture.captureHold.Execute(context.Background(), command); err != nil {
		t.Fatalf("capture hold: %v", err)
	}

	conflictingCommand := command
	conflictingCommand.RequestHash = "different-capture-hash"
	_, err := fixture.captureHold.Execute(
		context.Background(),
		conflictingCommand,
	)
	if !errors.Is(err, transaction.ErrIdempotencyKeyConflict) {
		t.Fatalf("conflicting capture error = %v, want idempotency conflict", err)
	}
}

func TestReleaseHoldRestoresAvailableBalance(t *testing.T) {
	fixture := newHoldCaptureFixture(t, "0198f3ec")
	releaseHold := wallet.NewReleaseHoldUseCase(fixture.holdRepository)

	if _, err := releaseHold.Execute(
		context.Background(),
		fixture.hold.ID(),
	); err != nil {
		t.Fatalf("release hold: %v", err)
	}

	snapshot, err := fixture.walletRepository.GetLedgerBalance(
		context.Background(),
		fixture.wallet.LedgerAccountID(),
	)
	if err != nil {
		t.Fatalf("get balance after release: %v", err)
	}
	if snapshot.ActiveHoldsMinorUnits != 0 {
		t.Fatalf("active holds after release = %d, want 0", snapshot.ActiveHoldsMinorUnits)
	}

	balance, err := wallet.CalculateWalletBalance(fixture.wallet, snapshot)
	if err != nil {
		t.Fatalf("calculate balance after release: %v", err)
	}
	if balance.AvailableBalanceMinorUnits != 10000 {
		t.Fatalf("available balance after release = %d, want 10000", balance.AvailableBalanceMinorUnits)
	}
}

func TestExpireHoldRestoresAvailableBalance(t *testing.T) {
	fixture := newHoldCaptureFixture(t, "0198f3ed")
	if _, err := fixture.pool.Exec(
		context.Background(),
		`UPDATE wallet_holds
         SET created_at = NOW() - INTERVAL '2 seconds',
             expires_at = NOW() - INTERVAL '1 second'
         WHERE id = $1::uuid`,
		fixture.hold.ID().String(),
	); err != nil {
		t.Fatalf("expire hold fixture: %v", err)
	}

	expireHold := wallet.NewExpireHoldUseCase(fixture.holdRepository)
	if _, err := expireHold.Execute(
		context.Background(),
		fixture.hold.ID(),
	); err != nil {
		t.Fatalf("expire hold: %v", err)
	}

	snapshot, err := fixture.walletRepository.GetLedgerBalance(
		context.Background(),
		fixture.wallet.LedgerAccountID(),
	)
	if err != nil {
		t.Fatalf("get balance after expiration: %v", err)
	}
	if snapshot.ActiveHoldsMinorUnits != 0 {
		t.Fatalf("active holds after expiration = %d, want 0", snapshot.ActiveHoldsMinorUnits)
	}
}

type holdCaptureFixture struct {
	pool                 *pgxpool.Pool
	walletRepository     *WalletRepository
	holdRepository       *HoldRepository
	captureHold          wallet.CaptureHoldUseCase
	wallet               domain.Wallet
	settlementAccount    domain.Account
	hold                 domain.Hold
	depositTransactionID string
}

func newHoldCaptureFixture(
	t *testing.T,
	baseID string,
) holdCaptureFixture {
	t.Helper()

	pool := openIntegrationPool(t)
	queries := db.New(pool)
	accountRepository := NewAccountRepository(queries)
	walletRepository := NewWalletRepository(queries)
	transactionRepository := NewTransactionRepository(pool, queries)
	holdRepository := NewHoldRepository(pool, queries, transactionRepository)

	walletAccount := newIntegrationAccountWithCode(
		t,
		baseID+"-7f0a-7b10-8def-123456789ab0",
		"integration-capture-"+baseID+"-wallet-account",
	)
	settlementAccount := newIntegrationAccountWithCode(
		t,
		baseID+"-7f0a-7b11-8def-123456789ab1",
		"integration-capture-"+baseID+"-settlement-account",
	)
	foundWallet := newIntegrationWallet(
		t,
		baseID+"-7f0a-7b12-8def-123456789ab2",
		"integration-capture-"+baseID+"-owner",
		walletAccount.ID().String(),
	)
	depositTransactionID := baseID + "-7f0a-7b13-8def-123456789ab3"
	holdID := baseID + "-7f0a-7b14-8def-123456789ab4"

	cleanupLedgerTransaction(t, pool, depositTransactionID)
	cleanupWalletHold(t, pool, holdID)
	cleanupWallet(t, pool, foundWallet.ID().String())
	cleanupAccount(t, pool, walletAccount.Code())
	cleanupAccount(t, pool, settlementAccount.Code())
	t.Cleanup(func() {
		cleanupLedgerTransaction(t, pool, depositTransactionID)
		cleanupWalletHold(t, pool, holdID)
		cleanupWallet(t, pool, foundWallet.ID().String())
		cleanupAccount(t, pool, walletAccount.Code())
		cleanupAccount(t, pool, settlementAccount.Code())
	})

	if err := accountRepository.Create(context.Background(), walletAccount); err != nil {
		t.Fatalf("create wallet account: %v", err)
	}
	if err := accountRepository.Create(context.Background(), settlementAccount); err != nil {
		t.Fatalf("create settlement account: %v", err)
	}
	if err := walletRepository.Create(context.Background(), foundWallet); err != nil {
		t.Fatalf("create wallet: %v", err)
	}

	deposit := newIntegrationTransaction(
		t,
		depositTransactionID,
		settlementAccount.ID(),
		walletAccount.ID(),
	)
	if err := transactionRepository.Post(
		context.Background(),
		deposit,
		"integration-capture-"+baseID+"-deposit-key",
		"integration-capture-"+baseID+"-deposit-hash",
	); err != nil {
		t.Fatalf("post deposit: %v", err)
	}

	createHold := wallet.NewCreateHoldUseCase(walletRepository, holdRepository)
	hold, err := createHold.Execute(
		context.Background(),
		wallet.CreateHoldCommand{
			ID:               domain.HoldID(holdID),
			WalletID:         foundWallet.ID(),
			AmountMinorUnits: 3000,
			ExpiresAt:        time.Now().UTC().Add(time.Hour),
			IdempotencyKey:   "integration-capture-" + baseID + "-hold-key",
			RequestHash:      "integration-capture-" + baseID + "-hold-hash",
		},
	)
	if err != nil {
		t.Fatalf("create hold: %v", err)
	}

	return holdCaptureFixture{
		pool:                 pool,
		walletRepository:     walletRepository,
		holdRepository:       holdRepository,
		captureHold:          wallet.NewCaptureHoldUseCase(walletRepository, holdRepository, holdRepository),
		wallet:               foundWallet,
		settlementAccount:    settlementAccount,
		hold:                 hold,
		depositTransactionID: depositTransactionID,
	}
}

func (f holdCaptureFixture) captureCommand(
	transactionID string,
	idempotencyKey string,
	requestHash string,
) wallet.CaptureHoldCommand {
	return wallet.CaptureHoldCommand{
		HoldID:              f.hold.ID(),
		SettlementAccountID: f.settlementAccount.ID(),
		TransactionID:       domain.TransactionID(transactionID),
		JournalEntryID:      domain.JournalEntryID(transactionID[:8] + "-7f0a-7b30-8def-123456789ab5"),
		WalletPostingID:     domain.PostingID(transactionID[:8] + "-7f0a-7b31-8def-123456789ab6"),
		SettlementPostingID: domain.PostingID(transactionID[:8] + "-7f0a-7b32-8def-123456789ab7"),
		Description:         "Integration hold capture",
		IdempotencyKey:      idempotencyKey,
		RequestHash:         requestHash,
	}
}
