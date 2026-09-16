package httpadapter

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthHandlerLiveDoesNotCheckDependencies(t *testing.T) {
	checkerCalled := false
	handler := NewHealthHandler(func(context.Context) error {
		checkerCalled = true
		return nil
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/health/live", nil)

	handler.Live(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: got %d", recorder.Code)
	}

	if recorder.Body.String() != "{\"status\":\"ok\"}\n" {
		t.Fatalf("unexpected response body: %q", recorder.Body.String())
	}

	if checkerCalled {
		t.Fatal("live endpoint must not check dependencies")
	}
}

func TestHealthHandlerReadyReturnsOKWhenDependencyIsAvailable(t *testing.T) {
	checkerCalled := false
	handler := NewHealthHandler(func(ctx context.Context) error {
		checkerCalled = true

		if _, ok := ctx.Deadline(); !ok {
			t.Fatal("readiness check must have a deadline")
		}

		return nil
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/health/ready", nil)

	handler.Ready(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: got %d", recorder.Code)
	}

	if recorder.Body.String() != "{\"status\":\"ready\"}\n" {
		t.Fatalf("unexpected response body: %q", recorder.Body.String())
	}

	if !checkerCalled {
		t.Fatal("expected readiness checker to be called")
	}
}

func TestHealthHandlerReadyReturnsServiceUnavailableWhenDependencyFails(t *testing.T) {
	handler := NewHealthHandler(func(context.Context) error {
		return errors.New("database unavailable")
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/health/ready", nil)

	handler.Ready(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("unexpected status code: got %d", recorder.Code)
	}

	if recorder.Body.String() != "{\"status\":\"not_ready\"}\n" {
		t.Fatalf("unexpected response body: %q", recorder.Body.String())
	}
}

func TestHealthHandlerReadyReturnsServiceUnavailableWhenCheckerIsMissing(t *testing.T) {
	handler := NewHealthHandler(nil)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/health/ready", nil)

	handler.Ready(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("unexpected status code: got %d", recorder.Code)
	}
}

func TestNewRouterRegistersHealthEndpoints(t *testing.T) {
	router := NewRouter(nil, func(context.Context) error { return nil }, nil, nil)

	tests := []struct {
		name       string
		method     string
		path       string
		statusCode int
	}{
		{name: "legacy health", method: http.MethodGet, path: "/health", statusCode: http.StatusOK},
		{name: "live", method: http.MethodGet, path: "/health/live", statusCode: http.StatusOK},
		{name: "ready", method: http.MethodGet, path: "/health/ready", statusCode: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(tt.method, tt.path, nil)

			router.ServeHTTP(recorder, request)

			if recorder.Code != tt.statusCode {
				t.Fatalf(
					"unexpected status code: got %d, want %d",
					recorder.Code,
					tt.statusCode,
				)
			}
		})
	}
}
