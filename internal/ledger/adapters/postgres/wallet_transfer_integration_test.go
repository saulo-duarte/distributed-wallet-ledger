//go:build integration

package postgres

import (
	"context"
	"errors"
	"sync"
	"testing"

	db "financial-ledger/internal/ledger/adapters/postgres/generated"
	"financial-ledger/internal/ledger/application/transaction"
	"financial-ledger/internal/ledger/application/wallet"
	"financial-ledger/internal/ledger/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestTransferWalletIntegrationPersistsAndReplaysIdempotently(t *testing.T) {
	fixture := newTransferIntegrationFixture(t, "0198f3d0")

	transferID := "0198f3d1-7f0a-7b75-8def-123456789ab4"
	command := newIntegrationTransferCommand(
		fixture.sourceWallet,
		fixture.destinationWallet,
		transferID,
		3000,
	)

	if _, err := fixture.transferWallet.Execute(
		context.Background(),
		command,
	); err != nil {
		t.Fatalf("execute transfer: %v", err)
	}

	if _, err := fixture.transferWallet.Execute(
		context.Background(),
		command,
	); err != nil {
		t.Fatalf("replay transfer: %v", err)
	}

	sourceBalance, err := fixture.walletRepository.GetLedgerBalance(
		context.Background(),
		fixture.sourceWallet.LedgerAccountID(),
	)
	if err != nil {
		t.Fatalf("get source balance: %v", err)
	}

	if sourceBalance.TotalCredits != 10000 ||
		sourceBalance.TotalDebits != 3000 {
		t.Fatalf(
			"unexpected source totals: credits=%d debits=%d",
			sourceBalance.TotalCredits,
			sourceBalance.TotalDebits,
		)
	}

	destinationBalance, err := fixture.walletRepository.GetLedgerBalance(
		context.Background(),
		fixture.destinationWallet.LedgerAccountID(),
	)
	if err != nil {
		t.Fatalf("get destination balance: %v", err)
	}

	if destinationBalance.TotalCredits != 3000 ||
		destinationBalance.TotalDebits != 0 {
		t.Fatalf(
			"unexpected destination totals: credits=%d debits=%d",
			destinationBalance.TotalCredits,
			destinationBalance.TotalDebits,
		)
	}

	var transactionCount int
	if err := fixture.pool.QueryRow(
		context.Background(),
		"SELECT COUNT(*) FROM transactions WHERE id = $1::uuid",
		transferID,
	).Scan(&transactionCount); err != nil {
		t.Fatalf("count transfer transactions: %v", err)
	}
	if transactionCount != 1 {
		t.Fatalf("transfer transaction count = %d, want 1", transactionCount)
	}
}

func TestTransferWalletIntegrationSerializesConcurrentTransfers(t *testing.T) {
	fixture := newTransferIntegrationFixture(t, "0198f3e0")

	commands := []wallet.TransferWalletCommand{
		newIntegrationTransferCommand(
			fixture.sourceWallet,
			fixture.destinationWallet,
			"0198f3e1-7f0a-7b76-8def-123456789ab5",
			7000,
		),
		newIntegrationTransferCommand(
			fixture.sourceWallet,
			fixture.destinationWallet,
			"0198f3e2-7f0a-7b77-8def-123456789ab6",
			7000,
		),
	}

	results := make(chan error, len(commands))
	var waitGroup sync.WaitGroup
	waitGroup.Add(len(commands))

	for _, command := range commands {
		command := command
		go func() {
			defer waitGroup.Done()
			_, err := fixture.transferWallet.Execute(
				context.Background(),
				command,
			)
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
			t.Fatalf("unexpected concurrent transfer error: %v", err)
		}
	}

	if successes != 1 {
		t.Fatalf("successful transfers = %d, want 1", successes)
	}
	if insufficientFunds != 1 {
		t.Fatalf("insufficient-funds transfers = %d, want 1", insufficientFunds)
	}

	sourceBalance, err := fixture.walletRepository.GetLedgerBalance(
		context.Background(),
		fixture.sourceWallet.LedgerAccountID(),
	)
	if err != nil {
		t.Fatalf("get source balance: %v", err)
	}
	if sourceBalance.TotalCredits != 10000 ||
		sourceBalance.TotalDebits != 7000 {
		t.Fatalf(
			"unexpected source totals: credits=%d debits=%d",
			sourceBalance.TotalCredits,
			sourceBalance.TotalDebits,
		)
	}

	destinationBalance, err := fixture.walletRepository.GetLedgerBalance(
		context.Background(),
		fixture.destinationWallet.LedgerAccountID(),
	)
	if err != nil {
		t.Fatalf("get destination balance: %v", err)
	}
	if destinationBalance.TotalCredits != 7000 {
		t.Fatalf(
			"destination credits = %d, want 7000",
			destinationBalance.TotalCredits,
		)
	}
}

type transferIntegrationFixture struct {
	pool              *pgxpool.Pool
	walletRepository  *WalletRepository
	transferWallet    wallet.TransferWalletUseCase
	sourceWallet      domain.Wallet
	destinationWallet domain.Wallet
}

func newTransferIntegrationFixture(
	t *testing.T,
	baseID string,
) transferIntegrationFixture {
	t.Helper()

	pool := openIntegrationPool(t)
	queries := db.New(pool)
	accountRepository := NewAccountRepository(queries)
	walletRepository := NewWalletRepository(queries)
	transactionRepository := NewTransactionRepository(pool, queries)
	postTransaction := transaction.NewPostTransactionUseCase(
		transactionRepository,
	)
	transferWallet := wallet.NewTransferWalletUseCase(
		walletRepository,
		walletRepository,
		postTransaction,
	)

	sourceAccount := newIntegrationAccountWithCode(
		t,
		baseID+"-7f0a-7b70-8def-123456789ab0",
		"integration-transfer-"+baseID+"-source-account",
	)
	destinationAccount := newIntegrationAccountWithCode(
		t,
		baseID+"-7f0a-7b71-8def-123456789ab1",
		"integration-transfer-"+baseID+"-destination-account",
	)
	clearingAccount := newIntegrationAccountWithCode(
		t,
		baseID+"-7f0a-7b72-8def-123456789ab2",
		"integration-transfer-"+baseID+"-clearing-account",
	)
	sourceWallet := newIntegrationWallet(
		t,
		baseID+"-7f0a-7b73-8def-123456789ab3",
		"integration-transfer-"+baseID+"-source-owner",
		sourceAccount.ID().String(),
	)
	destinationWallet := newIntegrationWallet(
		t,
		baseID+"-7f0a-7b74-8def-123456789ab4",
		"integration-transfer-"+baseID+"-destination-owner",
		destinationAccount.ID().String(),
	)
	depositID := baseID + "-7f0a-7b78-8def-123456789ab7"

	cleanupLedgerTransaction(t, pool, depositID)
	cleanupWallet(t, pool, sourceWallet.ID().String())
	cleanupWallet(t, pool, destinationWallet.ID().String())
	cleanupAccount(t, pool, sourceAccount.Code())
	cleanupAccount(t, pool, destinationAccount.Code())
	cleanupAccount(t, pool, clearingAccount.Code())
	t.Cleanup(func() {
		cleanupLedgerTransaction(t, pool, depositID)
		cleanupWallet(t, pool, sourceWallet.ID().String())
		cleanupWallet(t, pool, destinationWallet.ID().String())
		cleanupAccount(t, pool, sourceAccount.Code())
		cleanupAccount(t, pool, destinationAccount.Code())
		cleanupAccount(t, pool, clearingAccount.Code())
	})

	if err := accountRepository.Create(context.Background(), sourceAccount); err != nil {
		t.Fatalf("create source account: %v", err)
	}
	if err := accountRepository.Create(context.Background(), destinationAccount); err != nil {
		t.Fatalf("create destination account: %v", err)
	}
	if err := accountRepository.Create(context.Background(), clearingAccount); err != nil {
		t.Fatalf("create clearing account: %v", err)
	}
	if err := walletRepository.Create(context.Background(), sourceWallet); err != nil {
		t.Fatalf("create source wallet: %v", err)
	}
	if err := walletRepository.Create(context.Background(), destinationWallet); err != nil {
		t.Fatalf("create destination wallet: %v", err)
	}

	deposit := newIntegrationTransaction(
		t,
		depositID,
		clearingAccount.ID(),
		sourceAccount.ID(),
	)
	if err := transactionRepository.Post(
		context.Background(),
		deposit,
		"integration-transfer-"+baseID+"-deposit",
		"integration-transfer-"+baseID+"-deposit-hash",
	); err != nil {
		t.Fatalf("post source deposit: %v", err)
	}

	return transferIntegrationFixture{
		pool:              pool,
		walletRepository:  walletRepository,
		transferWallet:    transferWallet,
		sourceWallet:      sourceWallet,
		destinationWallet: destinationWallet,
	}
}

func newIntegrationTransferCommand(
	sourceWallet domain.Wallet,
	destinationWallet domain.Wallet,
	transactionID string,
	amountMinorUnits int64,
) wallet.TransferWalletCommand {
	return wallet.TransferWalletCommand{
		SourceWalletID:       sourceWallet.ID(),
		DestinationWalletID:  destinationWallet.ID(),
		TransactionID:        domain.TransactionID(transactionID),
		JournalEntryID:       domain.JournalEntryID(transactionID[:8] + "-7f0a-7b79-8def-123456789ab8"),
		SourcePostingID:      domain.PostingID(transactionID[:8] + "-7f0a-7b7a-8def-123456789ab9"),
		DestinationPostingID: domain.PostingID(transactionID[:8] + "-7f0a-7b7b-8def-123456789aba"),
		AmountMinorUnits:     amountMinorUnits,
		Description:          "Integration wallet transfer",
		IdempotencyKey:       "integration-transfer-" + transactionID,
		RequestHash:          "integration-transfer-hash-" + transactionID,
	}
}
