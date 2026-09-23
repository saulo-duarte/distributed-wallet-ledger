package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"financial-ledger/internal/bootstrap"
	httpadapter "financial-ledger/internal/ledger/adapters/http"
	"financial-ledger/internal/platform/config"
	"financial-ledger/internal/platform/observability"
)

func run(ctx context.Context) error {
	cfg, err := config.Load(ctx)
	if err != nil {
		return fmt.Errorf("load application config: %w", err)
	}

	logger, err := observability.NewLogger(observability.LoggingConfig{
		Level:  cfg.LogLevel,
		Format: cfg.LogFormat,
	})
	if err != nil {
		return fmt.Errorf("initialize logger: %w", err)
	}

	tracerShutdown, err := observability.InitTracer(ctx, observability.TracerConfig{
		ServiceName:  cfg.OTELServiceName,
		OTLPEndpoint: cfg.OTELEndpoint,
		Insecure:     true,
		Disabled:     cfg.OTELDisabled,
	})
	if err != nil {
		return fmt.Errorf("initialize tracer: %w", err)
	}
	defer func() {
		_ = tracerShutdown(context.Background())
	}()

	metrics := observability.DefaultMetrics()

	logger.Info("application_starting", slog.String("address", cfg.APIAddress))

	dependencies, err := bootstrap.New(ctx, cfg, logger)
	if err != nil {
		logger.Error(
			"application_dependencies_initialization_failed",
			slog.Any("error", err),
		)
		return fmt.Errorf("initialize application dependencies: %w", err)
	}
	defer func() {
		logger.Info("application_shutdown_started")
		dependencies.Close()
		logger.Info("application_shutdown_completed")
	}()

	accountHandler := httpadapter.NewAccountHandler(
		dependencies.CreateAccount,
		logger,
	)
	walletHandler := httpadapter.NewWalletHandlerWithHolds(
		dependencies.CreateWallet,
		dependencies.GetWallet,
		dependencies.ListWalletsByOwner,
		dependencies.GetWalletBalance,
		dependencies.DepositWallet,
		dependencies.WithdrawWallet,
		dependencies.TransferWallet,
		dependencies.CreateHold,
		dependencies.ReleaseHold,
		dependencies.ExpireHold,
		dependencies.CaptureHold,
		logger,
	)
	transactionHandler := httpadapter.NewTransactionHandler(
		dependencies.PostTransaction,
		dependencies.GetTransaction,
		logger,
	)
	accountEntriesHandler := httpadapter.NewAccountEntriesHandler(
		dependencies.ListAccountEntries,
		logger,
	)
	reversalHandler := httpadapter.NewReversalHandler(
		dependencies.ReverseTransaction,
		logger,
	)
	paymentHandler := httpadapter.NewPaymentHandler(
		dependencies.PaymentSaga,
		logger,
	)
	router := httpadapter.NewRouterWithSaga(
		accountHandler,
		dependencies.CheckReadiness,
		transactionHandler,
		accountEntriesHandler,
		walletHandler,
		paymentHandler,
		reversalHandler,
	)
	handler := observability.HTTPTracingMiddleware(
		metrics.HTTPMetricsMiddleware(
			observability.HTTPRequestLogger(logger)(router),
		),
	)

	server := &http.Server{
		Addr:              cfg.APIAddress,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("http_server_started", slog.String("address", cfg.APIAddress))
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErrors:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		logger.Error("http_server_failed", slog.Any("error", err))
		return fmt.Errorf("HTTP server failed: %w", err)

	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(
			context.Background(),
			10*time.Second,
		)
		defer cancel()

		logger.Info("http_server_shutdown_started")
		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("http_server_shutdown_failed", slog.Any("error", err))
			return fmt.Errorf("shutdown HTTP server: %w", err)
		}

		logger.Info("http_server_shutdown_completed")
		return nil
	}
}
