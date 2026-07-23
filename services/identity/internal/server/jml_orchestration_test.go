package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	ggidtenant "github.com/ggid/ggid/pkg/tenant"
	"github.com/google/uuid"
)

func TestBuildPhases_Joiner(t *testing.T) {
	phases := buildPhases("joiner")
	if len(phases) != 4 {
		t.Fatalf("expected 4 phases, got %d", len(phases))
	}
	if phases[0].Name != "create_account" {
		t.Errorf("expected first phase create_account, got %s", phases[0].Name)
	}
	if phases[3].Name != "mfa_enroll_guide" {
		t.Errorf("expected last phase mfa_enroll_guide, got %s", phases[3].Name)
	}
}

func TestBuildPhases_Mover(t *testing.T) {
	phases := buildPhases("mover")
	if len(phases) != 3 {
		t.Fatalf("expected 3 phases, got %d", len(phases))
	}
	if phases[0].Name != "recalc_permissions" {
		t.Errorf("expected first phase recalc_permissions, got %s", phases[0].Name)
	}
}

func TestBuildPhases_Leaver(t *testing.T) {
	phases := buildPhases("leaver")
	if len(phases) != 4 {
		t.Fatalf("expected 4 phases, got %d", len(phases))
	}
	if phases[0].Name != "disable_account" {
		t.Errorf("expected first phase disable_account, got %s", phases[0].Name)
	}
	if phases[3].Name != "archive_audit" {
		t.Errorf("expected last phase archive_audit, got %s", phases[3].Name)
	}
}

func TestTriggerToEventType(t *testing.T) {
	cases := map[string]string{
		"joiner": "user.created",
		"mover":  "user.role_changed",
		"leaver": "user.deleted",
		"":      "",
	}
	for trigger, expected := range cases {
		if got := triggerToEventType(trigger); got != expected {
			t.Errorf("triggerToEventType(%q) = %q, want %q", trigger, got, expected)
		}
	}
}

func TestJMLOrchestrate_InvalidTrigger(t *testing.T) {
	h := &HTTPHandler{}
	tenantID := uuid.New()
	tc := &ggidtenant.Context{TenantID: tenantID}

	body := `{"user_id":"` + uuid.New().String() + `","trigger":"invalid"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/identity/lifecycle/orchestrate", io.NopCloser(strings.NewReader(body)))
	req = req.WithContext(ggidtenant.WithContext(req.Context(), tc))

	rr := httptest.NewRecorder()
	h.handleJMLOrchestrate(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid trigger, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestJMLOrchestrate_InvalidUserID(t *testing.T) {
	h := &HTTPHandler{}
	tenantID := uuid.New()
	tc := &ggidtenant.Context{TenantID: tenantID}

	body := `{"user_id":"not-a-uuid","trigger":"joiner"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/identity/lifecycle/orchestrate", io.NopCloser(strings.NewReader(body)))
	req = req.WithContext(ggidtenant.WithContext(req.Context(), tc))

	rr := httptest.NewRecorder()
	h.handleJMLOrchestrate(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid user_id, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestJMLOrchestrate_Joiner_FallbackSuccess(t *testing.T) {
	// With nil engine and nil pool, phases execute via fallback path.
	h := &HTTPHandler{}
	tenantID := uuid.New()
	userID := uuid.New()
	tc := &ggidtenant.Context{TenantID: tenantID}

	body := `{"user_id":"` + userID.String() + `","trigger":"joiner","user_attrs":{"department":"engineering"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/identity/lifecycle/orchestrate", io.NopCloser(strings.NewReader(body)))
	req = req.WithContext(ggidtenant.WithContext(req.Context(), tc))

	rr := httptest.NewRecorder()
	h.handleJMLOrchestrate(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var run JMLOrchestration
	if err := json.Unmarshal(rr.Body.Bytes(), &run); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if run.Trigger != "joiner" {
		t.Errorf("expected trigger joiner, got %s", run.Trigger)
	}
	if run.Status != "completed" {
		t.Errorf("expected status completed, got %s", run.Status)
	}
	if len(run.Phases) != 4 {
		t.Errorf("expected 4 phases, got %d", len(run.Phases))
	}
	if run.UserID != userID.String() {
		t.Errorf("expected user_id %s, got %s", userID.String(), run.UserID)
	}
}

func TestJMLOrchestrate_Leaver_FallbackSuccess(t *testing.T) {
	h := &HTTPHandler{}
	tenantID := uuid.New()
	userID := uuid.New()
	tc := &ggidtenant.Context{TenantID: tenantID}

	body := `{"user_id":"` + userID.String() + `","trigger":"leaver"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/identity/lifecycle/orchestrate", io.NopCloser(strings.NewReader(body)))
	req = req.WithContext(ggidtenant.WithContext(req.Context(), tc))

	rr := httptest.NewRecorder()
	h.handleJMLOrchestrate(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var run JMLOrchestration
	if err := json.Unmarshal(rr.Body.Bytes(), &run); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if run.Status != "completed" {
		t.Errorf("expected completed, got %s", run.Status)
	}
	if len(run.Phases) != 4 {
		t.Fatalf("expected 4 phases, got %d", len(run.Phases))
	}
}

func TestJMLOrchestrate_GetStatus_NotFound(t *testing.T) {
	h := &HTTPHandler{}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/identity/lifecycle/orchestrate/nonexistent", nil)
	rr := httptest.NewRecorder()
	h.handleJMLOrchestrate(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", rr.Code, rr.Body.String())
	}
}
