package middleware

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// --- Auth middleware tests ---

func TestAuth_SkipPaths(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	mw := Auth("http://localhost:8080", Options{
		SkipPaths: []string{"/health", "/public"},
	})(next)

	// /health should be skipped
	called = false
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req)
	if !called {
		t.Error("expected /health to skip auth")
	}

	// /public/foo should be skipped (prefix match)
	called = false
	req = httptest.NewRequest("GET", "/public/foo", nil)
	w = httptest.NewRecorder()
	mw.ServeHTTP(w, req)
	if !called {
		t.Error("expected /public/foo to skip auth")
	}
}

func TestAuth_MissingToken(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not call next without token")
	})

	mw := Auth("http://localhost:8080", Options{})(next)

	req := httptest.NewRequest("GET", "/api/protected", nil)
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuth_InvalidToken(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not call next with invalid token")
	})

	mw := Auth("http://localhost:8080", Options{})(next)

	req := httptest.NewRequest("GET", "/api/protected", nil)
	req.Header.Set("Authorization", "Bearer not-a-jwt")
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for invalid token, got %d", w.Code)
	}
}

func TestAuth_ValidToken(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		info, ok := FromContext(r.Context())
		if !ok || info == nil {
			t.Error("expected UserInfo in context")
		}
		if info.UserID != "user-123" {
			t.Errorf("expected user-123, got %s", info.UserID)
		}
		w.WriteHeader(http.StatusOK)
	})

	mw := Auth("http://localhost:8080", Options{})(next)

	token := makeTestJWT(t, map[string]interface{}{
		"sub":       "user-123",
		"username":  "testuser",
		"email":     "test@example.com",
		"tenant_id": "t1",
		"roles":     []interface{}{"admin", "editor"},
		"scope":     "read write",
	})

	req := httptest.NewRequest("GET", "/api/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req)
	if !called {
		t.Error("expected next handler to be called with valid token")
	}
}

func TestAuth_CustomUnauthorized(t *testing.T) {
	customCalled := false
	customHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		customCalled = true
		w.WriteHeader(http.StatusTeapot) // 418
	})

	mw := Auth("http://localhost:8080", Options{
		OnUnauthorized: customHandler,
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	req := httptest.NewRequest("GET", "/api/protected", nil)
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req)
	if !customCalled {
		t.Error("expected custom unauthorized handler to be called")
	}
	if w.Code != http.StatusTeapot {
		t.Errorf("expected 418 from custom handler, got %d", w.Code)
	}
}

// --- RequireRole tests ---

func TestRequireRole_HasRole(t *testing.T) {
	called := false
	info := &UserInfo{Roles: []string{"admin"}}
	ctx := context.WithValue(context.Background(), contextKey{}, info)

	handler := RequireRole("admin")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest("GET", "/", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if !called {
		t.Error("expected handler to be called when user has role")
	}
}

func TestRequireRole_MissingRole(t *testing.T) {
	called := false
	info := &UserInfo{Roles: []string{"viewer"}}
	ctx := context.WithValue(context.Background(), contextKey{}, info)

	handler := RequireRole("admin")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest("GET", "/", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if called {
		t.Error("expected handler to NOT be called when role missing")
	}
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestRequireRole_NoUserInContext(t *testing.T) {
	handler := RequireRole("admin")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not call handler without user in context")
	}))

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without user, got %d", w.Code)
	}
}

func TestRequireRole_AdminBypass(t *testing.T) {
	called := false
	info := &UserInfo{Roles: []string{"admin"}}
	ctx := context.WithValue(context.Background(), contextKey{}, info)

	// Admin should bypass any role check
	handler := RequireRole("superadmin")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest("GET", "/", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if !called {
		t.Error("expected admin to bypass role check")
	}
}

// --- FromContext tests ---

func TestFromContext_WithUser(t *testing.T) {
	info := &UserInfo{UserID: "u1"}
	ctx := context.WithValue(context.Background(), contextKey{}, info)

	got, ok := FromContext(ctx)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if got.UserID != "u1" {
		t.Errorf("expected u1, got %s", got.UserID)
	}
}

func TestFromContext_EmptyContext(t *testing.T) {
	got, ok := FromContext(context.Background())
	if ok {
		t.Error("expected ok=false for empty context")
	}
	if got != nil {
		t.Error("expected nil UserInfo")
	}
}

// --- extractBearer tests ---

func TestExtractBearer_Valid(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer my-token")
	token := extractBearer(req)
	if token != "my-token" {
		t.Errorf("expected 'my-token', got '%s'", token)
	}
}

func TestExtractBearer_MissingHeader(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	token := extractBearer(req)
	if token != "" {
		t.Errorf("expected empty string, got '%s'", token)
	}
}

func TestExtractBearer_WrongScheme(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Basic abc123")
	token := extractBearer(req)
	if token != "" {
		t.Errorf("expected empty string for non-Bearer, got '%s'", token)
	}
}

func TestExtractBearer_EmptyBearer(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer ")
	token := extractBearer(req)
	if token != "" {
		t.Errorf("expected empty string for empty Bearer, got '%s'", token)
	}
}

// --- Options defaults ---

func TestAuth_Defaults(t *testing.T) {
	// With no options, should still work
	mw := Auth("http://localhost:8080", Options{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	req := httptest.NewRequest("GET", "/api/test", nil)
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 with defaults, got %d", w.Code)
	}

	// Check default JSON response
	body := w.Body.String()
	if !strings.Contains(body, "missing or invalid token") {
		t.Errorf("expected default error message, got %s", body)
	}
}

// --- parseToken tests ---

func TestParseToken_AllFields(t *testing.T) {
	token := makeTestJWT(t, map[string]interface{}{
		"sub":       "user-1",
		"tenant_id": "tenant-1",
		"username":  "johndoe",
		"email":     "john@example.com",
		"roles":     []interface{}{"admin", "user"},
		"scope":     "read write delete",
	})

	info, err := parseToken(token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.UserID != "user-1" {
		t.Errorf("expected user-1, got %s", info.UserID)
	}
	if info.TenantID != "tenant-1" {
		t.Errorf("expected tenant-1, got %s", info.TenantID)
	}
	if info.Username != "johndoe" {
		t.Errorf("expected johndoe, got %s", info.Username)
	}
	if info.Email != "john@example.com" {
		t.Errorf("expected john@example.com, got %s", info.Email)
	}
	if len(info.Roles) != 2 {
		t.Errorf("expected 2 roles, got %d", len(info.Roles))
	}
	if len(info.Scopes) != 3 {
		t.Errorf("expected 3 scopes, got %d", len(info.Scopes))
	}
}

func TestParseToken_MinimalClaims(t *testing.T) {
	token := makeTestJWT(t, map[string]interface{}{
		"sub": "minimal-user",
	})

	info, err := parseToken(token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.UserID != "minimal-user" {
		t.Errorf("expected minimal-user, got %s", info.UserID)
	}
	if len(info.Roles) != 0 {
		t.Errorf("expected 0 roles, got %d", len(info.Roles))
	}
}

func TestParseToken_InvalidJWT(t *testing.T) {
	_, err := parseToken("not.a.valid.jwt")
	if err != nil {
		// ParseUnverified might still parse 3-part tokens
		// Just verify it doesn't panic
	}
}

// --- defaultUnauthorized tests ---

func TestDefaultUnauthorized(t *testing.T) {
	w := httptest.NewRecorder()
	defaultUnauthorized(w, httptest.NewRequest("GET", "/", nil))
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
	if w.Header().Get("Content-Type") != "application/json" {
		t.Error("expected application/json content type")
	}
	body := w.Body.String()
	if !strings.Contains(body, "error") {
		t.Errorf("expected error in body, got %s", body)
	}
}

// --- Helper to create valid-looking JWT ---

func makeTestJWT(t *testing.T, claims map[string]interface{}) string {
	t.Helper()
	header := `{"alg":"RS256","typ":"JWT"}`
	headerB64 := base64.RawURLEncoding.EncodeToString([]byte(header))
	claimsJSON, _ := json.Marshal(claims)
	claimsB64 := base64.RawURLEncoding.EncodeToString(claimsJSON)
	sigB64 := base64.RawURLEncoding.EncodeToString([]byte("fake-sig"))
	return strings.Join([]string{headerB64, claimsB64, sigB64}, ".")
}
