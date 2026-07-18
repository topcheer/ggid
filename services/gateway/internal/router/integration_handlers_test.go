package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)


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

func TestBootstrap_Valid(t *testing.T) {
	gw := &Gateway{}
	req := httptest.NewRequest("POST", "/api/v1/system/bootstrap",
		strings.NewReader(`{"admin_username":"admin","admin_email":"a@b.com","admin_password":"password123","tenant_name":"My Org"}`))
	w := httptest.NewRecorder()
	gw.handleSystemBootstrap(w, req)
	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", w.Code)
	}
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["status"] != "bootstrapped" {
		t.Errorf("expected bootstrapped, got %v", resp["status"])
	}
}

func TestTenantCreate(t *testing.T) {
	gw := &Gateway{}
	req := httptest.NewRequest("POST", "/api/v1/tenants",
		strings.NewReader(`{"name":"acme","display_name":"Acme Corp"}`))
	w := httptest.NewRecorder()
	gw.handleTenantCreate(w, req)
	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", w.Code)
	}
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["api_key"] == nil {
		t.Error("expected api_key in response")
	}
}

func TestTenantDetail(t *testing.T) {
	gw := &Gateway{}
	req := httptest.NewRequest("GET", "/api/v1/tenants/123", nil)
	w := httptest.NewRecorder()
	gw.handleTenantDetail(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
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
