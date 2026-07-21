package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ggid/ggid/services/gateway/internal/config"
)

func newTestConfig(authURL string) *config.Config {
	return &config.Config{
		Routes: map[string]string{
			"/api/v1/auth": authURL,
		},
	}
}

func TestWebhookCatalog(t *testing.T) {
	gw := &Gateway{}
	req := httptest.NewRequest("GET", "/api/v1/webhooks/events/catalog", nil)
	w := httptest.NewRecorder()
	gw.handleWebhookCatalog(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	count, ok := resp["count"].(float64)
	if !ok || count < 1 {
		t.Errorf("expected count >= 1, got %v", resp["count"])
	}
}

func TestBootstrap_MissingFields(t *testing.T) {
	gw := &Gateway{}
	req := httptest.NewRequest("POST", "/api/v1/system/bootstrap",
		strings.NewReader(`{"admin_username":"a"}`))
	w := httptest.NewRecorder()
	gw.handleSystemBootstrap(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing fields, got %d", w.Code)
	}
}

// TestBootstrap_Valid tests that bootstrap with valid input attempts to call the auth service.
// Since there's no real auth service in unit tests, we expect 502 (Bad Gateway).
// Integration/E2E tests cover the full flow with real services.
func TestBootstrap_CallsAuthService(t *testing.T) {
	gw := &Gateway{}
	req := httptest.NewRequest("POST", "/api/v1/system/bootstrap",
		strings.NewReader(`{"admin_username":"admin","admin_email":"a@b.com","admin_password":"password123","tenant_name":"My Org"}`))
	w := httptest.NewRecorder()
	gw.handleSystemBootstrap(w, req)
	// Without a real auth service running, expect 502 Bad Gateway
	if w.Code != http.StatusBadGateway {
		t.Errorf("expected 502 (no auth service in test), got %d", w.Code)
	}
}

func TestSystemHealth(t *testing.T) {
	gw := &Gateway{}
	req := httptest.NewRequest("GET", "/api/v1/system/health", nil)
	w := httptest.NewRecorder()
	gw.handleSystemHealth(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["version"] == nil {
		t.Error("expected version field")
	}
}

func TestBootstrap_ShortPassword(t *testing.T) {
	gw := &Gateway{}
	req := httptest.NewRequest("POST", "/api/v1/system/bootstrap",
		strings.NewReader(`{"admin_username":"a","admin_email":"a@b.com","admin_password":"short"}`))
	w := httptest.NewRecorder()
	gw.handleSystemBootstrap(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for short password, got %d", w.Code)
	}
}

// TestBootstrap_WithMockAuthService tests the full bootstrap flow with a mock auth service.
func TestBootstrap_WithMockAuthService(t *testing.T) {
	// Create a mock auth service that handles register + verify
	mockAuth := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/auth/register" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"user_id":"test-user-id-123"}`))
			return
		}
		if r.URL.Path == "/api/v1/auth/verify" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"user_id":"test-user-id-123","tenant_id":"00000000-0000-0000-0000-000000000001","username":"admin","mfa_required":false}`))
			return
		}
		if r.URL.Path == "/api/v1/oauth/register" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"client_id":"ggid-console-test"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer mockAuth.Close()

	gw := &Gateway{}
	// Configure the gateway to use the mock auth service
	gw.cfg = newTestConfig(mockAuth.URL)

	req := httptest.NewRequest("POST", "/api/v1/system/bootstrap",
		strings.NewReader(`{"admin_username":"admin","admin_email":"a@b.com","admin_password":"password123","tenant_name":"My Org"}`))
	w := httptest.NewRecorder()
	gw.handleSystemBootstrap(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["status"] != "bootstrapped" {
		t.Errorf("expected bootstrapped, got %v", resp["status"])
	}
	// Bootstrap no longer returns access_token — users authenticate via OAuth flow.
	if resp["access_token"] != nil {
		t.Errorf("bootstrap should not return access_token, got %v", resp["access_token"])
	}
}
