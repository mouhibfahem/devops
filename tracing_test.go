package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// recordSpans swaps in a tracer provider that keeps finished spans in memory.
func recordSpans(t *testing.T) *tracetest.SpanRecorder {
	t.Helper()
	rec := tracetest.NewSpanRecorder()
	old := otel.GetTracerProvider()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec))
	otel.SetTracerProvider(tp)
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()); otel.SetTracerProvider(old) })
	return rec
}

func TestEachRequestCreatesANamedSpan(t *testing.T) {
	captureLogs(t)
	spans := recordSpans(t)
	withObservability(NewRouter(NewStore())).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/cards/1", nil))

	ended := spans.Ended()
	if len(ended) != 1 || ended[0].Name() != "GET /cards/{id}" {
		t.Fatalf("expected one span named after the route, got %v", ended)
	}
}

func TestIncomingTraceparentIsContinued(t *testing.T) {
	buf := captureLogs(t)
	recordSpans(t)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	withObservability(NewRouter(NewStore())).ServeHTTP(httptest.NewRecorder(), req)

	if entry := lastLog(t, buf); entry["trace_id"] != "4bf92f3577b34da6a3ce929d0e0e4736" {
		t.Fatalf("log should carry the incoming trace id, got %v", entry["trace_id"])
	}
}
