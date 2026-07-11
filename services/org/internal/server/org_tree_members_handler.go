package httpserver

import (
	"net/http"

	"github.com/google/uuid"
)

// GET /api/v1/organizations/{id}/tree?include_members=true
// Returns org tree with member_count, active_users, pending_invites per node.
// Note: /api/v1/orgs/{id}/tree already exists but without member counts.
// This handler adds member count support.
func (s *HTTPServer) handleOrgTreeWithMembers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract org ID from path
	orgIDStr := r.URL.Query().Get("org_id")
	if orgIDStr == "" {
		writeJSONError(w, http.StatusBadRequest, "org_id is required")
		return
	}
	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid org_id")
		return
	}

	includeMembers := r.URL.Query().Get("include_members") == "true"

	// Get sub-tree
	subTree, err := s.orgSvc.GetSubTree(r.Context(), s.getTenantID(r), orgID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to load org tree")
		return
	}

	// Build response with member counts
	nodes := make([]map[string]any, 0, len(subTree))
	for _, o := range subTree {
		node := map[string]any{
			"id":        o.ID.String(),
			"name":      o.Name,
			"parent_id": o.ParentID,
			"path":      o.Path,
		}
		if includeMembers {
			// Best-effort member counts from metadata
			mc := 0
			active := 0
			pending := 0
			if o.Metadata != nil {
				if v, ok := o.Metadata["member_count"].(float64); ok {
					mc = int(v)
				}
				if v, ok := o.Metadata["active_users"].(float64); ok {
					active = int(v)
				}
				if v, ok := o.Metadata["pending_invites"].(float64); ok {
					pending = int(v)
				}
			}
			node["member_count"] = mc
			node["active_users"] = active
			node["pending_invites"] = pending
		}
		nodes = append(nodes, node)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"organization_id": orgIDStr,
		"include_members": includeMembers,
		"tree":            nodes,
		"total_nodes":     len(nodes),
	})
}

func (s *HTTPServer) getTenantID(r *http.Request) uuid.UUID {
	idStr := r.Header.Get("X-Tenant-ID")
	if idStr == "" {
		return uuid.Nil
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		return uuid.Nil
	}
	return id
}
