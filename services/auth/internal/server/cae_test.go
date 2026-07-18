package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ggid/ggid/services/auth/internal/repository"
	"github.com/google/uuid"
)

func newTestHandlerWithCAE() *Handler {
	return &Handler{
		caeRepo: repository.NewCAERepository(nil),
		capRepo: repository.NewConditionalAccessRepository(nil),
	}
}

// Test 1: GET /cae/status returns 200 with stats.
func TestCAE_Status(t *testing.T) {
	h := newTestHandlerWithCAE()
	tid := uuid.New()

	req := httptest.NewRequest("GET", "/api/v1/auth/cae/status", nil)
	req = reqWithTenantContext(req, tid)
	w := httptest.NewRecorder()

	h.handleCAE(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// Test 2: POST /cae/run triggers a sweep and returns 200.
func TestCAE_Run(t *testing.T) {
	h := newTestHandlerWithCAE()
	tid := uuid.New()

	req := httptest.NewRequest("POST", "/api/v1/auth/cae/run", nil)
	req = reqWithTenantContext(req, tid)
	w := httptest.NewRecorder()

	h.handleCAE(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// Test 3: GET /cae/log returns 200 with array.
func TestCAE_Log(t *testing.T) {
	h := newTestHandlerWithCAE()
	tid := uuid.New()

	req := httptest.NewRequest("GET", "/api/v1/auth/cae/log", nil)
	req = reqWithTenantContext(req, tid)
	w := httptest.NewRecorder()

	h.handleCAE(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// Test 4: Missing tenant returns 401.
func TestCAE_NoTenant(t *testing.T) {
	h := newTestHandlerWithCAE()

	req := httptest.NewRequest("GET", "/api/v1/auth/cae/status", nil)
	w := httptest.NewRecorder()

	h.handleCAE(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

// Test 5: EvaluateSessionForCAE with nil pool returns "allow".
func TestCAE_EvaluateSession_NilPool(t *testing.T) {
	h := newTestHandlerWithCAE()

	action := h.EvaluateSessionForCAE(uuid.New(), "sess-1", "user-1", "1.2.3.4", 50)
	if action != "allow" {
		t.Errorf("expected allow with nil pool, got %s", action)
	}
}

// Test 6: CAE status with nil repos returns "not configured".
func TestCAE_StatusNilRepo(t *testing.T) {
	h := &Handler{} // no repos
	tid := uuid.New()

	req := httptest.NewRequest("GET", "/api/v1/auth/cae/status", nil)
	req = reqWithTenantContext(req, tid)
	w := httptest.NewRecorder()

	h.handleCAE(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

// Test 7: CAE run with nil repos returns "not configured".
func TestCAE_RunNilRepo(t *testing.T) {
	h := &Handler{}
	tid := uuid.New()

	req := httptest.NewRequest("POST", "/api/v1/auth/cae/run", nil)
	req = reqWithTenantContext(req, tid)
	w := httptest.NewRecorder()

	h.handleCAE(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

// Test 8: Wrong method returns 405.
func TestCAE_WrongMethod(t *testing.T) {
	h := newTestHandlerWithCAE()

	req := httptest.NewRequest("DELETE", "/api/v1/auth/cae/status", nil)
	w := httptest.NewRecorder()

	h.handleCAE(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}
