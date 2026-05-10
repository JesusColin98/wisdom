package observability

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/attribute"
)

var Tracer trace.Tracer = otel.Tracer("wisdom-engine")

// AttrString creates a string attribute.
func AttrString(key, value string) attribute.KeyValue {
	return attribute.String(key, value)
}

// AttrInt creates an int attribute.
func AttrInt(key string, value int) attribute.KeyValue {
	return attribute.Int(key, value)
}

// AttrFloat64 creates a float64 attribute.
func AttrFloat64(key string, value float64) attribute.KeyValue {
	return attribute.Float64(key, value)
}

// InitTracer initializes OpenTelemetry tracing with a Stdout exporter.
func InitTracer() func(context.Context) error {
	exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		slog.Error("Failed to create trace exporter", "error", err)
		return func(ctx context.Context) error { return nil }
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("wisdom-engine"),
		)),
	)
	otel.SetTracerProvider(tp)

	slog.Info("OTel Tracer initialized with Stdout exporter (L3 Deliberative active)")

	return tp.Shutdown
}
