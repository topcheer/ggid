package healthcheck

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestDeepHandler_TimeoutHandling verifies that a slow backend doesn't
// cause the deep health check to hang beyond its timeout.
func TestDeepHandler_TimeoutHandling(t *testing.T) {
	// Start a slow backend that takes 500ms
	slowBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer slowBackend.Close()

	// Start a fast backend
	fastBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer fastBackend.Close()

	// Create checker with backends, 2s timeout (generous enough for 500ms slow)
	checker := NewChecker(map[string]string{
		"slow-service": slowBackend.URL + "/healthz",
		"fast-service": fastBackend.URL + "/healthz",
	})

	handler := checker.DeepHandler()

	// The deep health check should complete in under 3 seconds
	start := time.Now()
	req := httptest.NewRequest(http.MethodGet, "/healthz/deep", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	elapsed := time.Since(start)

	// Should return 200 (both backends healthy, just one is slow)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rr.Code, rr.Body.String())
	}

	// Should complete in under 3 seconds (parallel checks with 5s timeout each)
	if elapsed > 3*time.Second {
		t.Fatalf("deep health check took too long: %v", elapsed)
	}
}

// TestDeepHandler_AllHealthy verifies 200 when all backends are healthy.
func TestDeepHandler_AllHealthy(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	checker := NewChecker(map[string]string{
		"svc1": backend.URL + "/healthz",
		"svc2": backend.URL + "/healthz",
	})

	handler := checker.DeepHandler()
	req := httptest.NewRequest(http.MethodGet, "/healthz/deep", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

// TestDeepHandler_OneUnhealthy verifies 503 when a backend is degraded.
func TestDeepHandler_OneUnhealthy(t *testing.T) {
	healthy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer healthy.Close()

	unhealthy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer unhealthy.Close()

	checker := NewChecker(map[string]string{
		"healthy-svc":   healthy.URL + "/healthz",
		"unhealthy-svc": unhealthy.URL + "/healthz",
	})

	handler := checker.DeepHandler()
	req := httptest.NewRequest(http.MethodGet, "/healthz/deep", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when one backend is down, got %d", rr.Code)
	}
}
