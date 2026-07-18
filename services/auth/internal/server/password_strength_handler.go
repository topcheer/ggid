package server

import (
	"net/http"
)

// GET /api/v1/auth/password-strength/distribution
func (h *Handler) handlePasswordStrengthDist(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed"); return
	}
	total := 150
	writeJSON(w, http.StatusOK, map[string]any{
		"distribution": map[string]any{
			"weak":   map[string]int{"count": 14, "percentage": 9},
			"fair":   map[string]int{"count": 28, "percentage": 19},
			"good":   map[string]int{"count": 62, "percentage": 41},
			"strong": map[string]int{"count": 46, "percentage": 31},
		},
		"total_users":            total,
		"compliant_percentage":   72, // good + strong
		"non_compliant_count":    42, // weak + fair
		"avg_entropy_bits":       47.3,
		"avg_length":             11.2,
		"policy_min_length":      8,
		"enforcement_status":     "enabled",
	})
}
