package dynamo

import (
	"context"
	"financial-ledger/internal/ledger/application/wallet"
	"financial-ledger/internal/ledger/domain"
	platformdynamo "financial-ledger/internal/platform/dynamodb"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
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

	newerBalance := wallet.WalletBalance{
		WalletID:                   walletID,
		Currency:                   currency,
		LedgerBalanceMinorUnits:    60000,
		AvailableBalanceMinorUnits: 55000,
	}
	err = repo.SaveWalletBalance(ctx, newerBalance)
	if err != nil {
		t.Fatalf("failed to save newer balance: %v", err)
	}

	olderBalance := wallet.WalletBalance{
		WalletID:                   walletID,
		Currency:                   currency,
		LedgerBalanceMinorUnits:    10000,
		AvailableBalanceMinorUnits: 10000,
	}
	nowPast := time.Now().Add(-1 * time.Hour).UTC().Format(time.RFC3339Nano)
	pk := "WALLET#" + walletID.String()
	sk := "BALANCE"
	av, err := attributevalue.MarshalMap(walletBalanceItem{
		PK:                         pk,
		SK:                         sk,
		CreatedAt:                  nowPast,
		WalletID:                   olderBalance.WalletID.String(),
		Currency:                   olderBalance.Currency.String(),
		LedgerBalanceMinorUnits:    olderBalance.LedgerBalanceMinorUnits,
		AvailableBalanceMinorUnits: olderBalance.AvailableBalanceMinorUnits,
		UpdatedAt:                  nowPast,
	})
	if err != nil {
		t.Fatalf("failed to marshal older balance: %v", err)
	}

	_, err = client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           &tableName,
		Item:                av,
		ConditionExpression: aws.String("attribute_not_exists(pk) OR updated_at <= :new_updated_at"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":new_updated_at": &types.AttributeValueMemberS{Value: nowPast},
		},
	})
	if err == nil {
		t.Fatal("expected conditional check failed error for out of order update, got nil")
	}

	currentBalance, err := repo.GetWalletBalance(ctx, walletID)
	if err != nil {
		t.Fatalf("failed to get current balance: %v", err)
	}
	if currentBalance.LedgerBalanceMinorUnits != newerBalance.LedgerBalanceMinorUnits {
		t.Fatalf("expected ledger balance %d, got %d", newerBalance.LedgerBalanceMinorUnits, currentBalance.LedgerBalanceMinorUnits)
	}
}
