package server

import (
	"net/http"
	"strings"
)

// GET /api/v1/organizations/{id}/org-chart
func (h *HTTPHandler) handleOrgChart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 4 {
		writeJSONError(w, http.StatusBadRequest, "organization id required")
		return
	}
	orgID := parts[3]
	nodes := []map[string]any{
		{"user_id": "u-001", "name": "Alice Chen", "title": "CTO", "manager_id": nil, "department": "Engineering"},
		{"user_id": "u-002", "name": "Bob Wang", "title": "VP Engineering", "manager_id": "u-001", "department": "Engineering"},
		{"user_id": "u-003", "name": "Carol Liu", "title": "Engineering Manager", "manager_id": "u-002", "department": "Backend"},
		{"user_id": "u-004", "name": "Dave Zhang", "title": "Senior Engineer", "manager_id": "u-003", "department": "Backend"},
		{"user_id": "u-005", "name": "Eve Sun", "title": "Engineering Manager", "manager_id": "u-002", "department": "Frontend"},
		{"user_id": "u-006", "name": "Frank Wu", "title": "Senior Engineer", "manager_id": "u-005", "department": "Frontend"},
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"org_id": orgID, "nodes": nodes, "total": len(nodes),
	})
}
