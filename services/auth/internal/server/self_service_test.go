package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Self-service endpoints require JWT authentication.
// These tests verify routing and auth-gating (no token → 401).

func TestSelfServiceDevices_Unauthorized(t *testing.T) {
	h := New(nil) // authSvc nil — we won't reach it; auth check fails first
	h.registerRoutes()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/self-service/devices", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unauthenticated request, got %d", rr.Code)
	}
}

func TestSelfServiceSessions_Unauthorized(t *testing.T) {
	h := New(nil)
	h.registerRoutes()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/self-service/sessions", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestMFASelfRemove_Unauthorized(t *testing.T) {
	h := New(nil)
	h.registerRoutes()

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/self-service/mfa/factor-123", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unauthenticated MFA removal, got %d", rr.Code)
	}
}

func TestRegistrationConfig_GetNoTenantID(t *testing.T) {
	h := New(nil)
	h.registerRoutes()

	// GET without tenant_id should return 400
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/registration/config", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing tenant_id, got %d", rr.Code)
	}
}
