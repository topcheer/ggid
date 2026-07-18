package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
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

	// Check for ggid_http_requests_total (registered in metrics.go init)
	if !strings.Contains(bodyStr, "ggid_http_requests_total") {
		t.Error("missing ggid_http_requests_total metric")
	}
	// Check for ggid_http_duration_seconds
	if !strings.Contains(bodyStr, "ggid_http_duration_seconds") {
		t.Error("missing ggid_http_duration_seconds metric")
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

// TestMetricsMiddleware_CounterIncrementAndLabels verifies that after making
// requests, the ggid_http_requests_total counter is incremented with the correct
// labels (method, path, status) visible in the /metrics output.
func TestMetricsMiddleware_CounterIncrementAndLabels(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mw := MetricsMiddleware(next)

	// Make 3 GET requests to /api/v1/users
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
		rr := httptest.NewRecorder()
		mw.ServeHTTP(rr, req)
	}

	// Make 1 POST request to /api/v1/orgs
	postReq := httptest.NewRequest(http.MethodPost, "/api/v1/orgs", nil)
	postRR := httptest.NewRecorder()
	mw.ServeHTTP(postRR, postReq)

	// Now scrape /metrics and verify labels appear
	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsRR := httptest.NewRecorder()
	promhttp.Handler().ServeHTTP(metricsRR, metricsReq)

	body, _ := io.ReadAll(metricsRR.Body)
	bodyStr := string(body)

	// Verify ggid_http_requests_total has the GET /api/v1/users 200 label combination
	if !strings.Contains(bodyStr, `ggid_http_requests_total{method="GET",path="/api/v1/users",status="200"}`) {
		t.Errorf("expected GET /api/v1/users 200 label in metrics output\nOutput:\n%s", bodyStr)
	}
	// Verify POST label
	if !strings.Contains(bodyStr, `ggid_http_requests_total{method="POST",path="/api/v1/orgs",status="200"}`) {
		t.Errorf("expected POST /api/v1/orgs 200 label in metrics output\nOutput:\n%s", bodyStr)
	}
	// Verify the counter value is >= 1 for the GET path.
	// Note: the prometheus counter is a package-level global, so other tests
	// in this package may have already incremented it. We only check >= 1
	// to confirm our requests were counted, not an exact value.
	found := false
	for _, line := range strings.Split(bodyStr, "\n") {
		if strings.Contains(line, `method="GET"`) && strings.Contains(line, `/api/v1/users`) {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				val := parts[len(parts)-1]
				if f, err := strconv.ParseFloat(val, 64); err == nil && f >= 1 {
					found = true
				}
			}
		}
	}
	if !found {
		t.Errorf("expected GET /api/v1/users counter >= 1\nOutput:\n%s", bodyStr)
	}
}

// TestMetricsMiddleware_StatusLabels verifies different status codes
// produce the correct label values.
func TestMetricsMiddleware_StatusLabels(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/forbidden" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		if r.URL.Path == "/notfound" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	mw := MetricsMiddleware(next)

	for _, path := range []string{"/ok", "/forbidden", "/notfound"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rr := httptest.NewRecorder()
		mw.ServeHTTP(rr, req)
	}

	// Scrape metrics
	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsRR := httptest.NewRecorder()
	promhttp.Handler().ServeHTTP(metricsRR, metricsReq)
	body, _ := io.ReadAll(metricsRR.Body)
	bodyStr := string(body)

	if !strings.Contains(bodyStr, `status="200"`) {
		t.Error("expected status=200 label")
	}
	if !strings.Contains(bodyStr, `status="403"`) {
		t.Error("expected status=403 label")
	}
	if !strings.Contains(bodyStr, `status="404"`) {
		t.Error("expected status=404 label")
	}
}

// TestPrometheusMetricNaming verifies all exposed metrics follow Prometheus
// naming conventions: snake_case names, HELP text present, TYPE defined.
func TestPrometheusMetricNaming(t *testing.T) {
	// Make a request to populate counters
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mw := MetricsMiddleware(next)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	mw.ServeHTTP(httptest.NewRecorder(), req)

	// Increment auth failure counter so it appears in metrics output
	authFailures.WithLabelValues("test").Inc()

	// Scrape metrics
	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsRR := httptest.NewRecorder()
	promhttp.Handler().ServeHTTP(metricsRR, metricsReq)
	body, _ := io.ReadAll(metricsRR.Body)
	bodyStr := string(body)

	// Required metric names that should be present
	requiredMetrics := []string{
		"ggid_http_requests_total",
		"ggid_http_duration_seconds",
		"ggid_auth_failures_total",
	}

	for _, name := range requiredMetrics {
		// Verify HELP text exists
		helpLine := "# HELP " + name
		if !strings.Contains(bodyStr, helpLine) {
			t.Errorf("metric %q missing HELP text", name)
		}
		// Verify TYPE line exists
		typeLine := "# TYPE " + name
		if !strings.Contains(bodyStr, typeLine) {
			t.Errorf("metric %q missing TYPE line", name)
		}
		// Verify metric name is snake_case (no camelCase, no hyphens)
		for _, ch := range name {
			if ch >= 'A' && ch <= 'Z' {
				t.Errorf("metric %q contains uppercase (not snake_case)", name)
				break
			}
			if ch == '-' {
				t.Errorf("metric %q contains hyphen (should be underscore)", name)
				break
			}
		}
	}
}
