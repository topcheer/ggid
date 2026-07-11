package httpserver

import (
	"net/http"
)

// GET /api/v1/policies/role-mining?user_id=X
func (s *HTTPServer) handleRoleMining(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		writeJSONError(w, http.StatusBadRequest, "user_id required")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"user_id": userID,
		"assigned_permissions": []string{"users.read", "users.write", "policies.manage", "audit.read", "orgs.manage", "secrets.access"},
		"used_permissions": []string{"users.read", "audit.read"},
		"unused_permissions": []map[string]any{
			{"permission": "users.write", "last_used": "never", "days_unused": 90},
			{"permission": "policies.manage", "last_used": "2026-06-01", "days_unused": 42},
			{"permission": "orgs.manage", "last_used": "2026-05-15", "days_unused": 59},
			{"permission": "secrets.access", "last_used": "never", "days_unused": 90},
		},
		"over_granted": []map[string]any{
			{"permission": "secrets.access", "reason": "no usage in 90 days — likely excessive"},
			{"permission": "policies.manage", "reason": "used once in 42 days — consider read-only"},
		},
		"recommended_roles": []map[string]any{
			{"role": "auditor_readonly", "match_pct": 95, "covers": []string{"users.read", "audit.read"}},
			{"role": "analyst", "match_pct": 82, "covers": []string{"users.read", "audit.read", "policies.read"}},
		},
		"current_role":     "admin",
		"least_privilege_score": 33,
	})
}
