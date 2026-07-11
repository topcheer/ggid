package server

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// POST /api/v1/users/{id}/reactivate — restore deactivated/deprovisioned user.
func (h *HTTPHandler) reactivateUser(ctx context.Context, userID uuid.UUID, w http.ResponseWriter, r *http.Request) {
	user, err := h.svc.GetUser(ctx, userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	activated, err := h.svc.ActivateUser(ctx, userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	restored := []string{"account_status"}
	if user.EmailVerified {
		restored = append(restored, "email_verified")
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":            "reactivated",
		"user_id":           userID.String(),
		"username":          activated.Username,
		"reactivated_at":    time.Now().UTC().Format(time.RFC3339),
		"restored_features": restored,
		"welcome_email":     "queued",
		"default_role":      "user",
	})
}
