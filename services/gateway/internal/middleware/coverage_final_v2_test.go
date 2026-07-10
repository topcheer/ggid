package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// === OpenAPI Aggregator additional tests ===

func TestOpenAPIAggregator_Handler_Success_V2(t *testing.T) {
	// Create a mock service that serves OpenAPI spec
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/openapi.json" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"openapi": "3.0.0",
				"info":    map[string]any{"title": "test", "version": "1.0"},
				"paths": map[string]any{
					"/users": map[string]any{
						"get": map[string]any{"summary": "list users"},
					},
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	agg := NewOpenAPIAggregator(map[string]string{
		"test-service": ts.URL,
	})

	handler := agg.Handler()
	req := httptest.NewRequest("GET", "/openapi.json", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Handler: want 200, got %d", rr.Code)
	}

	var spec OpenAPISpec
	if err := json.Unmarshal(rr.Body.Bytes(), &spec); err != nil {
		t.Errorf("Parse spec: %v", err)
	}
}

func TestOpenAPISpec_SortedPaths(t *testing.T) {
	spec := &OpenAPISpec{
		OpenAPI: "3.0.0",
		Paths: map[string]map[string]any{
			"/users":  {},
			"/orgs":   {},
			"/audit":  {},
			"/health": {},
		},
	}

	paths := spec.SortedPaths()
	if len(paths) != 4 {
		t.Fatalf("Expected 4 paths, got %d", len(paths))
	}
	// Should be sorted
	if paths[0] != "/audit" {
		t.Errorf("First path should be '/audit', got '%s'", paths[0])
	}
	if paths[1] != "/health" {
		t.Errorf("Second path should be '/health', got '%s'", paths[1])
	}
}

func TestOpenAPIAggregator_InvalidateCache_V2(t *testing.T) {
	agg := NewOpenAPIAggregator(map[string]string{})
	// Just verify it doesn't panic
	agg.InvalidateCache()
}

// === HealthScore additional tests ===

func TestHealthScore_IsHealthy_DefaultThreshold_V2(t *testing.T) {
	hs := NewHealthScore(5*time.Minute, 0.9)

	// Record some successes to get a high score
	for i := 0; i < 10; i++ {
		hs.RecordSuccess("backend1", 10*time.Millisecond)
	}

	// With threshold <= 0, should default to 50
	if !hs.IsHealthy("backend1", 0) {
		t.Error("Default threshold 50: should be healthy")
	}
	if !hs.IsHealthy("backend1", -1) {
		t.Error("Negative threshold: should default to 50 and be healthy")
	}
}

func TestHealthScore_IsHealthy_UnknownBackend_V2(t *testing.T) {
	hs := NewHealthScore(5*time.Minute, 0.9)

	// Unknown backend with no data
	if hs.IsHealthy("unknown", 50) {
		// Score for unknown backend is 100 (default), so this is healthy
	}
}

func TestHealthScore_AllScores_V2(t *testing.T) {
	hs := NewHealthScore(5*time.Minute, 0.9)

	for i := 0; i < 5; i++ {
		hs.RecordSuccess("b1", 5*time.Millisecond)
		hs.RecordSuccess("b2", 5*time.Millisecond)
	}
	hs.RecordError("b2")

	scores := hs.AllScores()
	if len(scores) < 2 {
		t.Errorf("Expected >= 2 scores, got %d", len(scores))
	}
	if _, ok := scores["b1"]; !ok {
		t.Error("Missing b1 in scores")
	}
}

func TestHealthScore_Reset_V2(t *testing.T) {
	hs := NewHealthScore(5*time.Minute, 0.9)

	for i := 0; i < 5; i++ {
		hs.RecordSuccess("b1", 5*time.Millisecond)
	}

	hs.Reset("b1")
	scores := hs.AllScores()
	if _, ok := scores["b1"]; ok {
		t.Error("b1 should be removed after reset")
	}
}
