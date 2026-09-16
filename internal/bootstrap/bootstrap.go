package bootstrap

import (
	"context"
	"errors"
	"fmt"

	"financial-ledger/internal/ledger/adapters/postgres"
	db "financial-ledger/internal/ledger/adapters/postgres/generated"
	"financial-ledger/internal/ledger/application/account"
	"financial-ledger/internal/ledger/application/transaction"
	"financial-ledger/internal/platform/config"
)

type Dependencies struct {
	CreateAccount      account.CreateAccountUseCase
	PostTransaction    transaction.PostTransactionUseCase
	GetTransaction     transaction.GetTransactionUseCase
	ReverseTransaction transaction.ReverseTransactionUseCase
	ListAccountEntries account.ListAccountEntriesUseCase

	readiness func(context.Context) error
	close     func()
}

func New(ctx context.Context, cfg config.Config) (*Dependencies, error) {
	connectionConfig := postgres.DefaultConnectionConfig(cfg.DatabaseURL)
	pool, err := postgres.OpenPool(ctx, connectionConfig)
	if err != nil {
		return nil, fmt.Errorf("initialize postgres pool: %w", err)
	}

	queries := db.New(pool)
	accountRepository := postgres.NewAccountRepository(queries)
	transactionRepository := postgres.NewTransactionRepository(pool, queries)

	return &Dependencies{
		CreateAccount:      account.NewCreateAccountUseCase(accountRepository),
		PostTransaction:    transaction.NewPostTransactionUseCase(transactionRepository),
		GetTransaction:     transaction.NewGetTransactionUseCase(transactionRepository),
		ReverseTransaction: transaction.NewReverseTransactionUseCase(transactionRepository, transactionRepository),
		ListAccountEntries: account.NewListAccountEntriesUseCase(accountRepository),
		readiness:          pool.Ping,
		close:              pool.Close,
	}, nil
}

func (d *Dependencies) CheckReadiness(ctx context.Context) error {
	if d == nil || d.readiness == nil {
		return errors.New("readiness check is not configured")
	}

	return d.readiness(ctx)
}

func (d *Dependencies) Close() {
	if d == nil || d.close == nil {
		return
	}

	d.close()
}
