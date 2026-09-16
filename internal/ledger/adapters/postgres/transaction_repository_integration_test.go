//go:build integration

package postgres

import (
	"context"
	"fmt"
	"testing"
	"time"

	db "financial-ledger/internal/ledger/adapters/postgres/generated"
	"financial-ledger/internal/ledger/application/account"
	"financial-ledger/internal/ledger/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestTransactionRepositoryPostPersistsAllRows(t *testing.T) {
	pool := openIntegrationPool(t)
	queries := db.New(pool)
	repository := NewTransactionRepository(pool, queries)

	debitAccount := newIntegrationAccountWithCode(
		t,
		"0198f3b2-7f0a-7ac0-8def-123456789ac0",
		"integration-ledger-persist-debit",
	)
	creditAccount := newIntegrationAccountWithCode(
		t,
		"0198f3b2-7f0a-7ac1-8def-123456789ac1",
		"integration-ledger-persist-credit",
	)

	accountRepository := NewAccountRepository(queries)
	cleanupLedgerTransaction(t, pool, "0198f3b2-7f0a-7ac2-8def-123456789ac2")
	cleanupAccount(t, pool, debitAccount.Code())
	cleanupAccount(t, pool, creditAccount.Code())
	t.Cleanup(func() {
		cleanupLedgerTransaction(t, pool, "0198f3b2-7f0a-7ac2-8def-123456789ac2")
		cleanupAccount(t, pool, debitAccount.Code())
		cleanupAccount(t, pool, creditAccount.Code())
	})

	if err := accountRepository.Create(context.Background(), debitAccount); err != nil {
		t.Fatalf("create debit account: %v", err)
	}
	if err := accountRepository.Create(context.Background(), creditAccount); err != nil {
		t.Fatalf("create credit account: %v", err)
	}

	postedTransaction := newIntegrationTransaction(
		t,
		"0198f3b2-7f0a-7ac2-8def-123456789ac2",
		debitAccount.ID(),
		creditAccount.ID(),
	)

	if err := repository.Post(
		context.Background(),
		postedTransaction,
		"integration-post-001",
		"integration-hash-001",
	); err != nil {
		t.Fatalf("post transaction: %v", err)
	}

	transactionID, err := transactionIDToUUID(postedTransaction.ID())
	if err != nil {
		t.Fatalf("convert transaction ID: %v", err)
	}

	persistedTransaction, err := queries.GetTransaction(
		context.Background(),
		transactionID,
	)
	if err != nil {
		t.Fatalf("get persisted transaction: %v", err)
	}

	if persistedTransaction.Description != postedTransaction.Description() {
		t.Fatalf(
			"unexpected description: got %q, want %q",
			persistedTransaction.Description,
			postedTransaction.Description(),
		)
	}

	journalEntryID, err := journalEntryIDToUUID(
		postedTransaction.JournalEntry().ID(),
	)
	if err != nil {
		t.Fatalf("convert journal entry ID: %v", err)
	}

	postings, err := queries.ListJournalEntryPostings(
		context.Background(),
		journalEntryID,
	)
	if err != nil {
		t.Fatalf("list persisted postings: %v", err)
	}

	if len(postings) != 2 {
		t.Fatalf("unexpected posting count: got %d, want 2", len(postings))
	}

	details, err := repository.Get(
		context.Background(),
		postedTransaction.ID(),
	)
	if err != nil {
		t.Fatalf("get transaction details: %v", err)
	}

	if details.ID != postedTransaction.ID() {
		t.Fatalf("unexpected transaction details ID: %q", details.ID)
	}
	if details.Description != postedTransaction.Description() {
		t.Fatalf("unexpected transaction details description: %q", details.Description)
	}
	if details.CreatedAt.IsZero() {
		t.Fatal("expected transaction created_at")
	}
	if details.JournalEntry.PostedAt.IsZero() {
		t.Fatal("expected journal entry posted_at")
	}
	if len(details.JournalEntry.Postings) != 2 {
		t.Fatalf(
			"unexpected transaction detail postings: got %d, want 2",
			len(details.JournalEntry.Postings),
		)
	}
}

func TestTransactionRepositoryPostRollsBackWhenPostingFails(t *testing.T) {
	pool := openIntegrationPool(t)
	queries := db.New(pool)
	repository := NewTransactionRepository(pool, queries)

	transactionID := "0198f3b2-7f0a-7ad0-8def-123456789ad0"
	cleanupLedgerTransaction(t, pool, transactionID)
	t.Cleanup(func() {
		cleanupLedgerTransaction(t, pool, transactionID)
	})

	missingAccountID, err := domain.NewAccountID(
		"0198f3b2-7f0a-7ad1-8def-123456789ad1",
	)
	if err != nil {
		t.Fatal(err)
	}

	validAccountID, err := domain.NewAccountID(
		"0198f3b2-7f0a-7ad2-8def-123456789ad2",
	)
	if err != nil {
		t.Fatal(err)
	}

	postedTransaction := newIntegrationTransaction(
		t,
		transactionID,
		missingAccountID,
		validAccountID,
	)

	if err := repository.Post(
		context.Background(),
		postedTransaction,
		"integration-rollback-001",
		"integration-rollback-hash-001",
	); err == nil {
		t.Fatal("expected posting failure")
	}

	var transactionCount int
	if err := pool.QueryRow(
		context.Background(),
		"SELECT COUNT(*) FROM transactions WHERE id = $1",
		transactionID,
	).Scan(&transactionCount); err != nil {
		t.Fatalf("count rolled back transaction: %v", err)
	}

	if transactionCount != 0 {
		t.Fatalf("expected rollback, found %d transaction rows", transactionCount)
	}
}

func TestTransactionRepositoryPostRejectsDuplicateIdempotencyKey(t *testing.T) {
	pool := openIntegrationPool(t)
	queries := db.New(pool)
	repository := NewTransactionRepository(pool, queries)

	debitAccount := newIntegrationAccountWithCode(
		t,
		"0198f3b2-7f0a-7ae0-8def-123456789ae0",
		"integration-ledger-duplicate-debit",
	)
	creditAccount := newIntegrationAccountWithCode(
		t,
		"0198f3b2-7f0a-7ae1-8def-123456789ae1",
		"integration-ledger-duplicate-credit",
	)
	accountRepository := NewAccountRepository(queries)

	cleanupLedgerTransaction(t, pool, "0198f3b2-7f0a-7ae2-8def-123456789ae2")
	cleanupLedgerTransaction(t, pool, "0198f3b2-7f0a-7ae3-8def-123456789ae3")
	cleanupAccount(t, pool, debitAccount.Code())
	cleanupAccount(t, pool, creditAccount.Code())
	t.Cleanup(func() {
		cleanupLedgerTransaction(t, pool, "0198f3b2-7f0a-7ae2-8def-123456789ae2")
		cleanupLedgerTransaction(t, pool, "0198f3b2-7f0a-7ae3-8def-123456789ae3")
		cleanupAccount(t, pool, debitAccount.Code())
		cleanupAccount(t, pool, creditAccount.Code())
	})

	if err := accountRepository.Create(context.Background(), debitAccount); err != nil {
		t.Fatalf("create debit account: %v", err)
	}
	if err := accountRepository.Create(context.Background(), creditAccount); err != nil {
		t.Fatalf("create credit account: %v", err)
	}

	first := newIntegrationTransaction(
		t,
		"0198f3b2-7f0a-7ae2-8def-123456789ae2",
		debitAccount.ID(),
		creditAccount.ID(),
	)
	second := newIntegrationTransaction(
		t,
		"0198f3b2-7f0a-7ae3-8def-123456789ae3",
		debitAccount.ID(),
		creditAccount.ID(),
	)

	if err := repository.Post(
		context.Background(),
		first,
		"integration-duplicate-001",
		"integration-duplicate-hash-001",
	); err != nil {
		t.Fatalf("post first transaction: %v", err)
	}

	if err := repository.Post(
		context.Background(),
		second,
		"integration-duplicate-001",
		"integration-duplicate-hash-001",
	); err == nil {
		t.Fatal("expected duplicate idempotency key error")
	}
}

func TestAccountRepositoryListEntriesUsesDescendingCursorPagination(t *testing.T) {
	pool := openIntegrationPool(t)
	queries := db.New(pool)
	accountRepository := NewAccountRepository(queries)
	transactionRepository := NewTransactionRepository(pool, queries)

	debitAccount := newIntegrationAccountWithCode(
		t,
		"0198f3b2-7f0a-7af0-8def-123456789af0",
		"integration-list-debit",
	)
	creditAccount := newIntegrationAccountWithCode(
		t,
		"0198f3b2-7f0a-7af1-8def-123456789af1",
		"integration-list-credit",
	)

	transactionIDs := []string{
		"0198f3b2-7f0a-7af2-8def-123456789af2",
		"0198f3b3-7f0a-7af3-8def-123456789af3",
		"0198f3b4-7f0a-7af4-8def-123456789af4",
	}

	for _, transactionID := range transactionIDs {
		cleanupLedgerTransaction(t, pool, transactionID)
	}
	cleanupAccount(t, pool, debitAccount.Code())
	cleanupAccount(t, pool, creditAccount.Code())
	t.Cleanup(func() {
		for _, transactionID := range transactionIDs {
			cleanupLedgerTransaction(t, pool, transactionID)
		}
		cleanupAccount(t, pool, debitAccount.Code())
		cleanupAccount(t, pool, creditAccount.Code())
	})

	if err := accountRepository.Create(context.Background(), debitAccount); err != nil {
		t.Fatalf("create debit account: %v", err)
	}
	if err := accountRepository.Create(context.Background(), creditAccount); err != nil {
		t.Fatalf("create credit account: %v", err)
	}

	for index, transactionID := range transactionIDs {
		postedTransaction := newIntegrationTransaction(
			t,
			transactionID,
			debitAccount.ID(),
			creditAccount.ID(),
		)

		if err := transactionRepository.Post(
			context.Background(),
			postedTransaction,
			fmt.Sprintf("integration-list-%d", index),
			fmt.Sprintf("integration-list-hash-%d", index),
		); err != nil {
			t.Fatalf("post transaction %d: %v", index, err)
		}

		time.Sleep(5 * time.Millisecond)
	}

	firstPage, err := accountRepository.ListEntries(
		context.Background(),
		debitAccount.ID(),
		2,
		nil,
	)
	if err != nil {
		t.Fatalf("list first account entries page: %v", err)
	}

	if len(firstPage) != 2 {
		t.Fatalf("unexpected first page size: got %d, want 2", len(firstPage))
	}

	if firstPage[0].TransactionID.String() != transactionIDs[2] {
		t.Fatalf(
			"unexpected newest transaction: got %q, want %q",
			firstPage[0].TransactionID,
			transactionIDs[2],
		)
	}

	if firstPage[1].TransactionID.String() != transactionIDs[1] {
		t.Fatalf(
			"unexpected second transaction: got %q, want %q",
			firstPage[1].TransactionID,
			transactionIDs[1],
		)
	}

	secondPage, err := accountRepository.ListEntries(
		context.Background(),
		debitAccount.ID(),
		2,
		&account.AccountEntriesCursor{
			CreatedAt: firstPage[1].CreatedAt,
			PostingID: firstPage[1].PostingID,
		},
	)
	if err != nil {
		t.Fatalf("list second account entries page: %v", err)
	}

	if len(secondPage) != 1 {
		t.Fatalf("unexpected second page size: got %d, want 1", len(secondPage))
	}

	if secondPage[0].TransactionID.String() != transactionIDs[0] {
		t.Fatalf(
			"unexpected oldest transaction: got %q, want %q",
			secondPage[0].TransactionID,
			transactionIDs[0],
		)
	}
}

func newIntegrationTransaction(
	t *testing.T,
	transactionID string,
	debitAccountID domain.AccountID,
	creditAccountID domain.AccountID,
) domain.Transaction {
	t.Helper()

	txID, err := domain.NewTransactionID(transactionID)
	if err != nil {
		t.Fatal(err)
	}

	journalID, err := domain.NewJournalEntryID(
		transactionID[:8] + "-7f0a-7abc-8def-123456789abc",
	)
	if err != nil {
		t.Fatal(err)
	}

	currency, err := domain.NewCurrency("BRL")
	if err != nil {
		t.Fatal(err)
	}

	debitAmount, err := domain.NewMoney(currency, 10000)
	if err != nil {
		t.Fatal(err)
	}

	creditAmount, err := domain.NewMoney(currency, 10000)
	if err != nil {
		t.Fatal(err)
	}

	debitPostingID, err := domain.NewPostingID(
		transactionID[:8] + "-7f0a-7abd-8def-123456789abd",
	)
	if err != nil {
		t.Fatal(err)
	}

	creditPostingID, err := domain.NewPostingID(
		transactionID[:8] + "-7f0a-7abe-8def-123456789abe",
	)
	if err != nil {
		t.Fatal(err)
	}

	debitPosting, err := domain.NewPosting(
		debitPostingID,
		debitAccountID,
		domain.PostingDirectionDebit,
		debitAmount,
	)
	if err != nil {
		t.Fatal(err)
	}

	creditPosting, err := domain.NewPosting(
		creditPostingID,
		creditAccountID,
		domain.PostingDirectionCredit,
		creditAmount,
	)
	if err != nil {
		t.Fatal(err)
	}

	journalEntry, err := domain.NewJournalEntry(
		journalID,
		txID,
		currency,
		[]domain.Posting{debitPosting, creditPosting},
	)
	if err != nil {
		t.Fatal(err)
	}

	postedTransaction, err := domain.NewTransaction(
		txID,
		fmt.Sprintf("Integration transaction %s", transactionID),
		journalEntry,
	)
	if err != nil {
		t.Fatal(err)
	}

	return postedTransaction
}

func cleanupLedgerTransaction(
	t *testing.T,
	pool *pgxpool.Pool,
	transactionID string,
) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if _, err := pool.Exec(
		ctx,
		"DELETE FROM idempotency_keys WHERE transaction_id = $1::uuid",
		transactionID,
	); err != nil {
		t.Fatalf("cleanup idempotency key for transaction %q: %v", transactionID, err)
	}

	if _, err := pool.Exec(
		ctx,
		`DELETE FROM postings
          WHERE journal_entry_id IN (
              SELECT id FROM journal_entries WHERE transaction_id = $1::uuid
          )`,
		transactionID,
	); err != nil {
		t.Fatalf("cleanup postings for transaction %q: %v", transactionID, err)
	}

	if _, err := pool.Exec(
		ctx,
		"DELETE FROM journal_entries WHERE transaction_id = $1::uuid",
		transactionID,
	); err != nil {
		t.Fatalf("cleanup journal entry for transaction %q: %v", transactionID, err)
	}

	if _, err := pool.Exec(
		ctx,
		"DELETE FROM transactions WHERE id = $1::uuid",
		transactionID,
	); err != nil {
		t.Fatalf("cleanup transaction %q: %v", transactionID, err)
	}
}
