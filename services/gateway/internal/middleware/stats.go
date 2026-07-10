// Package middleware implements HTTP middleware for the API Gateway.
package middleware

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// --- Request Statistics ---

// RequestStats holds aggregated request statistics.
type RequestStats struct {
	TotalRequests  atomic.Uint64 `json:"-"`
	TotalResponses atomic.Uint64 `json:"-"`
	Total2xx       atomic.Uint64 `json:"-"`
	Total3xx       atomic.Uint64 `json:"-"`
	Total4xx       atomic.Uint64 `json:"-"`
	Total5xx       atomic.Uint64 `json:"-"`
	TotalErrors    atomic.Uint64 `json:"-"` // proxy errors (502/503)
	TotalBytesSent atomic.Uint64 `json:"-"` // response body bytes
	StartTime      time.Time     `json:"-"`
}

// StatsCollector tracks request statistics for the gateway.
type StatsCollector struct {
	stats   *RequestStats
	perRoute map[string]*RequestStats
	mu      sync.RWMutex
}

// NewStatsCollector creates a new stats collector.
func NewStatsCollector() *StatsCollector {
	return &StatsCollector{
		stats:   &RequestStats{StartTime: time.Now()},
		perRoute: make(map[string]*RequestStats),
	}
}

// Record records a single request's stats.
func (sc *StatsCollector) Record(route, method string, status, responseSize int, duration time.Duration) {
	sc.stats.TotalRequests.Add(1)
	sc.stats.TotalResponses.Add(1)
	sc.stats.TotalBytesSent.Add(uint64(responseSize))

	switch {
	case status >= 500:
		sc.stats.Total5xx.Add(1)
		if status == 502 || status == 503 {
			sc.stats.TotalErrors.Add(1)
		}
	case status >= 400:
		sc.stats.Total4xx.Add(1)
	case status >= 300:
		sc.stats.Total3xx.Add(1)
	case status >= 200:
		sc.stats.Total2xx.Add(1)
	}

	if route != "" {
		sc.mu.Lock()
		rs, ok := sc.perRoute[route]
		if !ok {
			rs = &RequestStats{StartTime: time.Now()}
			sc.perRoute[route] = rs
		}
		sc.mu.Unlock()

		rs.TotalRequests.Add(1)
		rs.TotalResponses.Add(1)
		rs.TotalBytesSent.Add(uint64(responseSize))
		switch {
		case status >= 500:
			rs.Total5xx.Add(1)
		case status >= 400:
			rs.Total4xx.Add(1)
		case status >= 300:
			rs.Total3xx.Add(1)
		case status >= 200:
			rs.Total2xx.Add(1)
		}
	}
}

// StatsResponse is the JSON response for GET /api/v1/gateway/stats.
type StatsResponse struct {
	Uptime          string               `json:"uptime"`
	TotalRequests   uint64               `json:"total_requests"`
	TotalResponses  uint64               `json:"total_responses"`
	StatusBreakdown map[string]uint64    `json:"status_breakdown"`
	TotalErrors     uint64               `json:"total_errors"`
	TotalBytesSent  uint64               `json:"total_bytes_sent"`
	Routes          map[string]RouteStat `json:"routes"`
}

// RouteStat is per-route statistics.
type RouteStat struct {
	Requests  uint64            `json:"requests"`
	Responses uint64            `json:"responses"`
	Status    map[string]uint64 `json:"status"`
	Errors    uint64            `json:"errors"`
	BytesSent uint64            `json:"bytes_sent"`
}

// Snapshot returns a point-in-time copy of the statistics.
func (sc *StatsCollector) Snapshot() StatsResponse {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	resp := StatsResponse{
		Uptime:         time.Since(sc.stats.StartTime).Round(time.Second).String(),
		TotalRequests:  sc.stats.TotalRequests.Load(),
		TotalResponses: sc.stats.TotalResponses.Load(),
		TotalErrors:    sc.stats.TotalErrors.Load(),
		TotalBytesSent: sc.stats.TotalBytesSent.Load(),
		StatusBreakdown: map[string]uint64{
			"2xx": sc.stats.Total2xx.Load(),
			"3xx": sc.stats.Total3xx.Load(),
			"4xx": sc.stats.Total4xx.Load(),
			"5xx": sc.stats.Total5xx.Load(),
		},
		Routes: make(map[string]RouteStat),
	}

	for route, rs := range sc.perRoute {
		resp.Routes[route] = RouteStat{
			Requests:  rs.TotalRequests.Load(),
			Responses: rs.TotalResponses.Load(),
			BytesSent: rs.TotalBytesSent.Load(),
			Errors:    rs.Total5xx.Load(),
			Status: map[string]uint64{
				"2xx": rs.Total2xx.Load(),
				"3xx": rs.Total3xx.Load(),
				"4xx": rs.Total4xx.Load(),
				"5xx": rs.Total5xx.Load(),
			},
		}
	}

	return resp
}

// StatsMiddleware wraps a handler and records request statistics.
func StatsMiddleware(collector *StatsCollector, routeResolver func(path string) string) func(http.Handler) http.Handler {
	if routeResolver == nil {
		routeResolver = func(path string) string { return "" }
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			route := routeResolver(r.URL.Path)
			sr := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(sr, r)
			collector.Record(route, r.Method, sr.status, sr.size, 0)
		})
	}
}

// StatsHandler returns an http.HandlerFunc that returns stats JSON.
func (sc *StatsCollector) StatsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(sc.Snapshot())
	}
}

// --- Middleware Chain Inspection ---

// MiddlewareInfo describes a middleware in the chain.
type MiddlewareInfo struct {
	Name     string `json:"name"`
	Order    int    `json:"order"`
	Category string `json:"category"`
}

// MiddlewareChain describes the full middleware configuration.
type MiddlewareChain struct {
	Outer []MiddlewareInfo `json:"outer"`
	Inner []MiddlewareInfo `json:"inner"`
	Count int              `json:"count"`
}

// DefaultMiddlewareChain returns the default middleware chain for the gateway.
func DefaultMiddlewareChain() MiddlewareChain {
	outer := []MiddlewareInfo{
		{Name: "PanicRecovery", Order: 1, Category: "resilience"},
		{Name: "CORS", Order: 2, Category: "security"},
		{Name: "RequestID", Order: 3, Category: "tracing"},
		{Name: "RequestLogger", Order: 4, Category: "logging"},
		{Name: "TenantResolver", Order: 5, Category: "security"},
	}
	inner := []MiddlewareInfo{
		{Name: "GzipBrotli", Order: 1, Category: "compression"},
		{Name: "RateLimit", Order: 2, Category: "security"},
		{Name: "JWTAuth", Order: 3, Category: "security"},
		{Name: "BodySizeLimit", Order: 4, Category: "validation"},
		{Name: "BotDetect", Order: 5, Category: "security"},
		{Name: "Cache", Order: 6, Category: "performance"},
		{Name: "APIVersioning", Order: 7, Category: "routing"},
		{Name: "CircuitBreaker", Order: 8, Category: "resilience"},
		{Name: "ReverseProxy", Order: 9, Category: "routing"},
	}
	return MiddlewareChain{
		Outer: outer,
		Inner: inner,
		Count: len(outer) + len(inner),
	}
}

// MiddlewareChainHandler returns an http.HandlerFunc that returns the middleware chain.
func MiddlewareChainHandler() http.HandlerFunc {
	chain := DefaultMiddlewareChain()
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(chain)
	}
}

// --- Helpers ---

// ExtractRoutePrefix extracts the route prefix from a path.
// /api/v1/users/123 → /api/v1/users
func ExtractRoutePrefix(path string) string {
	parts := strings.SplitN(path, "/", 5)
	if len(parts) >= 4 {
		return strings.Join(parts[:4], "/")
	}
	return path
}
