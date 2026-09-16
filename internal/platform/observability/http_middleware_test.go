package observability

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPRequestLoggerGeneratesAndPropagatesRequestID(t *testing.T) {
	var output bytes.Buffer
	logger, err := NewLogger(LoggingConfig{
		Level:  "info",
		Writer: &output,
	})
	if err != nil {
		t.Fatalf("create logger: %v", err)
	}

	handler := HTTPRequestLogger(logger)(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		requestID := RequestIDFromContext(r.Context())
		if requestID == "" {
			t.Fatal("expected request ID in context")
		}

		w.WriteHeader(http.StatusNoContent)
	}))

	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	requestID := recorder.Header().Get(RequestIDHeader)
	if requestID == "" {
		t.Fatal("expected request ID response header")
	}

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("unexpected status code: got %d", recorder.Code)
	}

	var record map[string]any
	if err := json.Unmarshal(output.Bytes(), &record); err != nil {
		t.Fatalf("decode log: %v", err)
	}

	if record["msg"] != "http_request_completed" {
		t.Fatalf("unexpected log message: %v", record["msg"])
	}
	if record["request_id"] != requestID {
		t.Fatalf("unexpected log request ID: got %v, want %s", record["request_id"], requestID)
	}
	if record["status_code"] != float64(http.StatusNoContent) {
		t.Fatalf("unexpected logged status code: %v", record["status_code"])
	}
}

func TestHTTPRequestLoggerUsesIncomingRequestID(t *testing.T) {
	var output bytes.Buffer
	logger, err := NewLogger(LoggingConfig{
		Level:  "info",
		Writer: &output,
	})
	if err != nil {
		t.Fatalf("create logger: %v", err)
	}

	handler := HTTPRequestLogger(logger)(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		if got := RequestIDFromContext(r.Context()); got != "request-123" {
			t.Fatalf("unexpected context request ID: %q", got)
		}
		_, _ = w.Write([]byte("ok"))
	}))

	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	request.Header.Set(RequestIDHeader, "request-123")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if got := recorder.Header().Get(RequestIDHeader); got != "request-123" {
		t.Fatalf("unexpected response request ID: %q", got)
	}
}

func TestHTTPRequestLoggerLogsClientErrorsAsWarning(t *testing.T) {
	var output bytes.Buffer
	logger, err := NewLogger(LoggingConfig{
		Level:  "info",
		Writer: &output,
	})
	if err != nil {
		t.Fatalf("create logger: %v", err)
	}

	handler := HTTPRequestLogger(logger)(http.HandlerFunc(func(
		w http.ResponseWriter,
		_ *http.Request,
	) {
		w.WriteHeader(http.StatusBadRequest)
	}))

	handler.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest(http.MethodGet, "/invalid", nil),
	)

	var record map[string]any
	if err := json.Unmarshal(output.Bytes(), &record); err != nil {
		t.Fatalf("decode log: %v", err)
	}

	if record["level"] != slog.LevelWarn.String() {
		t.Fatalf("unexpected log level: %v", record["level"])
	}
}
