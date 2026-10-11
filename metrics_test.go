package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestMetricsUseRoutePatterns(t *testing.T) {
	captureLogs(t)
	h := withObservability(NewRouter(NewStore()))
	before := testutil.ToFloat64(httpRequests.WithLabelValues("GET", "GET /cards/{id}", "200"))

	for _, path := range []string{"/cards/1", "/cards/2"} {
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, path, nil))
	}
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/nope", nil))

	if got := testutil.ToFloat64(httpRequests.WithLabelValues("GET", "GET /cards/{id}", "200")) - before; got != 2 {
		t.Fatalf("expected 2 requests counted under the route pattern, got %v", got)
	}
	if got := testutil.ToFloat64(httpRequests.WithLabelValues("GET", "unmatched", "404")); got < 1 {
		t.Fatalf("expected unknown paths to be counted as 'unmatched', got %v", got)
	}
}

func TestMetricsEndpoint(t *testing.T) {
	captureLogs(t)
	h := withObservability(NewRouter(NewStore()))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/health", nil))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body, _ := io.ReadAll(w.Body)
	for _, want := range []string{"http_requests_total", "http_request_duration_seconds_bucket", "http_requests_in_flight", "go_goroutines"} {
		if !strings.Contains(string(body), want) {
			t.Errorf("/metrics does not expose %s", want)
		}
	}
}
