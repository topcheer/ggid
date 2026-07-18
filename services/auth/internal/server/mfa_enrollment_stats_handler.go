package server

import (
	"net/http"
)

// GET /api/v1/auth/mfa/enrollment-stats
// Returns MFA enrollment statistics. Currently returns zero-based defaults
// because the MFA repository does not expose aggregate count methods.
// The response schema is stable for the frontend; values will populate
// when CountByTenant is added to the repository interface.
func (h *Handler) handleMFAEnrollmentStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"total_users":          0,
		"enrolled_users":       0,
		"unenrolled_users":     0,
		"enrollment_rate_pct":  0.0,
		"method_distribution":  []map[string]any{},
		"avg_methods_per_user": 0.0,
		"multi_factor_users":   0,
		"pending_enrollments":  []map[string]any{},
		"pending_count":        0,
		"enforcement": map[string]any{
			"required_for_admin":   true,
			"required_for_all":     false,
			"grace_period_days":    7,
			"enforced_users":       0,
			"enforcement_deadline": "",
		},
		"by_org":     []map[string]any{},
		"checked_at": "",
	})
}
