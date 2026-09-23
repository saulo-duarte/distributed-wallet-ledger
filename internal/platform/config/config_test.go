package config

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

type mockSSMClient struct {
	params map[string]string
	err    error
}

func (m *mockSSMClient) GetParametersByPath(
	ctx context.Context,
	params *ssm.GetParametersByPathInput,
	optFns ...func(*ssm.Options),
) (*ssm.GetParametersByPathOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	var outParams []types.Parameter
	for k, v := range m.params {
		name := k
		val := v
		outParams = append(outParams, types.Parameter{
			Name:  &name,
			Value: &val,
		})
	}

	return &ssm.GetParametersByPathOutput{
		Parameters: outParams,
	}, nil
}

func TestLoadFromEnv(t *testing.T) {
	env := map[string]string{
		"API_ADDR":        ":9090",
		"DATABASE_URL":    "postgres://ledger:ledger@localhost:5432/ledger?sslmode=disable",
		"MIGRATIONS_PATH": "db/migrations",
		"LOG_LEVEL":       "debug",
	}

	cfg, err := LoadFromEnv(func(key string) (string, bool) {
		value, ok := env[key]
		return value, ok
	})
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.APIAddress != ":9090" {
		t.Fatalf("unexpected API address: %q", cfg.APIAddress)
	}

	if cfg.DatabaseURL != env["DATABASE_URL"] {
		t.Fatalf("unexpected database URL: %q", cfg.DatabaseURL)
	}

	if cfg.MigrationsPath != "db/migrations" {
		t.Fatalf("unexpected migrations path: %q", cfg.MigrationsPath)
	}

	if cfg.LogLevel != "debug" {
		t.Fatalf("unexpected log level: %q", cfg.LogLevel)
	}
}

func TestLoadFromEnvUsesDefaults(t *testing.T) {
	cfg, err := LoadFromEnv(func(key string) (string, bool) {
		if key == "DATABASE_URL" {
			return "postgres://ledger:ledger@localhost:5432/ledger?sslmode=disable", true
		}
		return "", false
	})
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.APIAddress != defaultAPIAddress {
		t.Fatalf("unexpected default API address: %q", cfg.APIAddress)
	}

	if cfg.MigrationsPath != defaultMigrationsPath {
		t.Fatalf("unexpected default migrations path: %q", cfg.MigrationsPath)
	}

	if cfg.LogLevel != defaultLogLevel {
		t.Fatalf("unexpected default log level: %q", cfg.LogLevel)
	}
}

func TestLoadFromEnvRequiresDatabaseURL(t *testing.T) {
	_, err := LoadFromEnv(func(string) (string, bool) {
		return "", false
	})
	if err == nil {
		t.Fatal("expected missing DATABASE_URL error")
	}
}

func TestLoadFromEnvRejectsEmptyAPIAddress(t *testing.T) {
	_, err := LoadFromEnv(func(key string) (string, bool) {
		switch key {
		case "API_ADDR":
			return " ", true
		case "DATABASE_URL":
			return "postgres://ledger:ledger@localhost:5432/ledger?sslmode=disable", true
		default:
			return "", false
		}
	})
	if err == nil {
		t.Fatal("expected empty API_ADDR error")
	}
}

func TestLoadFromEnvRejectsEmptyMigrationsPath(t *testing.T) {
	_, err := LoadFromEnv(func(key string) (string, bool) {
		switch key {
		case "DATABASE_URL":
			return "postgres://ledger:ledger@localhost:5432/ledger?sslmode=disable", true
		case "MIGRATIONS_PATH":
			return " ", true
		default:
			return "", false
		}
	})
	if err == nil {
		t.Fatal("expected empty MIGRATIONS_PATH error")
	}
}

func TestLoadFromSSMSuccess(t *testing.T) {
	mockClient := &mockSSMClient{
		params: map[string]string{
			"/goledge/DATABASE_URL":                 "postgres://ssm_user:pass@ssm_host:5432/db",
			"/goledge/API_ADDR":                     ":8888",
			"/goledge/MIGRATIONS_PATH":              "custom/migrations",
			"/goledge/LOG_LEVEL":                    "warn",
			"/goledge/LOG_FORMAT":                   "json",
			"/goledge/DYNAMODB_ENDPOINT":            "http://dynamo:4566",
			"/goledge/DYNAMODB_REGION":              "sa-east-1",
			"/goledge/DYNAMODB_TABLE":               "custom_projections",
			"/goledge/AWS_ENDPOINT":                 "http://aws:4566",
			"/goledge/AWS_REGION":                   "sa-east-1",
			"/goledge/SNS_TOPIC_ARN":                "arn:aws:sns:sa-east-1:000:custom-topic",
			"/goledge/SQS_WALLET_BALANCE_QUEUE_URL": "http://aws:4566/000/custom-queue",
		},
	}

	cfg, err := LoadFromSSM(context.Background(), mockClient, "/goledge", nil)
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}

	if cfg.DatabaseURL != "postgres://ssm_user:pass@ssm_host:5432/db" {
		t.Fatalf("unexpected database URL: %q", cfg.DatabaseURL)
	}
	if cfg.APIAddress != ":8888" {
		t.Fatalf("unexpected API address: %q", cfg.APIAddress)
	}
	if cfg.MigrationsPath != "custom/migrations" {
		t.Fatalf("unexpected migrations path: %q", cfg.MigrationsPath)
	}
	if cfg.LogLevel != "warn" {
		t.Fatalf("unexpected log level: %q", cfg.LogLevel)
	}
	if cfg.LogFormat != "json" {
		t.Fatalf("unexpected log format: %q", cfg.LogFormat)
	}
	if cfg.DynamoDBEndpoint != "http://dynamo:4566" {
		t.Fatalf("unexpected dynamo endpoint: %q", cfg.DynamoDBEndpoint)
	}
	if cfg.DynamoDBRegion != "sa-east-1" {
		t.Fatalf("unexpected dynamo region: %q", cfg.DynamoDBRegion)
	}
	if cfg.DynamoDBTable != "custom_projections" {
		t.Fatalf("unexpected dynamo table: %q", cfg.DynamoDBTable)
	}
	if cfg.AWSEndpoint != "http://aws:4566" {
		t.Fatalf("unexpected aws endpoint: %q", cfg.AWSEndpoint)
	}
	if cfg.AWSRegion != "sa-east-1" {
		t.Fatalf("unexpected aws region: %q", cfg.AWSRegion)
	}
	if cfg.SNSTopicARN != "arn:aws:sns:sa-east-1:000:custom-topic" {
		t.Fatalf("unexpected sns topic arn: %q", cfg.SNSTopicARN)
	}
	if cfg.SQSWalletBalanceQueueURL != "http://aws:4566/000/custom-queue" {
		t.Fatalf("unexpected sqs queue url: %q", cfg.SQSWalletBalanceQueueURL)
	}
}

func TestLoadFromSSMFallback(t *testing.T) {
	mockClient := &mockSSMClient{
		params: map[string]string{
			"/goledge/DATABASE_URL": "postgres://ssm_user:pass@ssm_host:5432/db",
		},
	}

	fallbackEnv := map[string]string{
		"API_ADDR": ":7777",
	}

	cfg, err := LoadFromSSM(context.Background(), mockClient, "/goledge", func(k string) (string, bool) {
		v, ok := fallbackEnv[k]
		return v, ok
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.DatabaseURL != "postgres://ssm_user:pass@ssm_host:5432/db" {
		t.Fatalf("unexpected database URL: %q", cfg.DatabaseURL)
	}
	if cfg.APIAddress != ":7777" {
		t.Fatalf("unexpected API address: %q", cfg.APIAddress)
	}
}

func TestLoadFromSSMError(t *testing.T) {
	mockClient := &mockSSMClient{
		err: errors.New("ssm unreachable"),
	}

	_, err := LoadFromSSM(context.Background(), mockClient, "/goledge", nil)
	if err == nil {
		t.Fatal("expected error but got nil")
	}
}
