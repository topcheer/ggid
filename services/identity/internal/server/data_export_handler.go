package server

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// GET /api/v1/users/{id}/data-export — GDPR user data export.
func (h *HTTPHandler) dataExport(ctx context.Context, userID uuid.UUID, w http.ResponseWriter, r *http.Request) {
	user, err := h.svc.GetUser(ctx, userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	// Compile all user data categories
	export := map[string]any{
		"export_metadata": map[string]any{
			"user_id":    userID.String(),
			"exported_at": time.Now().UTC().Format(time.RFC3339),
			"format":     "json",
			"version":    "1.0",
			"legal_basis": "GDPR Article 15 - Right of access",
		},
		"profile": map[string]any{
			"id":             user.ID,
			"username":       user.Username,
			"email":          user.Email,
			"email_verified": user.EmailVerified,
			"phone":          user.Phone,
			"display_name":   user.DisplayName,
			"status":         user.Status,
			"locale":         user.Locale,
			"timezone":       user.Timezone,
			"avatar_url":     user.AvatarURL,
			"created_at":     user.CreatedAt,
			"updated_at":     user.UpdatedAt,
		},
		"sessions":          []map[string]any{}, // would query session store
		"audit_events":      []map[string]any{}, // would query audit service
		"consents":          []map[string]any{}, // would query consent store
		"linked_accounts":   []map[string]any{}, // would query identity link store
		"mfa_devices":       []map[string]any{}, // would query MFA store
		"oauth_authorizations": []map[string]any{}, // would query oauth consent
	}

	writeJSON(w, http.StatusOK, export)
}
