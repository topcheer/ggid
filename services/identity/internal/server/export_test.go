package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExportUsers_InvalidFormat(t *testing.T) {
	h := &HTTPHandler{}
	h.mux = http.NewServeMux()
	h.mux.HandleFunc("/api/v1/users/export", h.handleExportUsers)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/export?format=xml", nil)
	req.Header.Set("X-Tenant-ID", "550e8400-e29b-41d4-a716-446655440000")
	rr := httptest.NewRecorder()
	h.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestExportUsers_MethodNotAllowed(t *testing.T) {
	h := &HTTPHandler{}
	h.mux = http.NewServeMux()
	h.mux.HandleFunc("/api/v1/users/export", h.handleExportUsers)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/export", nil)
	rr := httptest.NewRecorder()
	h.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}

func TestExportUsers_MissingTenant(t *testing.T) {
	h := &HTTPHandler{}
	h.mux = http.NewServeMux()
	h.mux.HandleFunc("/api/v1/users/export", h.handleExportUsers)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/export?format=csv", nil)
	rr := httptest.NewRecorder()
	h.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}
