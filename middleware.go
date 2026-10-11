// Observability middleware: request IDs, structured JSON logs and panic recovery.
// Kept out of main.go so the API itself stays small.
package main

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"os"
	"runtime/debug"
	"time"
)

const requestIDHeader = "X-Request-ID"

func init() {
	// Every log line (including the standard "log" package) is written as JSON on stdout.
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
}

// statusRecorder remembers the status code written by a handler.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func newRequestID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(b)
}

// withObservability wraps the API: it tags each request with an ID, recovers from
// panics and writes one structured log line per request.
func withObservability(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		id := r.Header.Get(requestIDHeader)
		if id == "" {
			id = newRequestID()
		}
		w.Header().Set(requestIDHeader, id)
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		defer func() {
			if err := recover(); err != nil {
				slog.Error("panic while handling request", "request_id", id, "error", err, "stack", string(debug.Stack()))
				writeError(rec, http.StatusInternalServerError, "internal server error")
			}
			logRequest(r, rec.status, time.Since(start), id)
		}()
		next.ServeHTTP(rec, r)
	})
}

// logRequest picks the level from the status: errors are easy to filter later.
func logRequest(r *http.Request, status int, d time.Duration, id string) {
	level := slog.LevelInfo
	switch {
	case status >= 500:
		level = slog.LevelError
	case status >= 400:
		level = slog.LevelWarn
	}
	slog.Log(r.Context(), level, "request",
		"request_id", id,
		"method", r.Method,
		"path", r.URL.Path,
		"query", r.URL.RawQuery,
		"status", status,
		"duration_ms", float64(d.Microseconds())/1000,
		"remote_addr", r.RemoteAddr,
		"user_agent", r.UserAgent(),
	)
}
