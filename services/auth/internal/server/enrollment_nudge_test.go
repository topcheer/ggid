package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ggid/ggid/services/auth/internal/repository"
	"github.com/google/uuid"
)

func newTestHandlerWithNudgeRepo() *Handler {
	return &Handler{
		enrollmentNudgeRepo:    repository.NewEnrollmentNudgeRepository(nil),
		passwordDeprecationRepo: repository.NewPasswordDeprecationRepository(nil),
	}
}

// Test 1: GET nudge check returns enrollment_nudge=false when deprecation is off.
func TestEnrollmentNudge_NotRequired(t *testing.T) {
	h := newTestHandlerWithNudgeRepo()
	tid := uuid.New()
	uid := uuid.New()

	req := httptest.NewRequest("GET", "/api/v1/auth/enrollment/nudge/"+uid.String(), nil)
	req = reqWithTenantContext(req, tid)
	w := httptest.NewRecorder()

	h.handleEnrollmentNudge(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp["enrollment_nudge"] != false {
		t.Errorf("expected enrollment_nudge=false when deprecation=off, got %v", resp["enrollment_nudge"])
	}
}

// Test 2: POST dismiss returns 200 with dismissed_until.
func TestEnrollmentNudge_Dismiss(t *testing.T) {
	h := newTestHandlerWithNudgeRepo()
	tid := uuid.New()
	uid := uuid.New()

	body := `{"user_id":"` + uid.String() + `","nudge_type":"passkey","days":7}`
	req := httptest.NewRequest("POST", "/api/v1/auth/enrollment/dismiss", bytes.NewBufferString(body))
	req = reqWithTenantContext(req, tid)
	w := httptest.NewRecorder()

	h.handleEnrollmentDismiss(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp["status"] != "dismissed" {
		t.Errorf("expected status=dismissed, got %v", resp["status"])
	}
	if resp["dismissed_until"] == nil {
		t.Error("expected dismissed_until to be set")
	}
}

// Test 3: POST dismiss with invalid user_id returns 400.
func TestEnrollmentNudge_DismissInvalidUser(t *testing.T) {
	h := newTestHandlerWithNudgeRepo()
	tid := uuid.New()

	body := `{"user_id":"not-a-uuid"}`
	req := httptest.NewRequest("POST", "/api/v1/auth/enrollment/dismiss", bytes.NewBufferString(body))
	req = reqWithTenantContext(req, tid)
	w := httptest.NewRecorder()

	h.handleEnrollmentDismiss(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// Test 4: GET with invalid user_id returns 400.
func TestEnrollmentNudge_InvalidUserID(t *testing.T) {
	h := newTestHandlerWithNudgeRepo()
	tid := uuid.New()

	req := httptest.NewRequest("GET", "/api/v1/auth/enrollment/nudge/not-a-uuid", nil)
	req = reqWithTenantContext(req, tid)
	w := httptest.NewRecorder()

	h.handleEnrollmentNudge(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// Test 5: GET without tenant returns 401.
func TestEnrollmentNudge_NoTenant(t *testing.T) {
	h := newTestHandlerWithNudgeRepo()
	uid := uuid.New()

	req := httptest.NewRequest("GET", "/api/v1/auth/enrollment/nudge/"+uid.String(), nil)
	w := httptest.NewRecorder()

	h.handleEnrollmentNudge(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

// Test 6: IsDismissed returns false with nil pool (no DB = not dismissed).
func TestEnrollmentNudge_IsDismissedNilPool(t *testing.T) {
	repo := repository.NewEnrollmentNudgeRepository(nil)
	dismissed, err := repo.IsDismissed(
		reqWithTenantContext(httptest.NewRequest("GET", "/", nil), uuid.New()).Context(),
		uuid.New(), uuid.New(), "passkey",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dismissed {
		t.Error("with nil pool, IsDismissed should return false")
	}
}

// Test 7: Dismiss defaults to 7 days when days=0.
func TestEnrollmentNudge_DismissDefaultDays(t *testing.T) {
	h := newTestHandlerWithNudgeRepo()
	tid := uuid.New()
	uid := uuid.New()

	body := `{"user_id":"` + uid.String() + `"}`
	req := httptest.NewRequest("POST", "/api/v1/auth/enrollment/dismiss", bytes.NewBufferString(body))
	req = reqWithTenantContext(req, tid)
	w := httptest.NewRecorder()

	h.handleEnrollmentDismiss(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp["days"] != float64(7) {
		t.Errorf("expected default days=7, got %v", resp["days"])
	}
}
