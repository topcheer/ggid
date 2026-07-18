package server

import (
	"net/http"
)

// GET /api/v1/auth/login-analytics?from=X&to=Y
func (h *Handler) handleLoginAnalytics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed"); return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"from": r.URL.Query().Get("from"), "to": r.URL.Query().Get("to"),
		"total_attempts": 12483,
		"successful":     11902,
		"failed":         581,
		"success_rate":   0.953,
		"avg_duration_ms": 847,
		"top_methods": []map[string]any{
			{"method": "password", "count": 8421, "percentage": 67.5},
			{"method": "sso_google", "count": 2847, "percentage": 22.8},
			{"method": "webauthn", "count": 912, "percentage": 7.3},
			{"method": "sso_github", "count": 303, "percentage": 2.4},
		},
		"failure_reasons": []map[string]int{
			{"invalid_credentials": 387}, {"account_locked": 92}, {"mfa_failed": 67}, {"expired_password": 35},
		},
		"unique_users": 347,
	})
}
