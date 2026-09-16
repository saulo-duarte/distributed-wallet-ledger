package observability

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

type LoggingConfig struct {
	Level     string
	AddSource bool
	Writer    io.Writer
}

func NewLogger(cfg LoggingConfig) (*slog.Logger, error) {
	level, err := ParseLevel(cfg.Level)
	if err != nil {
		return nil, err
	}

	writer := cfg.Writer
	if writer == nil {
		writer = os.Stdout
	}

	handler := slog.NewJSONHandler(
		writer,
		&slog.HandlerOptions{
			Level:     level,
			AddSource: cfg.AddSource,
		},
	)

	return slog.New(handler), nil
}

func ParseLevel(value string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "info":
		return slog.LevelInfo, nil
	case "debug":
		return slog.LevelDebug, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("invalid log level %q", value)
	}
}
