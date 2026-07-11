package server

import (
	"net/http"
)

// GET /api/v1/oauth/scope-delegation?token=X
func handleScopeDelegation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"}); return
	}
	token := r.URL.Query().Get("token")
	if token == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "token required"}); return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"token": token[:8] + "****",
		"delegation_chain": []map[string]any{
			{"step": 0, "actor": "admin", "scopes": []string{"openid", "profile", "admin.users", "admin.policies"}, "delegated_at": "2026-07-10T08:00:00Z"},
			{"step": 1, "actor": "manager", "scopes": []string{"openid", "profile", "users.read"}, "restricted_from": []string{"admin.policies"}, "delegated_at": "2026-07-11T10:00:00Z"},
			{"step": 2, "actor": "auditor", "scopes": []string{"openid", "audit.read"}, "restricted_from": []string{"users.read", "profile"}, "delegated_at": "2026-07-12T08:00:00Z"},
		},
		"current_scopes":   []string{"openid", "audit.read"},
		"original_scopes":  []string{"openid", "profile", "admin.users", "admin.policies"},
		"restriction_count": 2,
		"max_depth_reached": 2,
	})
}
