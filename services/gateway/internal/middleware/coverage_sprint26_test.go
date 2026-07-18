package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// --- gzip writer error paths ---

func TestCovS26_GzipWriter_WriteHeader(t *testing.T) {
	handler := Gzip(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestCovS26_GzipWriter_NoAcceptEncoding(t *testing.T) {
	handler := Gzip(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("plain"))
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Body.String() != "plain" {
		t.Fatalf("expected uncompressed 'plain', got %q", rr.Body.String())
	}
}

// --- CORS preflight matching ---

func TestCovS26_CORS_Preflight(t *testing.T) {
	handler := CORSWithConfig(CORSConfig{AllowedOrigins: []string{"*"}})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not call next on OPTIONS")
	}))

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/test", nil)
	req.Header.Set("Origin", "https://app.example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rr.Code)
	}
	if rr.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Fatal("expected ACAO header")
	}
}

func TestCovS26_CORS_ActualRequest(t *testing.T) {
	called := false
	handler := CORSWithConfig(CORSConfig{AllowedOrigins: []string{"*"}})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	req.Header.Set("Origin", "https://app.example.com")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if !called {
		t.Fatal("handler not called for GET")
	}
	if rr.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Fatal("expected ACAO header")
	}
}

// --- circuit breaker half-open transition ---

func TestCovS26_CircuitBreaker_HalfOpenFailure(t *testing.T) {
	cb := NewCircuitBreaker(CircuitConfig{
		MaxFailures:     2,
		Timeout:         30 * time.Millisecond,
		HalfOpenSuccess: 2,
	})

	cb.RecordFailure()
	cb.RecordFailure()
	if cb.State() != CircuitOpen {
		t.Fatalf("expected open, got %s", cb.State())
	}

	time.Sleep(40 * time.Millisecond)
	// State transitions to half-open on Allow() call
	if !cb.Allow() {
		t.Fatal("expected Allow() to return true after timeout (half-open)")
	}

	// Failure during half-open re-opens
	cb.RecordFailure()
	if cb.State() != CircuitOpen {
		t.Fatalf("expected re-opened, got %s", cb.State())
	}
}

// --- error_writer.go integration ---

func TestCovS26_ErrorWriter_AllCodes(t *testing.T) {
	codes := []struct {
		status int
		code   string
	}{
		{http.StatusBadRequest, "bad_request"},
		{http.StatusUnauthorized, "unauthorized"},
		{http.StatusForbidden, "forbidden"},
		{http.StatusNotFound, "not_found"},
		{http.StatusInternalServerError, "internal_error"},
		{http.StatusServiceUnavailable, "unavailable"},
	}

	for _, tc := range codes {
		req := httptest.NewRequest("GET", "/test", nil)
		rr := httptest.NewRecorder()
		WriteError(rr, req, tc.status, tc.code, "test message")

		if rr.Code != tc.status {
			t.Errorf("expected %d, got %d for code %s", tc.status, rr.Code, tc.code)
		}
		body := rr.Body.String()
		if !strings.Contains(body, tc.code) {
			t.Errorf("body %q missing code %q", body, tc.code)
		}
	}
}
