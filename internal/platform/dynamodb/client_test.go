package platformdynamo_test

import (
	"context"
	"os"
	"testing"
	"time"

	dynamo "financial-ledger/internal/platform/dynamodb"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

func TestEnsureTable_Integration(t *testing.T) {
	endpoint := os.Getenv("DYNAMODB_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://localhost:4566"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cfg := dynamo.Config{
		Endpoint: endpoint,
		Region:   "us-east-1",
		Table:    "goledge_projections_test",
	}

	client, err := dynamo.NewClient(ctx, cfg)
	if err != nil {
		t.Fatalf("failed to create dynamo client: %v", err)
	}

	err = dynamo.EnsureTable(ctx, client, cfg.Table)
	if err != nil {
		t.Skipf("skipping integration test: DynamoDB endpoint %s unreachable: %v", endpoint, err)
	}

	describeOutput, err := client.DescribeTable(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(cfg.Table),
	})
	if err != nil {
		t.Fatalf("failed to describe created table: %v", err)
	}

	if aws.ToString(describeOutput.Table.TableName) != cfg.Table {
		t.Fatalf("expected table name %q, got %q", cfg.Table, aws.ToString(describeOutput.Table.TableName))
	}

	if len(describeOutput.Table.LocalSecondaryIndexes) != 1 {
		t.Fatalf("expected 1 LSI, got %d", len(describeOutput.Table.LocalSecondaryIndexes))
	}
	if aws.ToString(describeOutput.Table.LocalSecondaryIndexes[0].IndexName) != "lsi_created_at" {
		t.Fatalf("expected LSI name %q, got %q", "lsi_created_at", aws.ToString(describeOutput.Table.LocalSecondaryIndexes[0].IndexName))
	}

	err = dynamo.EnsureTable(ctx, client, cfg.Table)
	if err != nil {
		t.Fatalf("expected EnsureTable to be idempotent, got error: %v", err)
	}
}
