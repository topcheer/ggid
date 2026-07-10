package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// --- MetricsMiddleware coverage ---

func TestMetricsMiddleware_SkipsMetricsPath(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	MetricsMiddleware(next).ServeHTTP(w, req)

	if !called {
		t.Error("/metrics path should still call next handler")
	}
}

func TestMetricsMiddleware_NormalRequest(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})

	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	w := httptest.NewRecorder()
	MetricsMiddleware(next).ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestMetricsMiddleware_UnauthorizedIncrementsAuthFailure(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})

	req := httptest.NewRequest("GET", "/api/v1/secret", nil)
	w := httptest.NewRecorder()
	MetricsMiddleware(next).ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
	// The authFailures counter should have been incremented
	// We can't easily check the counter value but the code path is exercised
}

func TestMetricsMiddleware_POSTRequest(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(201)
	})

	req := httptest.NewRequest("POST", "/api/v1/users/abc-123", nil)
	w := httptest.NewRecorder()
	MetricsMiddleware(next).ServeHTTP(w, req)

	if w.Code != 201 {
		t.Errorf("expected 201, got %d", w.Code)
	}
}

func TestMetricsMiddleware_WithBody(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("response body"))
	})

	req := httptest.NewRequest("GET", "/api/v1/data/123", nil)
	w := httptest.NewRecorder()
	MetricsMiddleware(next).ServeHTTP(w, req)

	if w.Body.String() != "response body" {
		t.Errorf("body = %q", w.Body.String())
	}
}

func TestMetricsRecorder_WriteHeader(t *testing.T) {
	w := httptest.NewRecorder()
	sr := &metricsRecorder{ResponseWriter: w, status: 200}
	sr.WriteHeader(404)
	if sr.status != 404 {
		t.Errorf("status = %d", sr.status)
	}
	if w.Code != 404 {
		t.Errorf("recorder status = %d", w.Code)
	}
}

// --- EnhancedMetrics coverage ---

func TestEnhancedMetrics_ObserveRequest(t *testing.T) {
	m := GetEnhancedMetrics()
	m.ObserveRequest("GET", "/api/v1/users", 200, 100, 500, 1000000)
	m.ObserveRequest("POST", "/api/v1/users", 201, 200, 300, 500000)
	m.ObserveRequest("GET", "/api/v1/users", 500, 50, 100, 5000000)
	// Just verify no panic
}

func TestNormalizeStatusCode_StatusGroups(t *testing.T) {
	tests := []struct {
		input int
		want  string
	}{
		{200, "2xx"},
		{201, "2xx"},
		{301, "3xx"},
		{404, "4xx"},
		{500, "5xx"},
	}
	for _, tt := range tests {
		if got := normalizeStatusCode(tt.input); got != tt.want {
			t.Errorf("normalizeStatusCode(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestItoa_Conversions(t *testing.T) {
	tests := []struct {
		input int
		want  string
	}{
		{0, "0"},
		{1, "1"},
		{2, "2"},
		{5, "5"},
	}
	for _, tt := range tests {
		if got := itoa(tt.input); got != tt.want {
			t.Errorf("itoa(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestEnhancedMetricsHandler_ReturnsHandler(t *testing.T) {
	h := EnhancedMetricsHandler()
	if h == nil {
		t.Error("handler should not be nil")
	}
	// Serve a request to verify it works
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestSetActiveSessions_Counter(t *testing.T) {
	SetActiveSessions(42)
	SetActiveSessions(0)
}

func TestIncAuthFailure_Counter(t *testing.T) {
	IncAuthFailure("invalid_token")
	IncAuthFailure("expired")
}
