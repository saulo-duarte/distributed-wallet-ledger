package httpadapter

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func NewRouter(
	accountHandler *AccountHandler,
	readinessChecker func(context.Context) error,
	transactionHandler *TransactionHandler,
	accountEntriesHandler *AccountEntriesHandler,
	reversalHandlers ...*ReversalHandler,
) http.Handler {
	router := chi.NewRouter()
	healthHandler := NewHealthHandler(readinessChecker)

	router.Get("/health", healthHandler.Live)
	router.Get("/health/live", healthHandler.Live)
	router.Get("/health/ready", healthHandler.Ready)
	router.Post("/accounts", accountHandler.Create)

	if transactionHandler != nil {
		router.Post("/transactions", transactionHandler.Post)
		router.Get("/transactions/{transactionID}", transactionHandler.Get)
	}
	if accountEntriesHandler != nil {
		router.Get("/accounts/{accountID}/entries", accountEntriesHandler.List)
	}
	if len(reversalHandlers) > 0 && reversalHandlers[0] != nil {
		router.Post(
			"/transactions/{transactionID}/reversal",
			reversalHandlers[0].Reverse,
		)
	}

	return router
}
