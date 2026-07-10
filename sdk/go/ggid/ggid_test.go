package ggid

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// helper to create a JWT
func makeTestJWT(claims map[string]interface{}) string {
	header := `{"alg":"RS256","typ":"JWT"}`
	headerB64 := base64.RawURLEncoding.EncodeToString([]byte(header))
	claimsJSON, _ := json.Marshal(claims)
	claimsB64 := base64.RawURLEncoding.EncodeToString(claimsJSON)
	sigB64 := base64.RawURLEncoding.EncodeToString([]byte("fake-sig"))
	return strings.Join([]string{headerB64, claimsB64, sigB64}, ".")
}

// --- JWTVerifier tests ---

func TestParseJWTClaims_Valid(t *testing.T) {
	token := makeTestJWT(map[string]interface{}{
		"sub": "user-1",
		"exp": float64(time.Now().Add(1 * time.Hour).Unix()),
	})

	claims, err := parseJWTClaims(token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if claims["sub"] != "user-1" {
		t.Errorf("expected sub=user-1, got %v", claims["sub"])
	}
}

func TestParseJWTClaims_InvalidFormat(t *testing.T) {
	_, err := parseJWTClaims("not-a-jwt")
	if err == nil {
		t.Fatal("expected error for non-JWT string")
	}
}

func TestParseJWTClaims_TwoParts(t *testing.T) {
	_, err := parseJWTClaims("a.b")
	if err == nil {
		t.Fatal("expected error for 2-part JWT")
	}
}

func TestParseJWTClaims_InvalidBase64(t *testing.T) {
	_, err := parseJWTClaims("header.!!!.signature")
	if err == nil {
		t.Fatal("expected error for invalid base64")
	}
}

func TestJWTVerifier_Verify_Expired(t *testing.T) {
	v := NewJWTVerifier("http://localhost:8080/.well-known/jwks.json")
	token := makeTestJWT(map[string]interface{}{
		"sub": "u",
		"exp": float64(time.Now().Add(-1 * time.Hour).Unix()),
	})

	_, err := v.Verify(context.Background(), token)
	if err != ErrTokenExpired {
		t.Errorf("expected ErrTokenExpired, got %v", err)
	}
}

func TestJWTVerifier_Verify_Valid(t *testing.T) {
	v := NewJWTVerifier("http://localhost:8080/.well-known/jwks.json")
	token := makeTestJWT(map[string]interface{}{
		"sub":      "user-1",
		"username": "testuser",
		"exp":      float64(time.Now().Add(1 * time.Hour).Unix()),
	})

	claims, err := v.Verify(context.Background(), token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if claims["sub"] != "user-1" {
		t.Errorf("expected sub=user-1, got %v", claims["sub"])
	}
}

func TestJWTVerifier_Verify_NoExp(t *testing.T) {
	v := NewJWTVerifier("http://localhost:8080/.well-known/jwks.json")
	token := makeTestJWT(map[string]interface{}{
		"sub": "no-exp-user",
	})

	claims, err := v.Verify(context.Background(), token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if claims["sub"] != "no-exp-user" {
		t.Errorf("expected sub=no-exp-user, got %v", claims["sub"])
	}
}

func TestJWTVerifier_Verify_InvalidToken(t *testing.T) {
	v := NewJWTVerifier("http://localhost:8080/.well-known/jwks.json")
	_, err := v.Verify(context.Background(), "invalid-token")
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}

func TestNewJWTVerifier_Defaults(t *testing.T) {
	v := NewJWTVerifier("http://example.com/jwks")
	if v.jwksURL != "http://example.com/jwks" {
		t.Errorf("unexpected URL: %s", v.jwksURL)
	}
	if v.ttl != 5*time.Minute {
		t.Errorf("expected 5m TTL, got %v", v.ttl)
	}
}

// --- Middleware tests ---

func TestMiddleware_PublicPaths(t *testing.T) {
	c := &Client{verifier: NewJWTVerifier("http://localhost:8080/.well-known/jwks.json")}
	called := false
	mw := c.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	for _, path := range []string{"/", "/healthz", "/docs", "/login", "/register"} {
		called = false
		req := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()
		mw.ServeHTTP(w, req)
		if !called {
			t.Errorf("expected handler called for public path %s", path)
		}
	}
}

func TestMiddleware_AuthEndpointSkipped(t *testing.T) {
	c := &Client{verifier: NewJWTVerifier("http://localhost:8080/.well-known/jwks.json")}
	called := false
	mw := c.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/api/v1/auth/login", nil)
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req)
	if !called {
		t.Error("expected handler called for /api/v1/auth/ path")
	}
}

func TestMiddleware_OAuthEndpointSkipped(t *testing.T) {
	c := &Client{verifier: NewJWTVerifier("http://localhost:8080/.well-known/jwks.json")}
	called := false
	mw := c.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/oauth/authorize", nil)
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req)
	if !called {
		t.Error("expected handler called for /oauth/ path")
	}
}

func TestMiddleware_MissingBearer(t *testing.T) {
	c := &Client{verifier: NewJWTVerifier("http://localhost:8080/.well-known/jwks.json")}
	called := false
	mw := c.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest("GET", "/api/protected", nil)
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req)
	if called {
		t.Error("expected handler NOT called without Bearer")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestMiddleware_InvalidBearerScheme(t *testing.T) {
	c := &Client{verifier: NewJWTVerifier("http://localhost:8080/.well-known/jwks.json")}
	mw := c.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not reach handler")
	}))

	req := httptest.NewRequest("GET", "/api/protected", nil)
	req.Header.Set("Authorization", "Basic abc123")
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestMiddleware_ValidToken(t *testing.T) {
	c := &Client{verifier: NewJWTVerifier("http://localhost:8080/.well-known/jwks.json")}
	called := false
	mw := c.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		claims := ClaimsFromContext(r.Context())
		if claims == nil {
			t.Error("expected claims in context")
		}
		w.WriteHeader(http.StatusOK)
	}))

	token := makeTestJWT(map[string]interface{}{
		"sub": "user-1",
		"exp": float64(time.Now().Add(1 * time.Hour).Unix()),
	})

	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req)
	if !called {
		t.Error("expected handler called with valid token")
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// --- ClaimsFromContext tests ---

func TestClaimsFromContext_WithClaims(t *testing.T) {
	claims := map[string]interface{}{"sub": "user-1"}
	ctx := context.WithValue(context.Background(), claimsKey{}, claims)
	got := ClaimsFromContext(ctx)
	if got == nil || got["sub"] != "user-1" {
		t.Error("expected to get claims")
	}
}

func TestClaimsFromContext_Empty(t *testing.T) {
	got := ClaimsFromContext(context.Background())
	if got != nil {
		t.Error("expected nil for empty context")
	}
}

func TestClaimsFromContext_WrongType(t *testing.T) {
	ctx := context.WithValue(context.Background(), claimsKey{}, "not-a-map")
	got := ClaimsFromContext(ctx)
	if got != nil {
		t.Error("expected nil for wrong type")
	}
}

// --- RequirePermission ---

func TestRequirePermission_NoClaims(t *testing.T) {
	c := &Client{verifier: NewJWTVerifier("http://localhost:8080/.well-known/jwks.json")}
	mw := c.RequirePermission("docs", "read")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not call handler")
	}))

	req := httptest.NewRequest("GET", "/api/docs", nil)
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}
