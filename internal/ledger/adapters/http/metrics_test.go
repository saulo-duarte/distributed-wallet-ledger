package httpadapter

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMetricsRouteIsAvailable(t *testing.T) {
	router := NewRouter(nil, func(context.Context) error { return nil }, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 on /metrics, got %d", rec.Code)
	}
}
