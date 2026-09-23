package observability

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	HTTPRequestsTotal           *prometheus.CounterVec
	HTTPRequestDurationSeconds  *prometheus.HistogramVec
	TransactionsTotal           *prometheus.CounterVec
	ActiveHoldsGauge            *prometheus.GaugeVec
	OutboxEventsPublishedTotal  *prometheus.CounterVec
	SQSMessagesConsumedTotal    *prometheus.CounterVec
	SagaExecutionsTotal         *prometheus.CounterVec
	CircuitBreakerStateGauge    *prometheus.GaugeVec
	CircuitBreakerRejectionsTotal *prometheus.CounterVec
}

func NewMetrics(reg prometheus.Registerer) *Metrics {
	factory := promauto.With(reg)

	return &Metrics{
		HTTPRequestsTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "ledger_http_requests_total",
				Help: "Total number of HTTP requests processed by the ledger API.",
			},
			[]string{"method", "route", "status_code"},
		),
		HTTPRequestDurationSeconds: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "ledger_http_request_duration_seconds",
				Help: "Histogram of HTTP request latency in seconds.",
				Buckets: []float64{
					0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0,
				},
			},
			[]string{"method", "route", "status_code"},
		),
		TransactionsTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "ledger_transactions_total",
				Help: "Total number of double-entry ledger transactions posted.",
			},
			[]string{"status", "currency"},
		),
		ActiveHoldsGauge: factory.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ledger_active_holds_count",
				Help: "Current number of active funds holds.",
			},
			[]string{"currency"},
		),
		OutboxEventsPublishedTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "ledger_outbox_events_published_total",
				Help: "Total number of events successfully published to SNS.",
			},
			[]string{"event_type", "status"},
		),
		SQSMessagesConsumedTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "ledger_sqs_messages_consumed_total",
				Help: "Total number of projection messages consumed from SQS.",
			},
			[]string{"queue_name", "status"},
		),
		SagaExecutionsTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "ledger_saga_executions_total",
				Help: "Total number of payment saga executions by scenario and result.",
			},
			[]string{"scenario", "status"},
		),
		CircuitBreakerStateGauge: factory.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ledger_circuit_breaker_state",
				Help: "State of circuit breakers (0=closed, 1=half_open, 2=open).",
			},
			[]string{"name", "state"},
		),
		CircuitBreakerRejectionsTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "ledger_circuit_breaker_rejections_total",
				Help: "Total number of requests rejected by an open circuit breaker.",
			},
			[]string{"name"},
		),
	}
}

var (
	defaultMetricsOnce sync.Once
	defaultMetricsInst *Metrics
)

func DefaultMetrics() *Metrics {
	defaultMetricsOnce.Do(func() {
		defaultMetricsInst = NewMetrics(prometheus.DefaultRegisterer)
	})
	return defaultMetricsInst
}

func MetricsHandler() http.Handler {
	return promhttp.Handler()
}

func (m *Metrics) HTTPMetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &statusRecordingResponseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(rw, r)

		duration := time.Since(start).Seconds()
		route := r.URL.Path
		if rCtx := chi.RouteContext(r.Context()); rCtx != nil && rCtx.RoutePattern() != "" {
			route = rCtx.RoutePattern()
		}
		statusStr := strconv.Itoa(rw.statusCode)

		m.HTTPRequestsTotal.WithLabelValues(r.Method, route, statusStr).Inc()
		m.HTTPRequestDurationSeconds.WithLabelValues(r.Method, route, statusStr).Observe(duration)
	})
}

type statusRecordingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *statusRecordingResponseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func (m *Metrics) RecordTransaction(status, currency string) {
	if m == nil || m.TransactionsTotal == nil {
		return
	}
	m.TransactionsTotal.WithLabelValues(status, currency).Inc()
}

func (m *Metrics) RecordSaga(scenario, status string) {
	if m == nil || m.SagaExecutionsTotal == nil {
		return
	}
	m.SagaExecutionsTotal.WithLabelValues(scenario, status).Inc()
}

func (m *Metrics) RecordOutboxPublish(eventType, status string) {
	if m == nil || m.OutboxEventsPublishedTotal == nil {
		return
	}
	m.OutboxEventsPublishedTotal.WithLabelValues(eventType, status).Inc()
}

func (m *Metrics) RecordSQSConsumed(queueName, status string) {
	if m == nil || m.SQSMessagesConsumedTotal == nil {
		return
	}
	m.SQSMessagesConsumedTotal.WithLabelValues(queueName, status).Inc()
}

func (m *Metrics) RecordCircuitBreakerState(name string, state string) {
	if m == nil || m.CircuitBreakerStateGauge == nil {
		return
	}
	for _, s := range []string{"closed", "half_open", "open"} {
		val := float64(0)
		if s == state {
			val = 1
		}
		m.CircuitBreakerStateGauge.WithLabelValues(name, s).Set(val)
	}
}

func (m *Metrics) RecordCircuitBreakerRejection(name string) {
	if m == nil || m.CircuitBreakerRejectionsTotal == nil {
		return
	}
	m.CircuitBreakerRejectionsTotal.WithLabelValues(name).Inc()
}
