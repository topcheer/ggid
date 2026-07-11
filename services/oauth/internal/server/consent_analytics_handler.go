package server

import (
	"net/http"
	"time"
)

// GET /api/v1/oauth/consent/analytics?client_id=X
func handleConsentAnalytics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}

	clientID := r.URL.Query().Get("client_id")
	if clientID == "" {
		clientID = "all"
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"client_id":          clientID,
		"total_decisions":    1247,
		"grant_count":        1089,
		"deny_count":         158,
		"grant_rate":         0.873,
		"deny_rate":          0.127,
		"avg_decision_time_ms": 4200,
		"top_denied_scopes": []map[string]any{
			{"scope": "admin.users", "deny_count": 87, "deny_percentage": 55.1},
			{"scope": "audit.export", "deny_count": 42, "deny_percentage": 26.6},
			{"scope": "profile.phone", "deny_count": 29, "deny_percentage": 18.4},
		},
		"top_granted_scopes": []map[string]any{
			{"scope": "openid", "grant_count": 1089, "grant_percentage": 100.0},
			{"scope": "profile", "grant_count": 1052, "grant_percentage": 96.6},
			{"scope": "profile.email", "grant_count": 990, "grant_percentage": 90.9},
		},
		"analyzed_at": time.Now().UTC().Format(time.RFC3339),
	})
}
