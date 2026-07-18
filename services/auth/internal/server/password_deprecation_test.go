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

func newTestHandlerWithDeprecationRepo() *Handler {
	return &Handler{
		passwordDeprecationRepo: repository.NewPasswordDeprecationRepository(nil),
	}
}

// Test 1: GET returns default config (level=off) when no DB.
func TestPasswordDeprecation_GetDefault(t *testing.T) {
	h := newTestHandlerWithDeprecationRepo()
	tid := uuid.New()

	req := httptest.NewRequest("GET", "/api/v1/auth/password-deprecation", nil)
	req = reqWithTenantContext(req, tid)
	w := httptest.NewRecorder()

	h.handlePasswordDeprecation(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var cfg repository.PasswordDeprecationConfig
	if err := json.Unmarshal(w.Body.Bytes(), &cfg); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if cfg.Level != repository.DeprecationOff {
		t.Errorf("expected level=off, got %s", cfg.Level)
	}
}

// Test 2: PUT with valid level returns 200.
func TestPasswordDeprecation_UpdateValid(t *testing.T) {
	h := newTestHandlerWithDeprecationRepo()
	tid := uuid.New()

	body := `{"level":"migration_required","grace_period_days":30}`
	req := httptest.NewRequest("PUT", "/api/v1/auth/password-deprecation", bytes.NewBufferString(body))
	req = reqWithTenantContext(req, tid)
	w := httptest.NewRecorder()

	h.handlePasswordDeprecation(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var cfg repository.PasswordDeprecationConfig
	if err := json.Unmarshal(w.Body.Bytes(), &cfg); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if cfg.Level != repository.DeprecationMigrationRequired {
		t.Errorf("expected level=migration_required, got %s", cfg.Level)
	}
	if cfg.GracePeriodDays != 30 {
		t.Errorf("expected grace_period_days=30, got %d", cfg.GracePeriodDays)
	}
}

// Test 3: PUT with invalid level returns 400.
func TestPasswordDeprecation_InvalidLevel(t *testing.T) {
	h := newTestHandlerWithDeprecationRepo()
	tid := uuid.New()

	body := `{"level":"bogus"}`
	req := httptest.NewRequest("PUT", "/api/v1/auth/password-deprecation", bytes.NewBufferString(body))
	req = reqWithTenantContext(req, tid)
	w := httptest.NewRecorder()

	h.handlePasswordDeprecation(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// Test 4: CheckPasswordLoginAllowed with nil pool returns allowed=true.
func TestPasswordDeprecation_CheckLoginAllowed_NilPool(t *testing.T) {
	repo := repository.NewPasswordDeprecationRepository(nil)
	allowed, mustEnroll, reason := repo.CheckPasswordLoginAllowed(reqWithTenantContext(
		httptest.NewRequest("GET", "/", nil), uuid.New()).Context(), uuid.New())

	if !allowed {
		t.Error("with nil pool, password login should be allowed (graceful degradation)")
	}
	if mustEnroll {
		t.Error("with nil pool, mustEnrollPasswordless should be false")
	}
	if reason != "" {
		t.Errorf("with nil pool, reason should be empty, got %s", reason)
	}
}

// Test 5: PUT with invalid enforcement_date returns 400.
func TestPasswordDeprecation_InvalidDate(t *testing.T) {
	h := newTestHandlerWithDeprecationRepo()
	tid := uuid.New()

	body := `{"level":"disabled","enforcement_date":"not-a-date"}`
	req := httptest.NewRequest("PUT", "/api/v1/auth/password-deprecation", bytes.NewBufferString(body))
	req = reqWithTenantContext(req, tid)
	w := httptest.NewRecorder()

	h.handlePasswordDeprecation(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad date, got %d: %s", w.Code, w.Body.String())
	}
}

// Test 6: Missing tenant context returns 401.
func TestPasswordDeprecation_NoTenant(t *testing.T) {
	h := newTestHandlerWithDeprecationRepo()

	req := httptest.NewRequest("GET", "/api/v1/auth/password-deprecation", nil)
	w := httptest.NewRecorder()

	h.handlePasswordDeprecation(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

// Test 7: All valid levels are accepted.
func TestPasswordDeprecation_AllLevelsAccepted(t *testing.T) {
	levels := []string{"off", "read_only", "migration_required", "disabled"}
	for _, level := range levels {
		h := newTestHandlerWithDeprecationRepo()
		tid := uuid.New()

		body := `{"level":"` + level + `"}`
		req := httptest.NewRequest("PUT", "/api/v1/auth/password-deprecation", bytes.NewBufferString(body))
		req = reqWithTenantContext(req, tid)
		w := httptest.NewRecorder()

		h.handlePasswordDeprecation(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("level %s: expected 200, got %d: %s", level, w.Code, w.Body.String())
		}
	}
}
