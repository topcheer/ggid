// Package healthcheck implements health check aggregation for the API Gateway.
package healthcheck

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"
)

// Checker pings backend services and aggregates their health status.
type Checker struct {
	services map[string]string // name → healthz URL
	client   *http.Client
}

// NewChecker creates a health checker with the given service map.
// The map keys are service names, values are their /healthz URLs.
func NewChecker(services map[string]string) *Checker {
	return &Checker{
		services: services,
		client:   &http.Client{Timeout: 2 * time.Second},
	}
}

// ServiceStatus is the health status of a single backend service.
type ServiceStatus struct {
	Name           string `json:"name"`
	Status         string `json:"status"`
	Latency        int64  `json:"latency_ms"`
	Version        string `json:"version,omitempty"`
	UptimeSeconds  int64  `json:"uptime_seconds,omitempty"`
	Error          string `json:"error,omitempty"`
}

// AggregatedStatus is the overall health response.
type AggregatedStatus struct {
	Status    string                   `json:"status"`
	Total     int                      `json:"total"`
	Healthy   int                      `json:"healthy"`
	Unhealthy int                      `json:"unhealthy"`
	Services  map[string]ServiceStatus `json:"services"`
	GatewayUptimeSeconds int64          `json:"gateway_uptime_seconds"`
	CheckedAt time.Time                `json:"checked_at"`
}

// gatewayStart tracks when the checker was created (proxy for gateway uptime).
var gatewayStart = time.Now()

// Handler returns an http.HandlerFunc that checks all backends.
func (c *Checker) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status := c.CheckAll(r.Context())
		code := http.StatusOK
		if status.Unhealthy > 0 {
			code = http.StatusServiceUnavailable
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		json.NewEncoder(w).Encode(status)
	}
}

// LiveHandler returns a lightweight liveness probe (Kubernetes /healthz/live).
// It does NOT check backends — it only verifies the gateway process itself is alive.
func (c *Checker) LiveHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "alive",
		})
	}
}

// ReadyHandler returns a readiness probe (Kubernetes /healthz/ready).
// It checks ALL backend services. Returns 200 only if every service is healthy.
// During startup, this lets the load balancer hold traffic until backends are up.
func (c *Checker) ReadyHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status := c.CheckAll(r.Context())
		code := http.StatusOK
		if status.Unhealthy > 0 {
			code = http.StatusServiceUnavailable
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		// For readiness, include the full detail so operators can see what's down
		json.NewEncoder(w).Encode(status)
	}
}

// DeepHandler returns a deep health check that pings ALL backends with a
// configurable timeout (default 5s). Returns 503 if any backend is down.
// Unlike ReadyHandler, this includes per-service latency and error detail.
func (c *Checker) DeepHandler() http.HandlerFunc {
	deepClient := &http.Client{Timeout: 5 * time.Second}
	return func(w http.ResponseWriter, r *http.Request) {
		// Temporarily swap client for deeper checks
		orig := c.client
		c.client = deepClient
		defer func() { c.client = orig }()

		status := c.CheckAll(r.Context())
		code := http.StatusOK
		if status.Unhealthy > 0 {
			code = http.StatusServiceUnavailable
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		json.NewEncoder(w).Encode(status)
	}
}

// HandlerWithMode dispatches based on the "mode" query parameter.
// mode=live  → liveness probe (process alive only)
// mode=ready → readiness probe (all backends healthy)
// default    → full aggregated health (backward compatible)
func (c *Checker) HandlerWithMode() http.HandlerFunc {
	live := c.LiveHandler()
	ready := c.ReadyHandler()
	full := c.Handler()

	return func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("mode") {
		case "live":
			live.ServeHTTP(w, r)
		case "ready":
			ready.ServeHTTP(w, r)
		default:
			full.ServeHTTP(w, r)
		}
	}
}

// CheckAll pings all services in parallel and returns aggregated status.
func (c *Checker) CheckAll(ctx context.Context) *AggregatedStatus {
	var mu sync.Mutex
	var wg sync.WaitGroup

	results := make(map[string]ServiceStatus, len(c.services))

	for name, url := range c.services {
		wg.Add(1)
		go func(name, healthURL string) {
			defer wg.Done()
			status := c.checkOne(ctx, name, healthURL)
			mu.Lock()
			results[name] = status
			mu.Unlock()
		}(name, url)
	}

	wg.Wait()

	healthy := 0
	for _, s := range results {
		if s.Status == "healthy" {
			healthy++
		}
	}

	overall := "healthy"
	if healthy < len(results) {
		overall = "degraded"
	}

	return &AggregatedStatus{
		Status:              overall,
		Total:               len(results),
		Healthy:             healthy,
		Unhealthy:           len(results) - healthy,
		Services:            results,
		GatewayUptimeSeconds: int64(time.Since(gatewayStart).Seconds()),
		CheckedAt:           time.Now().UTC(),
	}
}

func (c *Checker) checkOne(ctx context.Context, name, healthURL string) ServiceStatus {
	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, "GET", healthURL, nil)
	if err != nil {
		return ServiceStatus{Name: name, Status: "unhealthy", Error: err.Error(), Latency: 0}
	}

	resp, err := c.client.Do(req)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		return ServiceStatus{Name: name, Status: "unhealthy", Error: err.Error(), Latency: latency}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		// Try to parse version + uptime from response body
		status := ServiceStatus{Name: name, Status: "healthy", Latency: latency}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		var info struct {
			Version       string `json:"version"`
			UptimeSeconds int64  `json:"uptime_seconds"`
			Uptime        int64  `json:"uptime"`
		}
		if json.Unmarshal(body, &info) == nil {
			status.Version = info.Version
			status.UptimeSeconds = info.UptimeSeconds + info.Uptime
		}
		return status
	}
	return ServiceStatus{Name: name, Status: "unhealthy", Error: "non-200 status", Latency: latency}
}
