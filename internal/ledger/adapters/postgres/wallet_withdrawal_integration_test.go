//go:build integration

package postgres

import (
	"context"
	"testing"

	db "financial-ledger/internal/ledger/adapters/postgres/generated"
	"financial-ledger/internal/ledger/application/transaction"
	"financial-ledger/internal/ledger/application/wallet"
	"financial-ledger/internal/ledger/domain"
)

func TestWithdrawWalletIntegrationPersistsPostingsAndUpdatesBalance(t *testing.T) {
	pool := openIntegrationPool(t)
	queries := db.New(pool)

	walletAccount := newIntegrationAccountWithCode(
		t,
		"0198f3b2-7f0a-7b30-8def-123456789ab0",
		"integration-withdraw-wallet-account",
	)
	clearingAccount := newIntegrationAccountWithCode(
		t,
		"0198f3b2-7f0a-7b31-8def-123456789ab1",
		"integration-withdraw-clearing-account",
	)
	foundWallet := newIntegrationWallet(
		t,
		"0198f3b2-7f0a-7b32-8def-123456789ab2",
		"integration-withdraw-owner",
		walletAccount.ID().String(),
	)

	depositTransactionID := "0198f3b2-7f0a-7b33-8def-123456789ab3"
	withdrawTransactionID := "0198f3b2-7f0a-7b34-8def-123456789ab4"

	cleanupLedgerTransaction(t, pool, depositTransactionID)
	cleanupLedgerTransaction(t, pool, withdrawTransactionID)
	cleanupWallet(t, pool, foundWallet.ID().String())
	cleanupAccount(t, pool, walletAccount.Code())
	cleanupAccount(t, pool, clearingAccount.Code())
	t.Cleanup(func() {
		cleanupLedgerTransaction(t, pool, depositTransactionID)
		cleanupLedgerTransaction(t, pool, withdrawTransactionID)
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
		"integration-withdraw-deposit",
		"integration-withdraw-deposit-hash",
	); err != nil {
		t.Fatalf("post deposit: %v", err)
	}

	withdrawal, err := withdrawWallet.Execute(
		context.Background(),
		newIntegrationWithdrawCommand(
			foundWallet,
			clearingAccount.ID(),
			withdrawTransactionID,
			3000,
		),
	)
	if err != nil {
		t.Fatalf("execute withdrawal: %v", err)
	}

	if withdrawal.ID().String() != withdrawTransactionID {
		t.Fatalf(
			"unexpected withdrawal transaction ID: got %q, want %q",
			withdrawal.ID(),
			withdrawTransactionID,
		)
	}

	snapshot, err := walletRepository.GetLedgerBalance(
		context.Background(),
		foundWallet.LedgerAccountID(),
	)
	if err != nil {
		t.Fatalf("get wallet balance: %v", err)
	}

	if snapshot.TotalCredits != 10000 {
		t.Fatalf("wallet credits = %d, want 10000", snapshot.TotalCredits)
	}
	if snapshot.TotalDebits != 3000 {
		t.Fatalf("wallet debits = %d, want 3000", snapshot.TotalDebits)
	}
	if snapshot.TotalCredits-snapshot.TotalDebits != 7000 {
		t.Fatalf(
			"wallet balance = %d, want 7000",
			snapshot.TotalCredits-snapshot.TotalDebits,
		)
	}

	postings := withdrawal.JournalEntry().Postings()
	if postings[0].AccountID() != foundWallet.LedgerAccountID() {
		t.Fatalf("withdrawal debit account = %q, want %q", postings[0].AccountID(), foundWallet.LedgerAccountID())
	}
	if postings[0].Direction() != domain.PostingDirectionDebit {
		t.Fatalf("withdrawal first direction = %q, want debit", postings[0].Direction())
	}
	if postings[1].AccountID() != clearingAccount.ID() {
		t.Fatalf("withdrawal credit account = %q, want %q", postings[1].AccountID(), clearingAccount.ID())
	}
	if postings[1].Direction() != domain.PostingDirectionCredit {
		t.Fatalf("withdrawal second direction = %q, want credit", postings[1].Direction())
	}
}

func TestWithdrawWalletIntegrationReplaysSameIdempotentRequest(t *testing.T) {
	pool := openIntegrationPool(t)
	queries := db.New(pool)

	walletAccount := newIntegrationAccountWithCode(
		t,
		"0198f3b2-7f0a-7b40-8def-123456789ab0",
		"integration-idempotent-withdraw-wallet",
	)
	clearingAccount := newIntegrationAccountWithCode(
		t,
		"0198f3b2-7f0a-7b41-8def-123456789ab1",
		"integration-idempotent-withdraw-clearing",
	)
	foundWallet := newIntegrationWallet(
		t,
		"0198f3b2-7f0a-7b42-8def-123456789ab2",
		"integration-idempotent-withdraw-owner",
		walletAccount.ID().String(),
	)

	depositTransactionID := "0198f3b2-7f0a-7b43-8def-123456789ab3"
	withdrawTransactionID := "0198f3b2-7f0a-7b44-8def-123456789ab4"

	cleanupLedgerTransaction(t, pool, depositTransactionID)
	cleanupLedgerTransaction(t, pool, withdrawTransactionID)
	cleanupWallet(t, pool, foundWallet.ID().String())
	cleanupAccount(t, pool, walletAccount.Code())
	cleanupAccount(t, pool, clearingAccount.Code())
	t.Cleanup(func() {
		cleanupLedgerTransaction(t, pool, depositTransactionID)
		cleanupLedgerTransaction(t, pool, withdrawTransactionID)
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
		"integration-idempotent-deposit",
		"integration-idempotent-deposit-hash",
	); err != nil {
		t.Fatalf("post deposit: %v", err)
	}

	command := newIntegrationWithdrawCommand(
		foundWallet,
		clearingAccount.ID(),
		withdrawTransactionID,
		3000,
	)

	if _, err := withdrawWallet.Execute(context.Background(), command); err != nil {
		t.Fatalf("execute first withdrawal: %v", err)
	}

	if _, err := withdrawWallet.Execute(context.Background(), command); err != nil {
		t.Fatalf("replay withdrawal: %v", err)
	}

	snapshot, err := walletRepository.GetLedgerBalance(
		context.Background(),
		foundWallet.LedgerAccountID(),
	)
	if err != nil {
		t.Fatalf("get wallet balance: %v", err)
	}

	if snapshot.TotalDebits != 3000 {
		t.Fatalf(
			"wallet debits after replay = %d, want 3000",
			snapshot.TotalDebits,
		)
	}

	var transactionCount int
	if err := pool.QueryRow(
		context.Background(),
		"SELECT COUNT(*) FROM transactions WHERE id = $1::uuid",
		withdrawTransactionID,
	).Scan(&transactionCount); err != nil {
		t.Fatalf("count replayed withdrawal: %v", err)
	}
	if transactionCount != 1 {
		t.Fatalf("withdrawal transaction count = %d, want 1", transactionCount)
	}
}

func TestWithdrawWalletIntegrationRollsBackWhenClearingAccountIsInvalid(t *testing.T) {
	pool := openIntegrationPool(t)
	queries := db.New(pool)

	walletAccount := newIntegrationAccountWithCode(
		t,
		"0198f3b2-7f0a-7b50-8def-123456789ab0",
		"integration-rollback-withdraw-wallet",
	)
	clearingAccount := newIntegrationAccountWithCode(
		t,
		"0198f3b2-7f0a-7b51-8def-123456789ab1",
		"integration-rollback-withdraw-clearing",
	)
	foundWallet := newIntegrationWallet(
		t,
		"0198f3b2-7f0a-7b52-8def-123456789ab2",
		"integration-rollback-withdraw-owner",
		walletAccount.ID().String(),
	)

	depositTransactionID := "0198f3b2-7f0a-7b53-8def-123456789ab3"
	withdrawTransactionID := "0198f3b2-7f0a-7b54-8def-123456789ab4"

	cleanupLedgerTransaction(t, pool, depositTransactionID)
	cleanupLedgerTransaction(t, pool, withdrawTransactionID)
	cleanupWallet(t, pool, foundWallet.ID().String())
	cleanupAccount(t, pool, walletAccount.Code())
	cleanupAccount(t, pool, clearingAccount.Code())
	t.Cleanup(func() {
		cleanupLedgerTransaction(t, pool, depositTransactionID)
		cleanupLedgerTransaction(t, pool, withdrawTransactionID)
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
		"integration-rollback-deposit",
		"integration-rollback-deposit-hash",
	); err != nil {
		t.Fatalf("post deposit: %v", err)
	}

	missingClearingAccount := domain.AccountID(
		"0198f3b2-7f0a-7b5f-8def-123456789ab5",
	)
	_, err := withdrawWallet.Execute(
		context.Background(),
		newIntegrationWithdrawCommand(
			foundWallet,
			missingClearingAccount,
			withdrawTransactionID,
			3000,
		),
	)
	if err == nil {
		t.Fatal("expected withdrawal to fail for missing clearing account")
	}

	snapshot, err := walletRepository.GetLedgerBalance(
		context.Background(),
		foundWallet.LedgerAccountID(),
	)
	if err != nil {
		t.Fatalf("get wallet balance: %v", err)
	}
	if snapshot.TotalDebits != 0 || snapshot.TotalCredits != 10000 {
		t.Fatalf(
			"wallet balance changed after failed withdrawal: debits=%d credits=%d",
			snapshot.TotalDebits,
			snapshot.TotalCredits,
		)
	}

	var transactionCount int
	if err := pool.QueryRow(
		context.Background(),
		"SELECT COUNT(*) FROM transactions WHERE id = $1::uuid",
		withdrawTransactionID,
	).Scan(&transactionCount); err != nil {
		t.Fatalf("count rolled back withdrawal: %v", err)
	}
	if transactionCount != 0 {
		t.Fatalf("expected no withdrawal transaction, found %d", transactionCount)
	}
}

func newIntegrationWithdrawCommand(
	foundWallet domain.Wallet,
	clearingAccountID domain.AccountID,
	transactionID string,
	amountMinorUnits int64,
) wallet.WithdrawWalletCommand {
	return wallet.WithdrawWalletCommand{
		WalletID:          foundWallet.ID(),
		ClearingAccountID: clearingAccountID,
		TransactionID:     domain.TransactionID(transactionID),
		JournalEntryID:    domain.JournalEntryID(transactionID[:8] + "-7f0a-7b55-8def-123456789ab5"),
		WalletPostingID:   domain.PostingID(transactionID[:8] + "-7f0a-7b56-8def-123456789ab6"),
		ClearingPostingID: domain.PostingID(transactionID[:8] + "-7f0a-7b57-8def-123456789ab7"),
		AmountMinorUnits:  amountMinorUnits,
		Description:       "Integration wallet withdrawal",
		IdempotencyKey:    "integration-withdraw-" + transactionID,
		RequestHash:       "integration-withdraw-hash-" + transactionID,
	}
}
