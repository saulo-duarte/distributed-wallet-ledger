package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"financial-ledger/internal/ledger/adapters/dynamo"
	"financial-ledger/internal/ledger/adapters/gateway"
	"financial-ledger/internal/ledger/adapters/postgres"
	db "financial-ledger/internal/ledger/adapters/postgres/generated"
	"financial-ledger/internal/ledger/adapters/sns"
	"financial-ledger/internal/ledger/adapters/sqs"
	"financial-ledger/internal/ledger/adapters/worker"
	"financial-ledger/internal/ledger/application/account"
	"financial-ledger/internal/ledger/application/outbox"
	"financial-ledger/internal/ledger/application/saga"
	"financial-ledger/internal/ledger/application/transaction"
	"financial-ledger/internal/ledger/application/wallet"
	"financial-ledger/internal/platform/awsx"
	"financial-ledger/internal/platform/config"
	"financial-ledger/internal/platform/dynamodb"
	"financial-ledger/internal/platform/observability"
	"financial-ledger/internal/platform/resilience"
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
	PaymentSaga        saga.PaymentSagaOrchestrator
	PostTransaction    transaction.PostTransactionUseCase
	GetTransaction     transaction.GetTransactionUseCase
	ReverseTransaction transaction.ReverseTransactionUseCase
	ListAccountEntries account.ListAccountEntriesUseCase

	readiness func(context.Context) error
	close     func()
}

func New(ctx context.Context, cfg config.Config, logger *slog.Logger) (*Dependencies, error) {
	if logger == nil {
		logger = slog.Default()
	}

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
	metrics := observability.DefaultMetrics()

	var walletBalanceUseCase wallet.GetWalletBalanceUseCase
	var withdrawWalletUseCase wallet.WithdrawWalletUseCase

	dynamoClient, err := platformdynamo.NewClient(ctx, platformdynamo.Config{
		Endpoint: cfg.DynamoDBEndpoint,
		Region:   cfg.DynamoDBRegion,
		Table:    cfg.DynamoDBTable,
	})
	if err == nil {
		if err := platformdynamo.EnsureTable(ctx, dynamoClient, cfg.DynamoDBTable); err != nil {
			logger.Warn("dynamodb_projection_unavailable", slog.Any("error", err))
		}
		projectionRepo := dynamo.NewWalletBalanceProjectionRepository(dynamoClient, cfg.DynamoDBTable)
		projector := wallet.NewWalletBalanceProjector(walletRepository, walletRepository, projectionRepo)

		awsCfg, awsErr := awsx.LoadAWSConfig(ctx, awsx.Config{
			Endpoint: cfg.AWSEndpoint,
			Region:   cfg.AWSRegion,
		})
		if awsErr == nil {
			snsClient := awsx.NewSNSClient(awsCfg, cfg.AWSEndpoint)
			sqsClient := awsx.NewSQSClient(awsCfg, cfg.AWSEndpoint)

			snsPublisher, pubErr := sns.NewEventPublisher(snsClient, cfg.SNSTopicARN)
			if pubErr == nil {
				outboxRepo := postgres.NewOutboxRepository(queries, pool)
				relay := outbox.NewRelay(
					outboxRepo,
					snsPublisher,
					outbox.RelayConfig{},
					logger,
					metrics,
				)
				go func() {
					if err := relay.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
						logger.Error("outbox_relay_stopped", slog.Any("error", err))
					}
				}()
			} else {
				logger.Warn("sns_publisher_unavailable", slog.Any("error", pubErr))
			}

			consumer, consumerErr := sqs.NewProjectionConsumer(
				sqsClient,
				projector,
				sqs.ProjectionConsumerConfig{
					QueueURL: cfg.SQSWalletBalanceQueueURL,
				},
				logger,
				metrics,
			)
			if consumerErr == nil {
				go func() {
					if err := consumer.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
						logger.Error("sqs_projection_consumer_stopped", slog.Any("error", err))
					}
				}()
			} else {
				logger.Warn("sqs_projection_consumer_unavailable", slog.Any("error", consumerErr))
			}
		} else {
			logger.Warn("aws_eventing_unavailable", slog.Any("error", awsErr))
		}

		walletBalanceUseCase = wallet.NewGetWalletBalanceUseCaseWithProjection(
			walletRepository,
			walletRepository,
			projectionRepo,
			logger,
		)
		withdrawWalletUseCase = wallet.NewWithdrawWalletUseCase(
			walletRepository,
			walletRepository,
			postTransaction,
		)
	} else {
		logger.Warn("dynamodb_projection_unavailable", slog.Any("error", err))
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

	createHoldUseCase := wallet.NewCreateHoldUseCase(
		walletRepository,
		holdRepository,
	)
	releaseHoldUseCase := wallet.NewReleaseHoldUseCase(holdRepository)
	expireHoldUseCase := wallet.NewExpireHoldUseCase(holdRepository)
	captureHoldUseCase := wallet.NewCaptureHoldUseCase(
		walletRepository,
		holdRepository,
		holdRepository,
	)

	holdExpirationWorker := worker.NewHoldExpirationWorker(
		holdRepository,
		expireHoldUseCase,
		worker.HoldExpirationConfig{},
		logger,
	)
	go func() {
		if err := holdExpirationWorker.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
			logger.Error("hold_expiration_worker_stopped", slog.Any("error", err))
		}
	}()

	gatewayCB := resilience.NewCircuitBreaker(resilience.Config{
		Name:             "payment_gateway",
		FailureThreshold: 3,
		SuccessThreshold: 2,
		Timeout:          5 * time.Second,
		OnStateChange: func(name string, from, to resilience.State) {
			logger.Warn(
				"circuit_breaker_state_changed",
				slog.String("circuit_breaker", name),
				slog.String("from_state", from.String()),
				slog.String("to_state", to.String()),
			)
			metrics.RecordCircuitBreakerState(name, to.String())
		},
	})
	metrics.RecordCircuitBreakerState("payment_gateway", "closed")

	protectedGateway := gateway.NewCircuitBreakerPaymentGateway(
		gateway.NewMockPaymentGateway(),
		gatewayCB,
		metrics,
	)

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
		CreateHold:  createHoldUseCase,
		ReleaseHold: releaseHoldUseCase,
		ExpireHold:  expireHoldUseCase,
		CaptureHold: captureHoldUseCase,
		PaymentSaga: saga.NewPaymentSagaOrchestrator(
			createHoldUseCase,
			gateway.NewMockAntiFraudService(),
			captureHoldUseCase,
			releaseHoldUseCase,
			protectedGateway,
			metrics,
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
