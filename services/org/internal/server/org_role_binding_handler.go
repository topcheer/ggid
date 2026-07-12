package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// OrgRoleBinding links a user to a role within an organization context.
type OrgRoleBinding struct {
	ID        string    `json:"id"`
	OrgID     string    `json:"org_id"`
	UserID    string    `json:"user_id"`
	RoleID    string    `json:"role_id"`
	CreatedAt time.Time `json:"created_at"`
	CreatedBy string    `json:"created_by,omitempty"`
}

var (
	orgRoleBindingMu sync.RWMutex
	orgRoleBindings  = make(map[string]*OrgRoleBinding)
)

// POST /api/v1/organizations/{id}/role-bindings — bind role to user at org level.
// GET /api/v1/organizations/{id}/role-bindings — list org role bindings.
func (s *HTTPServer) handleOrgRoleBindings(w http.ResponseWriter, r *http.Request) {
	// Route budget paths to budget handler
	if strings.Contains(r.URL.Path, "budget-summary") || strings.Contains(r.URL.Path, "/budget") {
		s.handleOrgBudget(w, r)
		return
	}
	if strings.HasSuffix(r.URL.Path, "/access-report") {
		s.handleAccessReport(w, r)
		return
	}
	if strings.HasSuffix(r.URL.Path, "/restructure") {
		s.handleOrgRestructure(w, r)
		return
	}
	if strings.HasSuffix(r.URL.Path, "/members") {
		pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		orgUID, _ := uuid.Parse(pathParts[3])
		s.handleOrgMembers(w, r, orgUID)
		return
	}
	if strings.HasSuffix(r.URL.Path, "/access-matrix") {
		s.handleAccessMatrix(w, r)
		return
	}
	if strings.Contains(r.URL.Path, "/teams/export") {
		s.handleTeamsExport(w, r)
		return
	}

	// Extract org_id from path
	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")
	if len(parts) < 5 {
		writeJSONError(w, http.StatusBadRequest, "invalid path")
		return
	}
	orgID := parts[3] // api/v1/organizations/{id}/role-bindings

	switch r.Method {
	case http.MethodPost:
		var req struct {
			UserID    string `json:"user_id"`
			RoleID    string `json:"role_id"`
			CreatedBy string `json:"created_by"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if req.UserID == "" || req.RoleID == "" {
			writeJSONError(w, http.StatusBadRequest, "user_id and role_id are required")
			return
		}

		binding := &OrgRoleBinding{
			ID: uuid.New().String(), OrgID: orgID,
			UserID: req.UserID, RoleID: req.RoleID,
			CreatedBy: req.CreatedBy, CreatedAt: time.Now().UTC(),
		}
		orgRoleBindingMu.Lock()
		orgRoleBindings[binding.ID] = binding
		orgRoleBindingMu.Unlock()
		writeJSON(w, http.StatusCreated, binding)

	case http.MethodGet:
		userID := r.URL.Query().Get("user_id")
		roleID := r.URL.Query().Get("role_id")

		orgRoleBindingMu.RLock()
		result := []*OrgRoleBinding{}
		for _, b := range orgRoleBindings {
			if b.OrgID != orgID {
				continue
			}
			if userID != "" && b.UserID != userID {
				continue
			}
			if roleID != "" && b.RoleID != roleID {
				continue
			}
			result = append(result, b)
		}
		orgRoleBindingMu.RUnlock()
		writeJSON(w, http.StatusOK, map[string]any{"bindings": result, "count": len(result)})

	case http.MethodDelete:
		bindingID := r.URL.Query().Get("binding_id")
		if bindingID == "" {
			writeJSONError(w, http.StatusBadRequest, "binding_id is required")
			return
		}
		orgRoleBindingMu.Lock()
		if _, ok := orgRoleBindings[bindingID]; !ok {
			orgRoleBindingMu.Unlock()
			writeJSONError(w, http.StatusNotFound, "binding not found")
			return
		}
		delete(orgRoleBindings, bindingID)
		orgRoleBindingMu.Unlock()
		writeJSON(w, http.StatusOK, map[string]any{"status": "deleted", "binding_id": bindingID})

	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
