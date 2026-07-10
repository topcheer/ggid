package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	responseTimeHist = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gateway_response_time_seconds",
			Help:    "Response time per route in seconds",
			Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"method", "path", "status"},
	)
)

// ResponseTimeTracker is a latency tracker that records per-request timing
// into Prometheus histograms and sets the X-Response-Time response header.
// It also maintains an in-memory p50/p99 approximation for logging.
type ResponseTimeTracker struct {
	mu      sync.Mutex
	samples []float64 // milliseconds, ring buffer
	head    int
	size    int
}

// NewResponseTimeTracker creates a tracker with the given sample window size.
func NewResponseTimeTracker(windowSize int) *ResponseTimeTracker {
	if windowSize <= 0 {
		windowSize = 1000
	}
	return &ResponseTimeTracker{
		samples: make([]float64, windowSize),
	}
}

// ResponseTimeMiddleware wraps next with latency tracking.
func ResponseTimeMiddleware(tracker *ResponseTimeTracker) func(http.Handler) http.Handler {
	if tracker == nil {
		tracker = NewResponseTimeTracker(1000)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := &responseTimeWriter{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(rw, r)

			elapsed := time.Since(start)
			ms := float64(elapsed.Microseconds()) / 1000.0

			// Set response header
			w.Header().Set("X-Response-Time", fmt.Sprintf("%.3fms", ms))

			// Record in Prometheus
			responseTimeHist.WithLabelValues(r.Method, r.URL.Path, strconv.Itoa(rw.status)).Observe(elapsed.Seconds())

			// Record in tracker
			tracker.record(ms)
		})
	}
}

func (t *ResponseTimeTracker) record(ms float64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.samples[t.head] = ms
	t.head = (t.head + 1) % len(t.samples)
	if t.size < len(t.samples) {
		t.size++
	}
}

// Percentiles returns p50 and p99 in milliseconds.
func (t *ResponseTimeTracker) Percentiles() (p50, p99 float64) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.size == 0 {
		return 0, 0
	}

	// Copy and sort samples
	sorted := make([]float64, t.size)
	copy(sorted, t.samples[:t.size])
	sortFloat64s(sorted)

	p50Idx := t.size / 2
	p99Idx := t.size * 99 / 100
	if p99Idx >= t.size {
		p99Idx = t.size - 1
	}

	return sorted[p50Idx], sorted[p99Idx]
}

// responseTimeWriter captures the status code.
type responseTimeWriter struct {
	http.ResponseWriter
	status int
}

func (w *responseTimeWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *responseTimeWriter) Write(b []byte) (int, error) {
	return w.ResponseWriter.Write(b)
}

// sortFloat64s sorts in ascending order (simple insertion sort for small slices).
func sortFloat64s(a []float64) {
	for i := 1; i < len(a); i++ {
		key := a[i]
		j := i - 1
		for j >= 0 && a[j] > key {
			a[j+1] = a[j]
			j--
		}
		a[j+1] = key
	}
}
