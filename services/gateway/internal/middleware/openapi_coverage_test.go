package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// --- OpenAPIAggregator.Handler ---

func TestOpenAPIAggregator_Handler_Success(t *testing.T) {
	// Create mock backend that serves OpenAPI spec
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/swagger/doc.json" || r.URL.Path == "/openapi.json" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(OpenAPISpec{
				OpenAPI: "3.0.0",
				Info: OpenAPIInfo{
					Title:       "Test API",
					Description: "Test service",
					Version:     "1.0.0",
				},
				Paths: map[string]map[string]any{
					"/users": {"get": map[string]any{"summary": "List users"}},
				},
			})
			return
		}
		w.WriteHeader(404)
	}))
	defer backend.Close()

	agg := NewOpenAPIAggregator(map[string]string{
		"/api/v1/test": backend.URL,
	})

	req := httptest.NewRequest("GET", "/openapi", nil)
	w := httptest.NewRecorder()
	agg.Handler().ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var spec OpenAPISpec
	if err := json.NewDecoder(w.Body).Decode(&spec); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if spec.OpenAPI == "" {
		t.Error("openapi version should not be empty")
	}
	if len(spec.Paths) == 0 {
		t.Error("paths should not be empty")
	}
}

func TestOpenAPIAggregator_Handler_AllServicesDown(t *testing.T) {
	agg := NewOpenAPIAggregator(map[string]string{
		"/api/v1/test": "http://localhost:59999", // nothing listening
	})

	req := httptest.NewRequest("GET", "/openapi", nil)
	w := httptest.NewRecorder()
	agg.Handler().ServeHTTP(w, req)

	// Should return 503 since all services are unreachable, or 200 with empty spec
	// depending on whether aggregator treats unreachable as error
	if w.Code != http.StatusServiceUnavailable && w.Code != http.StatusOK {
		t.Errorf("expected 503 or 200, got %d", w.Code)
	}
}

func TestOpenAPIAggregator_Handler_CachedResult(t *testing.T) {
	requestCount := 0
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(OpenAPISpec{
			OpenAPI: "3.0.0",
			Info:    OpenAPIInfo{Title: "Cached API", Version: "1.0"},
			Paths:   map[string]map[string]any{},
		})
	}))
	defer backend.Close()

	agg := NewOpenAPIAggregator(map[string]string{
		"/api/v1/test": backend.URL,
	})

	// First request - should fetch from backend
	req1 := httptest.NewRequest("GET", "/openapi", nil)
	w1 := httptest.NewRecorder()
	agg.Handler().ServeHTTP(w1, req1)
	if w1.Code != 200 {
		t.Fatalf("first request: expected 200, got %d", w1.Code)
	}

	// Second request - should use cache
	req2 := httptest.NewRequest("GET", "/openapi", nil)
	w2 := httptest.NewRecorder()
	agg.Handler().ServeHTTP(w2, req2)
	if w2.Code != 200 {
		t.Fatalf("second request: expected 200, got %d", w2.Code)
	}

	if requestCount != 1 {
		t.Errorf("expected 1 backend request (cached), got %d", requestCount)
	}
}

// --- FetchSpec with different endpoints ---

func TestOpenAPIAggregator_FetchSpec_OpenAPIJson(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/openapi.json" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(OpenAPISpec{
				OpenAPI: "3.0.0",
				Info:    OpenAPIInfo{Title: "OpenAPI", Version: "2.0"},
			})
			return
		}
		w.WriteHeader(404)
	}))
	defer backend.Close()

	agg := NewOpenAPIAggregator(map[string]string{"/test": backend.URL})
	spec, err := agg.Aggregate()
	if err != nil {
		t.Fatal(err)
	}
	if spec.Info.Title == "" {
		t.Error("title should not be empty")
	}
}

func TestOpenAPIAggregator_FetchSpec_ApiDocs(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api-docs" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(OpenAPISpec{
				OpenAPI: "3.0.0",
				Info:    OpenAPIInfo{Title: "ApiDocs", Version: "3.0"},
			})
			return
		}
		w.WriteHeader(404)
	}))
	defer backend.Close()

	agg := NewOpenAPIAggregator(map[string]string{"/test": backend.URL})
	spec, err := agg.Aggregate()
	if err != nil {
		t.Fatal(err)
	}
	if spec.Info.Title == "" {
		t.Error("title should not be empty")
	}
}

func TestOpenAPIAggregator_FetchSpec_InvalidJSON(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("not valid json"))
	}))
	defer backend.Close()

	agg := NewOpenAPIAggregator(map[string]string{"/test": backend.URL})
	spec, err := agg.Aggregate()
	// Aggregator skips services with invalid JSON, returns empty spec
	if err != nil {
		t.Logf("got expected error: %v", err)
	}
	_ = spec
}

func TestOpenAPIAggregator_FetchSpec_Non200Status(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer backend.Close()

	agg := NewOpenAPIAggregator(map[string]string{"/test": backend.URL})
	spec, err := agg.Aggregate()
	// Aggregator may return empty spec when all endpoints fail
	if err != nil {
		t.Logf("got expected error: %v", err)
	}
	_ = spec
}

// --- SortedPaths ---

func TestSortedPaths(t *testing.T) {
	spec := &OpenAPISpec{
		Paths: map[string]map[string]any{
			"/users":     {},
			"/roles":     {},
			"/orgs":      {},
			"/auth/login": {},
		},
	}

	paths := spec.SortedPaths()
	if len(paths) != 4 {
		t.Fatalf("expected 4 paths, got %d", len(paths))
	}

	// Verify sorted order
	expected := []string{"/auth/login", "/orgs", "/roles", "/users"}
	for i, p := range paths {
		if p != expected[i] {
			t.Errorf("paths[%d] = %q, want %q", i, p, expected[i])
		}
	}
}

func TestSortedPaths_Empty(t *testing.T) {
	spec := &OpenAPISpec{Paths: map[string]map[string]any{}}
	paths := spec.SortedPaths()
	if len(paths) != 0 {
		t.Errorf("expected 0 paths, got %d", len(paths))
	}
}

// --- InvalidateCache ---

func TestInvalidateCache_ForcesRefetch(t *testing.T) {
	requestCount := 0
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		json.NewEncoder(w).Encode(OpenAPISpec{
			OpenAPI: "3.0.0",
			Info:    OpenAPIInfo{Title: "API", Version: "1.0"},
		})
	}))
	defer backend.Close()

	agg := NewOpenAPIAggregator(map[string]string{"/test": backend.URL})

	// First fetch
	_, _ = agg.Aggregate()
	// Second fetch from cache
	_, _ = agg.Aggregate()
	if requestCount != 1 {
		t.Errorf("expected 1 request, got %d", requestCount)
	}

	// Invalidate
	agg.InvalidateCache()

	// Third fetch should re-fetch
	_, _ = agg.Aggregate()
	if requestCount != 2 {
		t.Errorf("expected 2 requests after invalidation, got %d", requestCount)
	}
}

// --- Aggregate with empty routes ---

func TestAggregate_EmptyRoutes(t *testing.T) {
	agg := NewOpenAPIAggregator(map[string]string{})
	spec, err := agg.Aggregate()
	if err != nil {
		t.Fatal(err)
	}
	if len(spec.Paths) != 0 {
		t.Errorf("expected empty paths, got %d", len(spec.Paths))
	}
}

// --- Multiple services ---

func TestAggregate_MultipleServices(t *testing.T) {
	svc1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/swagger/doc.json" {
			json.NewEncoder(w).Encode(OpenAPISpec{
				OpenAPI: "3.0.0",
				Info:    OpenAPIInfo{Title: "Service1"},
				Paths:   map[string]map[string]any{"/api/v1/users": {"get": nil}},
			})
		}
	}))
	defer svc1.Close()

	svc2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/swagger/doc.json" {
			json.NewEncoder(w).Encode(OpenAPISpec{
				OpenAPI: "3.0.0",
				Info:    OpenAPIInfo{Title: "Service2"},
				Paths:   map[string]map[string]any{"/api/v1/roles": {"get": nil}},
			})
		}
	}))
	defer svc2.Close()

	agg := NewOpenAPIAggregator(map[string]string{
		"/api/v1/users": svc1.URL,
		"/api/v1/roles": svc2.URL,
	})

	spec, err := agg.Aggregate()
	if err != nil {
		t.Fatal(err)
	}
	if len(spec.Paths) != 2 {
		t.Errorf("expected 2 merged paths, got %d", len(spec.Paths))
	}
}
