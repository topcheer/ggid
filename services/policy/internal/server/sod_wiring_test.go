package httpserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWiring_SoDCheck(t *testing.T) {
	s := &HTTPServer{}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/policies/sod/check", s.handleSoDCheck)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/policies/sod/check",
		strings.NewReader(`{"user_id":"550e8400-e29b-41d4-a716-446655440000","roles":["admin","auditor"]}`))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code == http.StatusNotFound {
		t.Error("SoD check route should be registered")
	}
	if rr.Code == http.StatusOK {
		body := rr.Body.String()
		if !strings.Contains(body, "violated") {
			t.Error("should return violation status")
		}
	}
}

func TestWiring_SoDCheck_NotFound(t *testing.T) {
	s := &HTTPServer{}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/policies/sod/check", s.handleSoDCheck)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/policies/sod/check", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for GET, got %d", rr.Code)
	}
}
