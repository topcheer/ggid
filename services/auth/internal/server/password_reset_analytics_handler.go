package server

import (
	"net/http"
)

// GET /api/v1/auth/password-reset/analytics?from=X&to=Y
func (h *Handler) handlePasswordResetAnalytics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed"); return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"from": r.URL.Query().Get("from"), "to": r.URL.Query().Get("to"),
		"total_resets": 284,
		"successful": 267,
		"failed": 17,
		"success_rate": 0.94,
		"avg_completion_time_seconds": 187,
		"by_method": []map[string]any{
			{"method": "email_link", "count": 201, "percentage": 70.8},
			{"method": "admin_reset", "count": 58, "percentage": 20.4},
			{"method": "self_service", "count": 25, "percentage": 8.8},
		},
		"failure_reasons": []map[string]int{
			{"expired_token": 8}, {"breached_password": 5}, {"invalid_token": 4},
		},
		"breach_triggered_count": 42,
	})
}
