package server

import (
	"net/http"
)

// GET /api/v1/users/segments
func (h *HTTPHandler) handleUserSegments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"by_role": []map[string]any{
			{"role": "admin", "count": 5, "percentage": 3.3},
			{"role": "manager", "count": 12, "percentage": 8.0},
			{"role": "developer", "count": 68, "percentage": 45.3},
			{"role": "viewer", "count": 45, "percentage": 30.0},
			{"role": "service-account", "count": 20, "percentage": 13.3},
		},
		"by_activity": map[string]any{
			"active":   map[string]int{"count": 112, "last_24h": 87, "last_7d": 112},
			"dormant":  map[string]int{"count": 28, "inactive_days_min": 7, "inactive_days_max": 90},
			"inactive": map[string]int{"count": 10, "inactive_days_min": 90},
		},
		"by_risk_level": []map[string]any{
			{"level": "low", "count": 120, "percentage": 80.0},
			{"level": "medium", "count": 22, "percentage": 14.7},
			{"level": "high", "count": 8, "percentage": 5.3},
		},
		"by_mfa_status": map[string]int{
			"enabled":  108,
			"disabled": 42,
		},
		"total_users": 150,
	})
}
