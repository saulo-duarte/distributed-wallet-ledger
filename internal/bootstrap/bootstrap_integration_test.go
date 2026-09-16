//go:build integration

package bootstrap

import (
	"context"
	"os"
	"testing"
	"time"

	"financial-ledger/internal/platform/config"
)

func TestDependenciesCheckReadiness(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://ledger:ledger@localhost:5432/ledger?sslmode=disable"
	}

	dependencies, err := New(context.Background(), config.Config{
		DatabaseURL: databaseURL,
	})
	if err != nil {
		t.Fatalf("initialize dependencies: %v", err)
	}
	t.Cleanup(dependencies.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := dependencies.CheckReadiness(ctx); err != nil {
		t.Fatalf("check readiness: %v", err)
	}
}
