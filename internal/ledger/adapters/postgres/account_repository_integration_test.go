//go:build integration

package postgres

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	db "financial-ledger/internal/ledger/adapters/postgres/generated"
	"financial-ledger/internal/ledger/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestAccountRepositoryCreate(t *testing.T) {
	pool := openIntegrationPool(t)
	queries := db.New(pool)
	repository := NewAccountRepository(queries)

	account := newIntegrationAccount(
		t,
		"0198f3b2-7f0a-7abc-8def-123456789abc",
	)
	cleanupAccount(t, pool, account.Code())
	t.Cleanup(func() {
		cleanupAccount(t, pool, account.Code())
	})

	if err := repository.Create(context.Background(), account); err != nil {
		t.Fatalf("create account: %v", err)
	}

	id, err := accountIDToUUID(account.ID())
	if err != nil {
		t.Fatalf("convert account ID for query: %v", err)
	}

	persisted, err := queries.GetAccount(context.Background(), id)
	if err != nil {
		t.Fatalf("get persisted account: %v", err)
	}

	if persisted.ID.String() != account.ID().String() {
		t.Fatalf("unexpected persisted ID: got %q, want %q", persisted.ID.String(), account.ID())
	}

	if persisted.Code != account.Code() {
		t.Fatalf("unexpected persisted code: got %q, want %q", persisted.Code, account.Code())
	}

	if persisted.Name != account.Name() {
		t.Fatalf("unexpected persisted name: got %q, want %q", persisted.Name, account.Name())
	}

	if persisted.Currency != account.Currency().String() {
		t.Fatalf("unexpected persisted currency: got %q, want %q", persisted.Currency, account.Currency())
	}

	if persisted.Status != string(domain.AccountStatusOpen) {
		t.Fatalf("unexpected persisted status: got %q, want %q", persisted.Status, domain.AccountStatusOpen)
	}
}

func TestAccountRepositoryCreateRejectsDuplicateCode(t *testing.T) {
	pool := openIntegrationPool(t)
	queries := db.New(pool)
	repository := NewAccountRepository(queries)

	code := fmt.Sprintf("integration-duplicate-%d", time.Now().UnixNano())
	first := newIntegrationAccountWithCode(
		t,
		"0198f3b2-7f0a-7abd-8def-123456789abc",
		code,
	)
	second := newIntegrationAccountWithCode(
		t,
		"0198f3b2-7f0a-7abe-8def-123456789abc",
		code,
	)
	cleanupAccount(t, pool, first.Code())
	t.Cleanup(func() {
		cleanupAccount(t, pool, first.Code())
	})

	if err := repository.Create(context.Background(), first); err != nil {
		t.Fatalf("create first account: %v", err)
	}

	if err := repository.Create(context.Background(), second); err == nil {
		t.Fatal("expected duplicate account code error")
	}
}

func TestAccountRepositoryCreateRejectsInvalidUUID(t *testing.T) {
	pool := openIntegrationPool(t)
	queries := db.New(pool)
	repository := NewAccountRepository(queries)

	account := newIntegrationAccount(t, "account-001")

	if err := repository.Create(context.Background(), account); err == nil {
		t.Fatal("expected invalid UUID error")
	}
}

func TestAccountRepositoryCreatePropagatesCanceledContext(t *testing.T) {
	pool := openIntegrationPool(t)
	queries := db.New(pool)
	repository := NewAccountRepository(queries)

	account := newIntegrationAccount(
		t,
		"0198f3b2-7f0a-7abf-8def-123456789abc",
	)
	cleanupAccount(t, pool, account.Code())

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := repository.Create(ctx, account); err == nil {
		t.Fatal("expected canceled context error")
	}
}

func openIntegrationPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://ledger:ledger@localhost:5432/ledger?sslmode=disable"
	}

	cfg := DefaultConnectionConfig(databaseURL)
	cfg.MaxAttempts = 1
	cfg.ConnectTimeout = 3 * time.Second

	pool, err := OpenPool(context.Background(), cfg)
	if err != nil {
		t.Fatalf(
			"connect to integration PostgreSQL: %v; start it with "+
				"docker compose up -d postgres and apply migrations before running integration tests",
			err,
		)
	}

	t.Cleanup(pool.Close)
	return pool
}

func newIntegrationAccount(t *testing.T, id string) domain.Account {
	t.Helper()

	return newIntegrationAccountWithCode(
		t,
		id,
		fmt.Sprintf("integration-%d", time.Now().UnixNano()),
	)
}

func newIntegrationAccountWithCode(
	t *testing.T,
	id string,
	code string,
) domain.Account {
	t.Helper()

	accountID, err := domain.NewAccountID(id)
	if err != nil {
		t.Fatalf("create account ID: %v", err)
	}

	currency, err := domain.NewCurrency("BRL")
	if err != nil {
		t.Fatalf("create account currency: %v", err)
	}

	account, err := domain.NewAccount(
		accountID,
		code,
		"Integration Account",
		currency,
	)
	if err != nil {
		t.Fatalf("create domain account: %v", err)
	}

	return account
}

func cleanupAccount(t *testing.T, pool *pgxpool.Pool, code string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if _, err := pool.Exec(ctx, "DELETE FROM accounts WHERE code = $1", code); err != nil {
		t.Fatalf("cleanup account %q: %v", code, err)
	}
}
