package httpadapter

import (
	"context"
	"net/http"
	"time"

	"financial-ledger/internal/platform/httpx"
)

const readinessTimeout = 2 * time.Second

type HealthHandler struct {
	readinessChecker func(context.Context) error
}

func NewHealthHandler(
	readinessChecker func(context.Context) error,
) *HealthHandler {
	return &HealthHandler{
		readinessChecker: readinessChecker,
	}
}

func (h *HealthHandler) Live(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(
		w,
		http.StatusOK,
		healthResponse{Status: "ok"},
	)
}

func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	if h.readinessChecker == nil {
		httpx.WriteJSON(
			w,
			http.StatusServiceUnavailable,
			healthResponse{Status: "not_ready"},
		)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), readinessTimeout)
	defer cancel()

	if err := h.readinessChecker(ctx); err != nil {
		httpx.WriteJSON(
			w,
			http.StatusServiceUnavailable,
			healthResponse{Status: "not_ready"},
		)
		return
	}

	httpx.WriteJSON(
		w,
		http.StatusOK,
		healthResponse{Status: "ready"},
	)
}

type healthResponse struct {
	Status string `json:"status"`
}
