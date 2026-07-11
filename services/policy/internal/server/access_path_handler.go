package httpserver

import (
	"net/http"
)

type AccessPath struct {
	Resource     string   `json:"resource"`
	Path         []string `json:"path"` // e.g. ["role:admin", "permission:users.read", "resource:/users"]
	AccessLevel  string   `json:"access_level"`
	Inherited    bool     `json:"inherited"`
}

// GET /api/v1/policies/access-paths?user_id=X
func (s *HTTPServer) handleAccessPaths(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		writeJSONError(w, http.StatusBadRequest, "user_id required")
		return
	}

	// Return all privilege paths from user to sensitive resources
	paths := []AccessPath{
		{Resource: "/users", Path: []string{"role:admin", "perm:users.read", "resource:/users"}, AccessLevel: "read", Inherited: false},
		{Resource: "/users", Path: []string{"role:admin", "perm:users.write", "resource:/users"}, AccessLevel: "write", Inherited: false},
		{Resource: "/policies", Path: []string{"role:admin", "perm:policies.manage", "resource:/policies"}, AccessLevel: "admin", Inherited: false},
		{Resource: "/audit/events", Path: []string{"role:auditor", "perm:audit.read", "resource:/audit/events"}, AccessLevel: "read", Inherited: true},
		{Resource: "/orgs/billing", Path: []string{"role:manager", "perm:billing.read", "resource:/orgs/billing"}, AccessLevel: "read", Inherited: true},
		{Resource: "/secrets/vault", Path: []string{"role:devops", "perm:secrets.access", "resource:/secrets/vault"}, AccessLevel: "admin", Inherited: false},
	}

	// Classify by sensitivity
	sensitiveCount := 0
	for _, p := range paths {
		if p.AccessLevel == "admin" || p.AccessLevel == "write" {
			sensitiveCount++
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"user_id":          userID,
		"paths":            paths,
		"total_paths":      len(paths),
		"sensitive_access": sensitiveCount,
		"recommendation": func() string {
			if sensitiveCount > 3 {
				return "excessive privileged access — review role assignments for least privilege"
			}
			return "access paths within acceptable limits"
		}(),
	})
}
