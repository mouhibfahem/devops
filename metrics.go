// Prometheus metrics: request count and response time, exposed on /metrics.
package main

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	httpRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Number of HTTP requests handled, by method, route and status code.",
	}, []string{"method", "route", "status"})

	httpDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "Time spent handling HTTP requests, by method and route.",
		Buckets: []float64{.0005, .001, .0025, .005, .01, .025, .05, .1, .25, .5, 1},
	}, []string{"method", "route"})

	httpInFlight = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "http_requests_in_flight",
		Help: "Number of HTTP requests currently being handled.",
	})
)

// route returns the matched route pattern (e.g. "GET /cards/{id}") rather than the raw
// path, so /cards/1, /cards/2... share one label and the number of series stays small.
func route(r *http.Request) string {
	if r.Pattern == "" {
		return "unmatched"
	}
	return r.Pattern
}

func observeRequest(r *http.Request, status int, d time.Duration) {
	httpRequests.WithLabelValues(r.Method, route(r), strconv.Itoa(status)).Inc()
	httpDuration.WithLabelValues(r.Method, route(r)).Observe(d.Seconds())
}
