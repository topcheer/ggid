package server

import (
	"encoding/json"
	"net/http"
)

var scopeDeps = map[string][]string{
	"openid":       {},
	"profile":      {"openid"},
	"profile.email": {"profile"},
	"profile.name":  {"profile"},
	"audit":        {"openid"},
	"audit.read":   {"audit"},
	"audit.export": {"audit"},
	"admin":        {"openid"},
	"admin.users":   {"admin"},
	"admin.policies": {"admin"},
}

// POST /api/v1/oauth/scopes/resolve-dependencies
func handleResolveDependencies(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost { writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"}); return }
	var req struct{ RequestedScopes []string `json:"requested_scopes"` }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON"}); return }
	resolved := make(map[string]bool)
	var addWithDeps func(string)
	addWithDeps = func(scope string) {
		if resolved[scope] { return }
		resolved[scope] = true
		for _, dep := range scopeDeps[scope] { addWithDeps(dep) }
	}
	for _, s := range req.RequestedScopes { addWithDeps(s) }
	allScopes := make([]string, 0, len(resolved))
	for s := range resolved { allScopes = append(allScopes, s) }
	writeJSON(w, http.StatusOK, map[string]any{"requested_scopes": req.RequestedScopes, "resolved_scopes": allScopes, "added_dependencies": len(allScopes) - len(req.RequestedScopes)})
}
