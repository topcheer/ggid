package server

import (
	"net/http"
)

// GET /api/v1/auth/credential-exposure?user_id=X
func (h *Handler) handleCredentialExposure(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet { writeError(w, http.StatusMethodNotAllowed, "method not allowed"); return }
	userID := r.URL.Query().Get("user_id")
	if userID == "" { writeError(w, http.StatusBadRequest, "user_id required"); return }
	writeJSON(w, http.StatusOK, map[string]any{
		"user_id": userID,
		"active_tokens": 3, "active_sessions": 2, "linked_providers": []string{"google", "github"}, "api_keys": 1,
		"exposure_score": 42,
		"exposure_level": "moderate",
		"recommendations": []string{"Revoke 1 unused API key", "Review 1 stale session (last active >7d)", "Remove github provider if not used recently"},
		"detail": []map[string]any{
			{"type": "access_token", "id": "tok-001", "created": "2026-07-01", "last_used": "2026-07-12", "scopes": "openid profile"},
			{"type": "session", "id": "sess-001", "device": "Chrome/macOS", "ip": "192.168.1.50", "last_active": "2026-07-12T08:00:00Z"},
			{"type": "api_key", "id": "key-old", "created": "2025-01-01", "last_used": "never", "status": "stale"},
		},
	})
}
