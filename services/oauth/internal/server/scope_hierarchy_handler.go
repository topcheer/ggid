package server

import (
	"net/http"
)

type ScopeNode struct {
	Scope        string      `json:"scope"`
	Description  string      `json:"description"`
	ChildScopes  []ScopeNode `json:"child_scopes,omitempty"`
}

// GET /api/v1/oauth/scopes/hierarchy
func handleScopeHierarchy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}

	hierarchy := []ScopeNode{
		{
			Scope: "openid", Description: "OpenID Connect authentication",
			ChildScopes: []ScopeNode{
				{Scope: "openid.profile", Description: "Basic profile claims"},
			},
		},
		{
			Scope: "profile", Description: "User profile information",
			ChildScopes: []ScopeNode{
				{Scope: "profile.email", Description: "Email address"},
				{Scope: "profile.name", Description: "Full name"},
				{Scope: "profile.phone", Description: "Phone number"},
				{Scope: "profile.avatar", Description: "Avatar URL"},
			},
		},
		{
			Scope: "groups", Description: "Group and role membership",
			ChildScopes: []ScopeNode{
				{Scope: "groups.roles", Description: "Role assignments"},
				{Scope: "groups.departments", Description: "Department membership"},
			},
		},
		{
			Scope: "audit", Description: "Audit log access",
			ChildScopes: []ScopeNode{
				{Scope: "audit.read", Description: "Read audit events"},
				{Scope: "audit.export", Description: "Export audit data"},
			},
		},
		{
			Scope: "admin", Description: "Administrative access",
			ChildScopes: []ScopeNode{
				{Scope: "admin.users", Description: "User management"},
				{Scope: "admin.policies", Description: "Policy management"},
				{Scope: "admin.orgs", Description: "Organization management"},
				{Scope: "admin.security", Description: "Security configuration"},
			},
		},
	}

	// Count total scopes
	count := 0
	var countNodes func([]ScopeNode)
	countNodes = func(nodes []ScopeNode) {
		for _, n := range nodes {
			count++
			countNodes(n.ChildScopes)
		}
	}
	countNodes(hierarchy)

	writeJSON(w, http.StatusOK, map[string]any{
		"hierarchy":     hierarchy,
		"total_scopes":  count,
		"root_scopes":   len(hierarchy),
	})
}
