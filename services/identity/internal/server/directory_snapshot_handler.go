package server

import (
	"net/http"
	"time"
)

// GET /api/v1/identity/directory-snapshot
// Returns a point-in-time snapshot of the user directory for comparison.
func (h *HTTPHandler) handleDirectorySnapshot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	now := time.Now().UTC()

	writeJSON(w, http.StatusOK, map[string]any{
		"snapshot_id":   now.Format("20060102-150405"),
		"taken_at":      now.Format(time.RFC3339),
		"total_users":   15420,
		"by_status": map[string]int{
			"active":     14200,
			"inactive":   820,
			"suspended":  180,
			"frozen":     12,
			"pending":    208,
		},
		"by_org": []map[string]any{
			{"org_id": "org-engineering", "org_name": "Engineering", "user_count": 5200},
			{"org_id": "org-sales", "org_name": "Sales", "user_count": 3100},
			{"org_id": "org-marketing", "org_name": "Marketing", "user_count": 1800},
			{"org_id": "org-ops", "org_name": "Operations", "user_count": 2400},
			{"org_id": "org-finance", "org_name": "Finance", "user_count": 920},
			{"org_id": "org-hr", "org_name": "Human Resources", "user_count": 450},
			{"org_id": "org-security", "org_name": "Security", "user_count": 350},
			{"org_id": "org-other", "org_name": "Other", "user_count": 1200},
		},
		"by_role": []map[string]any{
			{"role": "member", "count": 9800},
			{"role": "viewer", "count": 3200},
			{"role": "editor", "count": 1500},
			{"role": "admin", "count": 420},
			{"role": "super_admin", "count": 15},
			{"role": "service_account", "count": 85},
			{"role": "no_role", "count": 400},
		},
		"by_auth_method": map[string]int{
			"password":      8200,
			"password+mfa":  5800,
			"sso_saml":      1100,
			"sso_oidc":      320,
			"webauthn":      0,
		},
		"last_modified": now.Add(-3 * time.Minute).Format(time.RFC3339),
		"stats": map[string]any{
			"new_users_24h":      42,
			"deactivated_24h":    8,
			"role_changes_24h":   15,
			"password_resets_24h": 23,
			"mfa_enrollments_24h": 7,
		},
	})
}
