// Request tracing with OpenTelemetry. Traces are sent to an OTLP collector (Jaeger)
// when OTEL_EXPORTER_OTLP_ENDPOINT is set; otherwise trace IDs are still generated
// so every log line can be correlated with its request.
package main

import (
	"context"
	"log/slog"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

const serviceName = "flashcards-api"

func init() {
	opts := []sdktrace.TracerProviderOption{sdktrace.WithResource(newResource())}
	if os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") != "" {
		exporter, err := otlptracehttp.New(context.Background())
		if err != nil {
			slog.Error("tracing exporter disabled", "error", err)
		} else {
			opts = append(opts, sdktrace.WithBatcher(exporter))
		}
	}
	otel.SetTracerProvider(sdktrace.NewTracerProvider(opts...))
	// Accept and forward the W3C "traceparent" header so traces can span several services.
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
}

// newResource describes this service; OTEL_SERVICE_NAME overrides the default name.
func newResource() *resource.Resource {
	res, err := resource.Merge(resource.Default(), resource.NewSchemaless(semconv.ServiceName(serviceName)))
	if err != nil {
		return resource.Default()
	}
	if name := os.Getenv("OTEL_SERVICE_NAME"); name != "" {
		res, _ = resource.Merge(res, resource.NewSchemaless(semconv.ServiceName(name)))
	}
	return res
}
