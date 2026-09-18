package dynamo

import (
	"context"
	"financial-ledger/internal/ledger/application/wallet"
	"financial-ledger/internal/ledger/domain"
	platformdynamo "financial-ledger/internal/platform/dynamodb"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestWalletBalanceProjectionRepository_Integration(t *testing.T) {
	endpoint := os.Getenv("DYNAMODB_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://localhost:4566"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	tableName := "goledge_projections_test"
	cfg := platformdynamo.Config{
		Endpoint: endpoint,
		Region:   "us-east-1",
		Table:    tableName,
	}

	client, err := platformdynamo.NewClient(ctx, cfg)
	if err != nil {
		t.Fatalf("failed to create dynamo client: %v", err)
	}
	err = platformdynamo.EnsureTable(ctx, client, tableName)
	if err != nil {
		t.Fatalf("failed to ensure table: %v", err)
	}
	repo := NewWalletBalanceProjectionRepository(client, tableName)
	walletID, err := domain.NewWalletID(uuid.NewString())
	if err != nil {
		t.Fatalf("failed to create wallet id: %v", err)
	}
	currency, err := domain.NewCurrency("BRL")
	if err != nil {
		t.Fatalf("failed to create currency: %v", err)
	}
	expectedBalance := wallet.WalletBalance{
		WalletID:                   walletID,
		Currency:                   currency,
		LedgerBalanceMinorUnits:    50000,
		AvailableBalanceMinorUnits: 45000,
	}
	err = repo.SaveWalletBalance(ctx, expectedBalance)
	if err != nil {
		t.Fatalf("failed to save wallet balance: %v", err)
	}
	actualBalance, err := repo.GetWalletBalance(ctx, walletID)
	if err != nil {
		t.Fatalf("failed to get wallet balance: %v", err)
	}
	if actualBalance.WalletID != expectedBalance.WalletID {
		t.Fatalf("expected wallet id %v, got %v", expectedBalance.WalletID, actualBalance.WalletID)
	}
	if actualBalance.Currency != expectedBalance.Currency {
		t.Fatalf("expected currency %v, got %v", expectedBalance.Currency, actualBalance.Currency)
	}
	if actualBalance.LedgerBalanceMinorUnits != expectedBalance.LedgerBalanceMinorUnits {
		t.Fatalf("expected ledger balance %d, got %d", expectedBalance.LedgerBalanceMinorUnits, actualBalance.LedgerBalanceMinorUnits)
	}
	if actualBalance.AvailableBalanceMinorUnits != expectedBalance.AvailableBalanceMinorUnits {
		t.Fatalf("expected available balance %d, got %d", expectedBalance.AvailableBalanceMinorUnits, actualBalance.AvailableBalanceMinorUnits)
	}
}
