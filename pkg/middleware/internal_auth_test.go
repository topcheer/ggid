package middleware

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

func testHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	})
}

func TestInternalAuth_ValidSignature(t *testing.T) {
	secret := []byte("test-secret")
	cfg := InternalAuthConfig{Secret: secret}
	ts := time.Now().Unix()
	reqID := "req-123"
	sig := ComputeSignatureBound(secret, "gateway", strconv.FormatInt(ts, 10), reqID, "GET", "/api/v1/users")

	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	req.Header.Set(InternalAuthHeaderService, "gateway")
	req.Header.Set(InternalAuthHeaderTimestamp, strconv.FormatInt(ts, 10))
	req.Header.Set(InternalAuthHeaderSignature, sig)
	req.Header.Set("X-Request-ID", reqID)

	w := httptest.NewRecorder()
	InternalAuth(cfg)(testHandler()).ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestInternalAuth_MissingHeaders(t *testing.T) {
	cfg := InternalAuthConfig{Secret: []byte("s")}
	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	w := httptest.NewRecorder()
	InternalAuth(cfg)(testHandler()).ServeHTTP(w, req)
	if w.Code != 403 {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestInternalAuth_ExpiredTimestamp(t *testing.T) {
	secret := []byte("test-secret")
	cfg := InternalAuthConfig{Secret: secret, ReplayWindow: 120}
	oldTs := time.Now().Unix() - 300 // 5 min ago
	sig := ComputeSignatureBound(secret, "gateway", strconv.FormatInt(oldTs, 10), "req-1", "GET", "/api/v1/users")

	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	req.Header.Set(InternalAuthHeaderService, "gateway")
	req.Header.Set(InternalAuthHeaderTimestamp, strconv.FormatInt(oldTs, 10))
	req.Header.Set(InternalAuthHeaderSignature, sig)

	w := httptest.NewRecorder()
	InternalAuth(cfg)(testHandler()).ServeHTTP(w, req)
	if w.Code != 403 {
		t.Errorf("expected 403 for expired timestamp, got %d", w.Code)
	}
}

func TestInternalAuth_WrongSignature(t *testing.T) {
	cfg := InternalAuthConfig{Secret: []byte("correct")}
	ts := time.Now().Unix()
	// Generate a valid signature with a DIFFERENT secret.
	wrongSig := ComputeSignatureBound([]byte("wrong-secret"), "gateway", strconv.FormatInt(ts, 10), "", "GET", "/api/v1/users")

	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	req.Header.Set(InternalAuthHeaderService, "gateway")
	req.Header.Set(InternalAuthHeaderTimestamp, strconv.FormatInt(ts, 10))
	req.Header.Set(InternalAuthHeaderSignature, wrongSig)

	w := httptest.NewRecorder()
	InternalAuth(cfg)(testHandler()).ServeHTTP(w, req)
	if w.Code != 403 {
		t.Errorf("expected 403 for wrong signature, got %d", w.Code)
	}
}

func TestInternalAuth_Whitelist(t *testing.T) {
	cfg := InternalAuthConfig{
		Secret:    []byte("s"),
		Whitelist: []string{"/healthz", "/metrics"},
	}

	for _, path := range []string{"/healthz", "/metrics"} {
		req := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()
		InternalAuth(cfg)(testHandler()).ServeHTTP(w, req)
		if w.Code != 200 {
			t.Errorf("whitelisted %s should be 200, got %d", path, w.Code)
		}
	}
}

func TestInternalAuth_PrevSecret(t *testing.T) {
	current := []byte("new-secret")
	prev := []byte("old-secret")
	cfg := InternalAuthConfig{Secret: current, PrevSecret: prev}
	ts := time.Now().Unix()
	// Sign with old secret.
	sig := ComputeSignatureBound(prev, "gateway", strconv.FormatInt(ts, 10), "r1", "GET", "/api/v1/users")

	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	req.Header.Set(InternalAuthHeaderService, "gateway")
	req.Header.Set(InternalAuthHeaderTimestamp, strconv.FormatInt(ts, 10))
	req.Header.Set(InternalAuthHeaderSignature, sig)
	req.Header.Set("X-Request-ID", "r1")

	w := httptest.NewRecorder()
	InternalAuth(cfg)(testHandler()).ServeHTTP(w, req)
	if w.Code != 200 {
		t.Errorf("prev secret should be accepted during rotation, got %d", w.Code)
	}
}

func TestInternalAuth_NilSecretAllows(t *testing.T) {
	cfg := InternalAuthConfig{Secret: nil}
	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	w := httptest.NewRecorder()
	InternalAuth(cfg)(testHandler()).ServeHTTP(w, req)
	if w.Code != 200 {
		t.Errorf("nil secret should allow all (dev mode), got %d", w.Code)
	}
}

func TestSignInternalRequest(t *testing.T) {
	secret := []byte("test-secret")
	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	req.Header.Set("X-Request-ID", "req-99")
	SignInternalRequest(req, "gateway", secret)

	if req.Header.Get(InternalAuthHeaderService) != "gateway" {
		t.Error("service header not set")
	}
	if req.Header.Get(InternalAuthHeaderTimestamp) == "" {
		t.Error("timestamp header not set")
	}
	if req.Header.Get(InternalAuthHeaderSignature) == "" {
		t.Error("signature header not set")
	}

	// Verify the signature is valid.
	ts := req.Header.Get(InternalAuthHeaderTimestamp)
	sig := req.Header.Get(InternalAuthHeaderSignature)
	expected := ComputeSignatureBound(secret, "gateway", ts, "req-99", "GET", "/api/v1/users")
	if sig != expected {
		t.Error("signature mismatch")
	}
}
