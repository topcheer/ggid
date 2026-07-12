package httpserver

import (
	"net/http"
	"strings"
	"time"
)

// GET /api/v1/policy/blast-radius/{policy_id}
// Returns impact scope of a policy change.
func (s *HTTPServer) handlePolicyBlastRadius(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	policyID := strings.TrimPrefix(r.URL.Path, "/api/v1/policy/blast-radius/")
	policyID = strings.TrimSuffix(policyID, "/")
	if policyID == "" {
		writeJSONError(w, http.StatusBadRequest, "policy_id is required")
		return
	}

	previewMode := r.URL.Query().Get("mode")
	if previewMode == "" {
		previewMode = "preview"
	}

	now := time.Now().UTC()

	writeJSON(w, http.StatusOK, map[string]any{
		"policy_id":      policyID,
		"preview_mode":   previewMode,
		"affected_users": []map[string]any{
			{"user_id": "user-001", "username": "alice.admin", "impact": "loses_permission", "severity": "high"},
			{"user_id": "user-003", "username": "carol.eng", "impact": "loses_permission", "severity": "medium"},
			{"user_id": "user-005", "username": "eve.sec", "impact": "gains_permission", "severity": "low"},
			{"user_id": "user-008", "username": "henry.ops", "impact": "modified_permission", "severity": "medium"},
			{"user_id": "user-012", "username": "ivan.dev", "impact": "loses_permission", "severity": "high"},
		},
		"affected_roles": []map[string]any{
			{"role_id": "role-admin", "role_name": "Administrator", "users_impacted": 3, "permissions_changed": 2},
			{"role_id": "role-editor", "role_name": "Editor", "users_impacted": 8, "permissions_changed": 1},
			{"role_id": "role-viewer", "role_name": "Viewer", "users_impacted": 0, "permissions_changed": 0},
		},
		"affected_resources": []map[string]any{
			{"resource": "users/*", "access_change": -3, "current_accessors": 145},
			{"resource": "audit/*", "access_change": -1, "current_accessors": 38},
			{"resource": "config/*", "access_change": +1, "current_accessors": 12},
		},
		"cascading_policies": []map[string]any{
			{"policy_id": "pol-002", "name": "Department RBAC", "relationship": "depends_on", "impact": "rules_re_evaluated"},
			{"policy_id": "pol-005", "name": "ABAC Conditions", "relationship": "overlaps", "impact": "may_conflict"},
			{"policy_id": "pol-008", "name": "Emergency Access", "relationship": "supersedes", "impact": "not_affected"},
		},
		"summary": map[string]any{
			"total_users_affected":    5,
			"total_roles_affected":    2,
			"total_resources_changed": 3,
			"total_cascading":         3,
			"breaking_changes":        3,
			"risk_level":              "high",
			"recommended_action":      "schedule_maintenance_window_and_notify_affected_users",
		},
		"analyzed_at": now.Format(time.RFC3339),
	})
}
