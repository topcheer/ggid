package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	ggidtenant "github.com/ggid/ggid/pkg/tenant"
	"github.com/google/uuid"
)

// TestLoginPolicyHandler tests GET/PUT /api/v1/auth/login-policy
func TestLoginPolicyHandler(t *testing.T) {
	h := &Handler{}

	// Reset global to known state
	loginPolicyMu.Lock()
	loginPolicy = LoginPolicy{MaxAttempts: 5, LockoutDurationMinutes: 30}
	loginPolicyMu.Unlock()

	// GET
	req := httptest.NewRequest("GET", "/api/v1/auth/login-policy", nil)
	w := httptest.NewRecorder()
	h.handleLoginPolicy(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET: expected 200, got %d", w.Code)
	}
	var policy LoginPolicy
	json.Unmarshal(w.Body.Bytes(), &policy)
	if policy.MaxAttempts != 5 {
		t.Errorf("expected max_attempts=5, got %d", policy.MaxAttempts)
	}

	// PUT valid
	body := `{"max_attempts":10,"lockout_duration_minutes":60}`
	req = httptest.NewRequest("PUT", "/api/v1/auth/login-policy", strings.NewReader(body))
	w = httptest.NewRecorder()
	h.handleLoginPolicy(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("PUT: expected 200, got %d", w.Code)
	}

	// PUT invalid — max_attempts < 1
	body = `{"max_attempts":0,"lockout_duration_minutes":30}`
	req = httptest.NewRequest("PUT", "/api/v1/auth/login-policy", strings.NewReader(body))
	w = httptest.NewRecorder()
	h.handleLoginPolicy(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid max_attempts, got %d: %s", w.Code, w.Body.String())
	}

	// Method not allowed
	req = httptest.NewRequest("DELETE", "/api/v1/auth/login-policy", nil)
	w = httptest.NewRecorder()
	h.handleLoginPolicy(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

// TestAPIKeysHandler tests that the API keys handler requires DB pool.
// Full CRUD tests require a live PostgreSQL instance (integration test).
func TestAPIKeysHandler(t *testing.T) {
	h := &Handler{} // nil pool

	// Without DB pool, API key operations should return 503.
	req := httptest.NewRequest("GET", "/api/v1/auth/api-keys", nil)
	w := httptest.NewRecorder()
	h.handleAPIKeys(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 (no DB), got %d: %s", w.Code, w.Body.String())
	}

	body := `{"name":"test-key","scopes":["read","write"],"expires_at":""}`
	req = httptest.NewRequest("POST", "/api/v1/auth/api-keys", strings.NewReader(body))
	w = httptest.NewRecorder()
	h.handleAPIKeys(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("POST expected 503 (no DB), got %d: %s", w.Code, w.Body.String())
	}
}

// TestBreakGlassHandler tests GET /api/v1/auth/break-glass/history
func TestBreakGlassHandler(t *testing.T) {
	h := &Handler{}

	// Inject tenant context (required by DB-backed handler).
	tc := &ggidtenant.Context{TenantID: uuid.New(), IsolationLevel: ggidtenant.IsolationShared}
	req := httptest.NewRequest("GET", "/api/v1/auth/break-glass/history", nil)
	req = req.WithContext(ggidtenant.WithContext(req.Context(), tc))
	w := httptest.NewRecorder()
	h.handleBreakGlass(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Method not allowed
	req = httptest.NewRequest("POST", "/api/v1/auth/break-glass/history", nil)
	req = req.WithContext(ggidtenant.WithContext(req.Context(), tc))
	w = httptest.NewRecorder()
	h.handleBreakGlass(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

// TestFeatureFlagsHandler tests GET/POST + toggle
func TestFeatureFlagsHandler(t *testing.T) {
	h := &Handler{}

	// GET
	req := httptest.NewRequest("GET", "/api/v1/admin/feature-flags", nil)
	w := httptest.NewRecorder()
	h.handleFeatureFlags(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET: expected 200, got %d", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	flags, ok := resp["flags"].([]any)
	if !ok || len(flags) == 0 {
		t.Fatal("expected non-empty flags array")
	}

	// POST create
	body := `{"name":"new-flag","enabled":false,"rollout_pct":0,"target_audience":"all"}`
	req = httptest.NewRequest("POST", "/api/v1/admin/feature-flags", strings.NewReader(body))
	w = httptest.NewRecorder()
	h.handleFeatureFlags(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("POST: expected 201, got %d", w.Code)
	}

	// Toggle
	req = httptest.NewRequest("POST", "/api/v1/admin/feature-flags/new-flag/toggle", nil)
	w = httptest.NewRecorder()
	h.handleFeatureFlags(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("toggle: expected 200, got %d", w.Code)
	}
}

// TestCertificatesV2Handler tests GET/POST certificates
func TestCertificatesV2Handler(t *testing.T) {
	h := &Handler{}

	// GET
	req := httptest.NewRequest("GET", "/api/v1/certificates", nil)
	w := httptest.NewRecorder()
	h.handleCertificatesV2(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET: expected 200, got %d", w.Code)
	}

	// POST sign
	body := `{"name":"test-cert","type":"TLS","domain":"test.local","cn":"test.local"}`
	req = httptest.NewRequest("POST", "/api/v1/certificates/sign", strings.NewReader(body))
	w = httptest.NewRecorder()
	h.handleCertificatesV2(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("POST sign: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var result map[string]any
	json.Unmarshal(w.Body.Bytes(), &result)
	if _, ok := result["certificate"]; !ok {
		t.Error("expected certificate in response")
	}
	if _, ok := result["cert_pem"]; !ok {
		t.Error("expected cert_pem in response")
	}
}
