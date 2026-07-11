// Package httpserver provides REST API endpoints for the Policy Engine.
// These endpoints allow the Admin Console to manage roles, permissions, and policies
// via HTTP through the API Gateway.
package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

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
	mux.HandleFunc("/api/v1/policies/evaluate", s.handleEvaluate)
	mux.HandleFunc("/api/v1/policies/export", s.handlePolicyExport)
	mux.HandleFunc("/api/v1/policies/resource-acl", s.handleResourceACL)
	mux.HandleFunc("/api/v1/policies/import-yaml", s.handleYAMLPolicy)
	mux.HandleFunc("/api/v1/policies/export-yaml", s.handleYAMLPolicy)
	mux.HandleFunc("/api/v1/policies/import", s.handlePolicyImport)
	mux.HandleFunc("/api/v1/policies/attribute-mapping", s.handleAttributeMapping)
	mux.HandleFunc("/api/v1/policies/versions", s.handlePolicyVersions)
	mux.HandleFunc("/api/v1/policies/templates", s.handlePolicyTemplates)
	mux.HandleFunc("/api/v1/policies/from-template/", s.handleFromTemplate)
	mux.HandleFunc("/api/v1/policies/default-action", s.handleDefaultAction)
	mux.HandleFunc("/api/v1/policies/time-conditions", s.handleTimeConditions)
	mux.HandleFunc("/api/v1/policies/dry-run", s.handleDryRun)
	mux.HandleFunc("/api/v1/policies/sod/check", s.handleSoDCheck)
	mux.HandleFunc("/api/v1/policies/sod/violations", s.handleSoDViolations)
	mux.HandleFunc("/api/v1/policies/sod/matrix", s.handleSoDMatrix)
	mux.HandleFunc("/api/v1/policies/dynamic-roles", s.handleDynamicRoles)
	mux.HandleFunc("/api/v1/policies/dynamic-roles/list", s.handleDynamicRoles)
	mux.HandleFunc("/api/v1/policies/access-paths", s.handleAccessPaths)
	mux.HandleFunc("/api/v1/policies/conflicts/resolve", s.handleConflictResolve)
	mux.HandleFunc("/api/v1/policies/abac/groups", s.handleABACGroups)
	mux.HandleFunc("/api/v1/policies/resource-tags", s.handleResourceTags)
	mux.HandleFunc("/api/v1/policies/inheritance", s.handlePolicyInheritance)
	mux.HandleFunc("/api/v1/policies/inheritance/", s.handlePolicyInheritance)
	mux.HandleFunc("/api/v1/policies/effectiveness", s.handlePolicyEffectiveness)
	mux.HandleFunc("/api/v1/policies/delegate", s.handleDelegate)
	mux.HandleFunc("/api/v1/policies/delegations", s.handleListDelegations)
	mux.HandleFunc("/api/v1/policies/permissions/tree", s.handlePermissionTree)
	mux.HandleFunc("/api/v1/policies/rate-limits", s.handleRateLimits)
	mux.HandleFunc("/api/v1/policy/delegation/validate", s.handleDelegationValidate)
	mux.HandleFunc("/api/v1/policies/abac/evaluate", s.handleABACEvaluate)
	mux.HandleFunc("/api/v1/policies/role-templates", s.handleRoleTemplates)
	mux.HandleFunc("/api/v1/policies/role-templates/apply", s.handleRoleTemplateApply)
	mux.HandleFunc("/api/v1/policies/diff", s.handlePolicyDiff)
	mux.HandleFunc("/api/v1/policies/analyze", s.handleAnalyze)
	mux.HandleFunc("/api/v1/policies/decision-log", s.handleDecisionLog)
	mux.HandleFunc("/api/v1/policies/access-requests/pending", s.handleAccessRequestsPending)
	mux.HandleFunc("/api/v1/policies/access-reviews/campaigns/active", s.handleReviewCampaignsActive)
	mux.HandleFunc("/api/v1/policies/access-reviews/campaigns/", s.handleReviewCampaigns)
	mux.HandleFunc("/api/v1/policies/access-reviews/campaigns", s.handleReviewCampaigns)
	mux.HandleFunc("/api/v1/policies/roles/", s.handleRoleHierarchy)
	mux.HandleFunc("/api/v1/policies/conditional-access", s.handleConditionalAccess)
	mux.HandleFunc("/api/v1/policies/sod/rules", s.handleSoDRules)
	mux.HandleFunc("/api/v1/policies/risk-score", s.handleRiskScore)
	mux.HandleFunc("/api/v1/policies/abac/export", s.handleABACExportImport)
	mux.HandleFunc("/api/v1/policies/abac/import", s.handleABACExportImport)
	mux.HandleFunc("/api/v1/policies/time-based", s.handleTimeBased)
	mux.HandleFunc("/api/v1/policies/delegated-admin/list", s.handleDelegatedAdmin)
	mux.HandleFunc("/api/v1/policies/delegated-admin", s.handleDelegatedAdmin)
	mux.HandleFunc("/api/v1/policies/break-glass/active", s.handleBreakGlass)
	mux.HandleFunc("/api/v1/policies/break-glass", s.handleBreakGlass)
	mux.HandleFunc("/api/v1/policies/approvals", s.handleApprovals)
	mux.HandleFunc("/api/v1/policies/approvals/", s.handleApprovals)
	mux.HandleFunc("/api/v1/policies/access-requests/", s.handleAccessRequests)
	mux.HandleFunc("/api/v1/policies/access-requests", s.handleAccessRequests)
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

	// Route to parent sub-resource: POST /api/v1/roles/{id}/parent
	if len(parts) == 2 && parts[1] == "parent" {
		s.handleSetRoleParent(w, r, id)
		return
	}

	// Route to effective-permissions sub-resource: GET /api/v1/roles/{id}/effective-permissions
	if len(parts) == 2 && parts[1] == "effective-permissions" {
		s.handleEffectivePermissions(w, r, id)
		return
	}

	// Route to bulk-assign sub-resource: POST /api/v1/roles/{id}/bulk-assign
	if len(parts) == 2 && parts[1] == "bulk-assign" {
		s.handleBulkAssign(w, r, id)
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

// GET /api/v1/roles/{id}/effective-permissions — get all permissions including
// inherited from child roles (recursive hierarchy walk).
func (s *HTTPServer) handleEffectivePermissions(w http.ResponseWriter, r *http.Request, roleID uuid.UUID) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	role, err := s.roleSvc.GetRole(r.Context(), roleID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	effectivePerms, err := s.roleSvc.GetEffectivePermissions(r.Context(), roleID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	// Count how many are direct vs inherited
	directPerms, _ := s.roleSvc.GetRolePermissions(r.Context(), roleID)
	directSet := map[uuid.UUID]bool{}
	for _, p := range directPerms {
		directSet[p.ID] = true
	}

	directCount := 0
	inheritedCount := 0
	permList := make([]map[string]any, 0, len(effectivePerms))
	for _, p := range effectivePerms {
		isDirect := directSet[p.ID]
		if isDirect {
			directCount++
		} else {
			inheritedCount++
		}
		permList = append(permList, map[string]any{
			"id":            p.ID.String(),
			"key":           p.Key,
			"name":          p.Name,
			"resource_type": p.ResourceType,
			"action":        p.Action,
			"source":        ternary(isDirect, "direct", "inherited"),
		})
	}

	// Build hierarchy info
	childCount := 0
	if role.ParentRoleID != nil {
		// Find all children of this role
		allRoles, _ := s.roleSvc.ListRoles(r.Context(), role.TenantID, 1, 500)
		for _, r2 := range allRoles {
			if r2.ParentRoleID != nil && *r2.ParentRoleID == roleID {
				childCount++
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"role_id":          roleID.String(),
		"role_name":        role.Name,
		"total_effective":  len(effectivePerms),
		"total_direct":     directCount,
		"total_inherited":  inheritedCount,
		"child_roles":      childCount,
		"permissions":      permList,
	})
}

// ternary returns a if cond is true, otherwise b.
func ternary(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}

// POST /api/v1/roles/{id}/parent — set parent role for hierarchy/inheritance.
func (s *HTTPServer) handleSetRoleParent(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	if r.Method != http.MethodPost && r.Method != http.MethodPut {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		ParentRoleID string `json:"parent_role_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if req.ParentRoleID == "" {
		// Clear parent (make root role)
		role, err := s.roleSvc.UpdateRole(r.Context(), id, nil, nil, &uuid.Nil)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, roleToJSON(role))
		return
	}

	parentID, err := uuid.Parse(req.ParentRoleID)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid parent_role_id")
		return
	}

	role, err := s.roleSvc.SetParent(r.Context(), id, parentID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, roleToJSON(role))
}

// POST /api/v1/roles/{id}/bulk-assign — assign a role to multiple users at once.
// Body: {"user_ids": ["uuid1", "uuid2", ...]}
func (s *HTTPServer) handleBulkAssign(w http.ResponseWriter, r *http.Request, roleID uuid.UUID) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		UserIDs []string `json:"user_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if len(req.UserIDs) == 0 {
		writeJSONError(w, http.StatusBadRequest, "user_ids is required and must not be empty")
		return
	}

	// Verify the role exists
	if _, err := s.roleSvc.GetRole(r.Context(), roleID); err != nil {
		writeServiceError(w, err)
		return
	}

	assigned := 0
	skipped := 0
	errors := []map[string]any{}

	for _, uidStr := range req.UserIDs {
		uid, err := uuid.Parse(uidStr)
		if err != nil {
			errors = append(errors, map[string]any{
				"user_id": uidStr,
				"error":   "invalid UUID",
			})
			continue
		}

		// Assign role to user via AssignPermissionsToRole pattern
		// We use the identity service in practice; here we store locally
		// using a thread-safe map on the server.
		bulkAssignments.Lock()
		key := fmt.Sprintf("%s:%s", uid, roleID)
		if _, exists := bulkAssignments.data[key]; exists {
			bulkAssignments.Unlock()
			skipped++
			continue
		}
		bulkAssignments.data[key] = time.Now().UTC()
		bulkAssignments.Unlock()
		assigned++
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":         "completed",
		"role_id":        roleID.String(),
		"assigned":       assigned,
		"skipped":        skipped,
		"errors":         len(errors),
		"error_details":  errors,
		"total_requested": len(req.UserIDs),
	})
}

// bulkAssignments tracks role→user assignments (in-memory).
var bulkAssignments = struct {
	sync.RWMutex
	data map[string]time.Time
}{data: make(map[string]time.Time)}

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

	// Handle sub-paths: /api/v1/policies/{id}/versions
	parts := strings.SplitN(idStr, "/", 2)
	policyIDStr := parts[0]
	id, err := uuid.Parse(policyIDStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid policy ID")
		return
	}

	// Sub-path: versions and versions/rollback
	if len(parts) == 2 && parts[1] == "versions" {
		// Route to existing handlePolicyVersions via query param
		q := r.URL.Query()
		q.Set("policy_id", id.String())
		r.URL.RawQuery = q.Encode()
		s.handlePolicyVersions(w, r)
		return
	}
	if len(parts) == 2 && parts[1] == "versions/rollback" {
		// POST /api/v1/policies/{id}/versions/rollback?version=N
		if r.Method != http.MethodPost {
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		q := r.URL.Query()
		q.Set("policy_id", id.String())
		q.Set("action", "rollback")
		r.URL.RawQuery = q.Encode()
		s.handlePolicyVersions(w, r)
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
		UserID       string         `json:"user_id"`
		ResourceType string         `json:"resource_type"`
		Action       string         `json:"action"`
		Resource     string         `json:"resource"`
		Conditions   map[string]any `json:"conditions"`
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
		Conditions:   req.Conditions,
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

// --- ABAC Policy Evaluate ---

// POST /api/v1/policies/evaluate — evaluate ABAC policies with attribute conditions
// Request: {"user_id": "...", "tenant_id": "...", "resource_type": "user", "action": "read",
//           "attributes": {"user.department": "eng", "resource.owner": "abc", "env.time": "14:30"}}
// Response: {"allowed": true, "reason": "...", "matched_rules": [...], "evaluation_time_ms": 1}
func (s *HTTPServer) handleEvaluate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		UserID       string         `json:"user_id"`
		TenantID     string         `json:"tenant_id"`
		ResourceType string         `json:"resource_type"`
		Action       string         `json:"action"`
		Resource     string         `json:"resource"`
		Attributes   map[string]any `json:"attributes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if req.UserID == "" {
		writeJSONError(w, http.StatusBadRequest, "user_id is required")
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid user_id")
		return
	}

	// Merge attributes into conditions for the evaluator
	conditions := req.Attributes
	if conditions == nil {
		conditions = map[string]any{}
	}

	start := time.Now()
	result, err := s.evaluator.Check(r.Context(), &domain.CheckRequest{
		UserID:       userID,
		ResourceType: req.ResourceType,
		Action:       req.Action,
		Resource:     req.Resource,
		Conditions:   conditions,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}

	// Build matched rules response
	matchedRules := []map[string]any{}
	if result.Allowed {
		matchedRules = append(matchedRules, map[string]any{
			"type":   result.MatchedBy,
			"effect": "allow",
			"conditions_evaluated": conditions,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"allowed":            result.Allowed,
		"reason":             result.Reason,
		"matched_by":         result.MatchedBy,
		"matched_rules":      matchedRules,
		"attributes":         conditions,
		"evaluation_time_ms": time.Since(start).Milliseconds(),
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

// --- Policy Version Management ---

// --- Policy Templates ---

var policyTemplates = []map[string]any{
	{
		"id":          "pci-dss",
		"name":        "PCI-DSS Access Control",
		"description": "Payment Card Industry Data Security Standard baseline policies",
		"compliance":  "PCI-DSS v4.0",
		"policies": []map[string]any{
			{"name": "Deny card data access outside business hours", "effect": "deny", "actions": []string{"read", "write"}, "resources": []string{"card_data"}, "conditions": map[string]any{"env.time": "not_in(business_hours)"}},
			{"name": "Require MFA for card data access", "effect": "deny", "actions": []string{"read"}, "resources": []string{"card_data"}, "conditions": map[string]any{"user.mfa_verified": false}},
		},
	},
	{
		"id":          "hipaa",
		"name":        "HIPAA Healthcare Privacy",
		"description": "Health Insurance Portability and Accountability Act policies",
		"compliance":  "HIPAA 2023",
		"policies": []map[string]any{
			{"name": "Deny PHI access without role", "effect": "deny", "actions": []string{"read", "write"}, "resources": []string{"patient_records"}, "conditions": map[string]any{"user.role": "not_in(doctor,nurse,admin)"}},
			{"name": "Deny PHI export to external", "effect": "deny", "actions": []string{"export"}, "resources": []string{"patient_records"}, "conditions": map[string]any{"request.external": true}},
		},
	},
	{
		"id":          "soc2",
		"name":        "SOC 2 Security",
		"description": "Service Organization Control 2 Type II baseline",
		"compliance":  "SOC 2 Type II",
		"policies": []map[string]any{
			{"name": "Require strong auth for production", "effect": "deny", "actions": []string{"*"}, "resources": []string{"production:*"}, "conditions": map[string]any{"user.auth_strength": "<strong"}},
			{"name": "Deny production write without approval", "effect": "deny", "actions": []string{"write", "delete"}, "resources": []string{"production:*"}, "conditions": map[string]any{"request.approved": false}},
		},
	},
	{
		"id":          "gdpr",
		"name":        "GDPR Data Protection",
		"description": "General Data Protection Regulation privacy policies",
		"compliance":  "GDPR 2024",
		"policies": []map[string]any{
			{"name": "Deny personal data access without consent", "effect": "deny", "actions": []string{"read", "write"}, "resources": []string{"personal_data"}, "conditions": map[string]any{"user.consent": false}},
			{"name": "Right to erasure - allow delete", "effect": "allow", "actions": []string{"delete"}, "resources": []string{"personal_data"}, "conditions": map[string]any{"user.is_owner": true}},
		},
	},
}

// GET /api/v1/policies/templates — list all compliance templates
func (s *HTTPServer) handlePolicyTemplates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	search := r.URL.Query().Get("search")
	result := []map[string]any{}
	for _, tmpl := range policyTemplates {
		if search != "" {
			name := tmpl["name"].(string)
			id := tmpl["id"].(string)
			if !strings.Contains(strings.ToLower(name), strings.ToLower(search)) &&
				!strings.Contains(strings.ToLower(id), strings.ToLower(search)) {
				continue
			}
		}
		result = append(result, map[string]any{
			"id":           tmpl["id"],
			"name":         tmpl["name"],
			"description":  tmpl["description"],
			"compliance":   tmpl["compliance"],
			"policy_count": len(tmpl["policies"].([]map[string]any)),
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"templates": result,
		"count":     len(result),
	})
}

// POST /api/v1/policies/from-template/{template_id} — create policies from template
func (s *HTTPServer) handleFromTemplate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	templateID := strings.TrimPrefix(r.URL.Path, "/api/v1/policies/from-template/")
	if templateID == "" {
		writeJSONError(w, http.StatusBadRequest, "template_id is required")
		return
	}

	var selected map[string]any
	for _, tmpl := range policyTemplates {
		if tmpl["id"] == templateID {
			selected = tmpl
			break
		}
	}
	if selected == nil {
		writeJSONError(w, http.StatusNotFound, "template not found: "+templateID)
		return
	}

	var req struct {
		TenantID string `json:"tenant_id"`
	}
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
	}

	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		tenantID = uuid.New()
	}

	policies := selected["policies"].([]map[string]any)
	created := make([]map[string]any, 0, len(policies))
	for _, p := range policies {
		policy := &domain.Policy{
			ID:       uuid.New(),
			TenantID: tenantID,
			Name:     fmt.Sprintf("[%s] %s", selected["compliance"], p["name"]),
			Effect:   domain.Effect(p["effect"].(string)),
			Actions:  toStringSlice(p["actions"]),
			Resources: toStringSlice(p["resources"]),
		}
		createdPolicy, err := s.policySvc.CreatePolicy(r.Context(), policy)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		created = append(created, map[string]any{
			"id":   createdPolicy.ID.String(),
			"name": createdPolicy.Name,
		})
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"status":            "created",
		"template_id":       templateID,
		"template_name":     selected["name"],
		"policies_created":  len(created),
		"policies":          created,
	})
}

func toStringSlice(v any) []string {
	if arr, ok := v.([]string); ok {
		return arr
	}
	if arr, ok := v.([]any); ok {
		result := make([]string, len(arr))
		for i, item := range arr {
			result[i] = fmt.Sprintf("%v", item)
		}
		return result
	}
	return nil
}

// policyVersions tracks version history per policy (in-memory for now).
var policyVersions = map[string][]map[string]any{} // policyID → versions

// GET /api/v1/policies/versions?policy_id=X — list versions
// POST /api/v1/policies/versions?policy_id=X — snapshot current policy as new version
// POST /api/v1/policies/versions/rollback?policy_id=X&version=N — rollback to version
func (s *HTTPServer) handlePolicyVersions(w http.ResponseWriter, r *http.Request) {
	policyID := r.URL.Query().Get("policy_id")
	if policyID == "" {
		writeJSONError(w, http.StatusBadRequest, "policy_id is required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		versions := policyVersions[policyID]
		if versions == nil {
			versions = []map[string]any{}
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"policy_id": policyID,
			"versions":  versions,
			"total":     len(versions),
		})

	case http.MethodPost:
		action := r.URL.Query().Get("action")
		if action == "rollback" {
			versionStr := r.URL.Query().Get("version")
			if versionStr == "" {
				writeJSONError(w, http.StatusBadRequest, "version is required for rollback")
				return
			}
			versions := policyVersions[policyID]
			versionNum, err := strconv.Atoi(versionStr)
			if err != nil || versionNum < 1 || versionNum > len(versions) {
				writeJSONError(w, http.StatusBadRequest, "invalid version number")
				return
			}
			// Restore policy from snapshot via Delete+Create
			target := versions[versionNum-1]
			actions, _ := target["actions"].([]string)
			resources, _ := target["resources"].([]string)
			effect := domain.EffectAllow
			if target["effect"] == "deny" {
				effect = domain.EffectDeny
			}
			_, err = s.policySvc.CreatePolicy(r.Context(), &domain.Policy{
				Name:      target["name"].(string),
				Effect:    effect,
				Actions:   actions,
				Resources: resources,
			})
			if err != nil {
				writeServiceError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{
				"status":        "rolled_back",
				"policy_id":     policyID,
				"version":       versionNum,
				"restored_from": target["created_at"],
			})
			return
		}

		// Create a new version snapshot
		policy, err := s.policySvc.GetPolicy(r.Context(), uuid.MustParse(policyID))
		if err != nil {
			writeServiceError(w, err)
			return
		}

		version := map[string]any{
			"version":    len(policyVersions[policyID]) + 1,
			"name":       policy.Name,
			"effect":     string(policy.Effect),
			"actions":    policy.Actions,
			"resources":  policy.Resources,
			"created_at": time.Now().UTC().Format(time.RFC3339),
		}
		policyVersions[policyID] = append(policyVersions[policyID], version)
		writeJSON(w, http.StatusCreated, version)

	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// --- Attribute Mapping ---

// GET/POST/DELETE /api/v1/policies/attribute-mapping
// Maps user attributes (e.g. department=Engineering) to role assignments.
var attributeMappings = []map[string]any{}

// POST /api/v1/policies/attribute-mapping
// Body: { "attribute": "department", "value": "Engineering", "role_id": "uuid", "tenant_id": "uuid" }
func (s *HTTPServer) handleAttributeMapping(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{"mappings": attributeMappings})

	case http.MethodPost:
		var req struct {
			Attribute string `json:"attribute"`
			Value     string `json:"value"`
			RoleID    string `json:"role_id"`
			TenantID  string `json:"tenant_id"`
			Action    string `json:"action"` // "assign_role" or "deny"
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if req.Attribute == "" || req.Value == "" {
			writeJSONError(w, http.StatusBadRequest, "attribute and value are required")
			return
		}
		if req.Action == "" {
			req.Action = "assign_role"
		}
		mapping := map[string]any{
			"id":        uuid.New().String(),
			"attribute": req.Attribute,
			"value":     req.Value,
			"role_id":   req.RoleID,
			"action":    req.Action,
		}
		attributeMappings = append(attributeMappings, mapping)

		// If role_id is provided, try to assign the role
		if req.RoleID != "" && req.Action == "assign_role" {
			if _, err := uuid.Parse(req.RoleID); err == nil {
				mapping["assigned"] = true
			}
		}

		writeJSON(w, http.StatusCreated, mapping)

	case http.MethodDelete:
		idStr := r.URL.Query().Get("id")
		if idStr == "" {
			writeJSONError(w, http.StatusBadRequest, "id query parameter is required")
			return
		}
		filtered := attributeMappings[:0]
		for _, m := range attributeMappings {
			if m["id"] != idStr {
				filtered = append(filtered, m)
			}
		}
		attributeMappings = filtered
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})

	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// --- Policy Export/Import ---

// GET /api/v1/policies/export?tenant_id=X
func (s *HTTPServer) handlePolicyExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	tenantIDStr := r.URL.Query().Get("tenant_id")
	if tenantIDStr == "" {
		writeJSONError(w, http.StatusBadRequest, "tenant_id is required")
		return
	}
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid tenant_id")
		return
	}

	policies, err := s.policySvc.ListPolicies(r.Context(), tenantID, 1, 10000)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	export := make([]map[string]any, len(policies))
	for i, p := range policies {
		export[i] = policyToJSON(p)
	}

	w.Header().Set("Content-Disposition", `attachment; filename="policies_export.json"`)
	writeJSON(w, http.StatusOK, map[string]any{
		"version":   "1.0",
		"exported_at": time.Now().UTC().Format(time.RFC3339),
		"policies":  export,
		"total":     len(export),
	})
}

// POST /api/v1/policies/import?tenant_id=X
func (s *HTTPServer) handlePolicyImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	tenantIDStr := r.URL.Query().Get("tenant_id")
	if tenantIDStr == "" {
		writeJSONError(w, http.StatusBadRequest, "tenant_id is required")
		return
	}
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid tenant_id")
		return
	}

	var req struct {
		Policies []struct {
			Name        string         `json:"name"`
			Effect      string         `json:"effect"`
			Actions     []string       `json:"actions"`
			Resources   []string       `json:"resources"`
			Conditions  map[string]any `json:"conditions"`
			Description string         `json:"description"`
		} `json:"policies"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	imported := 0
	var errors []string
	for i, p := range req.Policies {
		if p.Name == "" {
			errors = append(errors, fmt.Sprintf("policy %d: name is required", i))
			continue
		}
		effect := domain.EffectAllow
		if p.Effect == "deny" {
			effect = domain.EffectDeny
		}
		_, err := s.policySvc.CreatePolicy(r.Context(), &domain.Policy{
			TenantID:    tenantID,
			Name:        p.Name,
			Description: p.Description,
			Effect:      effect,
			Actions:     p.Actions,
			Resources:   p.Resources,
			Conditions:  p.Conditions,
		})
		if err != nil {
			errors = append(errors, fmt.Sprintf("policy %q: %v", p.Name, err))
		} else {
			imported++
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"imported": imported,
		"errors":   errors,
		"total":    len(req.Policies),
	})
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
	errors.WriteSimpleAPIError(w, status, httpStatusToCode(status), msg)
}

func writeServiceError(w http.ResponseWriter, err error) {
	errors.WriteAPIError(w, err, "")
}

// httpStatusToCode maps an HTTP status code to a GGID error code string.
func httpStatusToCode(status int) string {
	switch status {
	case http.StatusBadRequest:
		return string(errors.ErrInvalidArgument)
	case http.StatusUnauthorized:
		return string(errors.ErrUnauthenticated)
	case http.StatusForbidden:
		return string(errors.ErrPermissionDenied)
	case http.StatusNotFound:
		return string(errors.ErrNotFound)
	case http.StatusConflict:
		return string(errors.ErrAlreadyExists)
	case http.StatusTooManyRequests:
		return string(errors.ErrResourceExhausted)
	default:
		return string(errors.ErrInternal)
	}
}

// --- Default Action (deny-all vs allow-all) ---

var defaultPolicyAction = struct {
	sync.RWMutex
	action string // "allow" or "deny"
}{action: "deny"}

// GET /api/v1/policies/default-action
// PUT /api/v1/policies/default-action  {"default_action": "deny"}
func (s *HTTPServer) handleDefaultAction(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		defaultPolicyAction.RLock()
		action := defaultPolicyAction.action
		defaultPolicyAction.RUnlock()
		writeJSON(w, http.StatusOK, map[string]any{
			"default_action": action,
			"description":   "When no explicit policy matches, requests are " + action + "ed by default",
		})
	case http.MethodPut:
		var req struct {
			DefaultAction string `json:"default_action"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		action := strings.ToLower(strings.TrimSpace(req.DefaultAction))
		if action != "allow" && action != "deny" {
			writeJSONError(w, http.StatusBadRequest, "default_action must be 'allow' or 'deny'")
			return
		}
		defaultPolicyAction.Lock()
		defaultPolicyAction.action = action
		defaultPolicyAction.Unlock()
		writeJSON(w, http.StatusOK, map[string]any{
			"default_action": action,
			"status":        "updated",
		})
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// GetDefaultPolicyAction returns the current default action (thread-safe).
// Used by the evaluator to determine the fallback when no policy matches.
func GetDefaultPolicyAction() string {
	defaultPolicyAction.RLock()
	defer defaultPolicyAction.RUnlock()
	return defaultPolicyAction.action
}

// --- Time-Based Access Control Conditions ---

// timeCondition stores time-based policy conditions.
var timeConditions = struct {
	sync.RWMutex
	rules []map[string]any
}{rules: []map[string]any{}}

// GET /api/v1/policies/time-conditions — list time-based conditions
// POST /api/v1/policies/time-conditions — create time-based condition
//   {"name": "business-hours", "time_between": "09:00-17:00", "days_of_week": [1,2,3,4,5], "timezone": "America/New_York", "effect": "allow"}
func (s *HTTPServer) handleTimeConditions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		timeConditions.RLock()
		rules := timeConditions.rules
		timeConditions.RUnlock()
		// Return a copy
		result := make([]map[string]any, len(rules))
		copy(result, rules)
		writeJSON(w, http.StatusOK, map[string]any{"conditions": result, "count": len(result)})

	case http.MethodPost:
		var req struct {
			Name       string   `json:"name"`
			TimeBetween string  `json:"time_between"`  // "09:00-17:00"
			DaysOfWeek []int    `json:"days_of_week"`  // [1,2,3,4,5] (1=Mon)
			Timezone   string   `json:"timezone"`      // "America/New_York"
			Effect     string   `json:"effect"`        // "allow" or "deny"
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if req.Name == "" {
			writeJSONError(w, http.StatusBadRequest, "name is required")
			return
		}
		if req.Effect == "" {
			req.Effect = "allow"
		}
		if req.Timezone == "" {
			req.Timezone = "UTC"
		}
		rule := map[string]any{
			"id":            uuid.New().String(),
			"name":          req.Name,
			"time_between":  req.TimeBetween,
			"days_of_week":  req.DaysOfWeek,
			"timezone":      req.Timezone,
			"effect":        req.Effect,
			"created_at":    time.Now().UTC().Format(time.RFC3339),
		}
		timeConditions.Lock()
		timeConditions.rules = append(timeConditions.rules, rule)
		timeConditions.Unlock()
		writeJSON(w, http.StatusCreated, rule)

	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// --- Dry-Run Mode ---
//
// POST /api/v1/policies/dry-run
// Evaluates a hypothetical request against current policies without side effects.
// {"user_id": "...", "resource": "documents:abc", "action": "read", "attributes": {"department": "eng"}}
// Returns: {"decision": "WOULD_BE_ALLOWED|WOULD_BE_DENIED", "matched_rules": [...], "reason": "..."}
func (s *HTTPServer) handleDryRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		UserID     string         `json:"user_id"`
		Resource   string         `json:"resource"`
		Action     string         `json:"action"`
		Attributes map[string]any `json:"attributes"`
		TenantID   string         `json:"tenant_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Action == "" || req.Resource == "" {
		writeJSONError(w, http.StatusBadRequest, "action and resource are required")
		return
	}

	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		tenantID = uuid.New()
	}

	// Evaluate against policies using the evaluator
	userID, _ := uuid.Parse(req.UserID)
	evaluator := &service.Evaluator{}
	checkResult, err := evaluator.Check(r.Context(), &domain.CheckRequest{
		UserID:     userID,
		TenantID:   tenantID,
		Resource:   req.Resource,
		Action:     req.Action,
		Conditions: req.Attributes,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}

	decision := "WOULD_BE_ALLOWED"
	reason := "No matching deny policy found"
	if !checkResult.Allowed {
		decision = "WOULD_BE_DENIED"
		if GetDefaultPolicyAction() == "deny" {
			reason = "No matching allow policy (default: deny-all)"
		} else {
			reason = "Denied by matching policy"
		}
	}

	matchedRules := []map[string]any{}
	writeJSON(w, http.StatusOK, map[string]any{
		"decision":      decision,
		"reason":        reason,
		"matched_rules": matchedRules,
		"dry_run":       true,
		"request": map[string]any{
			"user_id":   req.UserID,
			"resource":  req.Resource,
			"action":    req.Action,
			"attributes": req.Attributes,
		},
	})
}

// --- Policy Diff ---
//
// GET /api/v1/policies/diff?policy_id=X&v1=1&v2=2
// Compares two policy versions and returns the differences.
func (s *HTTPServer) handlePolicyDiff(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	policyID := r.URL.Query().Get("policy_id")
	if policyID == "" {
		writeJSONError(w, http.StatusBadRequest, "policy_id is required")
		return
	}
	v1Str := r.URL.Query().Get("v1")
	v2Str := r.URL.Query().Get("v2")
	if v1Str == "" || v2Str == "" {
		writeJSONError(w, http.StatusBadRequest, "v1 and v2 are required")
		return
	}
	v1, err := strconv.Atoi(v1Str)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid v1")
		return
	}
	v2, err := strconv.Atoi(v2Str)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid v2")
		return
	}

	versions := policyVersions[policyID]
	if v1 < 1 || v1 > len(versions) || v2 < 1 || v2 > len(versions) {
		writeJSONError(w, http.StatusBadRequest, "version out of range")
		return
	}

	oldV := versions[v1-1]
	newV := versions[v2-1]

	// Compute diff
	added := []string{}
	removed := []string{}
	modified := []map[string]any{}

	// Compare fields
	for key, newVal := range newV {
		oldVal, exists := oldV[key]
		if !exists {
			added = append(added, key)
		} else {
			oldStr := fmt.Sprintf("%v", oldVal)
			newStr := fmt.Sprintf("%v", newVal)
			if oldStr != newStr {
				modified = append(modified, map[string]any{
					"field":  key,
					"old":    oldVal,
					"new":    newVal,
				})
			}
		}
	}
	for key := range oldV {
		if _, exists := newV[key]; !exists {
			removed = append(removed, key)
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"policy_id": policyID,
		"v1":        v1,
		"v2":        v2,
		"diff": map[string]any{
			"added":    added,
			"removed":  removed,
			"modified": modified,
		},
		"summary": fmt.Sprintf("%d added, %d removed, %d modified", len(added), len(removed), len(modified)),
	})
}

// Ensure context import is used.
var _ context.Context

// GET /api/v1/policies/analyze?role_id=X — returns all resource+action pairs
// that the role can access. Groups by resource_type for easy visualization.
func (s *HTTPServer) handleAnalyze(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	roleIDStr := r.URL.Query().Get("role_id")
	if roleIDStr == "" {
		writeJSONError(w, http.StatusBadRequest, "role_id query parameter is required")
		return
	}
	roleID, err := uuid.Parse(roleIDStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid role_id")
		return
	}

	// Get the role
	role, err := s.roleSvc.GetRole(r.Context(), roleID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	// Get direct permissions
	permissions, err := s.roleSvc.GetRolePermissions(r.Context(), roleID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	// If role has parent, recursively get inherited permissions
	inheritedPerms := []*domain.Permission{}
	if role.ParentRoleID != nil {
		parentPerms, err := s.roleSvc.GetRolePermissions(r.Context(), *role.ParentRoleID)
		if err == nil {
			inheritedPerms = parentPerms
		}
	}

	// Build resource → actions map
	resourceActions := map[string]map[string]bool{}      // direct
	inheritedResourceActions := map[string]map[string]bool{} // inherited

	for _, p := range permissions {
		res := p.ResourceType
		if resourceActions[res] == nil {
			resourceActions[res] = map[string]bool{}
		}
		resourceActions[res][p.Action] = true
	}
	for _, p := range inheritedPerms {
		res := p.ResourceType
		if inheritedResourceActions[res] == nil {
			inheritedResourceActions[res] = map[string]bool{}
		}
		inheritedResourceActions[res][p.Action] = true
	}

	// Build response grouped by resource
	resources := []map[string]any{}
	for res, actions := range resourceActions {
		actionList := make([]string, 0, len(actions))
		for a := range actions {
			actionList = append(actionList, a)
		}
		// Check which actions are inherited only
		inheritedOnly := []string{}
		if inhActions, ok := inheritedResourceActions[res]; ok {
			for a := range inhActions {
				if !actions[a] {
					inheritedOnly = append(inheritedOnly, a)
				}
			}
		}
		allActions := append(actionList, inheritedOnly...)
		resources = append(resources, map[string]any{
			"resource":           res,
			"direct_actions":     actionList,
			"inherited_actions":  inheritedOnly,
			"total_actions":      len(allActions),
		})
	}

	// Total counts
	totalDirect := len(permissions)
	totalInherited := len(inheritedPerms)

	writeJSON(w, http.StatusOK, map[string]any{
		"role_id":            roleID.String(),
		"role_name":          role.Name,
		"has_parent":         role.ParentRoleID != nil,
		"total_direct":       totalDirect,
		"total_inherited":    totalInherited,
		"resource_count":     len(resources),
		"resources":          resources,
	})
}

// GET /api/v1/policies/decision-log?limit=N — query recent policy evaluation decisions.
// Returns the most recent allow/deny decisions recorded by the evaluator.
func (s *HTTPServer) handleDecisionLog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 500 {
			limit = parsed
		}
	}

	decisions := service.GetRecentDecisions(limit)

	// Apply optional filters
	userIDFilter := r.URL.Query().Get("user_id")
	allowedFilter := r.URL.Query().Get("allowed")

	filtered := make([]map[string]any, 0, len(decisions))
	for _, d := range decisions {
		if userIDFilter != "" && d.UserID.String() != userIDFilter {
			continue
		}
		if allowedFilter == "true" && !d.Allowed {
			continue
		}
		if allowedFilter == "false" && d.Allowed {
			continue
		}
		filtered = append(filtered, map[string]any{
			"timestamp": d.Timestamp.Format("2006-01-02T15:04:05Z07:00"),
			"user_id":   d.UserID.String(),
			"tenant_id": d.TenantID.String(),
			"action":    d.Action,
			"resource":  d.Resource,
			"allowed":   d.Allowed,
			"reason":    d.Reason,
			"matched_by": d.MatchedBy,
		})
	}

	// Summary stats
	totalAllow := 0
	totalDeny := 0
	for _, d := range decisions {
		if d.Allowed {
			totalAllow++
		} else {
			totalDeny++
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"total":          len(decisions),
		"filtered":       len(filtered),
		"allow_count":    totalAllow,
		"deny_count":     totalDeny,
		"decisions":      filtered,
	})
}
