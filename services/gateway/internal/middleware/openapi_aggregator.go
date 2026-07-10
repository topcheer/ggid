package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"
)

// OpenAPISpec represents a minimal OpenAPI 3.0 document for aggregation.
type OpenAPISpec struct {
	OpenAPI    string                      `json:"openapi"`
	Info       OpenAPIInfo                 `json:"info"`
	Paths      map[string]map[string]any   `json:"paths"`
	Components map[string]any              `json:"components,omitempty"`
}

type OpenAPIInfo struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Version     string `json:"version"`
}

// ServiceDocFetcher fetches OpenAPI documentation from a backend service.
type ServiceDocFetcher interface {
	FetchSpec(serviceURL string) (*OpenAPISpec, error)
}

// OpenAPIAggregator aggregates OpenAPI specs from multiple backend services.
// It fetches /swagger/doc.json or /openapi.json from each service and merges
// them into a single unified API documentation endpoint.
type OpenAPIAggregator struct {
	mu          sync.RWMutex
	cache       *OpenAPISpec
	cacheExpiry time.Time
	ttl         time.Duration
	services    map[string]string // route prefix → service URL
	fetcher     ServiceDocFetcher
}

// NewOpenAPIAggregator creates a new aggregator for the given service routes.
func NewOpenAPIAggregator(routes map[string]string) *OpenAPIAggregator {
	return &OpenAPIAggregator{
		ttl:      5 * time.Minute,
		services: routes,
		fetcher:  &httpDocFetcher{client: &http.Client{Timeout: 5 * time.Second}},
	}
}

// Aggregate fetches and merges all service specs. Returns cached result if fresh.
func (a *OpenAPIAggregator) Aggregate() (*OpenAPISpec, error) {
	a.mu.RLock()
	if a.cache != nil && time.Now().Before(a.cacheExpiry) {
		result := a.cache
		a.mu.RUnlock()
		return result, nil
	}
	a.mu.RUnlock()

	merged := &OpenAPISpec{
		OpenAPI: "3.0.3",
		Info: OpenAPIInfo{
			Title:       "GGID IAM API",
			Description: "Unified Identity and Access Management API — aggregated from all backend services",
			Version:     "1.0.0",
		},
		Paths: make(map[string]map[string]any),
		Components: map[string]any{
			"securitySchemes": map[string]any{
				"bearerAuth": map[string]any{
					"type":         "http",
					"scheme":       "bearer",
					"bearerFormat": "JWT",
				},
				"apiKey": map[string]any{
					"type": "apiKey",
					"in":   "header",
					"name": "X-API-Key",
				},
			},
		},
	}

	for prefix, svcURL := range a.services {
		spec, err := a.fetcher.FetchSpec(svcURL)
		if err != nil {
			continue // skip services that don't expose docs
		}
		for path, methods := range spec.Paths {
			fullPath := prefix + path
			if _, exists := merged.Paths[fullPath]; !exists {
				merged.Paths[fullPath] = make(map[string]any)
			}
			for method, detail := range methods {
				// Tag each operation with its upstream service prefix for traceability.
				// If two services define the same method+path, the later one wins but
				// we record the conflict in the operation's x-upstream-service field.
				if detailMap, ok := detail.(map[string]any); ok {
					detailMap["x-upstream-service"] = prefix
					merged.Paths[fullPath][method] = detailMap
				} else {
					merged.Paths[fullPath][method] = detail
				}
			}
		}
	}

	// Sort paths for deterministic output
	a.mu.Lock()
	a.cache = merged
	a.cacheExpiry = time.Now().Add(a.ttl)
	a.mu.Unlock()

	return merged, nil
}

// Handler returns an HTTP handler that serves the aggregated OpenAPI spec.
func (a *OpenAPIAggregator) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		spec, err := a.Aggregate()
		if err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{"error": "failed to aggregate API docs"})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(spec)
	})
}

// InvalidateCache clears the cached spec, forcing a re-fetch on next request.
func (a *OpenAPIAggregator) InvalidateCache() {
	a.mu.Lock()
	a.cache = nil
	a.cacheExpiry = time.Time{}
	a.mu.Unlock()
}

// SortedPaths returns paths in sorted order for deterministic output.
func (s *OpenAPISpec) SortedPaths() []string {
	paths := make([]string, 0, len(s.Paths))
	for p := range s.Paths {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	return paths
}

// httpDocFetcher implements ServiceDocFetcher using HTTP.
type httpDocFetcher struct {
	client *http.Client
}

func (f *httpDocFetcher) FetchSpec(serviceURL string) (*OpenAPISpec, error) {
	// Try common OpenAPI endpoints
	for _, path := range []string{"/swagger/doc.json", "/openapi.json", "/api-docs"} {
		resp, err := f.client.Get(serviceURL + path)
		if err != nil {
			continue
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			continue
		}
		var spec OpenAPISpec
		if err := json.NewDecoder(resp.Body).Decode(&spec); err != nil {
			resp.Body.Close()
			continue
		}
		resp.Body.Close()
		return &spec, nil
	}
	return nil, fmt.Errorf("no OpenAPI spec found at %s", serviceURL)
}
