package observability

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestMetricsMiddlewareAndRecording(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics := NewMetrics(reg)

	metrics.RecordTransaction("posted", "BRL")
	metrics.RecordSaga("checkout", "success")
	metrics.RecordOutboxPublish("wallet.transferred", "published")
	metrics.RecordSQSConsumed("wallet-projections", "processed")
	metrics.RecordCircuitBreakerState("payment_gateway", "open")
	metrics.RecordCircuitBreakerRejection("payment_gateway")

	handler := metrics.HTTPMetricsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/test-metric", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	families, err := reg.Gather()
	if err != nil {
		t.Fatalf("failed to gather metrics: %v", err)
	}

	if len(families) == 0 {
		t.Fatal("expected registered metrics families, got 0")
	}
}
