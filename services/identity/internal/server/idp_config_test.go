package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ggid/ggid/services/identity/internal/idpconfig"
	"github.com/ggid/ggid/services/identity/internal/service"
)

func newIdPConfigTestHandler() *HTTPHandler {
	return &HTTPHandler{
		svc:          nil,
		brandingStore: service.NewBrandingStore(),
		idpConfigSvc: idpconfig.NewService(idpconfig.NewMemoryStore()),
	}
}

// valid tenant UUID for tests
const testTenantUUID = "550e8400-e29b-41d4-a716-446655440000"

func TestIdPConfig_ListEmpty(t *testing.T) {
	h := newIdPConfigTestHandler()
	h.mux = http.NewServeMux()
	h.mux.HandleFunc("/api/v1/tenants/", h.handleBranding)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tenants/"+testTenantUUID+"/idp-config", nil)
	rr := httptest.NewRecorder()
	h.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "configs") {
		t.Error("response should contain configs field")
	}
}

func TestIdPConfig_CreateAndGet(t *testing.T) {
	h := newIdPConfigTestHandler()
	h.mux = http.NewServeMux()
	h.mux.HandleFunc("/api/v1/tenants/", h.handleBranding)

	body := `{"idp_type":"saml","name":"Corp SAML","config_json":"{\"entity_id\":\"https://corp.example.com\"}"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tenants/"+testTenantUUID+"/idp-config", strings.NewReader(body))
	rr := httptest.NewRecorder()
	h.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	resp := rr.Body.String()
	if !strings.Contains(resp, "Corp SAML") {
		t.Error("response should contain config name")
	}
	if !strings.Contains(resp, "saml") {
		t.Error("response should contain idp_type saml")
	}
}

func TestIdPConfig_CreateInvalidType(t *testing.T) {
	h := newIdPConfigTestHandler()
	h.mux = http.NewServeMux()
	h.mux.HandleFunc("/api/v1/tenants/", h.handleBranding)

	body := `{"idp_type":"invalid","name":"Bad","config_json":"{}"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tenants/"+testTenantUUID+"/idp-config", strings.NewReader(body))
	rr := httptest.NewRecorder()
	h.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid type, got %d", rr.Code)
	}
}

func TestIdPConfig_CreateInvalidJSON(t *testing.T) {
	h := newIdPConfigTestHandler()
	h.mux = http.NewServeMux()
	h.mux.HandleFunc("/api/v1/tenants/", h.handleBranding)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tenants/"+testTenantUUID+"/idp-config", strings.NewReader("not json"))
	rr := httptest.NewRecorder()
	h.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid JSON, got %d", rr.Code)
	}
}

func TestIdPConfig_ListAfterCreate(t *testing.T) {
	h := newIdPConfigTestHandler()
	h.mux = http.NewServeMux()
	h.mux.HandleFunc("/api/v1/tenants/", h.handleBranding)

	// Create a config
	body := `{"idp_type":"oidc","name":"Google OIDC","config_json":"{}"}`
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/tenants/"+testTenantUUID+"/idp-config", strings.NewReader(body))
	h.mux.ServeHTTP(httptest.NewRecorder(), createReq)

	// List should contain the created config
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/tenants/"+testTenantUUID+"/idp-config", nil)
	listRR := httptest.NewRecorder()
	h.mux.ServeHTTP(listRR, listReq)

	if listRR.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", listRR.Code)
	}
	if !strings.Contains(listRR.Body.String(), "Google OIDC") {
		t.Error("list should contain created config")
	}
}

func TestIdPConfig_MethodNotAllowed(t *testing.T) {
	h := newIdPConfigTestHandler()
	h.mux = http.NewServeMux()
	h.mux.HandleFunc("/api/v1/tenants/", h.handleBranding)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/tenants/"+testTenantUUID+"/idp-config", nil)
	rr := httptest.NewRecorder()
	h.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for PATCH, got %d", rr.Code)
	}
}

func TestIdPConfig_InvalidTenantID(t *testing.T) {
	h := newIdPConfigTestHandler()
	h.mux = http.NewServeMux()
	h.mux.HandleFunc("/api/v1/tenants/", h.handleBranding)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tenants/not-a-uuid/idp-config", nil)
	rr := httptest.NewRecorder()
	h.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid tenant_id, got %d", rr.Code)
	}
}
