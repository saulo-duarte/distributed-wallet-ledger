package httpx

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDecodeJSON(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/resource",
		bytes.NewBufferString(`{"name":"ledger"}`),
	)

	var payload struct {
		Name string `json:"name"`
	}

	if err := DecodeJSON(recorder, request, &payload); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}

	if payload.Name != "ledger" {
		t.Fatalf("unexpected name: %q", payload.Name)
	}
}

func TestDecodeJSONRejectsUnknownFields(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/resource",
		bytes.NewBufferString(`{"unknown":true}`),
	)

	var payload struct {
		Name string `json:"name"`
	}

	err := DecodeJSON(recorder, request, &payload)
	if err != ErrInvalidRequestBody {
		t.Fatalf("expected invalid body error, got %v", err)
	}
}

func TestDecodeJSONRejectsMultipleValues(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/resource",
		bytes.NewBufferString(`{"name":"one"}{"name":"two"}`),
	)

	var payload map[string]string
	err := DecodeJSON(recorder, request, &payload)
	if err != ErrMultipleJSONValues {
		t.Fatalf("expected multiple values error, got %v", err)
	}
}

func TestWriteError(t *testing.T) {
	recorder := httptest.NewRecorder()

	WriteError(
		recorder,
		http.StatusBadRequest,
		"invalid_request",
		"invalid request body",
		"request-123",
	)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status code: %d", recorder.Code)
	}

	var response ErrorResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode error response: %v", err)
	}

	if response.Code != "invalid_request" {
		t.Fatalf("unexpected error code: %q", response.Code)
	}
	if response.RequestID != "request-123" {
		t.Fatalf("unexpected request ID: %q", response.RequestID)
	}
}
