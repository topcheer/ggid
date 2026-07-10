package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewOpenAPIAggregator_Handler(t *testing.T) {
	a := NewOpenAPIAggregator(map[string]string{
		"/api/v1/auth": "http://localhost:9001",
	})
	// Services are unreachable → handler should still return merged (empty paths)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/docs", nil)
	w := httptest.NewRecorder()
	a.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected application/json, got %s", ct)
	}
}

func TestOpenAPIAggregator_InvalidateCache(t *testing.T) {
	a := NewOpenAPIAggregator(map[string]string{})
	// First call populates cache
	spec1, _ := a.Aggregate()
	if spec1 == nil {
		t.Fatal("expected non-nil spec")
	}
	// Invalidate
	a.InvalidateCache()
	// Second call should re-fetch
	spec2, _ := a.Aggregate()
	if spec2 == nil {
		t.Fatal("expected non-nil spec after invalidate")
	}
}

func TestOpenAPIAggregator_SortedPaths(t *testing.T) {
	spec := &OpenAPISpec{
		Paths: map[string]map[string]any{
			"/api/v1/users": {},
			"/api/v1/auth":  {},
			"/api/v1/roles": {},
		},
	}
	paths := spec.SortedPaths()
	if len(paths) != 3 {
		t.Fatalf("expected 3 paths, got %d", len(paths))
	}
	if paths[0] != "/api/v1/auth" {
		t.Errorf("expected /api/v1/auth first, got %s", paths[0])
	}
}

func TestOpenAPIAggregator_WithMockFetcher(t *testing.T) {
	mock := &mockDocFetcher{
		specs: map[string]*OpenAPISpec{
			"http://auth:9001": {
				OpenAPI: "3.0.3",
				Info:    OpenAPIInfo{Title: "Auth Service", Version: "1.0"},
				Paths: map[string]map[string]any{
					"/login":          {"post": map[string]any{"summary": "Login"}},
					"/register":       {"post": map[string]any{"summary": "Register"}},
				},
			},
		},
	}
	a := &OpenAPIAggregator{
		ttl:      time.Minute,
		services: map[string]string{"/api/v1/auth": "http://auth:9001"},
		fetcher:  mock,
	}
	spec, err := a.Aggregate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(spec.Paths) != 2 {
		t.Errorf("expected 2 paths, got %d", len(spec.Paths))
	}
	if _, ok := spec.Paths["/api/v1/auth/login"]; !ok {
		t.Error("expected /api/v1/auth/login in merged paths")
	}
}

func TestOpenAPIAggregator_MultipleServices(t *testing.T) {
	mock := &mockDocFetcher{
		specs: map[string]*OpenAPISpec{
			"http://auth": {
				Paths: map[string]map[string]any{
					"/login": {"post": nil},
				},
			},
			"http://users": {
				Paths: map[string]map[string]any{
					"/list": {"get": nil},
				},
			},
		},
	}
	a := &OpenAPIAggregator{
		ttl: time.Minute,
		services: map[string]string{
			"/auth": "http://auth",
			"/users": "http://users",
		},
		fetcher: mock,
	}
	spec, _ := a.Aggregate()
	if len(spec.Paths) != 2 {
		t.Errorf("expected 2 merged paths, got %d", len(spec.Paths))
	}
}

type mockDocFetcher struct {
	specs map[string]*OpenAPISpec
}

func (m *mockDocFetcher) FetchSpec(svcURL string) (*OpenAPISpec, error) {
	if s, ok := m.specs[svcURL]; ok {
		return s, nil
	}
	return nil, errFetchNotFound
}

var errFetchNotFound = fmt.Errorf("no spec found")
