package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// TestMetricsEndpoint verifies that /metrics exposes Prometheus metrics.
func TestMetricsEndpoint(t *testing.T) {
	handler := promhttp.Handler()

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	body, _ := io.ReadAll(rr.Body)
	bodyStr := string(body)

	// Verify standard Go runtime metrics exist
	if !strings.Contains(bodyStr, "go_goroutines") {
		t.Error("missing go_goroutines metric")
	}
	if !strings.Contains(bodyStr, "process_resident_memory_bytes") {
		t.Error("missing process_resident_memory_bytes metric")
	}
}

// TestRequestMetricsRegistered verifies gateway-specific metric names are registered.
func TestRequestMetricsRegistered(t *testing.T) {
	// The middleware registers these in init(). We verify by checking
	// the /metrics output contains the metric names.
	handler := promhttp.Handler()

	// Generate some traffic first
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mw := MetricsMiddleware(next)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	rr := httptest.NewRecorder()
	mw.ServeHTTP(rr, req)

	// Now check /metrics
	req2 := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)

	body, _ := io.ReadAll(rr2.Body)
	bodyStr := string(body)

	// Check for http_requests_total (registered in metrics.go init)
	if !strings.Contains(bodyStr, "http_requests_total") {
		t.Error("missing http_requests_total metric")
	}
	// Check for http_request_duration_seconds
	if !strings.Contains(bodyStr, "http_request_duration_seconds") {
		t.Error("missing http_request_duration_seconds metric")
	}
}

// TestMetricsMiddleware_RecordsRequest verifies MetricsMiddleware wraps handler.
func TestMetricsMiddleware_RecordsRequest(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	mw := MetricsMiddleware(next)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	rr := httptest.NewRecorder()
	mw.ServeHTTP(rr, req)

	if !called {
		t.Fatal("handler not called")
	}
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}
