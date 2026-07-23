// Package metrics provides standardized Prometheus metrics for all GGID services.
// Each service imports this package and registers the /metrics endpoint.
package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Standard RED metrics (Rate, Errors, Duration) shared across all services.
var (
	// Request rate
	RequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ggid_requests_total",
			Help: "Total HTTP requests by service, method, path, status",
		},
		[]string{"service", "method", "path", "status"},
	)

	// Error rate
	ErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ggid_errors_total",
			Help: "Total HTTP 5xx errors by service",
		},
		[]string{"service", "path"},
	)

	// Request duration
	RequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ggid_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"service", "method", "path"},
	)

	// In-flight requests
	InFlightRequests = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ggid_inflight_requests",
			Help: "Current in-flight HTTP requests",
		},
		[]string{"service"},
	)
)

func init() {
	prometheus.MustRegister(RequestsTotal, ErrorsTotal, RequestDuration, InFlightRequests)
}

// Handler returns the standard /metrics HTTP handler.
func Handler() http.Handler {
	return promhttp.Handler()
}

// Middleware wraps an http.Handler with RED metrics collection.
// serviceName should be the service identifier (e.g., "gateway", "auth", "mcp").
func Middleware(serviceName string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip metrics endpoint itself
		if r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()
		InFlightRequests.WithLabelValues(serviceName).Inc()

		// Wrap response writer to capture status code
		ww := &statusWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(ww, r)

		duration := time.Since(start).Seconds()
		status := http.StatusText(ww.status)
		path := normalizePath(r.URL.Path)

		RequestsTotal.WithLabelValues(serviceName, r.Method, path, status).Inc()
		RequestDuration.WithLabelValues(serviceName, r.Method, path).Observe(duration)
		InFlightRequests.WithLabelValues(serviceName).Dec()

		if ww.status >= 500 {
			ErrorsTotal.WithLabelValues(serviceName, path).Inc()
		}
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

// normalizePath strips IDs from paths for better cardinality.
// /api/v1/users/550e8400-... → /api/v1/users/:id
func normalizePath(path string) string {
	parts := splitPath(path)
	for i, p := range parts {
		if looksLikeID(p) {
			parts[i] = ":id"
		}
	}
	return joinPath(parts)
}

func splitPath(path string) []string {
	var parts []string
	current := ""
	for _, c := range path {
		if c == '/' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

func joinPath(parts []string) string {
	result := ""
	for _, p := range parts {
		result += "/" + p
	}
	if result == "" {
		return "/"
	}
	return result
}

func looksLikeID(s string) bool {
	if len(s) < 8 {
		return false
	}
	// UUID-like
	if len(s) == 36 && s[8] == '-' {
		return true
	}
	// Numeric ID
	isNum := true
	for _, c := range s {
		if c < '0' || c > '9' {
			isNum = false
			break
		}
	}
	if isNum && len(s) > 3 {
		return true
	}
	return false
}
