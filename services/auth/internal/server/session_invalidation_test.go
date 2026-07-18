package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInvalidateSessions_InvalidReason(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/v1/auth/invalidate-sessions/550e8400-e29b-41d4-a716-446655440000",
		strings.NewReader(`{"reason":"bad_reason"}`))
	req.Header.Set("X-Tenant-ID", "550e8400-e29b-41d4-a716-446655440001")
	w := httptest.NewRecorder()

	h := &Handler{}
	h.handleInvalidateSessions(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid reason, got %d", w.Code)
	}
}

func TestInvalidateSessions_MissingUserID(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/v1/auth/invalidate-sessions/",
		strings.NewReader(`{"reason":"password_change"}`))
	req.Header.Set("X-Tenant-ID", "550e8400-e29b-41d4-a716-446655440001")
	w := httptest.NewRecorder()

	h := &Handler{}
	h.handleInvalidateSessions(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing user_id, got %d", w.Code)
	}
}

func TestInvalidateSessions_NoRevocationMgr(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/v1/auth/invalidate-sessions/550e8400-e29b-41d4-a716-446655440000",
		strings.NewReader(`{"reason":"password_change"}`))
	req.Header.Set("X-Tenant-ID", "550e8400-e29b-41d4-a716-446655440001")
	w := httptest.NewRecorder()

	h := &Handler{}
	h.handleInvalidateSessions(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "queued") {
		t.Errorf("expected 'queued' status when revocationMgr is nil, got: %s", body)
	}
}

func TestInvalidateSessions_WrongMethod(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/auth/invalidate-sessions/550e8400-e29b-41d4-a716-446655440000", nil)
	w := httptest.NewRecorder()

	h := &Handler{}
	h.handleInvalidateSessions(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestInvalidateSessions_BadUserID(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/v1/auth/invalidate-sessions/not-a-uuid",
		strings.NewReader(`{"reason":"password_change"}`))
	req.Header.Set("X-Tenant-ID", "550e8400-e29b-41d4-a716-446655440001")
	w := httptest.NewRecorder()

	h := &Handler{}
	h.handleInvalidateSessions(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad user_id, got %d", w.Code)
	}
}

func TestInvalidateSessions_BadTenantID(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/v1/auth/invalidate-sessions/550e8400-e29b-41d4-a716-446655440000",
		strings.NewReader(`{"reason":"password_change"}`))
	req.Header.Set("X-Tenant-ID", "not-a-uuid")
	w := httptest.NewRecorder()

	h := &Handler{}
	h.handleInvalidateSessions(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad tenant ID, got %d", w.Code)
	}
}

func TestInvalidateSessions_DefaultReason(t *testing.T) {
	// When reason is omitted, should default to "admin_action"
	req := httptest.NewRequest("POST", "/api/v1/auth/invalidate-sessions/550e8400-e29b-41d4-a716-446655440000",
		strings.NewReader(`{}`))
	req.Header.Set("X-Tenant-ID", "550e8400-e29b-41d4-a716-446655440001")
	w := httptest.NewRecorder()

	h := &Handler{}
	h.handleInvalidateSessions(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}
