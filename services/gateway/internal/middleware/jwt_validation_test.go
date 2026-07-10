package middleware

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// setupTestJWKS generates an RSA key pair and returns the private key and
// a JWKSClient configured with the corresponding public key.
func setupTestJWKS(t *testing.T) (*rsa.PrivateKey, *JWKSClient) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}

	pub := &priv.PublicKey
	kid := "test-key-id"

	jwks := &JWKSClient{
		publicKey:  pub,
		keyID:      kid,
		keys:       map[string]*rsa.PublicKey{kid: pub},
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}

	return priv, jwks
}

// signTestToken creates a signed JWT with the given claims.
func signTestToken(t *testing.T, priv *rsa.PrivateKey, claims jwt.MapClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = "test-key-id"
	str, err := token.SignedString(priv)
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}
	return str
}

func TestJWT_ValidToken(t *testing.T) {
	priv, jwks := setupTestJWKS(t)

	tokenStr := signTestToken(t, priv, jwt.MapClaims{
		"sub":        "user123",
		"tenant_id":  "tenant-abc",
		"iss":        "test-issuer",
		"aud":        "test-audience",
		"exp":        time.Now().Add(time.Hour).Unix(),
	})

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	w := httptest.NewRecorder()

	called := false
	handler := JWTAuth(jwks, true, "test-issuer", "test-audience")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(200)
	}))
	handler.ServeHTTP(w, req)

	if !called {
		t.Error("handler should be called with valid token")
	}
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestJWT_ExpiredToken(t *testing.T) {
	priv, jwks := setupTestJWKS(t)

	tokenStr := signTestToken(t, priv, jwt.MapClaims{
		"sub": "user123",
		"iss": "test-issuer",
		"aud": "test-audience",
		"exp": time.Now().Add(-time.Hour).Unix(), // expired 1 hour ago
	})

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	w := httptest.NewRecorder()

	called := false
	handler := JWTAuth(jwks, true, "test-issuer", "test-audience")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	handler.ServeHTTP(w, req)

	if called {
		t.Error("handler should NOT be called with expired token")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestJWT_NotBeforeInFuture(t *testing.T) {
	priv, jwks := setupTestJWKS(t)

	tokenStr := signTestToken(t, priv, jwt.MapClaims{
		"sub": "user123",
		"iss": "test-issuer",
		"aud": "test-audience",
		"nbf": time.Now().Add(time.Hour).Unix(), // not valid yet
		"exp": time.Now().Add(2 * time.Hour).Unix(),
	})

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	w := httptest.NewRecorder()

	called := false
	handler := JWTAuth(jwks, true, "test-issuer", "test-audience")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	handler.ServeHTTP(w, req)

	if called {
		t.Error("handler should NOT be called when nbf is in future")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestJWT_WrongIssuer(t *testing.T) {
	priv, jwks := setupTestJWKS(t)

	tokenStr := signTestToken(t, priv, jwt.MapClaims{
		"sub": "user123",
		"iss": "wrong-issuer", // wrong issuer
		"aud": "test-audience",
		"exp": time.Now().Add(time.Hour).Unix(),
	})

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	w := httptest.NewRecorder()

	called := false
	handler := JWTAuth(jwks, true, "test-issuer", "test-audience")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	handler.ServeHTTP(w, req)

	if called {
		t.Error("handler should NOT be called with wrong issuer")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestJWT_AudienceMismatch(t *testing.T) {
	priv, jwks := setupTestJWKS(t)

	tokenStr := signTestToken(t, priv, jwt.MapClaims{
		"sub": "user123",
		"iss": "test-issuer",
		"aud": "wrong-audience", // wrong audience
		"exp": time.Now().Add(time.Hour).Unix(),
	})

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	w := httptest.NewRecorder()

	called := false
	handler := JWTAuth(jwks, true, "test-issuer", "test-audience")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	handler.ServeHTTP(w, req)

	if called {
		t.Error("handler should NOT be called with wrong audience")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestJWT_MalformedToken(t *testing.T) {
	_, jwks := setupTestJWKS(t)

	tests := []struct {
		name string
		token string
	}{
		{"empty", ""},
		{"garbage", "not.a.jwt"},
		{"single_part", "abc"},
		{"two_parts", "abc.def"},
		{"not_base64", "!!!.@@@.###"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/test", nil)
			req.Header.Set("Authorization", "Bearer "+tt.token)
			w := httptest.NewRecorder()

			called := false
			handler := JWTAuth(jwks, true, "", "")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
			}))
			handler.ServeHTTP(w, req)

			if called {
				t.Error("handler should NOT be called with malformed token")
			}
			if w.Code != http.StatusUnauthorized {
				t.Errorf("expected 401, got %d", w.Code)
			}
		})
	}
}

func TestJWT_MissingAuthorizationHeader(t *testing.T) {
	_, jwks := setupTestJWKS(t)

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	w := httptest.NewRecorder()

	called := false
	handler := JWTAuth(jwks, true, "", "")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	handler.ServeHTTP(w, req)

	if called {
		t.Error("handler should NOT be called without Authorization header")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
	if w.Header().Get("WWW-Authenticate") == "" {
		t.Error("expected WWW-Authenticate header")
	}
}

func TestJWT_InvalidAuthHeaderFormat(t *testing.T) {
	_, jwks := setupTestJWKS(t)

	tests := []struct {
		name   string
		header string
	}{
		{"basic_auth", "Basic dXNlcjpwYXNz"},
		{"no_space", "justtoken"},
		{"wrong_scheme", "Token abc.def.ghi"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/test", nil)
			req.Header.Set("Authorization", tt.header)
			w := httptest.NewRecorder()

			called := false
			handler := JWTAuth(jwks, true, "", "")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
			}))
			handler.ServeHTTP(w, req)

			if called {
				t.Error("handler should NOT be called")
			}
			if w.Code != http.StatusUnauthorized {
				t.Errorf("expected 401, got %d", w.Code)
			}
		})
	}
}

func TestJWT_OptionalNoToken(t *testing.T) {
	_, jwks := setupTestJWKS(t)

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	w := httptest.NewRecorder()

	called := false
	handler := JWTAuth(jwks, false, "", "")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(200)
	}))
	handler.ServeHTTP(w, req)

	if !called {
		t.Error("handler should be called when token is optional and absent")
	}
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestJWT_OptionalInvalidToken(t *testing.T) {
	_, jwks := setupTestJWKS(t)

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	req.Header.Set("Authorization", "Bearer malformed.token.here")
	w := httptest.NewRecorder()

	called := false
	handler := JWTAuth(jwks, false, "", "")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(200)
	}))
	handler.ServeHTTP(w, req)

	if !called {
		t.Error("handler should be called when token is optional and invalid")
	}
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestJWT_WrongSigningMethod(t *testing.T) {
	// Generate HMAC key (should be rejected since we require RS256)
	_, jwks := setupTestJWKS(t)

	// Create a token with HS256 instead of RS256
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "user123",
		"iss": "test-issuer",
		"aud": "test-audience",
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	tokenStr, err := token.SignedString([]byte("secret-key"))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	w := httptest.NewRecorder()

	called := false
	handler := JWTAuth(jwks, true, "test-issuer", "test-audience")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	handler.ServeHTTP(w, req)

	if called {
		t.Error("handler should NOT be called with HS256 token (wrong signing method)")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestJWT_ClaimsInjection(t *testing.T) {
	priv, jwks := setupTestJWKS(t)

	tokenStr := signTestToken(t, priv, jwt.MapClaims{
		"sub":       "550e8400-e29b-41d4-a716-446655440000",
		"tenant_id": "tenant-123",
		"iss":       "test-issuer",
		"aud":       "test-audience",
		"exp":       time.Now().Add(time.Hour).Unix(),
	})

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)

	var capturedUserID, capturedTenantID string
	handler := JWTAuth(jwks, true, "test-issuer", "test-audience")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if uid, ok := UserIDFromRequest(r); ok {
			capturedUserID = uid.String()
		}
		if tid, ok := TenantIDFromRequest(r); ok {
			capturedTenantID = tid
		}
		w.WriteHeader(200)
	}))

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if capturedUserID == "" {
		t.Error("user ID should be injected into context")
	}
	if capturedTenantID == "" {
		t.Error("tenant ID should be injected into context")
	}
}

// Test PEM-based JWKS loading
func TestJWKS_LoadPublicKey(t *testing.T) {
	priv, _ := setupTestJWKS(t)

	// Marshal public key to PEM
	pubDER, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		t.Fatalf("failed to marshal public key: %v", err)
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubDER,
	})

	// Write to temp file
	tmpFile := t.TempDir() + "/pubkey.pem"
	if err := os.WriteFile(tmpFile, pemBytes, 0644); err != nil {
		t.Fatal(err)
	}

	// Load it
	pub, kid, err := loadPublicKey(tmpFile)
	if err != nil {
		t.Fatalf("loadPublicKey failed: %v", err)
	}
	if pub == nil {
		t.Error("expected non-nil public key")
	}
	if kid == "" {
		t.Error("expected non-empty key ID")
	}
}


