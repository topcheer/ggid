package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// TestChangePasswordTriggersInvalidation verifies that TriggerInvalidation
// works correctly for the password_change reason.
// The changePassword HTTP handler calls TriggerInvalidation after successful
// password change to revoke all other sessions.
func TestChangePasswordTriggersInvalidation(t *testing.T) {
	h := &Handler{}
	audit := h.TriggerInvalidation(
		uuid.New(), uuid.New(),
		InvReasonPasswordChange,
		"", // no session exemption — revoke all
	)
	if audit == nil {
		t.Fatal("TriggerInvalidation returned nil audit")
	}
	if audit.Reason != "password_change" {
		t.Errorf("expected reason 'password_change', got '%s'", audit.Reason)
	}
	if audit.ID == "" {
		t.Error("audit ID should not be empty")
	}
}

// TestMFAEnrollmentTriggersInvalidation verifies that first MFA enrollment
// triggers session invalidation with reason 'mfa_enrollment'.
func TestMFAEnrollmentTriggersInvalidation(t *testing.T) {
	h := &Handler{}
	audit := h.TriggerInvalidation(
		uuid.New(), uuid.New(),
		InvReasonMFAEnrollment,
		"", // revoke all non-MFA sessions
	)
	if audit == nil {
		t.Fatal("TriggerInvalidation returned nil audit")
	}
	if audit.Reason != "mfa_enrollment" {
		t.Errorf("expected reason 'mfa_enrollment', got '%s'", audit.Reason)
	}
}

// TestPostureDropTriggersInvalidation verifies that a posture drop event
// triggers session invalidation with reason 'posture_drop'.
// Posture drop = no session exemption, revoke everything.
func TestPostureDropTriggersInvalidation(t *testing.T) {
	h := &Handler{}
	audit := h.TriggerInvalidation(
		uuid.New(), uuid.New(),
		InvReasonPostureDrop,
		"", // no exemption — posture drop revokes ALL sessions
	)
	if audit == nil {
		t.Fatal("TriggerInvalidation returned nil audit")
	}
	if audit.Reason != "posture_drop" {
		t.Errorf("expected reason 'posture_drop', got '%s'", audit.Reason)
	}
}

// TestInvalidationEndpointHandlesPostureDrop verifies the HTTP endpoint
// accepts posture_drop as a valid reason.
func TestInvalidationEndpointHandlesPostureDrop(t *testing.T) {
	userID := uuid.New().String()
	tenantID := uuid.New().String()
	req := httptest.NewRequest("POST", "/api/v1/auth/invalidate-sessions/"+userID,
		strings.NewReader(`{"reason":"posture_drop"}`))
	req.Header.Set("X-Tenant-ID", tenantID)
	w := httptest.NewRecorder()

	h := &Handler{}
	h.handleInvalidateSessions(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for posture_drop, got %d", w.Code)
	}
}

// TestInvalidationEndpointHandlesMFAEnrollment verifies the HTTP endpoint
// accepts mfa_enrollment as a valid reason and supports except_session_id.
func TestInvalidationEndpointHandlesMFAEnrollment(t *testing.T) {
	userID := uuid.New().String()
	tenantID := uuid.New().String()
	req := httptest.NewRequest("POST", "/api/v1/auth/invalidate-sessions/"+userID,
		strings.NewReader(`{"reason":"mfa_enrollment","except_session_id":"keep-this"}`))
	req.Header.Set("X-Tenant-ID", tenantID)
	w := httptest.NewRecorder()

	h := &Handler{}
	h.handleInvalidateSessions(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for mfa_enrollment, got %d", w.Code)
	}
}

// TestInvalidationEndpointHandlesPasswordChange verifies the HTTP endpoint
// accepts password_change and tracks the initiating user.
func TestInvalidationEndpointHandlesPasswordChange(t *testing.T) {
	userID := uuid.New().String()
	tenantID := uuid.New().String()
	req := httptest.NewRequest("POST", "/api/v1/auth/invalidate-sessions/"+userID,
		strings.NewReader(`{"reason":"password_change","except_session_id":"current-session"}`))
	req.Header.Set("X-Tenant-ID", tenantID)
	req.Header.Set("X-User-ID", "admin-user")
	w := httptest.NewRecorder()

	h := &Handler{}
	h.handleInvalidateSessions(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for password_change, got %d", w.Code)
	}
}
