package observability

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"
)

func TestParseLevel(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  slog.Level
	}{
		{name: "empty defaults to info", input: "", want: slog.LevelInfo},
		{name: "info", input: "info", want: slog.LevelInfo},
		{name: "debug", input: "DEBUG", want: slog.LevelDebug},
		{name: "warning alias", input: "warning", want: slog.LevelWarn},
		{name: "error", input: "error", want: slog.LevelError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseLevel(tt.input)
			if err != nil {
				t.Fatalf("parse level: %v", err)
			}
			if got != tt.want {
				t.Fatalf("unexpected level: got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseLevelRejectsUnknownValue(t *testing.T) {
	if _, err := ParseLevel("verbose"); err == nil {
		t.Fatal("expected invalid log level error")
	}
}

func TestNewLoggerWritesJSON(t *testing.T) {
	var output bytes.Buffer

	logger, err := NewLogger(LoggingConfig{
		Level:  "info",
		Format: "json",
		Writer: &output,
	})
	if err != nil {
		t.Fatalf("create logger: %v", err)
	}

	logger.Info("account_created", slog.String("account_code", "cash"))

	var record map[string]any
	if err := json.Unmarshal(output.Bytes(), &record); err != nil {
		t.Fatalf("decode JSON log: %v", err)
	}

	if record["msg"] != "account_created" {
		t.Fatalf("unexpected message: %v", record["msg"])
	}
	if record["level"] != "INFO" {
		t.Fatalf("unexpected level: %v", record["level"])
	}
	if record["account_code"] != "cash" {
		t.Fatalf("unexpected account_code: %v", record["account_code"])
	}
}

func TestNewLoggerFiltersMessagesBelowConfiguredLevel(t *testing.T) {
	var output bytes.Buffer

	logger, err := NewLogger(LoggingConfig{
		Level:  "info",
		Writer: &output,
	})
	if err != nil {
		t.Fatalf("create logger: %v", err)
	}

	logger.Debug("debug_details")

	if output.Len() != 0 {
		t.Fatalf("expected debug message to be filtered, got %q", output.String())
	}
}
