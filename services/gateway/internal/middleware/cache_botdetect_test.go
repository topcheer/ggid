package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// --- Cache Tests ---

func TestCache_GET_CachesResponse(t *testing.T) {
	called := 0
	cache := NewCache(1 * time.Minute)
	handler := cache.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called++
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":"hello"}`))
	}))

	req := httptest.NewRequest("GET", "/api/v1/data", nil)
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, req)

	req2 := httptest.NewRequest("GET", "/api/v1/data", nil)
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)

	if called != 1 {
		t.Errorf("expected 1 upstream call, got %d", called)
	}
	if w2.Header().Get("X-Cache") != "HIT" {
		t.Error("expected cache HIT")
	}
	if w2.Header().Get("ETag") == "" {
		t.Error("expected ETag header")
	}
}

func TestCache_PostNotCached(t *testing.T) {
	called := 0
	cache := NewCache(1 * time.Minute)
	handler := cache.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called++
		w.Write([]byte("ok"))
	}))

	req1 := httptest.NewRequest("POST", "/test", nil)
	handler.ServeHTTP(httptest.NewRecorder(), req1)
	req2 := httptest.NewRequest("POST", "/test", nil)
	handler.ServeHTTP(httptest.NewRecorder(), req2)

	if called != 2 {
		t.Errorf("expected 2 calls for POST (not cached), got %d", called)
	}
}

func TestCache_IfNoneMatch_304(t *testing.T) {
	cache := NewCache(1 * time.Minute)
	handler := cache.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":"cached"}`))
	}))

	// First request populates cache
	req1 := httptest.NewRequest("GET", "/api/v1/data", nil)
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, req1)
	etag := w1.Header().Get("ETag")
	if etag == "" {
		t.Fatal("expected ETag on first response")
	}

	// Second request with If-None-Match
	req2 := httptest.NewRequest("GET", "/api/v1/data", nil)
	req2.Header.Set("If-None-Match", etag)
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)

	if w2.Code != 304 {
		t.Errorf("expected 304, got %d", w2.Code)
	}
}

func TestCache_Invalidate(t *testing.T) {
	called := 0
	cache := NewCache(1 * time.Minute)
	handler := cache.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called++
		w.Write([]byte("ok"))
	}))

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/test", nil))
	cache.Invalidate()
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/test", nil))

	if called != 2 {
		t.Errorf("expected 2 calls after invalidate, got %d", called)
	}
}

// --- Bot Detection Tests ---

func TestBotDetect_BlocksMalicious(t *testing.T) {
	handler := BotDetect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not reach handler")
	}))

	tests := []string{"sqlmap/1.0", "nikto scanner", "nmap scripting"}
	for _, ua := range tests {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("User-Agent", ua)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != 403 {
			t.Errorf("expected 403 for %s, got %d", ua, w.Code)
		}
	}
}

func TestBotDetect_TagsCrawler(t *testing.T) {
	called := false
	handler := BotDetect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Googlebot/2.1)")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !called {
		t.Error("Googlebot should not be blocked")
	}
	if w.Header().Get("X-Bot-Detected") != "googlebot" {
		t.Errorf("expected X-Bot-Detected=googlebot, got %s", w.Header().Get("X-Bot-Detected"))
	}
}

func TestBotDetect_NormalBrowser(t *testing.T) {
	called := false
	handler := BotDetect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15)")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !called {
		t.Error("normal browser should pass")
	}
	if w.Header().Get("X-Bot-Detected") != "" {
		t.Error("should not tag normal browser")
	}
}

func TestBehavioralBotDetect_BlocksHighRate(t *testing.T) {
	detector := NewBehavioralBotDetect(3, time.Minute)
	handler := detector.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	// First 3 pass
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "1.2.3.4:5678"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Errorf("attempt %d: expected 200, got %d", i+1, w.Code)
		}
	}

	// 4th blocked
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "1.2.3.4:5678"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != 429 {
		t.Errorf("expected 429 for 4th request, got %d", w.Code)
	}
}

// --- Gzip Coverage (additional) ---

func TestGzip_CompressesSVG(t *testing.T) {
	handler := GzipBrotli(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/svg+xml")
		w.Write([]byte(`<svg xmlns="http://www.w3.org/2000/svg"><rect/></svg>`))
	}))

	req := httptest.NewRequest("GET", "/logo.svg", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Header().Get("Content-Encoding") != "gzip" {
		t.Error("expected gzip for SVG")
	}
}

func TestGzip_SkipsOctetStream(t *testing.T) {
	handler := GzipBrotli(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(make([]byte, 512))
	}))

	req := httptest.NewRequest("GET", "/download", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Header().Get("Content-Encoding") == "gzip" {
		t.Error("should not compress octet-stream")
	}
}

// --- RateLimiter Coverage (additional) ---

func TestRateLimiter_AllPathsHeaders(t *testing.T) {
	rl := NewRateLimiter(RateLimitConfig{APILimit: 10, Window: time.Minute})
	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/api/v1/anything", nil)
	req.RemoteAddr = "7.7.7.7:1234"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Header().Get("X-RateLimit-Limit") != "10" {
		t.Errorf("expected limit 10, got %s", w.Header().Get("X-RateLimit-Limit"))
	}
}

// --- API Key Coverage (additional) ---

func TestAPIKeyAuth_ExpiredKey_401(t *testing.T) {
	validator := NewMemoryAPIKeyValidator()
	validator.AddKey("ggid_expired", "t1", "u1", nil)
	// Manually expire
	validator.keys["ggid_expired"].expires = time.Now().Add(-1 * time.Hour)

	handler := APIKeyAuth(validator)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "ggid_expired")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Errorf("expected 401 for expired key, got %d", w.Code)
	}
}
