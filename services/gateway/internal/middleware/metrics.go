package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	requestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	requestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	authFailures = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "auth_failures_total",
			Help: "Total number of authentication failures",
		},
		[]string{"reason"},
	)

	activeSessions = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "active_sessions",
			Help: "Number of active sessions",
		},
	)
)

func init() {
	prometheus.MustRegister(requestsTotal, requestDuration, authFailures, activeSessions)
}

// MetricsHandler returns the Prometheus metrics endpoint handler.
func MetricsHandler() http.Handler {
	return promhttp.Handler()
}

// MetricsMiddleware records request count and duration for all requests.
func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip metrics endpoint itself
		if r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()
		rw := &metricsRecorder{ResponseWriter: w, status: 200}
		next.ServeHTTP(rw, r)

		duration := time.Since(start).Seconds()
		status := strconv.Itoa(rw.status)
		path := normalizePath(r.URL.Path)

		requestsTotal.WithLabelValues(r.Method, path, status).Inc()
		requestDuration.WithLabelValues(r.Method, path).Observe(duration)
		if rw.status == http.StatusUnauthorized {
			authFailures.WithLabelValues("unauthorized").Inc()
		}
	})
}

// IncAuthFailure increments the auth failure counter.
func IncAuthFailure(reason string) {
	authFailures.WithLabelValues(reason).Inc()
}

// SetActiveSessions sets the active sessions gauge.
func SetActiveSessions(n int) {
	activeSessions.Set(float64(n))
}

// statusRecorder captures the HTTP status code.
type metricsRecorder struct {
	http.ResponseWriter
	status int
}

func (sr *metricsRecorder) WriteHeader(code int) {
	sr.status = code
	sr.ResponseWriter.WriteHeader(code)
}

// normalizePath strips IDs from paths for metrics labels.
// /api/v1/users/123 → /api/v1/users/{id}
func normalizePath(path string) string {
	parts := splitPath(path)
	for i, p := range parts {
		if isID(p) && i > 0 {
			parts[i] = "{id}"
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
			parts = append(parts, "/")
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

func isID(s string) bool {
	if len(s) != 36 {
		return false
	}
	// UUID format: 8-4-4-4-12
	return s[8] == '-' && s[13] == '-' && s[18] == '-' && s[23] == '-'
}

func joinPath(parts []string) string {
	result := ""
	for _, p := range parts {
		result += p
	}
	return result
}
