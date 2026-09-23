package observability

import (
	"context"
	"testing"
)

func TestInitTracerDisabled(t *testing.T) {
	shutdown, err := InitTracer(context.Background(), TracerConfig{Disabled: true})
	if err != nil {
		t.Fatalf("unexpected init error: %v", err)
	}
	defer func() {
		_ = shutdown(context.Background())
	}()

	ctx, span := StartSpan(context.Background(), "test-span")
	defer span.End()

	traceID := TraceIDFromContext(ctx)
	if traceID != "" {
		t.Fatalf("expected empty trace id for disabled tracer, got %q", traceID)
	}
}

func TestInitTracerStdout(t *testing.T) {
	shutdown, err := InitTracer(context.Background(), TracerConfig{
		ServiceName: "test-service",
	})
	if err != nil {
		t.Fatalf("unexpected init error: %v", err)
	}
	defer func() {
		_ = shutdown(context.Background())
	}()

	ctx, span := StartSpan(context.Background(), "test-span-stdout")
	defer span.End()

	traceID := TraceIDFromContext(ctx)
	if traceID == "" {
		t.Fatal("expected valid trace id")
	}
}
