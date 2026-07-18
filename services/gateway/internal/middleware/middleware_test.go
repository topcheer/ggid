package middleware

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ggid/ggid/pkg/tenant"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// --- Helpers ---

// generateTestRSAKey creates a temp RSA key pair and writes the public key to a file.
// Returns the private key for signing test JWTs and the public key file path.
func generateTestRSAKey(t *testing.T) (*rsa.PrivateKey, string) {
	t.Helper()
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}

	dir := t.TempDir()
	pubPath := filepath.Join(dir, "test_pub.pem")
	writeTestPublicKey(t, pubPath, &privKey.PublicKey)

	return privKey, pubPath
}

func writeTestPublicKey(t *testing.T, path string, pub *rsa.PublicKey) {
	t.Helper()
	// Write PKIX PEM
	import_pem_write_pub(t, path, pub)
}

// signTestJWT creates an RS256 JWT for testing.
func signTestJWT(t *testing.T, privKey *rsa.PrivateKey, claims jwt.MapClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := token.SignedString(privKey)
	if err != nil {
		t.Fatalf("sign JWT: %v", err)
	}
	return signed
}

// --- Request ID Tests ---

func TestRequestID_GeneratesNewID(t *testing.T) {
	called := false
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID, ok := r.Context().Value(RequestIDKey).(string)
		if !ok || requestID == "" {
			t.Error("expected non-empty request ID in context")
		}
		if w.Header().Get("X-Request-ID") == "" {
			t.Error("expected X-Request-ID in response header")
		}
		called = true
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !called {
		t.Fatal("handler not called")
	}
}

func TestRequestID_PreservesIncomingID(t *testing.T) {
	incomingID := "test-request-id-12345"
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID, _ := r.Context().Value(RequestIDKey).(string)
		if requestID != incomingID {
			t.Errorf("expected %s, got %s", incomingID, requestID)
		}
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Request-ID", incomingID)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Header().Get("X-Request-ID") != incomingID {
		t.Errorf("expected response header %s, got %s", incomingID, w.Header().Get("X-Request-ID"))
	}
}

// --- CORS Tests ---

func TestCORS_SetsHeaders(t *testing.T) {
	handler := CORSWithConfig(CORSConfig{AllowedOrigins: []string{"*"}})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("missing CORS origin header")
	}
	if w.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("missing CORS methods header")
	}
}

func TestCORS_PreflightReturns204(t *testing.T) {
	handler := CORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called for OPTIONS")
	}))

	req := httptest.NewRequest("OPTIONS", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204 for OPTIONS, got %d", w.Code)
	}
}

// --- Tenant Resolver Tests ---

func TestTenantResolver_FromHeader(t *testing.T) {
	tenantID := uuid.New()
	handler := TenantResolver("")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tc, err := tenant.FromContext(r.Context())
		if err != nil {
			t.Fatalf("expected tenant context: %v", err)
		}
		if tc.TenantID != tenantID {
			t.Errorf("expected %s, got %s", tenantID, tc.TenantID)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Tenant-ID", tenantID.String())
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
}

func TestTenantResolver_NoTenant(t *testing.T) {
	handler := TenantResolver("")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := tenant.FromContext(r.Context())
		if err == nil {
			t.Error("expected no tenant context when none provided")
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
}

func TestTenantResolver_InvalidUUID(t *testing.T) {
	handler := TenantResolver("")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := tenant.FromContext(r.Context())
		if err == nil {
			t.Error("expected no tenant for invalid UUID")
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Tenant-ID", "not-a-uuid")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
}

// --- JWT Auth Tests ---

func TestJWTAuth_MissingAuthHeader(t *testing.T) {
	privKey, pubPath := generateTestRSAKey(t)
	_ = privKey

	jwks, err := NewJWKSClient("", pubPath)
	if err != nil {
		t.Fatalf("NewJWKSClient: %v", err)
	}

	handler := JWTAuth(jwks, true, "test-issuer", "test-aud")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach handler without token")
	}))

	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestJWTAuth_InvalidTokenFormat(t *testing.T) {
	privKey, pubPath := generateTestRSAKey(t)
	_ = privKey

	jwks, _ := NewJWKSClient("", pubPath)

	handler := JWTAuth(jwks, true, "", "")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach handler with bad token format")
	}))

	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	req.Header.Set("Authorization", "Basic abc123")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestJWTAuth_ValidToken(t *testing.T) {
	privKey, pubPath := generateTestRSAKey(t)

	jwks, err := NewJWKSClient("", pubPath)
	if err != nil {
		t.Fatalf("NewJWKSClient: %v", err)
	}

	userID := uuid.New()
	tenantID := uuid.New()

	token := signTestJWT(t, privKey, jwt.MapClaims{
		"sub":        userID.String(),
		"tenant_id":  tenantID.String(),
		"iss":        "test-issuer",
		"exp":        time.Now().Add(15 * time.Minute).Unix(),
		"iat":        time.Now().Unix(),
	})

	handler := JWTAuth(jwks, true, "test-issuer", "")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify user ID is injected into context
		userIDStr, ok := r.Context().Value(UserIDKey).(string)
		if !ok || userIDStr != userID.String() {
			t.Errorf("expected user ID %s in context, got %v", userID, userIDStr)
		}
		// Verify tenant ID is injected
		tenantIDStr, _ := r.Context().Value(TenantIDKey).(string)
		if tenantIDStr != tenantID.String() {
			t.Errorf("expected tenant ID %s, got %s", tenantID, tenantIDStr)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestJWTAuth_ExpiredToken(t *testing.T) {
	privKey, pubPath := generateTestRSAKey(t)

	jwks, _ := NewJWKSClient("", pubPath)

	userID := uuid.New()
	token := signTestJWT(t, privKey, jwt.MapClaims{
		"sub": userID.String(),
		"iss": "test-issuer",
		"exp": time.Now().Add(-1 * time.Hour).Unix(), // expired
		"iat": time.Now().Add(-2 * time.Hour).Unix(),
	})

	handler := JWTAuth(jwks, true, "", "")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach handler with expired token")
	}))

	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for expired token, got %d", w.Code)
	}
}

func TestJWTAuth_WrongIssuer(t *testing.T) {
	privKey, pubPath := generateTestRSAKey(t)
	jwks, _ := NewJWKSClient("", pubPath)

	token := signTestJWT(t, privKey, jwt.MapClaims{
		"sub": uuid.New().String(),
		"iss": "wrong-issuer",
		"exp": time.Now().Add(15 * time.Minute).Unix(),
		"iat": time.Now().Unix(),
	})

	handler := JWTAuth(jwks, true, "expected-issuer", "")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach handler with wrong issuer")
	}))

	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for wrong issuer, got %d", w.Code)
	}
}

func TestJWTAuth_OptionalNoToken(t *testing.T) {
	privKey, pubPath := generateTestRSAKey(t)
	_ = privKey

	jwks, _ := NewJWKSClient("", pubPath)

	called := false
	handler := JWTAuth(jwks, false, "", "")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !called {
		t.Fatal("handler should be called when required=false and no token")
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestJWTAuth_WrongSigningKey(t *testing.T) {
	// Generate two separate key pairs
	privKey1, _ := generateTestRSAKey(t)
	_, pubPath2 := generateTestRSAKey(t)

	jwks, err := NewJWKSClient("", pubPath2)
	if err != nil {
		t.Fatalf("NewJWKSClient: %v", err)
	}

	// Sign with key1, verify with key2's public key — should fail
	token := signTestJWT(t, privKey1, jwt.MapClaims{
		"sub": uuid.New().String(),
		"exp": time.Now().Add(15 * time.Minute).Unix(),
		"iat": time.Now().Unix(),
	})

	handler := JWTAuth(jwks, true, "", "")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach handler with wrong key")
	}))

	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for wrong signing key, got %d", w.Code)
	}
}

func TestJWTAuth_TamperedToken(t *testing.T) {
	privKey, pubPath := generateTestRSAKey(t)
	jwks, _ := NewJWKSClient("", pubPath)

	token := signTestJWT(t, privKey, jwt.MapClaims{
		"sub": uuid.New().String(),
		"exp": time.Now().Add(15 * time.Minute).Unix(),
		"iat": time.Now().Unix(),
	})

	// Tamper: replace the signature (last segment) with garbage
	parts := strings.SplitN(token, ".", 3)
	if len(parts) != 3 {
		t.Fatalf("expected 3-part JWT, got %d parts", len(parts))
	}
	tampered := parts[0] + "." + parts[1] + ".INVALID_SIGNATURE"

	handler := JWTAuth(jwks, true, "", "")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach handler with tampered token")
	}))

	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	req.Header.Set("Authorization", "Bearer "+tampered)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for tampered token, got %d", w.Code)
	}
}

// --- JWKS Client Tests ---

func TestJWKSClient_LoadPublicKey(t *testing.T) {
	_, pubPath := generateTestRSAKey(t)

	jwks, err := NewJWKSClient("", pubPath)
	if err != nil {
		t.Fatalf("NewJWKSClient: %v", err)
	}
	if jwks.KeyID() == "" {
		t.Error("expected non-empty key ID")
	}
	if jwks.publicKey == nil {
		t.Error("expected non-nil public key")
	}
}

func TestJWKSClient_NonExistentKeyFile(t *testing.T) {
	_, err := NewJWKSClient("", "/nonexistent/path/key.pem")
	if err == nil {
		t.Error("expected error for non-existent key file")
	}
}

func TestJWKSClient_JWKSHandler(t *testing.T) {
	_, pubPath := generateTestRSAKey(t)
	jwks, _ := NewJWKSClient("", pubPath)

	req := httptest.NewRequest("GET", "/.well-known/jwks.json", nil)
	w := httptest.NewRecorder()
	jwks.JWKSHandler()(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	// Just check that we got some JSON response
	body := w.Body.String()
	if body == "" {
		t.Error("expected non-empty JWKS response")
	}
}

// --- Context Helper Tests ---

func TestUserIDFromRequest(t *testing.T) {
	userID := uuid.New()
	req := httptest.NewRequest("GET", "/", nil)
	ctx := context.WithValue(req.Context(), UserIDKey, userID.String())
	req = req.WithContext(ctx)

	got, ok := UserIDFromRequest(req)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if got != userID {
		t.Errorf("expected %s, got %s", userID, got)
	}
}

func TestTenantIDFromRequest(t *testing.T) {
	tenantID := uuid.New().String()
	req := httptest.NewRequest("GET", "/", nil)
	ctx := context.WithValue(req.Context(), TenantIDKey, tenantID)
	req = req.WithContext(ctx)

	got, ok := TenantIDFromRequest(req)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if got != tenantID {
		t.Errorf("expected %s, got %s", tenantID, got)
	}
}

// --- Helpers for tests ---

func generateSeparateKey(t *testing.T) *rsa.PrivateKey { //nolint:unused // test helper, may be used in future tests
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	return key
}

// import_pem_write_pub writes a public key to a PEM file.
func import_pem_write_pub(t *testing.T, path string, pub *rsa.PublicKey) {
	t.Helper()

	// Marshal to PKIX DER
	derBytes, err := x509.MarshalPKIXPublicKey(pub) // Note: x509 is imported at file top

	if err != nil {
		t.Fatalf("marshal public key: %v", err)
	}

	// Encode as PEM
	pemData := pemEncode("PUBLIC KEY", derBytes)

	err = os.WriteFile(path, pemData, 0o644)
	if err != nil {
		t.Fatalf("write public key: %v", err)
	}

	_ = base64.StdEncoding // keep import
}

// pemEncode wraps DER bytes in a PEM block.
func pemEncode(blockType string, der []byte) []byte {
	// Simple PEM encoder for tests
	encoded := base64.StdEncoding.EncodeToString(der)
	var result []byte
	header := []byte("-----BEGIN " + blockType + "-----\n")
	footer := []byte("-----END " + blockType + "-----\n")
	result = append(result, header...)
	for i := 0; i < len(encoded); i += 64 {
		end := i + 64
		if end > len(encoded) {
			end = len(encoded)
		}
		result = append(result, []byte(encoded[i:end]+"\n")...)
	}
	result = append(result, footer...)
	return result
}

// Ensure imports are used
var _ = big.NewInt

// --- CORSWithConfig Tests ---

func TestCORSWithConfig_SpecificOrigin(t *testing.T) {
	cfg := CORSConfig{
		AllowedOrigins:   []string{"https://app.ggid.dev"},
		AllowCredentials: true,
	}
	handler := CORSWithConfig(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://app.ggid.dev")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "https://app.ggid.dev" {
		t.Errorf("expected specific origin, got %s", w.Header().Get("Access-Control-Allow-Origin"))
	}
	if w.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Error("expected credentials header")
	}
}

func TestCORSWithConfig_UnlistedOrigin(t *testing.T) {
	cfg := CORSConfig{
		AllowedOrigins: []string{"https://app.ggid.dev"},
	}
	handler := CORSWithConfig(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://evil.com")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("expected no origin for unlisted domain, got %s", w.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORSWithConfig_Preflight(t *testing.T) {
	handler := CORSWithConfig(DefaultCORSConfig())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not call next for OPTIONS")
	}))

	req := httptest.NewRequest("OPTIONS", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
	if w.Header().Get("Access-Control-Expose-Headers") == "" {
		t.Error("expected expose headers")
	}
}

// --- SecurityHeaders Tests ---

func TestSecurityHeaders_SetsAllHeaders(t *testing.T) {
	handler := SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	tests := []struct{ header, expected string }{
		{"X-Content-Type-Options", "nosniff"},
		{"X-Frame-Options", "DENY"},
		{"Referrer-Policy", "strict-origin-when-cross-origin"},
		{"Strict-Transport-Security", "max-age=31536000; includeSubDomains"},
	}
	for _, tt := range tests {
		if got := w.Header().Get(tt.header); got != tt.expected {
			t.Errorf("expected %s=%s, got %s", tt.header, tt.expected, got)
		}
	}
}

// --- CSRF Protection Tests ---

func TestCSRFProtect_GETSetsCookie(t *testing.T) {
	called := false
	handler := CSRFProtect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !called {
		t.Error("handler should be called for GET")
	}
	cookies := w.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == "csrf_token" && c.Value != "" {
			found = true
			if !c.Secure {
				t.Error("csrf cookie should be Secure")
			}
			if c.SameSite != http.SameSiteLaxMode {
				t.Error("csrf cookie should be SameSite=Lax")
			}
		}
	}
	if !found {
		t.Error("expected csrf_token cookie to be set")
	}
}

func TestCSRFProtect_POSTWithoutToken_403(t *testing.T) {
	called := false
	handler := CSRFProtect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest("POST", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if called {
		t.Error("handler should NOT be called without CSRF token")
	}
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestCSRFProtect_POSTWithMatchingTokens_Passes(t *testing.T) {
	token := "test-csrf-token-12345"
	called := false
	handler := CSRFProtect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/test", nil)
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: token})
	req.Header.Set("X-CSRF-Token", token)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !called {
		t.Error("handler should be called with matching tokens")
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestCSRFProtect_POSTWithMismatchedTokens_403(t *testing.T) {
	called := false
	handler := CSRFProtect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest("POST", "/test", nil)
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: "token-a"})
	req.Header.Set("X-CSRF-Token", "token-b")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if called {
		t.Error("handler should NOT be called with mismatched tokens")
	}
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}
