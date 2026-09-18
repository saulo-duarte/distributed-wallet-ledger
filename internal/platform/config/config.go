package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

const (
	defaultAPIAddress     = ":8080"
	defaultMigrationsPath = "migrations"
	defaultLogLevel       = "info"
	defaultLogFormat      = "text"
	defaultDynamoEndpoint = "http://localhost:4566"
	defaultDynamoRegion   = "us-east-1"
	defaultDynamoTable    = "goledge_projections"
)

type Config struct {
	APIAddress       string
	DatabaseURL      string
	MigrationsPath   string
	LogLevel         string
	LogFormat        string
	DynamoDBEndpoint string
	DynamoDBRegion   string
	DynamoDBTable    string
}

func Load() (Config, error) {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return Config{}, fmt.Errorf("load .env file: %w", err)
	}

	return LoadFromEnv(os.LookupEnv)
}

func LoadFromEnv(getenv func(string) (string, bool)) (Config, error) {
	apiAddress := valueOrDefault(getenv, "API_ADDR", defaultAPIAddress)
	databaseURL := strings.TrimSpace(value(getenv, "DATABASE_URL"))
	migrationsPath := valueOrDefault(
		getenv,
		"MIGRATIONS_PATH",
		defaultMigrationsPath,
	)
	logLevel := valueOrDefault(getenv, "LOG_LEVEL", defaultLogLevel)
	logFormat := valueOrDefault(getenv, "LOG_FORMAT", defaultLogFormat)
	dynamoEndpoint := valueOrDefault(getenv, "DYNAMODB_ENDPOINT", defaultDynamoEndpoint)
	dynamoRegion := valueOrDefault(getenv, "DYNAMODB_REGION", defaultDynamoRegion)
	dynamoTable := valueOrDefault(getenv, "DYNAMODB_TABLE", defaultDynamoTable)

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
		APIAddress:       apiAddress,
		DatabaseURL:      databaseURL,
		MigrationsPath:   migrationsPath,
		LogLevel:         logLevel,
		LogFormat:        logFormat,
		DynamoDBEndpoint: dynamoEndpoint,
		DynamoDBRegion:   dynamoRegion,
		DynamoDBTable:    dynamoTable,
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
