package main

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// captureLogs redirects the default logger to a buffer for the duration of a test.
func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	old := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(old) })
	return &buf
}

// lastLog decodes the last JSON log line written to buf.
func lastLog(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	var entry map[string]any
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &entry); err != nil {
		t.Fatalf("log line is not JSON: %q (%v)", lines[len(lines)-1], err)
	}
	return entry
}

func serve(h http.Handler, req *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	withObservability(h).ServeHTTP(w, req)
	return w
}

func TestRequestLogIsStructured(t *testing.T) {
	buf := captureLogs(t)
	w := serve(NewRouter(NewStore()), httptest.NewRequest(http.MethodGet, "/cards?topic=docker", nil))

	entry := lastLog(t, buf)
	if entry["msg"] != "request" || entry["level"] != "INFO" || entry["method"] != "GET" ||
		entry["path"] != "/cards" || entry["query"] != "topic=docker" || entry["status"] != float64(200) {
		t.Fatalf("unexpected log entry: %v", entry)
	}
	if _, ok := entry["duration_ms"].(float64); !ok {
		t.Fatalf("duration_ms missing: %v", entry)
	}
	if id := w.Header().Get(requestIDHeader); id == "" || entry["request_id"] != id {
		t.Fatalf("request id header %q does not match log %v", id, entry["request_id"])
	}
}

func TestRequestIDIsPropagated(t *testing.T) {
	captureLogs(t)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set(requestIDHeader, "abc123")
	if got := serve(NewRouter(NewStore()), req).Header().Get(requestIDHeader); got != "abc123" {
		t.Fatalf("expected incoming request id to be reused, got %q", got)
	}
}

func TestClientErrorsAreLoggedAsWarnings(t *testing.T) {
	buf := captureLogs(t)
	serve(NewRouter(NewStore()), httptest.NewRequest(http.MethodGet, "/cards/999", nil))
	if entry := lastLog(t, buf); entry["level"] != "WARN" || entry["status"] != float64(404) {
		t.Fatalf("expected a WARN entry with status 404, got %v", entry)
	}
}

func TestPanicIsRecoveredAndLogged(t *testing.T) {
	buf := captureLogs(t)
	boom := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("boom") })
	w := serve(boom, httptest.NewRequest(http.MethodGet, "/", nil))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
	if !strings.Contains(buf.String(), `"msg":"panic while handling request"`) {
		t.Fatalf("panic was not logged: %s", buf.String())
	}
	if entry := lastLog(t, buf); entry["level"] != "ERROR" || entry["status"] != float64(500) {
		t.Fatalf("expected an ERROR request entry with status 500, got %v", entry)
	}
}

func TestSecurityHeaders(t *testing.T) {
	captureLogs(t)
	w := serve(NewRouter(NewStore()), httptest.NewRequest(http.MethodGet, "/cards", nil))
	for header, want := range map[string]string{
		"X-Content-Type-Options":       "nosniff",
		"Cross-Origin-Resource-Policy": "same-origin",
		"Cache-Control":                "no-store",
		"Content-Security-Policy":      "default-src 'none'; frame-ancestors 'none'",
	} {
		if got := w.Header().Get(header); got != want {
			t.Errorf("%s = %q, want %q", header, got, want)
		}
	}
}
