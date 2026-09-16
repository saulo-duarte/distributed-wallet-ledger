package httpadapter

import (
	"context"
	"net/http"
	"time"
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
	writeJSON(
		w,
		http.StatusOK,
		healthResponse{Status: "ok"},
	)
}

func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	if h.readinessChecker == nil {
		writeJSON(
			w,
			http.StatusServiceUnavailable,
			healthResponse{Status: "not_ready"},
		)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), readinessTimeout)
	defer cancel()

	if err := h.readinessChecker(ctx); err != nil {
		writeJSON(
			w,
			http.StatusServiceUnavailable,
			healthResponse{Status: "not_ready"},
		)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		healthResponse{Status: "ready"},
	)
}

type healthResponse struct {
	Status string `json:"status"`
}
