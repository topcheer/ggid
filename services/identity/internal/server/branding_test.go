package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ggid/ggid/services/identity/internal/service"
)

func newBrandingTestHandler() *HTTPHandler {
	return &HTTPHandler{
		svc:           nil, // not needed for branding
		brandingStore: service.NewBrandingStore(),
	}
}

func TestBranding_GetDefault(t *testing.T) {
	h := newBrandingTestHandler()
	h.mux = http.NewServeMux()
	h.mux.HandleFunc("/api/v1/tenants/", h.handleBranding)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tenants/tenant-123/branding", nil)
	rr := httptest.NewRecorder()
	h.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	if !strings.Contains(body, "primary_color") {
		t.Error("response should contain primary_color")
	}
	if !strings.Contains(body, "#2563eb") {
		t.Error("default primary_color should be #2563eb")
	}
}

func TestBranding_Update(t *testing.T) {
	h := newBrandingTestHandler()
	h.mux = http.NewServeMux()
	h.mux.HandleFunc("/api/v1/tenants/", h.handleBranding)

	body := `{"logo_url":"https://example.com/logo.png","primary_color":"#ff0000","secondary_color":"#cc0000","custom_domain":"login.example.com","email_template":"custom"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/tenants/t1/branding", strings.NewReader(body))
	rr := httptest.NewRecorder()
	h.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	resp := rr.Body.String()
	if !strings.Contains(resp, "#ff0000") {
		t.Error("primary_color should be updated to #ff0000")
	}
	if !strings.Contains(resp, "login.example.com") {
		t.Error("custom_domain should be set")
	}
}

func TestBranding_GetAfterUpdate(t *testing.T) {
	h := newBrandingTestHandler()
	h.mux = http.NewServeMux()
	h.mux.HandleFunc("/api/v1/tenants/", h.handleBranding)

	// Update first
	body := `{"logo_url":"https://example.com/logo.png","primary_color":"#00ff00"}`
	putReq := httptest.NewRequest(http.MethodPut, "/api/v1/tenants/t2/branding", strings.NewReader(body))
	putRR := httptest.NewRecorder()
	h.mux.ServeHTTP(putRR, putReq)
	if putRR.Code != http.StatusOK {
		t.Fatalf("update failed: %d", putRR.Code)
	}

	// Then GET
	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/tenants/t2/branding", nil)
	getRR := httptest.NewRecorder()
	h.mux.ServeHTTP(getRR, getReq)
	if getRR.Code != http.StatusOK {
		t.Fatalf("get failed: %d", getRR.Code)
	}
	resp := getRR.Body.String()
	if !strings.Contains(resp, "#00ff00") {
		t.Error("GET after PUT should return updated primary_color")
	}
	if !strings.Contains(resp, "https://example.com/logo.png") {
		t.Error("GET after PUT should return updated logo_url")
	}
}

func TestBranding_MethodNotAllowed(t *testing.T) {
	h := newBrandingTestHandler()
	h.mux = http.NewServeMux()
	h.mux.HandleFunc("/api/v1/tenants/", h.handleBranding)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tenants/t3/branding", nil)
	rr := httptest.NewRecorder()
	h.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}

func TestBranding_InvalidJSON(t *testing.T) {
	h := newBrandingTestHandler()
	h.mux = http.NewServeMux()
	h.mux.HandleFunc("/api/v1/tenants/", h.handleBranding)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/tenants/t4/branding", strings.NewReader("not json"))
	rr := httptest.NewRecorder()
	h.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestBranding_TenantIsolation(t *testing.T) {
	h := newBrandingTestHandler()
	h.mux = http.NewServeMux()
	h.mux.HandleFunc("/api/v1/tenants/", h.handleBranding)

	// Update tenant A
	bodyA := `{"primary_color":"#aaaaaa"}`
	putReq := httptest.NewRequest(http.MethodPut, "/api/v1/tenants/tenantA/branding", strings.NewReader(bodyA))
	h.mux.ServeHTTP(httptest.NewRecorder(), putReq)

	// Update tenant B
	bodyB := `{"primary_color":"#bbbbbb"}`
	putReqB := httptest.NewRequest(http.MethodPut, "/api/v1/tenants/tenantB/branding", strings.NewReader(bodyB))
	h.mux.ServeHTTP(httptest.NewRecorder(), putReqB)

	// GET tenant A — should have #aaaaaa
	getReqA := httptest.NewRequest(http.MethodGet, "/api/v1/tenants/tenantA/branding", nil)
	getRRA := httptest.NewRecorder()
	h.mux.ServeHTTP(getRRA, getReqA)
	if !strings.Contains(getRRA.Body.String(), "#aaaaaa") {
		t.Error("tenant A should have its own branding")
	}

	// GET tenant B — should have #bbbbbb
	getReqB := httptest.NewRequest(http.MethodGet, "/api/v1/tenants/tenantB/branding", nil)
	getRRB := httptest.NewRecorder()
	h.mux.ServeHTTP(getRRB, getReqB)
	if !strings.Contains(getRRB.Body.String(), "#bbbbbb") {
		t.Error("tenant B should have its own branding")
	}
}
