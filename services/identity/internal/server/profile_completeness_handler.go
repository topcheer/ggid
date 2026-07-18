package server

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

// GET /api/v1/users/{id}/profile-completeness
func (h *HTTPHandler) handleProfileCompleteness(ctx context.Context, userID uuid.UUID, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet { writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed"); return }
	user, _ := h.svc.GetUser(ctx, userID)
	total := 8; filled := 0; var missing []string
	check := func(val any, name string) {
		if val != nil && val != "" { filled++ } else { missing = append(missing, name) }
	}
	if user != nil {
		check(user.Username, "username"); check(user.Email, "email")
		check(user.Phone, "phone"); check(user.DisplayName, "display_name")
		check(user.Locale, "locale"); check(user.Timezone, "timezone")
		check(user.AvatarURL, "avatar_url")
		if user.EmailVerified { filled++ } else { missing = append(missing, "email_verified") }
	} else { missing = []string{"username", "email", "phone", "display_name", "locale", "timezone", "avatar_url", "email_verified"} }
	pct := float64(filled) / float64(total) * 100
	var warnings []string
	if pct < 50 { warnings = append(warnings, "profile incomplete — required for compliance") }
	if !user.EmailVerified { warnings = append(warnings, "email not verified — access restricted") }
	writeJSON(w, http.StatusOK, map[string]any{"user_id": userID.String(), "completion_pct": int(pct), "filled_fields": filled, "total_fields": total, "missing_fields": missing, "warnings": warnings})
}
