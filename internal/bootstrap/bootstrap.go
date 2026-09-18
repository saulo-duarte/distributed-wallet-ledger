package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"financial-ledger/internal/ledger/adapters/dynamo"
	"financial-ledger/internal/ledger/adapters/postgres"
	db "financial-ledger/internal/ledger/adapters/postgres/generated"
	"financial-ledger/internal/ledger/application/account"
	"financial-ledger/internal/ledger/application/transaction"
	"financial-ledger/internal/ledger/application/wallet"
	"financial-ledger/internal/platform/config"
	"financial-ledger/internal/platform/dynamodb"
)

type Dependencies struct {
	CreateAccount      account.CreateAccountUseCase
	CreateWallet       wallet.CreateWalletUseCase
	GetWallet          wallet.GetWalletUseCase
	ListWalletsByOwner wallet.ListWalletsByOwnerUseCase
	GetWalletBalance   wallet.GetWalletBalanceUseCase
	DepositWallet      wallet.DepositWalletUseCase
	WithdrawWallet     wallet.WithdrawWalletUseCase
	TransferWallet     wallet.TransferWalletUseCase
	CreateHold         wallet.CreateHoldUseCase
	ReleaseHold        wallet.ReleaseHoldUseCase
	ExpireHold         wallet.ExpireHoldUseCase
	CaptureHold        wallet.CaptureHoldUseCase
	PostTransaction    transaction.PostTransactionUseCase
	GetTransaction     transaction.GetTransactionUseCase
	ReverseTransaction transaction.ReverseTransactionUseCase
	ListAccountEntries account.ListAccountEntriesUseCase

	readiness func(context.Context) error
	close     func()
}

func New(ctx context.Context, cfg config.Config, logger *slog.Logger) (*Dependencies, error) {
	connectionConfig := postgres.DefaultConnectionConfig(cfg.DatabaseURL)
	pool, err := postgres.OpenPool(ctx, connectionConfig)
	if err != nil {
		return nil, fmt.Errorf("initialize postgres pool: %w", err)
	}

	queries := db.New(pool)
	accountRepository := postgres.NewAccountRepository(queries)
	walletRepository := postgres.NewWalletRepository(queries)
	transactionRepository := postgres.NewTransactionRepository(pool, queries)
	holdRepository := postgres.NewHoldRepository(
		pool,
		queries,
		transactionRepository,
	)
	postTransaction := transaction.NewPostTransactionUseCase(
		transactionRepository,
	)

	var walletBalanceUseCase wallet.GetWalletBalanceUseCase
	var withdrawWalletUseCase wallet.WithdrawWalletUseCase

	dynamoClient, err := platformdynamo.NewClient(ctx, platformdynamo.Config{
		Endpoint: cfg.DynamoDBEndpoint,
		Region:   cfg.DynamoDBRegion,
		Table:    cfg.DynamoDBTable,
	})
	if err == nil {
		_ = platformdynamo.EnsureTable(ctx, dynamoClient, cfg.DynamoDBTable)
		projectionRepo := dynamo.NewWalletBalanceProjectionRepository(dynamoClient, cfg.DynamoDBTable)
		projector := wallet.NewWalletBalanceProjector(walletRepository, walletRepository, projectionRepo)

		walletBalanceUseCase = wallet.NewGetWalletBalanceUseCaseWithProjection(
			walletRepository,
			walletRepository,
			projectionRepo,
			logger,
		)
		withdrawWalletUseCase = wallet.NewWithdrawWalletUseCaseWithProjector(
			walletRepository,
			walletRepository,
			postTransaction,
			projector,
		)
	} else {
		walletBalanceUseCase = wallet.NewGetWalletBalanceUseCase(
			walletRepository,
			walletRepository,
		)
		withdrawWalletUseCase = wallet.NewWithdrawWalletUseCase(
			walletRepository,
			walletRepository,
			postTransaction,
		)
	}

	return &Dependencies{
		CreateAccount:      account.NewCreateAccountUseCase(accountRepository),
		CreateWallet:       wallet.NewCreateWalletUseCase(walletRepository),
		GetWallet:          wallet.NewGetWalletUseCase(walletRepository),
		ListWalletsByOwner: wallet.NewListWalletsByOwnerUseCase(walletRepository),
		GetWalletBalance:   walletBalanceUseCase,
		DepositWallet: wallet.NewDepositWalletUseCase(
			walletRepository,
			postTransaction,
		),
		WithdrawWallet: withdrawWalletUseCase,
		TransferWallet: wallet.NewTransferWalletUseCase(
			walletRepository,
			walletRepository,
			postTransaction,
		),
		CreateHold: wallet.NewCreateHoldUseCase(
			walletRepository,
			holdRepository,
		),
		ReleaseHold: wallet.NewReleaseHoldUseCase(holdRepository),
		ExpireHold:  wallet.NewExpireHoldUseCase(holdRepository),
		CaptureHold: wallet.NewCaptureHoldUseCase(
			walletRepository,
			holdRepository,
			holdRepository,
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
