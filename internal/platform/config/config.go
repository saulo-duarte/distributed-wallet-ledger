package config

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"financial-ledger/internal/platform/awsx"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/joho/godotenv"
)

const (
	defaultAPIAddress            = ":8080"
	defaultMigrationsPath        = "migrations"
	defaultLogLevel              = "info"
	defaultLogFormat             = "text"
	defaultDynamoEndpoint        = "http://localhost:4566"
	defaultDynamoRegion          = "us-east-1"
	defaultDynamoTable           = "goledge_projections"
	defaultAWSEndpoint           = "http://localhost:4566"
	defaultAWSRegion             = "us-east-1"
	defaultSNSTopicARN           = "arn:aws:sns:us-east-1:000000000000:ledger-events"
	defaultSQSWalletBalanceQueue = "http://localhost:4566/000000000000/wallet-balance-projections-queue"
)

type SSMClient interface {
	GetParametersByPath(ctx context.Context, params *ssm.GetParametersByPathInput, optFns ...func(*ssm.Options)) (*ssm.GetParametersByPathOutput, error)
}

type Config struct {
	APIAddress               string
	DatabaseURL              string
	MigrationsPath           string
	LogLevel                 string
	LogFormat                string
	DynamoDBEndpoint         string
	DynamoDBRegion           string
	DynamoDBTable            string
	AWSEndpoint              string
	AWSRegion                string
	SNSTopicARN              string
	SQSWalletBalanceQueueURL string
	OTELServiceName          string
	OTELEndpoint             string
	OTELDisabled             bool
	MetricsEnabled           bool
}

func Load(ctx ...context.Context) (Config, error) {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return Config{}, fmt.Errorf("load .env file: %w", err)
	}

	c := context.Background()
	if len(ctx) > 0 && ctx[0] != nil {
		c = ctx[0]
	}

	ssmPath := os.Getenv("SSM_PARAMETER_PATH")
	ssmEnabled := strings.EqualFold(os.Getenv("AWS_SSM_ENABLED"), "true")
	if ssmPath != "" || ssmEnabled {
		if ssmPath == "" {
			ssmPath = "/goledge"
		}
		awsEndpoint := os.Getenv("AWS_ENDPOINT")
		awsRegion := os.Getenv("AWS_REGION")
		awsCfg, err := awsx.LoadAWSConfig(c, awsx.Config{
			Endpoint: awsEndpoint,
			Region:   awsRegion,
		})
		if err != nil {
			return Config{}, fmt.Errorf("load aws config for ssm: %w", err)
		}
		ssmClient := awsx.NewSSMClient(awsCfg, awsEndpoint)
		return LoadFromSSM(c, ssmClient, ssmPath, os.LookupEnv)
	}

	return LoadFromEnv(os.LookupEnv)
}

func LoadFromSSM(
	ctx context.Context,
	client SSMClient,
	path string,
	fallback func(string) (string, bool),
) (Config, error) {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	ssmMap, err := fetchSSMParameters(ctx, client, path)
	if err != nil {
		return Config{}, err
	}

	lookup := func(key string) (string, bool) {
		if val, ok := ssmMap[strings.ToUpper(key)]; ok && strings.TrimSpace(val) != "" {
			return val, true
		}
		if fallback != nil {
			return fallback(key)
		}
		return "", false
	}

	return LoadFromEnv(lookup)
}

func fetchSSMParameters(
	ctx context.Context,
	client SSMClient,
	path string,
) (map[string]string, error) {
	ssmMap := make(map[string]string)
	var nextToken *string

	for {
		out, err := client.GetParametersByPath(ctx, &ssm.GetParametersByPathInput{
			Path:           aws.String(path),
			WithDecryption: aws.Bool(true),
			Recursive:      aws.Bool(true),
			NextToken:      nextToken,
		})
		if err != nil {
			return nil, fmt.Errorf("fetch ssm parameters by path %q: %w", path, err)
		}

		for _, p := range out.Parameters {
			if p.Name != nil && p.Value != nil {
				key := strings.TrimPrefix(*p.Name, path)
				key = strings.TrimPrefix(key, "/")
				ssmMap[strings.ToUpper(key)] = *p.Value
			}
		}

		if out.NextToken == nil || *out.NextToken == "" {
			break
		}
		nextToken = out.NextToken
	}

	return ssmMap, nil
}

func LoadFromEnv(getenv func(string) (string, bool)) (Config, error) {
	apiAddress := value(getenv, "API_ADDR")
	if apiAddress == "" {
		apiAddress = value(getenv, "API_ADDRESS")
	}
	if apiAddress == "" {
		apiAddress = defaultAPIAddress
	}
	apiAddress = strings.TrimSpace(apiAddress)
	databaseURL := strings.TrimSpace(value(getenv, "DATABASE_URL"))
	migrationsPath := valueOrDefault(getenv, "MIGRATIONS_PATH", defaultMigrationsPath)
	logLevel := valueOrDefault(getenv, "LOG_LEVEL", defaultLogLevel)
	logFormat := valueOrDefault(getenv, "LOG_FORMAT", defaultLogFormat)
	dynamoEndpoint := valueOrDefault(getenv, "DYNAMODB_ENDPOINT", defaultDynamoEndpoint)
	dynamoRegion := valueOrDefault(getenv, "DYNAMODB_REGION", defaultDynamoRegion)
	dynamoTable := valueOrDefault(getenv, "DYNAMODB_TABLE", defaultDynamoTable)
	awsEndpoint := valueOrDefault(getenv, "AWS_ENDPOINT", defaultAWSEndpoint)
	awsRegion := valueOrDefault(getenv, "AWS_REGION", defaultAWSRegion)
	snsTopicARN := valueOrDefault(getenv, "SNS_TOPIC_ARN", defaultSNSTopicARN)
	sqsWalletBalanceQueueURL := valueOrDefault(
		getenv,
		"SQS_WALLET_BALANCE_QUEUE_URL",
		defaultSQSWalletBalanceQueue,
	)
	otelServiceName := valueOrDefault(getenv, "OTEL_SERVICE_NAME", "goledge-api")
	otelEndpoint := value(getenv, "OTEL_EXPORTER_OTLP_ENDPOINT")
	if otelEndpoint == "" {
		otelEndpoint = value(getenv, "OTEL_ENDPOINT")
	}
	otelDisabled := strings.EqualFold(value(getenv, "OTEL_DISABLED"), "true")
	metricsEnabled := !strings.EqualFold(value(getenv, "PROMETHEUS_METRICS_ENABLED"), "false")

	if databaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL cannot be empty")
	}

	if strings.TrimSpace(apiAddress) == "" {
		return Config{}, fmt.Errorf("API_ADDR cannot be empty")
	}

	if strings.TrimSpace(migrationsPath) == "" {
		return Config{}, fmt.Errorf("MIGRATIONS_PATH cannot be empty")
	}

	return Config{
		APIAddress:               apiAddress,
		DatabaseURL:              databaseURL,
		MigrationsPath:           migrationsPath,
		LogLevel:                 logLevel,
		LogFormat:                logFormat,
		DynamoDBEndpoint:         dynamoEndpoint,
		DynamoDBRegion:           dynamoRegion,
		DynamoDBTable:            dynamoTable,
		AWSEndpoint:              awsEndpoint,
		AWSRegion:                awsRegion,
		SNSTopicARN:              snsTopicARN,
		SQSWalletBalanceQueueURL: sqsWalletBalanceQueueURL,
		OTELServiceName:          otelServiceName,
		OTELEndpoint:             otelEndpoint,
		OTELDisabled:             otelDisabled,
		MetricsEnabled:           metricsEnabled,
	}, nil
}

func value(getenv func(string) (string, bool), key string) string {
	value, _ := getenv(key)
	return value
}

func valueOrDefault(
	getenv func(string) (string, bool),
	key string,
	defaultValue string,
) string {
	value, ok := getenv(key)
	if !ok {
		return defaultValue
	}
	return strings.TrimSpace(value)
}
