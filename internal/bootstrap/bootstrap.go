package bootstrap

import (
	"context"
	"errors"
	"fmt"

	"financial-ledger/internal/ledger/adapters/postgres"
	db "financial-ledger/internal/ledger/adapters/postgres/generated"
	"financial-ledger/internal/ledger/application/account"
	"financial-ledger/internal/ledger/application/transaction"
	"financial-ledger/internal/ledger/application/wallet"
	"financial-ledger/internal/platform/config"
)

type Dependencies struct {
	CreateAccount      account.CreateAccountUseCase
	CreateWallet       wallet.CreateWalletUseCase
	GetWallet          wallet.GetWalletUseCase
	ListWalletsByOwner wallet.ListWalletsByOwnerUseCase
	GetWalletBalance   wallet.GetWalletBalanceUseCase
	WithdrawWallet     wallet.WithdrawWalletUseCase
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
	walletRepository := postgres.NewWalletRepository(queries)
	transactionRepository := postgres.NewTransactionRepository(pool, queries)
	postTransaction := transaction.NewPostTransactionUseCase(
		transactionRepository,
	)

	return &Dependencies{
		CreateAccount:      account.NewCreateAccountUseCase(accountRepository),
		CreateWallet:       wallet.NewCreateWalletUseCase(walletRepository),
		GetWallet:          wallet.NewGetWalletUseCase(walletRepository),
		ListWalletsByOwner: wallet.NewListWalletsByOwnerUseCase(walletRepository),
		GetWalletBalance: wallet.NewGetWalletBalanceUseCase(
			walletRepository,
			walletRepository,
		),
		WithdrawWallet: wallet.NewWithdrawWalletUseCase(
			walletRepository,
			walletRepository,
			postTransaction,
		),
		PostTransaction:    postTransaction,
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
