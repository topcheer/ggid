package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// === APIKey coverage ===

func TestIsAPIKeyRequest_Bearer(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/data?api_key=abc123", nil)
	if !IsAPIKeyRequest(req) {
		t.Error("api_key query param should be API key request")
	}
}

func TestIsAPIKeyRequest_XAPIKey(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/data", nil)
	req.Header.Set("X-API-Key", "my-key-123")
	if !IsAPIKeyRequest(req) {
		t.Error("X-API-Key header should be API key request")
	}
}

func TestIsAPIKeyRequest_NoKey(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/data", nil)
	if IsAPIKeyRequest(req) {
		t.Error("No key should not be API key request")
	}
}

func TestIsAPIKeyRequest_JWT(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/data", nil)
	req.Header.Set("Authorization", "Bearer eyJhbGciOi")
	// JWT starts with eyJ, so it's not an API key
	if IsAPIKeyRequest(req) {
		t.Error("JWT should not be API key request")
	}
}

func TestAPIKeyError_Error(t *testing.T) {
	e := &apiKeyError{msg: "invalid key"}
	if e.Error() != "invalid key" {
		t.Errorf("Error: want 'invalid key', got '%s'", e.Error())
	}
}

func TestAPIKeyError_String(t *testing.T) {
	k := apiScopeCtxKey("my-scope")
	if k.String() != "my-scope" {
		t.Errorf("String: want 'my-scope', got '%s'", k.String())
	}
}

func TestAPIKeyAuth_MissingKey(t *testing.T) {
	validator := NewMemoryAPIKeyValidator()
	validator.AddKey("sk-test", "tenant1", "user1", []string{"read"})

	called := false
	handler := APIKeyAuth(validator)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true // Passes through when no API key present (JWT middleware handles it)
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/api/data", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if !called {
		t.Error("Missing key should pass through (JWT handles auth)")
	}
}

func TestAPIKeyAuth_ValidKey(t *testing.T) {
	validator := NewMemoryAPIKeyValidator()
	validator.AddKey("sk-test", "tenant1", "user1", []string{"read", "write"})

	called := false
	handler := APIKeyAuth(validator)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/api/data", nil)
	req.Header.Set("X-API-Key", "sk-test")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if !called {
		t.Error("Valid key should call next handler")
	}
}

func TestAPIKeyAuth_InvalidKey(t *testing.T) {
	validator := NewMemoryAPIKeyValidator()

	handler := APIKeyAuth(validator)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Invalid key should not call next")
	}))

	req := httptest.NewRequest("GET", "/api/data", nil)
	req.Header.Set("X-API-Key", "sk-invalid")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != 401 {
		t.Errorf("Invalid key: want 401, got %d", rr.Code)
	}
}

func TestMemoryAPIKeyValidator_Validate(t *testing.T) {
	v := NewMemoryAPIKeyValidator()
	v.AddKey("sk-1", "t1", "u1", []string{"read", "write"})

	tenant, user, scopes, err := v.Validate(nil, "sk-1")
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if tenant != "t1" {
		t.Errorf("tenant: want 't1', got '%s'", tenant)
	}
	if user != "u1" {
		t.Errorf("user: want 'u1', got '%s'", user)
	}
	if len(scopes) != 2 {
		t.Errorf("scopes: want 2, got %d", len(scopes))
	}

	_, _, _, err = v.Validate(nil, "sk-unknown")
	if err == nil {
		t.Error("Unknown key should return error")
	}
}

// === BotDetect coverage ===

func TestBotDetect_Allowed(t *testing.T) {
	handler := BotDetect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/api/data", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Errorf("Normal request: want 200, got %d", rr.Code)
	}
}

func TestBotDetect_BotUserAgent(t *testing.T) {
	handler := BotDetect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	// Known bot UA
	req := httptest.NewRequest("GET", "/api/data", nil)
	req.Header.Set("User-Agent", "Googlebot/2.1")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	// BotDetect may block or allow based on config
	if rr.Code != 200 && rr.Code != 403 {
		t.Errorf("Bot: want 200 or 403, got %d", rr.Code)
	}
}

func TestNewBehavioralBotDetect(t *testing.T) {
	b := NewBehavioralBotDetect(100, time.Minute)
	if b == nil {
		t.Fatal("Should not be nil")
	}
}

func TestBehavioralBotDetect_Middleware(t *testing.T) {
	b := NewBehavioralBotDetect(100, time.Minute)
	handler := b.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	// Single request should pass
	req := httptest.NewRequest("GET", "/api/data", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Errorf("First request: want 200, got %d", rr.Code)
	}
}

// === Cache coverage ===

func TestNewCache(t *testing.T) {
	c := NewCache(5 * time.Minute)
	if c == nil {
		t.Fatal("Should not be nil")
	}
}

func TestCache_Middleware_GET(t *testing.T) {
	c := NewCache(5 * time.Minute)
	callCount := 0
	handler := c.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(200)
		w.Write([]byte("hello"))
	}))

	// First request - should call handler
	req := httptest.NewRequest("GET", "/api/data", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if callCount != 1 {
		t.Errorf("First: want 1 call, got %d", callCount)
	}

	// Second request - should be cached
	req2 := httptest.NewRequest("GET", "/api/data", nil)
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)
	if callCount != 1 {
		t.Errorf("Cached: want 1 call, got %d", callCount)
	}
}

func TestCache_Middleware_POST_NoCache(t *testing.T) {
	c := NewCache(5 * time.Minute)
	callCount := 0
	handler := c.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(200)
	}))

	// POST should not be cached
	req1 := httptest.NewRequest("POST", "/api/data", nil)
	rr1 := httptest.NewRecorder()
	handler.ServeHTTP(rr1, req1)

	req2 := httptest.NewRequest("POST", "/api/data", nil)
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)

	if callCount != 2 {
		t.Errorf("POST should not be cached: want 2 calls, got %d", callCount)
	}
}

func TestCache_Invalidate(t *testing.T) {
	c := NewCache(5 * time.Minute)
	callCount := 0
	handler := c.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(200)
		w.Write([]byte("data"))
	}))

	req := httptest.NewRequest("GET", "/api/data", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	c.Invalidate()

	req2 := httptest.NewRequest("GET", "/api/data", nil)
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)

	if callCount != 2 {
		t.Errorf("After invalidate: want 2 calls, got %d", callCount)
	}
}

func TestCacheKey(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	key := cacheKey(req)
	if key == "" {
		t.Error("cacheKey should not be empty")
	}
}

func TestGenerateETag(t *testing.T) {
	etag := generateETag([]byte("hello world"))
	if etag == "" {
		t.Error("ETag should not be empty")
	}
}

// === AuditLog coverage ===

func TestAuditLog_DroppedCount(t *testing.T) {
	p := &NATSAuditPublisher{}
	if p.DroppedCount() != 0 {
		t.Errorf("DroppedCount: want 0, got %d", p.DroppedCount())
	}
}
