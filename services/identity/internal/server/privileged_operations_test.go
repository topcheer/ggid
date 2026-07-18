package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPrivilegedOps_NotConfigured(t *testing.T) {
	h := &HTTPHandler{}
	req := httptest.NewRequest("GET", "/api/v1/identity/privileged-operations", nil)
	w := httptest.NewRecorder()
	h.handlePrivilegedOperations(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

func TestPrivilegedOps_WrongMethod(t *testing.T) {
	h := &HTTPHandler{}
	req := httptest.NewRequest("DELETE", "/api/v1/identity/privileged-operations", nil)
	w := httptest.NewRecorder()
	h.handlePrivilegedOperations(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestNilIfEmpty(t *testing.T) {
	if nilIfEmpty("") != nil {
		t.Error("empty string should return nil")
	}
	if nilIfEmpty("x") != "x" {
		t.Error("non-empty should return itself")
	}
}

func TestIntToStr(t *testing.T) {
	if intToStr(0) != "0" {
		t.Error("0")
	}
	if intToStr(100) != "100" {
		t.Error("100")
	}
	if intToStr(42) != "42" {
		t.Error("42")
	}
}
