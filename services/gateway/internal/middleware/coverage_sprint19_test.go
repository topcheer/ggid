package middleware

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// ==================== Task 1: Coverage Boost ====================

// --- cache.go: Invalidate + If-None-Match ---

func TestCache_Invalidate_C19(t *testing.T) {
	cache := NewCache(5 * time.Minute)
	h := cache.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("cached"))
	}))
	// First request: cache miss, stores entry
	rr1 := httptest.NewRecorder()
	req1 := httptest.NewRequest("GET", "/data", nil)
	h.ServeHTTP(rr1, req1)
	if rr1.Header().Get("X-Cache") != "" {
		t.Error("first request should be cache miss")
	}
	// Second request: cache hit
	rr2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/data", nil)
	h.ServeHTTP(rr2, req2)
	if rr2.Header().Get("X-Cache") != "HIT" {
		t.Error("second request should be cache hit")
	}
	// Invalidate
	cache.Invalidate()
	// Third request: cache miss again
	rr3 := httptest.NewRecorder()
	req3 := httptest.NewRequest("GET", "/data", nil)
	h.ServeHTTP(rr3, req3)
	if rr3.Header().Get("X-Cache") == "HIT" {
		t.Error("third request after invalidate should be miss")
	}
}

func TestCache_IfNoneMatch_NotModified_C19(t *testing.T) {
	cache := NewCache(5 * time.Minute)
	h := cache.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("etag-data"))
	}))
	// First request: get ETag
	rr1 := httptest.NewRecorder()
	req1 := httptest.NewRequest("GET", "/etag", nil)
	h.ServeHTTP(rr1, req1)
	etag := rr1.Header().Get("ETag")
	if etag == "" {
		t.Fatal("expected ETag header")
	}
	// Second request with If-None-Match → 304
	rr2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/etag", nil)
	req2.Header.Set("If-None-Match", etag)
	h.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusNotModified {
		t.Errorf("status = %d, want %d", rr2.Code, http.StatusNotModified)
	}
}

func TestCache_PostNotCached_C19(t *testing.T) {
	cache := NewCache(5 * time.Minute)
	called := 0
	h := cache.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("data"))
	}))
	// POST should not be cached
	rr1 := httptest.NewRecorder()
	req1 := httptest.NewRequest("POST", "/submit", nil)
	h.ServeHTTP(rr1, req1)
	rr2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("POST", "/submit", nil)
	h.ServeHTTP(rr2, req2)
	if called != 2 {
		t.Errorf("POST should not be cached, called %d times", called)
	}
}

// --- botdetect.go: behavioral threshold boundary ---

func TestBehavioralBotDetect_ThresholdBoundary_C19(t *testing.T) {
	detector := NewBehavioralBotDetect(3, 10*time.Second)
	h := detector.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	// Requests 1-3: allowed (threshold=3, count>3 triggers)
	for i := 0; i < 3; i++ {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("request %d: status = %d, want 200", i+1, rr.Code)
		}
	}
	// Request 4: blocked (count=4 > threshold=3)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("request 4: status = %d, want %d", rr.Code, http.StatusTooManyRequests)
	}
}

func TestBehavioralBotDetect_EmptyIP_C19(t *testing.T) {
	detector := NewBehavioralBotDetect(3, 10*time.Second)
	called := false
	h := detector.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = ""
	h.ServeHTTP(rr, req)
	if !called {
		t.Error("should pass through with empty IP")
	}
}

// --- retry.go: max retries boundary ---

func TestRetryMiddleware_MaxRetriesExhausted_C19(t *testing.T) {
	cfg := &RetryConfig{
		MaxAttempts:     3,
		InitialDelay:    1 * time.Millisecond,
		MaxDelay:        5 * time.Millisecond,
		RetryableStatus: []int{503},
	}
	callCount := 0
	h := RetryMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(503)
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	h.ServeHTTP(rr, req)
	if callCount != 3 {
		t.Errorf("callCount = %d, want 3", callCount)
	}
	if rr.Code != 503 {
		t.Errorf("status = %d, want 503", rr.Code)
	}
}

func TestRetryMiddleware_NonRetryableMethod_C19(t *testing.T) {
	cfg := &RetryConfig{
		MaxAttempts:     3,
		InitialDelay:    1 * time.Millisecond,
		RetryableStatus: []int{503},
	}
	callCount := 0
	h := RetryMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(503)
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/", nil)
	h.ServeHTTP(rr, req)
	if callCount != 1 {
		t.Errorf("non-idempotent should not retry, callCount = %d", callCount)
	}
}

func TestRetryMiddleware_RetryHeader_C19(t *testing.T) {
	cfg := &RetryConfig{
		MaxAttempts:     2,
		InitialDelay:    1 * time.Millisecond,
		RetryableStatus: []int{503},
	}
	callCount := 0
	h := RetryMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			w.WriteHeader(503)
			return
		}
		w.WriteHeader(200)
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	h.ServeHTTP(rr, req)
	if rr.Header().Get("X-Retry-Count") != "1" {
		t.Errorf("X-Retry-Count = %q", rr.Header().Get("X-Retry-Count"))
	}
}

func TestIsRetryableStatus_C19(t *testing.T) {
	cfg := &RetryConfig{RetryableStatus: []int{502, 503, 504}}
	if !cfg.isRetryableStatus(503) {
		t.Error("503 should be retryable")
	}
	if cfg.isRetryableStatus(200) {
		t.Error("200 should not be retryable")
	}
}

// --- request_logging.go: Info/Warn/Error methods at 0% ---

func TestJSONLogger_Levels_C19(t *testing.T) {
	var output []string
	logger := &JSONLogger{writer: func(s string) { output = append(output, s) }}
	logger.Info(LogEntry{Method: "GET", Path: "/info"})
	logger.Warn(LogEntry{Method: "GET", Path: "/warn"})
	logger.Error(LogEntry{Method: "GET", Path: "/error"})
	if len(output) != 3 {
		t.Errorf("expected 3 outputs, got %d", len(output))
	}
}

// --- session.go: Middleware pass ---

func TestSessionMiddleware_PublicPath_C19(t *testing.T) {
	sm := NewSessionManager(nil)
	called := false
	h := sm.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/healthz", nil)
	h.ServeHTTP(rr, req)
	if !called {
		t.Error("public path should pass through")
	}
}

// --- apiversion.go: versionFromPath + stripVersionPrefix ---

func TestVersionFromPath_C19(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/api/v1/users", "1"},
		{"/api/v2/orgs", "2"},
		{"/api/v10/things", "10"},
		{"/api/users", ""},
		{"/other/v1", ""},
	}
	for _, tt := range tests {
		if got := VersionFromPath(tt.path); got != tt.want {
			t.Errorf("VersionFromPath(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestStripVersionPrefix_C19(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v2/users?limit=10", nil)
	stripped := stripVersionPrefix(req)
	if stripped.URL.Path != "/api/users" {
		t.Errorf("path = %q, want /api/users", stripped.URL.Path)
	}
}

// --- security_headers.go: mergeSecurityHeaders ---

func TestMergeSecurityHeaders_Full_C19(t *testing.T) {
	base := &SecurityHeadersConfig{Enabled: true, ContentTypeNosniff: true, HSTSMaxAge: 3600}
	override := &SecurityHeadersConfig{
		CSP:       "default-src 'none'",
		FrameDeny: true,
	}
	merged := mergeSecurityHeaders(base, override)
	if merged.CSP != "default-src 'none'" {
		t.Error("should pick up override CSP")
	}
	if !merged.ContentTypeNosniff {
		t.Error("should preserve base nosniff")
	}
	if !merged.FrameDeny {
		t.Error("should pick up override FrameDeny")
	}
}

// ==================== Task 2: Response Time Tracking ====================

func TestResponseTimeMiddleware_Basic_C19(t *testing.T) {
	tracker := NewResponseTimeTracker(100)
	h := ResponseTimeMiddleware(tracker)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/test", nil)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("status = %d", rr.Code)
	}
	if rr.Header().Get("X-Response-Time") == "" {
		t.Error("missing X-Response-Time header")
	}
}

func TestResponseTimeTracker_Percentiles_C19(t *testing.T) {
	tracker := NewResponseTimeTracker(100)
	for i := 1; i <= 100; i++ {
		tracker.record(float64(i))
	}
	p50, p99 := tracker.Percentiles()
	if p50 < 40 || p50 > 60 {
		t.Errorf("p50 = %.1f, expected ~50", p50)
	}
	if p99 < 90 || p99 > 100 {
		t.Errorf("p99 = %.1f, expected ~99", p99)
	}
}

func TestResponseTimeTracker_Empty_C19(t *testing.T) {
	tracker := NewResponseTimeTracker(10)
	p50, p99 := tracker.Percentiles()
	if p50 != 0 || p99 != 0 {
		t.Errorf("empty tracker should return 0,0 got %f,%f", p50, p99)
	}
}

func TestResponseTimeMiddleware_NilTracker_C19(t *testing.T) {
	h := ResponseTimeMiddleware(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("nil tracker should still work, status = %d", rr.Code)
	}
}

// ==================== Task 3: API Versioning (deprecation headers) ====================

func TestAPIVersioningMiddleware_Deprecation_C19(t *testing.T) {
	sunset := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	cfg := DefaultAPIVersionConfig()
	deprecations := map[string]*DeprecationInfo{
		"1": {Version: "1", Sunset: sunset, Deprecation: true, Link: "https://docs.example.com/migrate"},
	}
	h := APIVersioningMiddleware(cfg, deprecations)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	h.ServeHTTP(rr, req)
	if rr.Header().Get("Deprecation") != "true" {
		t.Error("missing Deprecation header")
	}
	if rr.Header().Get("Sunset") == "" {
		t.Error("missing Sunset header")
	}
	if rr.Header().Get("Link") == "" {
		t.Error("missing Link header")
	}
}

func TestAPIVersioningMiddleware_NoDeprecation_C19(t *testing.T) {
	cfg := DefaultAPIVersionConfig()
	deprecations := map[string]*DeprecationInfo{}
	h := APIVersioningMiddleware(cfg, deprecations)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v2/users", nil)
	h.ServeHTTP(rr, req)
	if rr.Header().Get("Deprecation") != "" {
		t.Error("should not set Deprecation for non-deprecated version")
	}
	if rr.Header().Get("X-API-Version") != "2" {
		t.Errorf("X-API-Version = %q, want '2'", rr.Header().Get("X-API-Version"))
	}
}

func TestExtractVersionFromPath_C19(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/v1/users", "1"},
		{"/v2/orgs", "2"},
		{"/v10/items", "10"},
		{"/api/users", ""},
		{"", ""},
	}
	for _, tt := range tests {
		if got := ExtractVersionFromPath(tt.path); got != tt.want {
			t.Errorf("ExtractVersionFromPath(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

// ==================== Task 4: GeoIP ====================

func TestGeoIPMiddleware_NoDB_C19(t *testing.T) {
	cfg := &GeoIPConfig{}
	h := GeoIPMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.1:1234"
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("without DB, should pass through, status = %d", rr.Code)
	}
}

func TestGeoIPMiddleware_BlockCountry_C19(t *testing.T) {
	// Inject mock country lookup
	old := SetCountryLookup(func(ip string) string {
		if ip == "1.2.3.4" {
			return "CN"
		}
		return ""
	})
	defer SetCountryLookup(old)

	cfg := &GeoIPConfig{BlockedCountries: []string{"CN"}}
	h := GeoIPMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called for blocked country")
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "1.2.3.4:5678"
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}
}

func TestGeoIPMiddleware_AllowCountry_C19(t *testing.T) {
	old := SetCountryLookup(func(ip string) string {
		if ip == "1.2.3.4" {
			return "US"
		}
		return ""
	})
	defer SetCountryLookup(old)

	cfg := &GeoIPConfig{AllowedCountries: []string{"US", "CA"}}
	h := GeoIPMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "1.2.3.4:5678"
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("allowed country should pass, status = %d", rr.Code)
	}
	if rr.Header().Get("X-Geo-Country") != "US" {
		t.Errorf("X-Geo-Country = %q", rr.Header().Get("X-Geo-Country"))
	}
}

func TestGeoIPMiddleware_DenyNotInAllowList_C19(t *testing.T) {
	old := SetCountryLookup(func(ip string) string {
		if ip == "1.2.3.4" {
			return "BR"
		}
		return ""
	})
	defer SetCountryLookup(old)

	cfg := &GeoIPConfig{AllowedCountries: []string{"US", "CA"}}
	h := GeoIPMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not reach handler")
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "1.2.3.4:5678"
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}
}

func TestGeoIPMiddleware_UpstreamCountryHeader_C19(t *testing.T) {
	cfg := &GeoIPConfig{}
	h := GeoIPMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Geo-Country", "DE")
	req.RemoteAddr = "192.168.1.1:1234"
	h.ServeHTTP(rr, req)
	if rr.Header().Get("X-Geo-Country") != "DE" {
		t.Error("should preserve upstream X-Geo-Country header")
	}
}

func TestGeoIPMiddleware_XForwardedFor_C19(t *testing.T) {
	old := SetCountryLookup(func(ip string) string {
		if ip == "5.6.7.8" {
			return "GB"
		}
		return ""
	})
	defer SetCountryLookup(old)

	cfg := &GeoIPConfig{TrustXForwardedFor: true}
	h := GeoIPMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Forwarded-For", "5.6.7.8, 10.0.0.1")
	req.RemoteAddr = "10.0.0.1:1234"
	h.ServeHTTP(rr, req)
	if rr.Header().Get("X-Geo-Country") != "GB" {
		t.Errorf("X-Geo-Country = %q, want GB", rr.Header().Get("X-Geo-Country"))
	}
}

func TestExtractGeoIPClientIP_C19(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "1.2.3.4:5678"
	if ip := extractGeoIPClientIP(r, false); ip != "1.2.3.4" {
		t.Errorf("got %q", ip)
	}
}

// ==================== Task 5: Shadow Traffic Mirror ====================

func TestShadowMirror_BasicPass_C19(t *testing.T) {
	cfg := &ShadowMirrorConfig{
		TargetURL: "",
		Percentage: 100,
	}
	sm := NewShadowMirror(cfg)
	called := false
	h := sm.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("primary"))
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/data", nil)
	h.ServeHTTP(rr, req)
	if !called {
		t.Error("primary handler must always be called")
	}
	if rr.Body.String() != "primary" {
		t.Errorf("body = %q", rr.Body.String())
	}
}

func TestShadowMirror_WithBackend_C19(t *testing.T) {
	shadowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Shadow-Traffic") != "true" {
			t.Error("shadow request should have X-Shadow-Traffic header")
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("shadow-response"))
	}))
	defer shadowServer.Close()

	var resultCount atomic.Int32
	cfg := &ShadowMirrorConfig{
		TargetURL:  shadowServer.URL,
		Percentage: 100,
		Timeout:    2 * time.Second,
		OnResult: func(primaryLatency, shadowLatency time.Duration, status int) {
			resultCount.Add(1)
			if status != 200 {
				t.Errorf("shadow status = %d", status)
			}
		},
	}
	sm := NewShadowMirror(cfg)
	h := sm.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("primary"))
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/data", nil)
	h.ServeHTTP(rr, req)

	if rr.Body.String() != "primary" {
		t.Errorf("primary body should be 'primary', got %q", rr.Body.String())
	}

	// Wait for async shadow
	time.Sleep(200 * time.Millisecond)
	if resultCount.Load() == 0 {
		t.Error("shadow OnResult callback should have been called")
	}
}

func TestShadowMirror_Percentage0_C19(t *testing.T) {
	shadowCalled := atomic.Int32{}
	shadowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		shadowCalled.Add(1)
	}))
	defer shadowServer.Close()

	cfg := &ShadowMirrorConfig{
		TargetURL:  shadowServer.URL,
		Percentage: 0,
	}
	sm := NewShadowMirror(cfg)
	h := sm.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 10; i++ {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		h.ServeHTTP(rr, req)
	}
	time.Sleep(100 * time.Millisecond)
	if shadowCalled.Load() > 0 {
		t.Error("0% should never mirror")
	}
}

func TestShadowMirror_NeverBlocksPrimary_C19(t *testing.T) {
	// Use a non-responsive shadow backend to verify it doesn't block
	cfg := &ShadowMirrorConfig{
		TargetURL:  "http://127.0.0.1:1", // connection refused
		Percentage: 100,
		Timeout:    100 * time.Millisecond,
	}
	sm := NewShadowMirror(cfg)
	start := time.Now()
	h := sm.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	h.ServeHTTP(rr, req)
	elapsed := time.Since(start)
	if elapsed > 50*time.Millisecond {
		t.Errorf("primary took %v, shadow should be async", elapsed)
	}
	if rr.Code != http.StatusOK {
		t.Errorf("status = %d", rr.Code)
	}
}

func TestShadowMirror_Stats_C19(t *testing.T) {
	cfg := &ShadowMirrorConfig{
		TargetURL:  "",
		Percentage: 100,
	}
	sm := NewShadowMirror(cfg)
	h := sm.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	for i := 0; i < 5; i++ {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		h.ServeHTTP(rr, req)
	}
	total, mirrored := sm.Stats()
	if total == 0 {
		t.Error("total should be > 0")
	}
	if mirrored != 0 {
		t.Errorf("mirrored = %d, want 0 (no target)", mirrored)
	}
}
