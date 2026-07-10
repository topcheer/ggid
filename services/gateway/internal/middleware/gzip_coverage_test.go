package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// --- Gzip Tests ---

func TestGzip_CompressesJSON(t *testing.T) {
	handler := Gzip(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"message":"hello world this is a long enough message to compress"}`))
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Header().Get("Content-Encoding") != "gzip" {
		t.Error("expected gzip encoding for JSON response")
	}
}

func TestGzip_SkipsWhenNotAccepted(t *testing.T) {
	called := false
	handler := Gzip(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true}`))
		called = true
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	// No Accept-Encoding: gzip
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !called {
		t.Error("handler should be called")
	}
	if w.Header().Get("Content-Encoding") == "gzip" {
		t.Error("should not set gzip without Accept-Encoding")
	}
}

func TestGzip_SkipsImages(t *testing.T) {
	handler := Gzip(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write(make([]byte, 1024))
	}))

	req := httptest.NewRequest("GET", "/image.png", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Header().Get("Content-Encoding") == "gzip" {
		t.Error("should not compress images")
	}
}

func TestGzip_SkipsVideo(t *testing.T) {
	handler := Gzip(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "video/mp4")
		w.Write(make([]byte, 2048))
	}))

	req := httptest.NewRequest("GET", "/video.mp4", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Header().Get("Content-Encoding") == "gzip" {
		t.Error("should not compress video")
	}
}

func TestGzip_CompressesHTML(t *testing.T) {
	html := "<html><body><h1>Hello</h1></body></html>"
	handler := Gzip(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(html))
	}))

	req := httptest.NewRequest("GET", "/page", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Header().Get("Content-Encoding") != "gzip" {
		t.Error("expected gzip for HTML")
	}
}

// --- Metrics Tests ---

func TestMetricsMiddleware_RecordsRequests(t *testing.T) {
	handler := MetricsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Verify metrics endpoint returns data
	metricsReq := httptest.NewRequest("GET", "/metrics", nil)
	metricsW := httptest.NewRecorder()
	MetricsHandler().ServeHTTP(metricsW, metricsReq)

	body := metricsW.Body.String()
	if !strings.Contains(body, "http_requests_total") {
		t.Error("expected http_requests_total in metrics")
	}
}

func TestMetricsHandler_ReturnsPrometheusFormat(t *testing.T) {
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	MetricsHandler().ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// --- CSRF Tests (additional coverage) ---

func TestCSRFProtect_PUTRequiresToken(t *testing.T) {
	called := false
	handler := CSRFProtect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest("PUT", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if called {
		t.Error("PUT without CSRF should be blocked")
	}
	if w.Code != 403 {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestCSRFProtect_DELETERequiresToken(t *testing.T) {
	handler := CSRFProtect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	req := httptest.NewRequest("DELETE", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 403 {
		t.Errorf("expected 403 for DELETE without CSRF, got %d", w.Code)
	}
}

// --- Security Headers Tests (additional coverage) ---

func TestSecurityHeaders_AllPresent(t *testing.T) {
	handler := SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	headers := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":       "DENY",
	}
	for h, expected := range headers {
		if got := w.Header().Get(h); got != expected {
			t.Errorf("expected %s=%s, got %s", h, expected, got)
		}
	}
}

// --- CORSWithConfig Tests (additional) ---

func TestCORSWithConfig_WildcardOrigin(t *testing.T) {
	cfg := CORSConfig{AllowedOrigins: []string{"*"}}
	handler := CORSWithConfig(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://anything.com")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("expected wildcard, got %s", w.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORSWithConfig_CredentialsWithOrigin(t *testing.T) {
	cfg := CORSConfig{
		AllowedOrigins:   []string{"https://app.example.com"},
		AllowCredentials: true,
	}
	handler := CORSWithConfig(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://app.example.com")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "https://app.example.com" {
		t.Error("expected specific origin echoed")
	}
	if w.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Error("expected credentials header")
	}
}

// --- RateLimiter Tests (additional) ---

func TestRateLimiter_XForwardedFor(t *testing.T) {
	rl := NewRateLimiter(RateLimitConfig{LoginLimit: 1, Window: 60 * 1000000000})
	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	// First request from IP via X-Forwarded-For
	req := httptest.NewRequest("POST", "/api/v1/auth/login", nil)
	req.Header.Set("X-Forwarded-For", "5.6.7.8")
	handler.ServeHTTP(httptest.NewRecorder(), req)

	// Second from same forwarded IP
	req2 := httptest.NewRequest("POST", "/api/v1/auth/login", nil)
	req2.Header.Set("X-Forwarded-For", "5.6.7.8")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req2)

	if w.Code != 429 {
		t.Errorf("expected 429 for second request from same forwarded IP, got %d", w.Code)
	}
}

// --- normalizePath Tests ---

func TestNormalizePath_PlainPath(t *testing.T) {
	result := normalizePath("/api/v1/users")
	if result != "/api/v1/users" {
		t.Errorf("expected /api/v1/users, got %s", result)
	}
}

func TestNormalizePath_WithUUID(t *testing.T) {
	result := normalizePath("/api/v1/users/550e8400-e29b-41d4-a716-446655440000")
	if result != "/api/v1/users/{id}" {
		t.Errorf("expected /api/v1/users/{id}, got %s", result)
	}
}

// --- Session Manager Tests (additional) ---

func TestSessionManager_PublicPathPasses(t *testing.T) {
	sm := NewSessionManager(nil)
	called := false
	handler := sm.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest("POST", "/api/v1/auth/login", nil)
	handler.ServeHTTP(httptest.NewRecorder(), req)
	if !called {
		t.Error("public path should pass through")
	}
}

func TestSessionManager_NilRedisOnProtectedPath(t *testing.T) {
	sm := NewSessionManager(nil)
	called := false
	handler := sm.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	// Protected path with nil Redis → should pass through (fail-open)
	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	handler.ServeHTTP(httptest.NewRecorder(), req)
	if !called {
		t.Error("should fail-open with nil Redis")
	}
}

// --- IsAPIKeyRequest Tests ---

func TestIsAPIKeyRequest_Header(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "ggid_test123")
	if !IsAPIKeyRequest(req) {
		t.Error("should detect API key in header")
	}
}

func TestIsAPIKeyRequest_QueryParam(t *testing.T) {
	req := httptest.NewRequest("GET", "/test?api_key=ggid_test", nil)
	if !IsAPIKeyRequest(req) {
		t.Error("should detect API key in query")
	}
}

func TestIsAPIKeyRequest_None(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	if IsAPIKeyRequest(req) {
		t.Error("should not detect API key when absent")
	}
}
