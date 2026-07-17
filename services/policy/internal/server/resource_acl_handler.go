package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

type ResourceACL struct {
	ID            string    `json:"id"`
	TenantID      string    `json:"tenant_id"`
	ResourcePath  string    `json:"resource_path"`
	Principal     string    `json:"principal"`
	PrincipalType string    `json:"principal_type"`
	Effect        string    `json:"effect"`
	Priority      int       `json:"priority"`
	CreatedAt     time.Time `json:"created_at"`
}

func (s *HTTPServer) handleResourceACL(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var req struct {
			TenantID      string `json:"tenant_id"`
			ResourcePath  string `json:"resource_path"`
			Principal     string `json:"principal"`
			PrincipalType string `json:"principal_type"`
			Effect        string `json:"effect"`
			Priority      int    `json:"priority"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		if req.ResourcePath == "" || req.Principal == "" {
			writeJSONError(w, http.StatusBadRequest, "resource_path and principal required")
			return
		}
		if req.Effect == "" {
			req.Effect = "allow"
		}
		if req.PrincipalType == "" {
			req.PrincipalType = "user"
		}
		acl := &ResourceACL{
			ID: uuid.New().String(), TenantID: req.TenantID, ResourcePath: req.ResourcePath,
			Principal: req.Principal, PrincipalType: req.PrincipalType, Effect: req.Effect,
			Priority: req.Priority, CreatedAt: time.Now().UTC(),
		}
		if s.policyMap != nil {
			s.policyMap.Store(r.Context(), "policy_resource_acls", acl.ID, map[string]any{
				"tenant_id": acl.TenantID, "resource_path": acl.ResourcePath,
				"principal": acl.Principal, "principal_type": acl.PrincipalType,
				"effect": acl.Effect, "priority": acl.Priority,
			})
		}
		writeJSON(w, http.StatusCreated, acl)
	case http.MethodGet:
		resource := r.URL.Query().Get("resource")
		var result []*ResourceACL
		if s.policyMap != nil {
			rows, _ := s.policyMap.List(r.Context(), "policy_resource_acls")
			for _, row := range rows {
				acl := &ResourceACL{
					ID: pmGetString(row, "id"), TenantID: pmGetString(row, "tenant_id"),
					ResourcePath: pmGetString(row, "resource_path"), Principal: pmGetString(row, "principal"),
					PrincipalType: pmGetString(row, "principal_type"), Effect: pmGetString(row, "effect"),
				}
				if resource != "" && !pathMatch(acl.ResourcePath, resource) {
					continue
				}
				result = append(result, acl)
			}
		}
		if result == nil {
			result = []*ResourceACL{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"acls": result, "count": len(result)})
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func pathMatch(pattern, path string) bool {
	if strings.HasSuffix(pattern, "/*") {
		return strings.HasPrefix(path, strings.TrimSuffix(pattern, "*"))
	}
	return pattern == path
}
