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
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load application config: %w", err)
	}

	logger, err := observability.NewLogger(observability.LoggingConfig{
		Level: cfg.LogLevel,
	})
	if err != nil {
		return fmt.Errorf("initialize logger: %w", err)
	}

	logger.Info("application_starting", slog.String("address", cfg.APIAddress))

	dependencies, err := bootstrap.New(ctx, cfg)
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
	router := httpadapter.NewRouter(
		accountHandler,
		dependencies.CheckReadiness,
	)
	handler := observability.HTTPRequestLogger(logger)(router)

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
