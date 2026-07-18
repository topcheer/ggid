package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/ggid/ggid/pkg/errors"
	"github.com/ggid/ggid/pkg/tenant"
	"github.com/ggid/ggid/services/auth/internal/repository"
	"github.com/google/uuid"
)

// enrollmentDismissRequest is the DTO for the dismiss endpoint.
type enrollmentDismissRequest struct {
	UserID    string `json:"user_id"`
	NudgeType string `json:"nudge_type"`
	Days      int    `json:"days"`
}

// SetEnrollmentNudgeRepo injects the DB-backed repository.
func (h *Handler) SetEnrollmentNudgeRepo(repo *repository.EnrollmentNudgeRepository) {
	h.enrollmentNudgeRepo = repo
}

// handleEnrollmentNudge handles GET /api/v1/auth/enrollment/nudge/:user_id.
func (h *Handler) handleEnrollmentNudge(w http.ResponseWriter, r *http.Request) {
	tc, err := tenant.FromContext(r.Context())
	if err != nil {
		errors.WriteSimpleAPIError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context required")
		return
	}

	if r.Method != http.MethodGet {
		errors.WriteSimpleAPIError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	// Extract user_id from path.
	path := r.URL.Path
	idx := strings.LastIndex(path, "/")
	if idx < 0 {
		errors.WriteSimpleAPIError(w, http.StatusBadRequest, "INVALID_PATH", "user_id required")
		return
	}
	userIDStr := path[idx+1:]
	if strings.HasPrefix(userIDStr, "nudge") {
		errors.WriteSimpleAPIError(w, http.StatusBadRequest, "INVALID_PATH", "user_id required")
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		errors.WriteSimpleAPIError(w, http.StatusBadRequest, "INVALID_ID", "invalid user_id")
		return
	}

	nudgeType := r.URL.Query().Get("type")
	if nudgeType == "" {
		nudgeType = "passkey"
	}

	// Check if password deprecation requires enrollment.
	var deprecationLevel string
	if h.passwordDeprecationRepo != nil {
		cfg, _ := h.passwordDeprecationRepo.Get(r.Context(), tc.TenantID)
		if cfg != nil {
			deprecationLevel = cfg.Level
		}
	}

	// Check if nudge is dismissed.
	isDismissed := false
	if h.enrollmentNudgeRepo != nil {
		isDismissed, _ = h.enrollmentNudgeRepo.IsDismissed(r.Context(), tc.TenantID, userID, nudgeType)
	}

	// Determine if nudge should be shown.
	// Conditions: deprecation policy requires migration AND nudge not dismissed.
	shouldNudge := false
	if deprecationLevel == repository.DeprecationMigrationRequired || deprecationLevel == repository.DeprecationDisabled {
		shouldNudge = !isDismissed
	}

	// Record shown if nudging.
	if shouldNudge && h.enrollmentNudgeRepo != nil {
		_ = h.enrollmentNudgeRepo.RecordShown(r.Context(), tc.TenantID, userID, nudgeType)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"enrollment_nudge":     shouldNudge,
		"nudge_type":           nudgeType,
		"deprecation_level":    deprecationLevel,
		"dismissed":            isDismissed,
		"message":              "Please enroll a Passkey to improve your account security.",
	})
}

// handleEnrollmentDismiss handles POST /api/v1/auth/enrollment/dismiss.
func (h *Handler) handleEnrollmentDismiss(w http.ResponseWriter, r *http.Request) {
	tc, err := tenant.FromContext(r.Context())
	if err != nil {
		errors.WriteSimpleAPIError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context required")
		return
	}

	if r.Method != http.MethodPost {
		errors.WriteSimpleAPIError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	var req enrollmentDismissRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.WriteSimpleAPIError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		errors.WriteSimpleAPIError(w, http.StatusBadRequest, "INVALID_ID", "invalid user_id")
		return
	}

	nudgeType := req.NudgeType
	if nudgeType == "" {
		nudgeType = "passkey"
	}

	days := req.Days
	if days <= 0 {
		days = 7
	}

	// Ensure the nudge row exists first.
	if h.enrollmentNudgeRepo != nil {
		_, _ = h.enrollmentNudgeRepo.GetOrCreate(r.Context(), tc.TenantID, userID, nudgeType)
		if err := h.enrollmentNudgeRepo.Dismiss(r.Context(), tc.TenantID, userID, nudgeType, days); err != nil {
			errors.WriteSimpleAPIError(w, http.StatusInternalServerError, "INTERNAL", "failed to dismiss")
			return
		}
	}

	dismissedUntil := time.Now().AddDate(0, 0, days)
	writeJSON(w, http.StatusOK, map[string]any{
		"status":          "dismissed",
		"dismissed_until": dismissedUntil.Format(time.RFC3339),
		"days":            days,
	})
}
