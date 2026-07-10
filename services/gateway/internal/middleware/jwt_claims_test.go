package middleware

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func makeJWT(payload map[string]any) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	body, _ := json.Marshal(payload)
	payloadB64 := base64.RawURLEncoding.EncodeToString(body)
	sig := base64.RawURLEncoding.EncodeToString([]byte("fake-sig"))
	return header + "." + payloadB64 + "." + sig
}

func TestExtractJWTClaims_FullToken(t *testing.T) {
	token := makeJWT(map[string]any{
		"sub":       "user-123",
		"tenant_id": "tenant-abc",
		"scope":     "read write",
		"email":     "test@example.com",
		"iss":       "ggid",
	})
	req := httptest.NewRequest("GET", "/api/data", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	claims := ExtractJWTClaims(req)
	if claims.Subject != "user-123" {
		t.Errorf("sub: want 'user-123', got '%s'", claims.Subject)
	}
	if claims.TenantID != "tenant-abc" {
		t.Errorf("tenant_id: want 'tenant-abc', got '%s'", claims.TenantID)
	}
	if len(claims.Scopes) != 2 {
		t.Errorf("scopes: want 2, got %d", len(claims.Scopes))
	}
	if claims.Email != "test@example.com" {
		t.Errorf("email: got '%s'", claims.Email)
	}
	if claims.Issuer != "ggid" {
		t.Errorf("iss: got '%s'", claims.Issuer)
	}
}

func TestExtractJWTClaims_NoToken(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/data", nil)
	claims := ExtractJWTClaims(req)
	if claims.Subject != "" {
		t.Error("Should be empty without token")
	}
}

func TestExtractJWTClaims_Malformed(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/data", nil)
	req.Header.Set("Authorization", "Bearer not-a-jwt")
	claims := ExtractJWTClaims(req)
	if claims.Subject != "" {
		t.Error("Malformed token should return empty claims")
	}
}

func TestExtractJWTClaims_ScopesArray(t *testing.T) {
	token := makeJWT(map[string]any{
		"sub":    "user-1",
		"scopes": []any{"read", "write", "admin"},
	})
	req := httptest.NewRequest("GET", "/api/data", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	claims := ExtractJWTClaims(req)
	if len(claims.Scopes) != 3 {
		t.Errorf("scopes array: want 3, got %d", len(claims.Scopes))
	}
}

func TestExtractJWTClaims_NoAuthHeader(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/data", nil)
	claims := ExtractJWTClaims(req)
	if claims.Subject != "" || claims.TenantID != "" {
		t.Error("Empty claims expected")
	}
}

func TestJWTClaimExtraction_SetsHeaders(t *testing.T) {
	token := makeJWT(map[string]any{
		"sub":       "user-1",
		"tenant_id": "t-1",
		"scope":     "read",
	})

	var capturedHeaders http.Header
	handler := JWTClaimExtraction(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = r.Header
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/api/data", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if capturedHeaders.Get("X-User-ID") != "user-1" {
		t.Errorf("X-User-ID: got '%s'", capturedHeaders.Get("X-User-ID"))
	}
	if capturedHeaders.Get("X-Tenant-ID") != "t-1" {
		t.Errorf("X-Tenant-ID: got '%s'", capturedHeaders.Get("X-Tenant-ID"))
	}
	if capturedHeaders.Get("X-Scopes") != "read" {
		t.Errorf("X-Scopes: got '%s'", capturedHeaders.Get("X-Scopes"))
	}
}

func TestJWTClaimExtraction_NoToken(t *testing.T) {
	called := false
	handler := JWTClaimExtraction(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/api/data", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if !called {
		t.Error("Should still call next handler")
	}
}

func TestClaimsFromContext(t *testing.T) {
	token := makeJWT(map[string]any{"sub": "ctx-test"})

	var ctxClaims JWTCClaims
	handler := JWTClaimExtraction(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxClaims = ClaimsFromContext(r.Context())
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/api/data", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if ctxClaims.Subject != "ctx-test" {
		t.Errorf("Context claims sub: got '%s'", ctxClaims.Subject)
	}
}

func TestClaimsFromContext_Empty(t *testing.T) {
	claims := ClaimsFromContext(nil)
	if claims.Subject != "" {
		t.Error("Should be empty")
	}
}

