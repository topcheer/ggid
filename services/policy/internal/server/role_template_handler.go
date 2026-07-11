package httpserver

import (
	"encoding/json"
	"net/http"
)

var roleTemplates = map[string][]string{
	"admin":    {"users:read", "users:write", "roles:read", "roles:write", "policies:read", "policies:write", "audit:read"},
	"operator": {"users:read", "users:write", "roles:read", "policies:read"},
	"viewer":   {"users:read", "roles:read", "policies:read", "audit:read"},
	"auditor":  {"users:read", "audit:read", "policies:read"},
}

func (s *HTTPServer) handleRoleTemplates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"templates": roleTemplates})
}

func (s *HTTPServer) handleRoleTemplateApply(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	var req struct {
		Template string `json:"template"`
		RoleName string `json:"role_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	perms, ok := roleTemplates[req.Template]
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unknown template"})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"role_name":   req.RoleName,
		"template":    req.Template,
		"permissions": perms,
		"created":     true,
	})
}
