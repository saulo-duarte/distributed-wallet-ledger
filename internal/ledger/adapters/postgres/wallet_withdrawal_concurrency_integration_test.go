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
)

func TestWithdrawWalletIntegrationSerializesConcurrentWithdrawals(t *testing.T) {
	pool := openIntegrationPool(t)
	queries := db.New(pool)

	walletAccount := newIntegrationAccountWithCode(
		t,
		"0198f3c0-7f0a-7b60-8def-123456789ab0",
		"integration-concurrent-withdraw-wallet",
	)
	clearingAccount := newIntegrationAccountWithCode(
		t,
		"0198f3c0-7f0a-7b61-8def-123456789ab1",
		"integration-concurrent-withdraw-clearing",
	)
	foundWallet := newIntegrationWallet(
		t,
		"0198f3c0-7f0a-7b62-8def-123456789ab2",
		"integration-concurrent-withdraw-owner",
		walletAccount.ID().String(),
	)

	depositTransactionID := "0198f3c0-7f0a-7b63-8def-123456789ab3"
	firstWithdrawalID := "0198f3c1-7f0a-7b64-8def-123456789ab4"
	secondWithdrawalID := "0198f3c2-7f0a-7b65-8def-123456789ab5"

	cleanupLedgerTransaction(t, pool, depositTransactionID)
	cleanupLedgerTransaction(t, pool, firstWithdrawalID)
	cleanupLedgerTransaction(t, pool, secondWithdrawalID)
	cleanupWallet(t, pool, foundWallet.ID().String())
	cleanupAccount(t, pool, walletAccount.Code())
	cleanupAccount(t, pool, clearingAccount.Code())
	t.Cleanup(func() {
		cleanupLedgerTransaction(t, pool, depositTransactionID)
		cleanupLedgerTransaction(t, pool, firstWithdrawalID)
		cleanupLedgerTransaction(t, pool, secondWithdrawalID)
		cleanupWallet(t, pool, foundWallet.ID().String())
		cleanupAccount(t, pool, walletAccount.Code())
		cleanupAccount(t, pool, clearingAccount.Code())
	})

	accountRepository := NewAccountRepository(queries)
	walletRepository := NewWalletRepository(queries)
	transactionRepository := NewTransactionRepository(pool, queries)
	postTransaction := transaction.NewPostTransactionUseCase(
		transactionRepository,
	)
	withdrawWallet := wallet.NewWithdrawWalletUseCase(
		walletRepository,
		walletRepository,
		postTransaction,
	)

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
		"integration-concurrent-deposit",
		"integration-concurrent-deposit-hash",
	); err != nil {
		t.Fatalf("post deposit: %v", err)
	}

	commands := []wallet.WithdrawWalletCommand{
		newIntegrationWithdrawCommand(
			foundWallet,
			clearingAccount.ID(),
			firstWithdrawalID,
			7000,
		),
		newIntegrationWithdrawCommand(
			foundWallet,
			clearingAccount.ID(),
			secondWithdrawalID,
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
			_, err := withdrawWallet.Execute(context.Background(), command)
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
			t.Fatalf("unexpected concurrent withdrawal error: %v", err)
		}
	}

	if successes != 1 {
		t.Fatalf("successful withdrawals = %d, want 1", successes)
	}
	if insufficientFunds != 1 {
		t.Fatalf("insufficient-funds withdrawals = %d, want 1", insufficientFunds)
	}

	snapshot, err := walletRepository.GetLedgerBalance(
		context.Background(),
		foundWallet.LedgerAccountID(),
	)
	if err != nil {
		t.Fatalf("get wallet balance: %v", err)
	}
	if snapshot.TotalCredits != 10000 || snapshot.TotalDebits != 7000 {
		t.Fatalf(
			"unexpected wallet totals: credits=%d debits=%d",
			snapshot.TotalCredits,
			snapshot.TotalDebits,
		)
	}
	if snapshot.TotalCredits-snapshot.TotalDebits != 3000 {
		t.Fatalf(
			"final wallet balance = %d, want 3000",
			snapshot.TotalCredits-snapshot.TotalDebits,
		)
	}
}
