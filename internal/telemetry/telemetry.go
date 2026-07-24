package telemetry

import (
	"context"
	"log"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

// InitTracer initializes an OTLP exporter, and configures the corresponding trace provider.
// It returns a cleanup function to be called when the application shuts down.
func InitTracer(ctx context.Context, serviceName string) (func(context.Context) error, error) {
	noOpShutdown := func(context.Context) error { return nil }
	if os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") == "" {
		return noOpShutdown, nil
	}

	// The otlptracehttp exporter automatically reads OTEL_EXPORTER_OTLP_ENDPOINT
	// and OTEL_EXPORTER_OTLP_HEADERS from the environment.
	exporter, err := otlptracehttp.New(ctx)
	if err != nil {
		log.Printf("Failed to create OTLP trace exporter: %v", err)
		return nil, err
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
		),
	)
	if err != nil {
		log.Printf("Failed to create resource: %v", err)
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	// Set the global TracerProvider
	otel.SetTracerProvider(tp)

	return tp.Shutdown, nil
}
