package server

import (
	"net/http"
	"strings"
	"time"
)

// GET /api/v1/identity/groups/{id}/membership-graph
func (h *HTTPHandler) handleMembershipGraph(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	groupID := strings.TrimPrefix(r.URL.Path, "/api/v1/identity/groups/")
	groupID = strings.TrimSuffix(groupID, "/membership-graph")
	groupID = strings.TrimSuffix(groupID, "/")
	if groupID == "" {
		writeError(w, http.StatusBadRequest, "group ID is required")
		return
	}

	now := time.Now().UTC()

	writeJSON(w, http.StatusOK, map[string]any{
		"group_id":     groupID,
		"analyzed_at":  now.Format(time.RFC3339),
		"direct_members": []map[string]any{
			{"user_id": "user-001", "username": "alice.eng", "added_at": now.Add(-90 * 24 * time.Hour).Format(time.RFC3339)},
			{"user_id": "user-003", "username": "carol.sec", "added_at": now.Add(-45 * 24 * time.Hour).Format(time.RFC3339)},
			{"user_id": "user-005", "username": "eve.dev", "added_at": now.Add(-12 * 24 * time.Hour).Format(time.RFC3339)},
			{"user_id": "user-008", "username": "henry.ops", "added_at": now.Add(-5 * 24 * time.Hour).Format(time.RFC3339)},
		},
		"direct_count": 4,
		"nested_groups": []map[string]any{
			{"group_id": "grp-backend", "name": "Backend Team", "members": 12, "depth": 1},
			{"group_id": "grp-frontend", "name": "Frontend Team", "members": 8, "depth": 1},
			{"group_id": "grp-devops", "name": "DevOps Team", "members": 5, "depth": 1},
			{"group_id": "grp-contractors", "name": "Contractors", "members": 3, "depth": 2},
		},
		"parent_groups": []map[string]any{
			{"group_id": "grp-engineering", "name": "Engineering (parent)", "depth": 1},
			{"group_id": "grp-all-staff", "name": "All Staff (root)", "depth": 2},
		},
		"total_depth":      2,
		"total_members":    32,
		"circular_detection": map[string]any{
			"has_circular":   false,
			"circular_paths": []string{},
			"checked_nodes":  8,
		},
		"graph_summary": map[string]any{
			"total_nodes":    12,
			"total_edges":    10,
			"max_depth":      2,
			"orphans":        0,
		},
	})
}
