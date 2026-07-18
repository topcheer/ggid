package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ggid/ggid/services/auth/internal/repository"
	"github.com/ggid/ggid/services/auth/internal/tap"
	"github.com/google/uuid"
)

func newTestHandlerWithTAP() *Handler {
	return &Handler{
		tapEngine:     tap.NewEngine(nil),
		tapPolicyRepo: repository.NewTAPPolicyRepository(nil),
	}
}

// Test 1: POST /tap — single issue returns 201 with code.
func TestTAP_Issue(t *testing.T) {
	h := newTestHandlerWithTAP()
	tid := uuid.New()

	body := `{"user_id":"user-123","reason":"onboarding"}`
	req := httptest.NewRequest("POST", "/api/v1/auth/tap", bytes.NewBufferString(body))
	req = reqWithTenantContext(req, tid)
	w := httptest.NewRecorder()

	h.handleTAP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp["code"] == nil || resp["code"] == "" {
		t.Error("expected code to be set")
	}
	if resp["tap_id"] == nil || resp["tap_id"] == "" {
		t.Error("expected tap_id to be set")
	}
}

// Test 2: POST /tap/batch — batch issue for multiple users.
func TestTAP_BatchIssue(t *testing.T) {
	h := newTestHandlerWithTAP()
	tid := uuid.New()

	body := `{"user_ids":["user-1","user-2","user-3"],"reason":"bulk onboarding"}`
	req := httptest.NewRequest("POST", "/api/v1/auth/tap/batch", bytes.NewBufferString(body))
	req = reqWithTenantContext(req, tid)
	w := httptest.NewRecorder()

	h.handleTAP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp tapBatchResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if len(resp.Issued) != 3 {
		t.Errorf("expected 3 issued, got %d", len(resp.Issued))
	}
	for _, r := range resp.Issued {
		if r.Code == "" {
			t.Error("issued TAP code should not be empty")
		}
	}
}

// Test 3: GET /tap/policy — returns default policy.
func TestTAP_GetPolicy(t *testing.T) {
	h := newTestHandlerWithTAP()
	tid := uuid.New()

	req := httptest.NewRequest("GET", "/api/v1/auth/tap/policy", nil)
	req = reqWithTenantContext(req, tid)
	w := httptest.NewRecorder()

	h.handleTAP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var policy repository.TAPPolicy
	if err := json.Unmarshal(w.Body.Bytes(), &policy); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if policy.MaxPerDay != 10 {
		t.Errorf("expected max_per_day=10 default, got %d", policy.MaxPerDay)
	}
}

// Test 4: PUT /tap/policy — update policy.
func TestTAP_UpdatePolicy(t *testing.T) {
	h := newTestHandlerWithTAP()
	tid := uuid.New()

	body := `{"allowed_groups":["admins","ops"],"max_per_day":20,"ttl_minutes":30}`
	req := httptest.NewRequest("PUT", "/api/v1/auth/tap/policy", bytes.NewBufferString(body))
	req = reqWithTenantContext(req, tid)
	w := httptest.NewRecorder()

	h.handleTAP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var policy repository.TAPPolicy
	if err := json.Unmarshal(w.Body.Bytes(), &policy); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if policy.MaxPerDay != 20 {
		t.Errorf("expected max_per_day=20, got %d", policy.MaxPerDay)
	}
	if len(policy.AllowedGroups) != 2 {
		t.Errorf("expected 2 allowed_groups, got %d", len(policy.AllowedGroups))
	}
}

// Test 5: POST /tap without user_id returns 400.
func TestTAP_IssueValidation(t *testing.T) {
	h := newTestHandlerWithTAP()
	tid := uuid.New()

	body := `{"reason":"test"}`
	req := httptest.NewRequest("POST", "/api/v1/auth/tap", bytes.NewBufferString(body))
	req = reqWithTenantContext(req, tid)
	w := httptest.NewRecorder()

	h.handleTAP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// Test 6: POST /tap/batch with empty array returns 400.
func TestTAP_BatchValidation(t *testing.T) {
	h := newTestHandlerWithTAP()
	tid := uuid.New()

	body := `{"user_ids":[]}`
	req := httptest.NewRequest("POST", "/api/v1/auth/tap/batch", bytes.NewBufferString(body))
	req = reqWithTenantContext(req, tid)
	w := httptest.NewRecorder()

	h.handleTAP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// Test 7: GET /tap/audit with user_id returns 200.
func TestTAP_AuditQuery(t *testing.T) {
	h := newTestHandlerWithTAP()
	tid := uuid.New()

	req := httptest.NewRequest("GET", "/api/v1/auth/tap/audit?user_id=user-123", nil)
	req = reqWithTenantContext(req, tid)
	w := httptest.NewRecorder()

	h.handleTAP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// Test 8: Missing tenant returns 401.
func TestTAP_NoTenant(t *testing.T) {
	h := newTestHandlerWithTAP()

	req := httptest.NewRequest("GET", "/api/v1/auth/tap/policy", nil)
	w := httptest.NewRecorder()

	h.handleTAP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

// Test 9: IsGroupAllowed with nil pool returns true (allow all).
func TestTAP_Policy_IsGroupAllowedNilPool(t *testing.T) {
	repo := repository.NewTAPPolicyRepository(nil)
	if !repo.IsGroupAllowed(reqWithTenantContext(httptest.NewRequest("GET", "/", nil), uuid.New()).Context(), uuid.New(), "any-group") {
		t.Error("with nil pool, all groups should be allowed")
	}
}
