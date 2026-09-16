package httpadapter

import (
	"context"
	"net/http"
)

func NewRouter(
	accountHandler *AccountHandler,
	readinessChecker func(context.Context) error,
	transactionHandler *TransactionHandler,
) http.Handler {
	mux := http.NewServeMux()
	healthHandler := NewHealthHandler(readinessChecker)

	mux.HandleFunc("GET /health", healthHandler.Live)
	mux.HandleFunc("GET /health/live", healthHandler.Live)
	mux.HandleFunc("GET /health/ready", healthHandler.Ready)
	mux.HandleFunc("POST /accounts", accountHandler.Create)

	if transactionHandler != nil {
		mux.HandleFunc("POST /transactions", transactionHandler.Post)
	}

	return mux
}
