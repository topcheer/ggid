// Package httpserver provides REST API endpoints for the Policy Engine.
// These endpoints allow the Admin Console to manage roles, permissions, and policies
// via HTTP through the API Gateway.
package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/ggid/ggid/pkg/errors"
	"github.com/ggid/ggid/services/policy/internal/domain"
	"github.com/ggid/ggid/services/policy/internal/service"
	"github.com/google/uuid"
)

// HTTPServer exposes the Policy Engine as a REST API.
type HTTPServer struct {
	roleSvc   *service.RoleService
	policySvc *service.PolicyService
	evaluator *service.Evaluator
}

// NewHTTPServer creates a new Policy Engine HTTP server.
func NewHTTPServer(roleSvc *service.RoleService, policySvc *service.PolicyService, evaluator *service.Evaluator) *HTTPServer {
	return &HTTPServer{roleSvc: roleSvc, policySvc: policySvc, evaluator: evaluator}
}

// RegisterRoutes registers all Policy Engine HTTP routes on the given mux.
func (s *HTTPServer) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/roles", s.handleRoles)
	mux.HandleFunc("/api/v1/roles/", s.handleRoleByID)
	mux.HandleFunc("/api/v1/permissions", s.handlePermissions)
	mux.HandleFunc("/api/v1/policies", s.handlePolicies)
	mux.HandleFunc("/api/v1/policies/", s.handlePolicyByID)
	mux.HandleFunc("/api/v1/policies/check", s.handleCheck)
}

// --- Roles ---

func (s *HTTPServer) handleRoles(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		s.createRole(w, r)
	case http.MethodGet:
		s.listRoles(w, r)
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *HTTPServer) handleRoleByID(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/roles/")
	if idStr == "" {
		writeJSONError(w, http.StatusBadRequest, "role ID is required")
		return
	}

	// Handle sub-paths: /api/v1/roles/{id}/permissions
	parts := strings.SplitN(idStr, "/", 2)
	id, err := uuid.Parse(parts[0])
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid role ID")
		return
	}

	// Route to permissions sub-resource
	if len(parts) == 2 && parts[1] == "permissions" {
		s.handleRolePermissions(w, r, id)
		return
	}

	switch r.Method {
	case http.MethodGet:
		role, err := s.roleSvc.GetRole(r.Context(), id)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, roleToJSON(role))
	case http.MethodDelete:
		if err := s.roleSvc.DeleteRole(r.Context(), id); err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *HTTPServer) createRole(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TenantID     string `json:"tenant_id"`
		Key          string `json:"key"`
		Name         string `json:"name"`
		Description  string `json:"description"`
		ParentRoleID string `json:"parent_role_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid tenant_id")
		return
	}

	var parentID *uuid.UUID
	if req.ParentRoleID != "" {
		pid, err := uuid.Parse(req.ParentRoleID)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid parent_role_id")
			return
		}
		parentID = &pid
	}

	role, err := s.roleSvc.CreateRole(r.Context(), tenantID, req.Key, req.Name, req.Description, parentID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, roleToJSON(role))
}

func (s *HTTPServer) listRoles(w http.ResponseWriter, r *http.Request) {
	tenantIDStr := r.URL.Query().Get("tenant_id")
	if tenantIDStr == "" {
		writeJSONError(w, http.StatusBadRequest, "tenant_id query parameter is required")
		return
	}
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid tenant_id")
		return
	}

	roles, err := s.roleSvc.ListRoles(r.Context(), tenantID, 1, 50)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	result := make([]map[string]any, len(roles))
	for i, role := range roles {
		result[i] = roleToJSON(role)
	}
	writeJSON(w, http.StatusOK, map[string]any{"roles": result})
}

// --- Permissions ---

func (s *HTTPServer) handlePermissions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listPermissions(w, r)
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *HTTPServer) listPermissions(w http.ResponseWriter, r *http.Request) {
	tenantIDStr := r.URL.Query().Get("tenant_id")
	if tenantIDStr == "" {
		writeJSONError(w, http.StatusBadRequest, "tenant_id query parameter is required")
		return
	}
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid tenant_id")
		return
	}

	perms, err := s.roleSvc.ListPermissions(r.Context(), tenantID, 1, 100)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	result := make([]map[string]any, len(perms))
	for i, p := range perms {
		result[i] = permissionToJSON(p)
	}
	writeJSON(w, http.StatusOK, map[string]any{"permissions": result})
}

// --- Policies ---

func (s *HTTPServer) handlePolicies(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		s.createPolicy(w, r)
	case http.MethodGet:
		s.listPolicies(w, r)
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *HTTPServer) handlePolicyByID(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/policies/")
	// Prevent /api/v1/policies/check from matching here
	if idStr == "" || idStr == "check" {
		return
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid policy ID")
		return
	}

	switch r.Method {
	case http.MethodDelete:
		if err := s.policySvc.DeletePolicy(r.Context(), id); err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *HTTPServer) createPolicy(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TenantID    string   `json:"tenant_id"`
		Name        string   `json:"name"`
		Description string   `json:"description"`
		Effect      string   `json:"effect"`
		Actions     []string `json:"actions"`
		Resources   []string `json:"resources"`
		Priority    int      `json:"priority"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid tenant_id")
		return
	}

	policy, err := s.policySvc.CreatePolicy(r.Context(), &domain.Policy{
		TenantID:    tenantID,
		Name:        req.Name,
		Description: req.Description,
		Effect:      domain.Effect(req.Effect),
		Actions:     req.Actions,
		Resources:   req.Resources,
		Priority:    req.Priority,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, policyToJSON(policy))
}

func (s *HTTPServer) listPolicies(w http.ResponseWriter, r *http.Request) {
	tenantIDStr := r.URL.Query().Get("tenant_id")
	if tenantIDStr == "" {
		writeJSONError(w, http.StatusBadRequest, "tenant_id query parameter is required")
		return
	}
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid tenant_id")
		return
	}

	policies, err := s.policySvc.ListPolicies(r.Context(), tenantID, 1, 50)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	result := make([]map[string]any, len(policies))
	for i, p := range policies {
		result[i] = policyToJSON(p)
	}
	writeJSON(w, http.StatusOK, map[string]any{"policies": result})
}

// --- Permission Check ---

func (s *HTTPServer) handleCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		UserID       string `json:"user_id"`
		ResourceType string `json:"resource_type"`
		Action       string `json:"action"`
		Resource     string `json:"resource"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid user_id")
		return
	}

	result, err := s.evaluator.Check(r.Context(), &domain.CheckRequest{
		UserID:       userID,
		ResourceType: req.ResourceType,
		Action:       req.Action,
		Resource:     req.Resource,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"allowed":    result.Allowed,
		"reason":     result.Reason,
		"matched_by": result.MatchedBy,
	})
}

// --- Role-Permission management ---

func (s *HTTPServer) handleRolePermissions(w http.ResponseWriter, r *http.Request, roleID uuid.UUID) {
	switch r.Method {
	case http.MethodGet:
		perms, err := s.roleSvc.GetRolePermissions(r.Context(), roleID)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		result := make([]map[string]any, len(perms))
		for i, p := range perms {
			result[i] = permissionToJSON(p)
		}
		writeJSON(w, http.StatusOK, map[string]any{"permissions": result})
	case http.MethodPost:
		var req struct {
			PermissionIDs []string `json:"permission_ids"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		permIDs := make([]uuid.UUID, 0, len(req.PermissionIDs))
		for _, idStr := range req.PermissionIDs {
			pid, err := uuid.Parse(idStr)
			if err != nil {
				writeJSONError(w, http.StatusBadRequest, "invalid permission_id")
				return
			}
			permIDs = append(permIDs, pid)
		}
		if err := s.roleSvc.GrantPermissionsToRole(r.Context(), roleID, permIDs); err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "granted"})
	case http.MethodDelete:
		var req struct {
			PermissionIDs []string `json:"permission_ids"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		permIDs := make([]uuid.UUID, 0, len(req.PermissionIDs))
		for _, idStr := range req.PermissionIDs {
			pid, err := uuid.Parse(idStr)
			if err != nil {
				writeJSONError(w, http.StatusBadRequest, "invalid permission_id")
				return
			}
			permIDs = append(permIDs, pid)
		}
		if err := s.roleSvc.RevokePermissionsFromRole(r.Context(), roleID, permIDs); err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// --- Helpers ---

func roleToJSON(r *domain.Role) map[string]any {
	m := map[string]any{
		"id":          r.ID.String(),
		"tenant_id":   r.TenantID.String(),
		"key":         r.Key,
		"name":        r.Name,
		"description": r.Description,
		"system_role": r.SystemRole,
	}
	if r.ParentRoleID != nil {
		m["parent_role_id"] = r.ParentRoleID.String()
	}
	return m
}

func permissionToJSON(p *domain.Permission) map[string]any {
	return map[string]any{
		"id":            p.ID.String(),
		"tenant_id":     p.TenantID.String(),
		"key":           p.Key,
		"name":          p.Name,
		"resource_type": p.ResourceType,
		"action":        p.Action,
		"system_perm":   p.SystemPerm,
	}
}

func policyToJSON(p *domain.Policy) map[string]any {
	return map[string]any{
		"id":          p.ID.String(),
		"tenant_id":   p.TenantID.String(),
		"name":        p.Name,
		"description": p.Description,
		"effect":      string(p.Effect),
		"actions":     p.Actions,
		"resources":   p.Resources,
		"priority":    p.Priority,
	}
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func writeServiceError(w http.ResponseWriter, err error) {
	if ge, ok := errors.AsGGIDError(err); ok {
		switch ge.Code {
		case errors.ErrNotFound:
			writeJSONError(w, http.StatusNotFound, ge.Message)
		case errors.ErrAlreadyExists:
			writeJSONError(w, http.StatusConflict, ge.Message)
		case errors.ErrInvalidArgument:
			writeJSONError(w, http.StatusBadRequest, ge.Message)
		case errors.ErrPermissionDenied:
			writeJSONError(w, http.StatusForbidden, ge.Message)
		case errors.ErrFailedPrecondition:
			writeJSONError(w, http.StatusPreconditionFailed, ge.Message)
		default:
			writeJSONError(w, http.StatusInternalServerError, ge.Message)
		}
		return
	}
	writeJSONError(w, http.StatusInternalServerError, err.Error())
}

// Ensure context import is used.
var _ context.Context
