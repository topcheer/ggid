package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestIsListEndpoint(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"/api/v1/users", true},
		{"/api/v1/roles", true},
		{"/api/v1/oauth/clients", true},
		{"/api/v1/policies", true},
		{"/api/v1/sessions", true},
		{"/api/v1/audit/events", true},
		// Not list endpoints
		{"/api/v1/users/550e8400-e29b-41d4-a716-446655440000", false},
		{"/api/v1/auth/login", false}, // login doesn't end with plural
		{"/api/v1/metrics", false},   // excluded
		{"/api/v1/stats", false},     // excluded
		{"/healthz", false},          // not /api/v1/
		{"/api/v1/user", false},      // singular
	}
	for _, tt := range tests {
		got := isListEndpoint(tt.path)
		if got != tt.want {
			t.Errorf("isListEndpoint(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestSplitURLPath(t *testing.T) {
	tests := []struct {
		path string
		want []string
	}{
		{"/api/v1/users", []string{"api", "v1", "users"}},
		{"/api/v1/oauth/clients", []string{"api", "v1", "oauth", "clients"}},
		{"/", []string{}},
		{"", []string{}},
	}
	for _, tt := range tests {
		got := splitURLPath(tt.path)
		if len(got) != len(tt.want) {
			t.Errorf("splitPath(%q) = %v, want %v", tt.path, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("splitPath(%q)[%d] = %q, want %q", tt.path, i, got[i], tt.want[i])
			}
		}
	}
}

func TestFnv1aShort(t *testing.T) {
	// Same input → same hash
	h1 := fnv1aShort("Bearer abc123")
	h2 := fnv1aShort("Bearer abc123")
	if h1 != h2 {
		t.Errorf("fnv1aShort not deterministic: %s vs %s", h1, h2)
	}
	// Different input → different hash
	h3 := fnv1aShort("Bearer xyz789")
	if h1 == h3 {
		t.Error("fnv1aShort collision for different inputs")
	}
	// Empty → "anon"
	if fnv1aShort("") != "anon" {
		t.Errorf("fnv1aShort('') = %q, want 'anon'", fnv1aShort(""))
	}
	// Length is 8 hex chars
	if len(h1) != 8 {
		t.Errorf("fnv1aShort length = %d, want 8", len(h1))
	}
}

func TestListCacheMiddleware_NonGetBypasses(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(200)
	})
	handler := ListCacheMiddleware(nil, DefaultListCacheConfig())(next)

	// nil rdb = pass-through wrapper, but should still call next
	req := httptest.NewRequest("POST", "/api/v1/users", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if !called {
		t.Error("POST should bypass cache and call handler")
	}
}

func TestListCacheMiddleware_NonListEndpointBypasses(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})
	handler := ListCacheMiddleware(nil, DefaultListCacheConfig())(next)

	// Non-list endpoint with GET
	req := httptest.NewRequest("GET", "/api/v1/auth/login", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	// nil rdb means handler = next (pass-through), should still work
	if rr.Code != 200 {
		t.Errorf("non-list GET: want 200, got %d", rr.Code)
	}
}

func TestListCacheConfig_Defaults(t *testing.T) {
	cfg := DefaultListCacheConfig()
	if cfg.TTL != 30*time.Second {
		t.Errorf("TTL: want 30s, got %v", cfg.TTL)
	}
	if cfg.MaxBodySize != 256*1024 {
		t.Errorf("MaxBodySize: want 256KB, got %d", cfg.MaxBodySize)
	}
}

func TestListCacheKey_TenantIsolation(t *testing.T) {
	req1 := httptest.NewRequest("GET", "/api/v1/users", nil)
	req1.Header.Set("X-Tenant-ID", "tenant-A")
	req1.Header.Set("Authorization", "Bearer token-A")

	req2 := httptest.NewRequest("GET", "/api/v1/users", nil)
	req2.Header.Set("X-Tenant-ID", "tenant-B")
	req2.Header.Set("Authorization", "Bearer token-B")

	k1 := listCacheKey(req1)
	k2 := listCacheKey(req2)

	if k1 == k2 {
		t.Error("different tenants should have different cache keys")
	}
}
