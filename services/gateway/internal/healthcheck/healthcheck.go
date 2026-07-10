// Package healthcheck implements health check aggregation for the API Gateway.
package healthcheck

import (
	"context"
	"encoding/json"
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
	Name    string `json:"name"`
	Status  string `json:"status"`
	Latency int64  `json:"latency_ms"`
	Error   string `json:"error,omitempty"`
}

// AggregatedStatus is the overall health response.
type AggregatedStatus struct {
	Status   string                    `json:"status"`
	Total    int                       `json:"total"`
	Healthy  int                       `json:"healthy"`
	Unhealthy int                      `json:"unhealthy"`
	Services map[string]ServiceStatus  `json:"services"`
	CheckedAt time.Time                `json:"checked_at"`
}

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
		Status:    overall,
		Total:     len(results),
		Healthy:   healthy,
		Unhealthy: len(results) - healthy,
		Services:  results,
		CheckedAt: time.Now().UTC(),
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
	resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return ServiceStatus{Name: name, Status: "healthy", Latency: latency}
	}
	return ServiceStatus{Name: name, Status: "unhealthy", Error: "non-200 status", Latency: latency}
}
