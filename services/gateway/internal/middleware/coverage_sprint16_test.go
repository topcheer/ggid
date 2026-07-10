package middleware

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// === Health Check tests ===

func TestHealthChecker_DefaultHealthy(t *testing.T) {
	hc := NewHealthChecker(nil)
	if !hc.IsHealthy("unknown-backend") {
		t.Error("Unknown backend should default to healthy")
	}
}

func TestHealthChecker_MarkSuccess(t *testing.T) {
	cfg := &HealthCheckConfig{SuccessThreshold: 2, FailureThreshold: 3, Interval: time.Second, Timeout: time.Second}
	hc := NewHealthChecker(cfg)

	hc.MarkSuccess("b1")
	hc.MarkSuccess("b1")

	if !hc.IsHealthy("b1") {
		t.Error("After 2 successes should be healthy")
	}
}

func TestHealthChecker_MarkFailureTrip(t *testing.T) {
	cfg := &HealthCheckConfig{SuccessThreshold: 2, FailureThreshold: 3, Interval: time.Second, Timeout: time.Second}
	hc := NewHealthChecker(cfg)

	hc.MarkFailure("b1", "timeout")
	hc.MarkFailure("b1", "timeout")
	if !hc.IsHealthy("b1") {
		t.Error("2 failures should not trip (threshold=3)")
	}

	hc.MarkFailure("b1", "timeout")
	if hc.IsHealthy("b1") {
		t.Error("3 failures should trip circuit")
	}
}

func TestHealthChecker_Recovery(t *testing.T) {
	cfg := &HealthCheckConfig{SuccessThreshold: 2, FailureThreshold: 2, Interval: time.Second, Timeout: time.Second}
	hc := NewHealthChecker(cfg)

	// Trip it
	hc.MarkFailure("b1", "err")
	hc.MarkFailure("b1", "err")
	if hc.IsHealthy("b1") {
		t.Error("Should be unhealthy after 2 failures")
	}

	// Recover
	hc.MarkSuccess("b1")
	if hc.IsHealthy("b1") {
		t.Error("1 success should not recover (threshold=2)")
	}
	hc.MarkSuccess("b1")
	if !hc.IsHealthy("b1") {
		t.Error("2 successes should recover")
	}
}

func TestHealthChecker_GetHealth(t *testing.T) {
	hc := NewHealthChecker(nil)
	hc.MarkFailure("b1", "conn refused")
	h := hc.GetHealth("b1")
	if h == nil {
		t.Fatal("Should not be nil")
	}
	if h.LastError != "conn refused" {
		t.Errorf("LastError: got '%s'", h.LastError)
	}
	if h.ConsecutiveFails != 1 {
		t.Errorf("Fails: want 1, got %d", h.ConsecutiveFails)
	}
}

func TestHealthChecker_AllHealth(t *testing.T) {
	hc := NewHealthChecker(nil)
	hc.MarkSuccess("b1")
	hc.MarkFailure("b2", "err")
	all := hc.AllHealth()
	if len(all) != 2 {
		t.Errorf("want 2 backends, got %d", len(all))
	}
}

func TestHealthChecker_ResetsCounters(t *testing.T) {
	hc := NewHealthChecker(nil)
	hc.MarkFailure("b1", "err")
	hc.MarkFailure("b1", "err")
	hc.MarkSuccess("b1") // resets fail counter
	h := hc.GetHealth("b1")
	if h.ConsecutiveFails != 0 {
		t.Errorf("Fails after success: want 0, got %d", h.ConsecutiveFails)
	}
	if h.ConsecutiveOKs != 1 {
		t.Errorf("OKs: want 1, got %d", h.ConsecutiveOKs)
	}
}

func TestDefaultHealthCheckConfig(t *testing.T) {
	cfg := DefaultHealthCheckConfig()
	if cfg.Interval != 10*time.Second {
		t.Errorf("Interval: got %v", cfg.Interval)
	}
	if cfg.FailureThreshold != 3 {
		t.Errorf("FailureThreshold: got %d", cfg.FailureThreshold)
	}
}

// === Tenant Context tests ===

func TestInjectTenantContext_FromHeader(t *testing.T) {
	var tid string
	handler := InjectTenantContext(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tid = TenantIDFromContext(r.Context())
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/api/data", nil)
	req.Header.Set("X-Tenant-ID", "from-header")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if tid != "from-header" {
		t.Errorf("want 'from-header', got '%s'", tid)
	}
}

func TestInjectTenantContext_FromJWT(t *testing.T) {
	// Build JWT with tenant_id
	payload, _ := json.Marshal(map[string]string{"tenant_id": "from-jwt"})
	payloadB64 := base64.RawURLEncoding.EncodeToString(payload)
	token := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256"}`)) + "." + payloadB64 + "." + base64.RawURLEncoding.EncodeToString([]byte("sig"))

	var tid string
	handler := InjectTenantContext(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tid = TenantIDFromContext(r.Context())
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/api/data", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if tid != "from-jwt" {
		t.Errorf("want 'from-jwt', got '%s'", tid)
	}
}

func TestInjectTenantContext_NoTenant(t *testing.T) {
	handler := InjectTenantContext(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/api/data", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Errorf("want 200, got %d", rr.Code)
	}
}

func TestInjectTenantContext_HeaderPriorityOverJWT(t *testing.T) {
	payload, _ := json.Marshal(map[string]string{"tenant_id": "from-jwt"})
	payloadB64 := base64.RawURLEncoding.EncodeToString(payload)
	token := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256"}`)) + "." + payloadB64 + "." + base64.RawURLEncoding.EncodeToString([]byte("sig"))

	var tid string
	handler := InjectTenantContext(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tid = TenantIDFromContext(r.Context())
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/api/data", nil)
	req.Header.Set("X-Tenant-ID", "from-header")
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if tid != "from-header" {
		t.Errorf("header should win: want 'from-header', got '%s'", tid)
	}
}

func TestTenantIDFromContext_Empty(t *testing.T) {
	tid := TenantIDFromContext(context.Background())
	if tid != "" {
		t.Error("Should be empty")
	}
}

// === Body size limiter coverage ===

func TestMaxBodySize_Allowed16(t *testing.T) {
	handler := MaxBodySize(1024)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("POST", "/api/data", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Errorf("want 200, got %d", rr.Code)
	}
}

func TestParseMaxBodySize16(t *testing.T) {
	tests := []struct {
		input string
		want  int64
	}{
		{"10MB", 10 * 1024 * 1024},
		{"1GB", 1024 * 1024 * 1024},
		{"512KB", 512 * 1024},
		{"invalid", 10 << 20}, // invalid returns default 10MB
	}
	for _, tt := range tests {
		if got := ParseMaxBodySize(tt.input); got != tt.want {
			t.Errorf("ParseMaxBodySize(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

// === Response cache coverage ===

func TestResponseCache_DefaultConfig16(t *testing.T) {
	cfg := DefaultResponseCacheConfig()
	if cfg.TTL <= 0 {
		t.Error("TTL should be positive")
	}
}

func TestResponseCache_Clear16(t *testing.T) {
	rc := NewResponseCache(DefaultResponseCacheConfig())
	rc.Clear() // should not panic
}
