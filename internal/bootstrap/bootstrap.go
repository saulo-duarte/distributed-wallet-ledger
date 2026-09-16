package bootstrap

import (
	"context"
	"fmt"

	"financial-ledger/internal/ledger/adapters/postgres"
	db "financial-ledger/internal/ledger/adapters/postgres/generated"
	"financial-ledger/internal/ledger/application/account"
	"financial-ledger/internal/platform/config"
)

type Dependencies struct {
	CreateAccount account.CreateAccountUseCase

	close func()
}

func New(ctx context.Context, cfg config.Config) (*Dependencies, error) {
	connectionConfig := postgres.DefaultConnectionConfig(cfg.DatabaseURL)
	pool, err := postgres.OpenPool(ctx, connectionConfig)
	if err != nil {
		return nil, fmt.Errorf("initialize postgres pool: %w", err)
	}

	queries := db.New(pool)
	accountRepository := postgres.NewAccountRepository(queries)

	return &Dependencies{
		CreateAccount: account.NewCreateAccountUseCase(accountRepository),
		close:         pool.Close,
	}, nil
}

func (d *Dependencies) Close() {
	if d == nil || d.close == nil {
		return
	}

	d.close()
}
