package httpadapter

import (
	"context"
	"net/http"

	"financial-ledger/internal/platform/observability"

	"github.com/go-chi/chi/v5"
)

func NewRouter(
	accountHandler *AccountHandler,
	readinessChecker func(context.Context) error,
	transactionHandler *TransactionHandler,
	accountEntriesHandler *AccountEntriesHandler,
	reversalHandlers ...*ReversalHandler,
) http.Handler {
	return newRouter(
		accountHandler,
		readinessChecker,
		transactionHandler,
		accountEntriesHandler,
		nil,
		nil,
		reversalHandlers...,
	)
}

func NewRouterWithWallet(
	accountHandler *AccountHandler,
	readinessChecker func(context.Context) error,
	transactionHandler *TransactionHandler,
	accountEntriesHandler *AccountEntriesHandler,
	walletHandler *WalletHandler,
	reversalHandlers ...*ReversalHandler,
) http.Handler {
	return newRouter(
		accountHandler,
		readinessChecker,
		transactionHandler,
		accountEntriesHandler,
		walletHandler,
		nil,
		reversalHandlers...,
	)
}

func NewRouterWithSaga(
	accountHandler *AccountHandler,
	readinessChecker func(context.Context) error,
	transactionHandler *TransactionHandler,
	accountEntriesHandler *AccountEntriesHandler,
	walletHandler *WalletHandler,
	paymentHandler *PaymentHandler,
	reversalHandlers ...*ReversalHandler,
) http.Handler {
	return newRouter(
		accountHandler,
		readinessChecker,
		transactionHandler,
		accountEntriesHandler,
		walletHandler,
		paymentHandler,
		reversalHandlers...,
	)
}

func newRouter(
	accountHandler *AccountHandler,
	readinessChecker func(context.Context) error,
	transactionHandler *TransactionHandler,
	accountEntriesHandler *AccountEntriesHandler,
	walletHandler *WalletHandler,
	paymentHandler *PaymentHandler,
	reversalHandlers ...*ReversalHandler,
) http.Handler {
	router := chi.NewRouter()

	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Idempotency-Key, Authorization, X-Request-ID")
			w.Header().Set("Access-Control-Expose-Headers", "X-Trace-ID, X-Request-ID")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	})

	healthHandler := NewHealthHandler(readinessChecker)

	router.Get("/health", healthHandler.Live)
	router.Get("/health/live", healthHandler.Live)
	router.Get("/health/ready", healthHandler.Ready)
	router.Get("/metrics", observability.MetricsHandler().ServeHTTP)
	router.Post("/accounts", accountHandler.Create)
	if walletHandler != nil {
		router.Post("/wallets", walletHandler.Create)
		if walletHandler.balanceEnabled {
			router.Get("/wallets/{walletID}/balance", walletHandler.GetBalance)
		}
		if walletHandler.depositEnabled {
			router.Post(
				"/wallets/{walletID}/deposits",
				walletHandler.Deposit,
			)
		}
		if walletHandler.holdEnabled {
			router.Post(
				"/wallets/{walletID}/holds",
				walletHandler.CreateHold,
			)
			router.Post(
				"/holds/{holdID}/release",
				walletHandler.ReleaseHold,
			)
			router.Post(
				"/holds/{holdID}/expire",
				walletHandler.ExpireHold,
			)
			router.Post(
				"/holds/{holdID}/capture",
				walletHandler.CaptureHold,
			)
		}
		if walletHandler.withdrawEnabled {
			router.Post(
				"/wallets/{walletID}/withdrawals",
				walletHandler.Withdraw,
			)
		}
		if walletHandler.transferEnabled {
			router.Post(
				"/wallets/{walletID}/transfers",
				walletHandler.Transfer,
			)
		}
		if walletHandler.queriesEnabled {
			router.Get("/wallets/{walletID}", walletHandler.Get)
			router.Get("/owners/{ownerID}/wallets", walletHandler.ListByOwner)
		}
	}

	if paymentHandler != nil {
		router.Post("/payments/checkout", paymentHandler.Checkout)
	}

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
