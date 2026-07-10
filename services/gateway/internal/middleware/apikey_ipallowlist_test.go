package middleware

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

// --- API Key Tests ---

func TestAPIKeyAuth_NoKey_PassesThrough(t *testing.T) {
	validator := NewMemoryAPIKeyValidator()
	called := false
	handler := APIKeyAuth(validator)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	handler.ServeHTTP(httptest.NewRecorder(), req)
	if !called {
		t.Error("handler should be called when no API key")
	}
}

func TestAPIKeyAuth_ValidKey_InjectsIdentity(t *testing.T) {
	validator := NewMemoryAPIKeyValidator()
	validator.AddKey("ggid_test123", "tenant-1", "user-1", []string{"read", "write"})

	var gotTenant, gotUser string
	handler := APIKeyAuth(validator)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotTenant, _ = r.Context().Value(TenantIDKey).(string)
		gotUser, _ = r.Context().Value(UserIDKey).(string)
	}))

	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	req.Header.Set("X-API-Key", "ggid_test123")
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if gotTenant != "tenant-1" {
		t.Errorf("expected tenant-1, got %s", gotTenant)
	}
	if gotUser != "user-1" {
		t.Errorf("expected user-1, got %s", gotUser)
	}
}

func TestAPIKeyAuth_InvalidKey_401(t *testing.T) {
	validator := NewMemoryAPIKeyValidator()
	called := false
	handler := APIKeyAuth(validator)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	req.Header.Set("X-API-Key", "invalid-key")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if called {
		t.Error("handler should NOT be called with invalid key")
	}
	if w.Code != 401 {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAPIKeyAuth_QueryParamKey(t *testing.T) {
	validator := NewMemoryAPIKeyValidator()
	validator.AddKey("ggid_query", "t1", "u1", nil)

	called := false
	handler := APIKeyAuth(validator)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest("GET", "/api/v1/users?api_key=ggid_query", nil)
	handler.ServeHTTP(httptest.NewRecorder(), req)
	if !called {
		t.Error("handler should be called with valid query param key")
	}
}

func TestHasScope(t *testing.T) {
	ctx := context.WithValue(context.Background(), APIKeyScopesKey, []string{"read", "write"})
	if !HasScope(ctx, "read") {
		t.Error("expected read scope")
	}
	if HasScope(ctx, "admin") {
		t.Error("should not have admin scope")
	}

	// No scopes in context = unrestricted
	if !HasScope(context.Background(), "anything") {
		t.Error("expected unrestricted without scopes")
	}
}

// --- IP Allowlist Tests ---

func TestIPAllowlist_NoConfig_AllowsAll(t *testing.T) {
	al := NewIPAllowlist(nil)
	called := false
	handler := al.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "1.2.3.4:1234"
	handler.ServeHTTP(httptest.NewRecorder(), req)
	if !called {
		t.Error("should allow when no CIDRs configured")
	}
}

func TestIPAllowlist_IPInList_Allows(t *testing.T) {
	cidrs := ParseCIDRs("10.0.0.0/8")
	al := NewIPAllowlist(map[string][]*net.IPNet{"tenant-1": cidrs})
	called := false
	handler := al.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "10.1.2.3:1234"
	req.Header.Set("X-Tenant-ID", "tenant-1")
	// Need to set tenant in context
	ctx := context.WithValue(req.Context(), TenantIDKey, "tenant-1")
	req = req.WithContext(ctx)
	handler.ServeHTTP(httptest.NewRecorder(), req)
	if !called {
		t.Error("should allow IP in CIDR range")
	}
}

func TestIPAllowlist_IPNotInList_Denies(t *testing.T) {
	cidrs := ParseCIDRs("10.0.0.0/8")
	al := NewIPAllowlist(map[string][]*net.IPNet{"tenant-1": cidrs})
	called := false
	handler := al.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:1234"
	ctx := context.WithValue(req.Context(), TenantIDKey, "tenant-1")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if called {
		t.Error("should deny IP outside CIDR")
	}
	if w.Code != 403 {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestIPAllowlist_XForwardedFor(t *testing.T) {
	cidrs := ParseCIDRs("10.0.0.0/8")
	al := NewIPAllowlist(map[string][]*net.IPNet{"t1": cidrs})

	allowed := false
	handler := al.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allowed = true
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", "10.5.5.5")
	ctx := context.WithValue(req.Context(), TenantIDKey, "t1")
	req = req.WithContext(ctx)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if !allowed {
		t.Error("should allow 10.x via X-Forwarded-For")
	}
}

func TestParseCIDRs(t *testing.T) {
	cidrs := ParseCIDRs("192.168.1.0/24", "10.0.0.0/8")
	if len(cidrs) != 2 {
		t.Fatalf("expected 2 CIDRs, got %d", len(cidrs))
	}
	if cidrs[0].String() != "192.168.1.0/24" {
		t.Errorf("unexpected CIDR: %s", cidrs[0].String())
	}
}

func TestParseCIDRs_SingleIP(t *testing.T) {
	cidrs := ParseCIDRs("1.2.3.4")
	if len(cidrs) != 1 {
		t.Fatalf("expected 1 CIDR, got %d", len(cidrs))
	}
	if cidrs[0].String() != "1.2.3.4/32" {
		t.Errorf("expected /32 for single IP, got %s", cidrs[0].String())
	}
}

func TestParseCIDRs_Invalid(t *testing.T) {
	cidrs := ParseCIDRs("not-an-ip")
	if len(cidrs) != 0 {
		t.Errorf("expected 0 CIDRs for invalid input, got %d", len(cidrs))
	}
}
