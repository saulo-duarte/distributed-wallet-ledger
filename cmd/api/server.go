package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"financial-ledger/internal/bootstrap"
	httpadapter "financial-ledger/internal/ledger/adapters/http"
	"financial-ledger/internal/platform/config"
)

func run(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load application config: %w", err)
	}

	dependencies, err := bootstrap.New(ctx, cfg)
	if err != nil {
		return fmt.Errorf("initialize application dependencies: %w", err)
	}
	defer dependencies.Close()

	accountHandler := httpadapter.NewAccountHandler(
		dependencies.CreateAccount,
	)
	router := httpadapter.NewRouter(accountHandler)

	server := &http.Server{
		Addr:              cfg.APIAddress,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	serverErrors := make(chan error, 1)
	go func() {
		log.Printf("financial-ledger API listening on %s", cfg.APIAddress)
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErrors:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("HTTP server failed: %w", err)

	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(
			context.Background(),
			10*time.Second,
		)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown HTTP server: %w", err)
		}

		return nil
	}
}
