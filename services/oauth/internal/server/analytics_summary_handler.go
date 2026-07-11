package server

import (
	"net/http"
)

// GET /api/v1/oauth/analytics/summary?from=X&to=Y
func handleAnalyticsSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"}); return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"from": r.URL.Query().Get("from"), "to": r.URL.Query().Get("to"),
		"total_clients": 24, "active_clients": 18, "inactive_clients": 6,
		"tokens_issued": 48392,
		"unique_users": 347,
		"avg_tokens_per_client": 2016,
		"top_clients": []map[string]any{
			{"client_id": "c-001", "name": "Dashboard App", "tokens": 18420, "active_users": 142, "percentage": 38.1},
			{"client_id": "c-002", "name": "CLI Tool", "tokens": 12750, "active_users": 89, "percentage": 26.3},
			{"client_id": "c-003", "name": "API Gateway", "tokens": 8900, "active_users": 67, "percentage": 18.4},
		},
		"top_scopes": []map[string]any{
			{"scope": "openid", "count": 48392}, {"scope": "profile", "count": 45120}, {"scope": "profile.email", "count": 39800},
		},
		"error_rate": 0.018,
	})
}
