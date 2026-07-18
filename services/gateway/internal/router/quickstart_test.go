package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestQuickstart_Default(t *testing.T) {
	// Reset state for this test.
	quickstartInitialized = false
	gw := &Gateway{}

	req := httptest.NewRequest("POST", "/api/v1/system/quickstart", strings.NewReader(`{}`))
	w := httptest.NewRecorder()
	gw.handleQuickstart(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var resp QuickstartResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Status != "initialized" {
		t.Errorf("expected initialized, got %s", resp.Status)
	}
	if resp.TenantID == "" {
		t.Error("expected tenant_id")
	}
	if resp.OAuthClientSecret == "" {
		t.Error("expected oauth_client_secret")
	}
	if len(resp.SampleCurl) < 3 {
		t.Error("expected sample curl commands")
	}
}

func TestQuickstart_Idempotent(t *testing.T) {
	quickstartInitialized = true
	gw := &Gateway{}

	req := httptest.NewRequest("POST", "/api/v1/system/quickstart", strings.NewReader(`{}`))
	w := httptest.NewRecorder()
	gw.handleQuickstart(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for idempotent call, got %d", w.Code)
	}
	var resp QuickstartResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Status != "already_initialized" {
		t.Errorf("expected already_initialized, got %s", resp.Status)
	}
}

func TestQuickstart_ShortPassword(t *testing.T) {
	quickstartInitialized = false
	gw := &Gateway{}

	req := httptest.NewRequest("POST", "/api/v1/system/quickstart",
		strings.NewReader(`{"admin_password":"short"}`))
	w := httptest.NewRecorder()
	gw.handleQuickstart(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for short password, got %d", w.Code)
	}
}

func TestQuickstart_CustomFields(t *testing.T) {
	quickstartInitialized = false
	gw := &Gateway{}

	body := `{"admin_username":"root","admin_email":"root@test.com","admin_password":"StrongPass1!","tenant_name":"TestOrg"}`
	req := httptest.NewRequest("POST", "/api/v1/system/quickstart", strings.NewReader(body))
	w := httptest.NewRecorder()
	gw.handleQuickstart(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	var resp QuickstartResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.AdminUsername != "root" {
		t.Errorf("expected root, got %s", resp.AdminUsername)
	}
}

func TestSystemStatus_NotInitialized(t *testing.T) {
	quickstartInitialized = false
	gw := &Gateway{}

	req := httptest.NewRequest("GET", "/api/v1/system/status", nil)
	w := httptest.NewRecorder()
	gw.handleSystemStatus(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var status SystemStatus
	_ = json.Unmarshal(w.Body.Bytes(), &status)
	if status.Initialized {
		t.Error("expected not initialized")
	}
	if status.Version == "" {
		t.Error("expected version")
	}
}

func TestSystemStatus_Initialized(t *testing.T) {
	quickstartInitialized = true
	gw := &Gateway{}

	req := httptest.NewRequest("GET", "/api/v1/system/status", nil)
	w := httptest.NewRecorder()
	gw.handleSystemStatus(w, req)

	var status SystemStatus
	_ = json.Unmarshal(w.Body.Bytes(), &status)
	if !status.Initialized {
		t.Error("expected initialized")
	}
	if status.UserCount != 1 {
		t.Errorf("expected 1 user, got %d", status.UserCount)
	}
}
