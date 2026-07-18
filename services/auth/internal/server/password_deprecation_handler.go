package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/ggid/ggid/pkg/errors"
	"github.com/ggid/ggid/pkg/tenant"
	"github.com/ggid/ggid/services/auth/internal/repository"
	"github.com/google/uuid"
)

// passwordDeprecationRequest is the DTO for PUT operations.
type passwordDeprecationRequest struct {
	Level            string `json:"level"`
	EnforcementDate  string `json:"enforcement_date,omitempty"`
	GracePeriodDays  int    `json:"grace_period_days"`
}

// SetPasswordDeprecationRepo injects the DB-backed repository.
func (h *Handler) SetPasswordDeprecationRepo(repo *repository.PasswordDeprecationRepository) {
	h.passwordDeprecationRepo = repo
}

// handlePasswordDeprecation handles GET/PUT for /api/v1/auth/password-deprecation.
func (h *Handler) handlePasswordDeprecation(w http.ResponseWriter, r *http.Request) {
	tc, err := tenant.FromContext(r.Context())
	if err != nil {
		errors.WriteSimpleAPIError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getPasswordDeprecation(w, r, tc.TenantID)
	case http.MethodPut:
		h.updatePasswordDeprecation(w, r, tc.TenantID)
	default:
		errors.WriteSimpleAPIError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
	}
}

func (h *Handler) getPasswordDeprecation(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	if h.passwordDeprecationRepo == nil {
		writeJSON(w, http.StatusOK, &repository.PasswordDeprecationConfig{
			TenantID:        tenantID,
			Level:           repository.DeprecationOff,
			GracePeriodDays: 30,
		})
		return
	}
	cfg, err := h.passwordDeprecationRepo.Get(r.Context(), tenantID)
	if err != nil {
		errors.WriteSimpleAPIError(w, http.StatusInternalServerError, "INTERNAL", "failed to get config")
		return
	}
	writeJSON(w, http.StatusOK, cfg)
}

func (h *Handler) updatePasswordDeprecation(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	var req passwordDeprecationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.WriteSimpleAPIError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	// Validate level.
	if !repository.ValidDeprecationLevels[req.Level] {
		errors.WriteSimpleAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR",
			"invalid level; must be one of: off, read_only, migration_required, disabled")
		return
	}

	if req.GracePeriodDays < 0 {
		req.GracePeriodDays = 30
	}

	cfg := &repository.PasswordDeprecationConfig{
		TenantID:        tenantID,
		Level:           req.Level,
		GracePeriodDays: req.GracePeriodDays,
	}

	// Parse optional enforcement date.
	if req.EnforcementDate != "" {
		t, err := time.Parse(time.RFC3339, req.EnforcementDate)
		if err != nil {
			errors.WriteSimpleAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid enforcement_date format; use RFC3339")
			return
		}
		cfg.EnforcementDate = &t
	}

	if err := h.passwordDeprecationRepo.Upsert(r.Context(), cfg); err != nil {
		errors.WriteSimpleAPIError(w, http.StatusInternalServerError, "INTERNAL", "failed to update config")
		return
	}

	// Re-read to get the updated_at timestamp. If the DB round-trip didn't
	// persist (nil pool), use the request config directly.
	saved, _ := h.passwordDeprecationRepo.Get(r.Context(), tenantID)
	if saved != nil && saved.Level == cfg.Level {
		cfg = saved
	}
	writeJSON(w, http.StatusOK, cfg)
}
