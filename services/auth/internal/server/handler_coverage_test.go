package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ggid/ggid/pkg/i18n"
)

// newTestHandler creates a Handler with required fields initialized for testing.
func newTestHandler() *Handler {
	return &Handler{
		translator: i18n.NewTranslator("en"),
	}
}

func TestLogin_EmptyBody(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	rr := httptest.NewRecorder()
	h.login(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty body, got %d", rr.Code)
	}
}

func TestLogin_InvalidJSON(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader("not json"))
	rr := httptest.NewRecorder()
	h.login(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid JSON, got %d", rr.Code)
	}
}

func TestLogin_GetMethod(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/login", nil)
	rr := httptest.NewRecorder()
	h.login(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for GET, got %d", rr.Code)
	}
}

func TestRegister_EmptyBody(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", nil)
	req.Header.Set("X-Tenant-ID", "00000000-0000-0000-0000-000000000001")
	rr := httptest.NewRecorder()
	h.register(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty body, got %d", rr.Code)
	}
}

func TestRegister_GetMethod(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/register", nil)
	rr := httptest.NewRecorder()
	h.register(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for GET, got %d", rr.Code)
	}
}

func TestLogout_EmptyBody(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	rr := httptest.NewRecorder()
	h.logout(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty body, got %d", rr.Code)
	}
}

func TestChangePassword_GetMethod(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/change-password", nil)
	rr := httptest.NewRecorder()
	h.changePassword(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for GET, got %d", rr.Code)
	}
}

func TestResetPassword_EmptyBody(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/reset-password", nil)
	rr := httptest.NewRecorder()
	h.resetPassword(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty body, got %d", rr.Code)
	}
}
