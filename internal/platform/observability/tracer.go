package observability

import (
	"context"
	"fmt"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

const defaultServiceName = "goledge-api"

type TracerConfig struct {
	ServiceName  string
	OTLPEndpoint string
	Insecure     bool
	Disabled     bool
}

func InitTracer(ctx context.Context, cfg TracerConfig) (func(context.Context) error, error) {
	if cfg.Disabled {
		otel.SetTracerProvider(noop.NewTracerProvider())
		return func(context.Context) error { return nil }, nil
	}

	if strings.TrimSpace(cfg.ServiceName) == "" {
		cfg.ServiceName = defaultServiceName
	}

	res, err := resource.Merge(
		resource.Default(),
		resource.NewSchemaless(
			semconv.ServiceNameKey.String(cfg.ServiceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("create tracer resource: %w", err)
	}

	exporter, err := newExporter(ctx, cfg)
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	return tp.Shutdown, nil
}

func newExporter(ctx context.Context, cfg TracerConfig) (sdktrace.SpanExporter, error) {
	if strings.TrimSpace(cfg.OTLPEndpoint) == "" {
		exp, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
		if err != nil {
			return nil, fmt.Errorf("create stdout tracer exporter: %w", err)
		}
		return exp, nil
	}

	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(cfg.OTLPEndpoint),
	}
	if cfg.Insecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	}

	exp, err := otlptracegrpc.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("create otlp grpc tracer exporter: %w", err)
	}
	return exp, nil
}

func Tracer(name ...string) trace.Tracer {
	tracerName := defaultServiceName
	if len(name) > 0 && strings.TrimSpace(name[0]) != "" {
		tracerName = name[0]
	}
	return otel.GetTracerProvider().Tracer(tracerName)
}

func StartSpan(
	ctx context.Context,
	spanName string,
	opts ...trace.SpanStartOption,
) (context.Context, trace.Span) {
	return Tracer().Start(ctx, spanName, opts...)
}

func TraceIDFromContext(ctx context.Context) string {
	spanCtx := trace.SpanFromContext(ctx).SpanContext()
	if spanCtx.IsValid() {
		return spanCtx.TraceID().String()
	}
	return ""
}
