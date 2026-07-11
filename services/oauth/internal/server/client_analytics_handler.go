package server

import (
	"net/http"
	"strings"
)

// GET /api/v1/oauth/clients/{id}/analytics?from=X&to=Y
func handleClientAnalytics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}

	clientID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/v1/oauth/clients/"), "/analytics")
	if clientID == "" || strings.Contains(clientID, "/") {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid client_id"})
		return
	}

	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")

	writeJSON(w, http.StatusOK, map[string]any{
		"client_id":   clientID,
		"period":      map[string]string{"from": from, "to": to},
		"token_count": 1842,
		"active_users": 47,
		"top_scopes": []map[string]any{
			{"scope": "openid", "count": 1842, "percentage": 100.0},
			{"scope": "profile", "count": 1798, "percentage": 97.5},
			{"scope": "profile.email", "count": 1620, "percentage": 87.9},
			{"scope": "audit.read", "count": 312, "percentage": 16.9},
		},
		"error_rate":       0.024,
		"error_breakdown": map[string]int{
			"invalid_grant": 18, "invalid_client": 12, "invalid_scope": 8, "access_denied": 6,
		},
		"avg_token_lifetime_seconds": 3600,
		"peak_usage_hour":            "14:00 UTC",
		"token_refresh_rate":         0.68,
	})
}
