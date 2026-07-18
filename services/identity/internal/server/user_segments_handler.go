package server

import (
	"net/http"
)

// GET /api/v1/users/segments
// Returns user segmentation analytics. Returns zero-based defaults until
// aggregate DB queries are implemented.
func (h *HTTPHandler) handleUserSegments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"by_role":       []map[string]any{},
		"by_activity":   map[string]any{},
		"by_risk_level": []map[string]any{},
		"by_mfa_status": map[string]int{
			"enabled":  0,
			"disabled": 0,
		},
		"total_users": 0,
	})
}
