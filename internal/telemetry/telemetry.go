package telemetry

import (
	"context"
	"log/slog"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

// Init setup distributed tracing and structured logging.
func Init(ctx context.Context, serviceName string) (*sdktrace.TracerProvider, error) {
	// 1. Initialize OpenTelemetry Exporter (OTLP gRPC)
	// This will use the OTEL_EXPORTER_OTLP_ENDPOINT environment variable if set
	exporter, err := otlptracegrpc.New(ctx)
	if err != nil {
		return nil, err
	}

	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
		),
	)
	if err != nil {
		return nil, err
	}

	// 2. Initialize TracerProvider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	// Set global TracerProvider and TextMapPropagator
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// 3. Initialize slog with custom handler that injects TraceID
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}
	handler := &TraceContextHandler{
		Handler: slog.NewJSONHandler(os.Stdout, opts),
	}
	logger := slog.New(handler)
	slog.SetDefault(logger)

	return tp, nil
}

// TraceContextHandler is a slog.Handler that adds trace_id and span_id to logs.
type TraceContextHandler struct {
	slog.Handler
}

// Handle adds trace and span ids to the record if they are present in the context.
func (h *TraceContextHandler) Handle(ctx context.Context, r slog.Record) error {
	span := trace.SpanContextFromContext(ctx)
	if span.IsValid() {
		r.AddAttrs(
			slog.String("trace_id", span.TraceID().String()),
			slog.String("span_id", span.SpanID().String()),
		)
	}
	return h.Handler.Handle(ctx, r)
}
