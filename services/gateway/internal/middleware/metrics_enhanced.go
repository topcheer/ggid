package middleware

import (
	"net/http"
	"runtime"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// EnhancedMetrics holds optimized Prometheus metrics with per-API histograms.
type EnhancedMetrics struct {
	requestDuration *prometheus.HistogramVec
	requestSize    *prometheus.HistogramVec
	responseSize   *prometheus.HistogramVec
	runtimeMetrics *RuntimeCollector
}

// RuntimeCollector collects Go runtime metrics.
type RuntimeCollector struct {
	goroutines prometheus.GaugeFunc
	memoryAlloc prometheus.GaugeFunc
	gcDuration  prometheus.GaugeFunc
	numGC       prometheus.CounterFunc
}

var enhancedMetrics *EnhancedMetrics

func init() {
	enhancedMetrics = &EnhancedMetrics{
		requestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "ggid_http_request_duration_seconds",
				Help:    "HTTP request duration by route and method",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
			},
			[]string{"method", "route", "status"},
		),
		requestSize: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "ggid_http_request_size_bytes",
				Help:    "HTTP request size by route",
				Buckets: prometheus.ExponentialBuckets(100, 10, 7), // 100b to 100MB
			},
			[]string{"method", "route"},
		),
		responseSize: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "ggid_http_response_size_bytes",
				Help:    "HTTP response size by route",
				Buckets: prometheus.ExponentialBuckets(100, 10, 7),
			},
			[]string{"method", "route", "status"},
		),
		runtimeMetrics: &RuntimeCollector{},
	}

	// Register all metrics
	prometheus.MustRegister(
		enhancedMetrics.requestDuration,
		enhancedMetrics.requestSize,
		enhancedMetrics.responseSize,
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "ggid_go_goroutines",
			Help: "Number of goroutines",
		}, func() float64 { return float64(runtime.NumGoroutine()) }),
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "ggid_go_memory_alloc_bytes",
			Help: "Memory allocated bytes",
		}, func() float64 {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			return float64(m.Alloc)
		}),
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "ggid_go_gc_duration_seconds",
			Help: "Last GC pause duration",
		}, func() float64 {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			if m.NumGC > 0 {
				return float64(m.PauseNs[(m.NumGC+255)%256]) / 1e9
			}
			return 0
		}),
		prometheus.NewCounterFunc(prometheus.CounterOpts{
			Name: "ggid_go_gc_total",
			Help: "Total number of GC runs",
		}, func() float64 {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			return float64(m.NumGC)
		}),
		// CPU count
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "ggid_go_cpu_count",
			Help: "Number of logical CPUs",
		}, func() float64 { return float64(runtime.NumCPU()) }),
		// Heap objects
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "ggid_go_heap_objects",
			Help: "Number of heap objects",
		}, func() float64 {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			return float64(m.HeapObjects)
		}),
		// Go version info
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "ggid_go_info",
			Help: "Go version info (value=1)",
			ConstLabels: prometheus.Labels{"version": runtime.Version()},
		}, func() float64 { return 1 }),
	)
}

// EnhancedMetricsHandler returns the Prometheus metrics handler with runtime metrics.
func EnhancedMetricsHandler() http.Handler {
	return promhttp.Handler()
}

// ObserveRequest records request metrics.
func (m *EnhancedMetrics) ObserveRequest(method, route string, status, reqSize, respSize int, duration time.Duration) {
	statusStr := normalizeStatusCode(status)
	m.requestDuration.WithLabelValues(method, route, statusStr).Observe(duration.Seconds())
	m.requestSize.WithLabelValues(method, route).Observe(float64(reqSize))
	m.responseSize.WithLabelValues(method, route, statusStr).Observe(float64(respSize))
}

func normalizeStatusCode(code int) string {
	return itoa(code/100) + "xx"
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	return string(buf[pos:])
}

// GetEnhancedMetrics returns the singleton metrics instance.
func GetEnhancedMetrics() *EnhancedMetrics {
	return enhancedMetrics
}
