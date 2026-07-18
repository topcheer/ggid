package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/ggid/ggid/pkg/errors"
		"github.com/ggid/ggid/pkg/tenant"
	"github.com/ggid/ggid/services/auth/internal/repository"
	"github.com/google/uuid"
)

// authMethodPolicyRequest is the DTO for create/update operations.
type authMethodPolicyRequest struct {
	GroupID          string   `json:"group_id"`
	RequiredMethods  []string `json:"required_methods"`
	ForbiddenMethods []string `json:"forbidden_methods"`
	Priority         int      `json:"priority"`
}

// SetAuthMethodPolicyRepo injects the DB-backed repository.
func (h *Handler) SetAuthMethodPolicyRepo(repo *repository.AuthMethodPolicyRepository) {
	h.authMethodPolicyRepo = repo
}

// handleAuthMethodPolicies handles CRUD for /api/v1/auth/method-policies.
func (h *Handler) handleAuthMethodPolicies(w http.ResponseWriter, r *http.Request) {
	// Extract tenant context.
	tc, err := tenant.FromContext(r.Context())
	if err != nil {
		errors.WriteSimpleAPIError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context required")
		return
	}

	// Route by method + path suffix.
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/auth/method-policies")

	switch {
	case r.Method == http.MethodGet && path == "" || path == "/":
		h.listAuthMethodPolicies(w, r, tc.TenantID)
	case r.Method == http.MethodPost && path == "" || path == "/":
		h.createAuthMethodPolicy(w, r, tc.TenantID)
	case r.Method == http.MethodPut && strings.HasPrefix(path, "/"):
		h.updateAuthMethodPolicy(w, r, tc.TenantID, strings.TrimPrefix(path, "/"))
	case r.Method == http.MethodDelete && strings.HasPrefix(path, "/"):
		h.deleteAuthMethodPolicy(w, r, tc.TenantID, strings.TrimPrefix(path, "/"))
	default:
		errors.WriteSimpleAPIError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
	}
}

func (h *Handler) listAuthMethodPolicies(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	if h.authMethodPolicyRepo == nil {
		writeJSON(w, http.StatusOK, []interface{}{})
		return
	}
	policies, err := h.authMethodPolicyRepo.ListByTenant(r.Context(), tenantID)
	if err != nil {
		errors.WriteSimpleAPIError(w, http.StatusInternalServerError, "INTERNAL", "failed to list policies")
		return
	}
	writeJSON(w, http.StatusOK, policies)
}

func (h *Handler) createAuthMethodPolicy(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	var req authMethodPolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.WriteSimpleAPIError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	if req.GroupID == "" {
		errors.WriteSimpleAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "group_id is required")
		return
	}

	policy := &repository.AuthMethodPolicy{
		ID:               uuid.New(),
		TenantID:         tenantID,
		GroupID:          req.GroupID,
		RequiredMethods:  req.RequiredMethods,
		ForbiddenMethods: req.ForbiddenMethods,
		Priority:         req.Priority,
	}

	if err := h.authMethodPolicyRepo.Create(r.Context(), policy); err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			errors.WriteSimpleAPIError(w, http.StatusConflict, "CONFLICT", "policy already exists for this group")
			return
		}
		errors.WriteSimpleAPIError(w, http.StatusInternalServerError, "INTERNAL", "failed to create policy")
		return
	}
	writeJSON(w, http.StatusCreated, policy)
}

func (h *Handler) updateAuthMethodPolicy(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID, idStr string) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		errors.WriteSimpleAPIError(w, http.StatusBadRequest, "INVALID_ID", "invalid policy ID")
		return
	}

	var req authMethodPolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.WriteSimpleAPIError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	policy := &repository.AuthMethodPolicy{
		ID:               id,
		TenantID:         tenantID,
		GroupID:          req.GroupID,
		RequiredMethods:  req.RequiredMethods,
		ForbiddenMethods: req.ForbiddenMethods,
		Priority:         req.Priority,
	}

	if err := h.authMethodPolicyRepo.Update(r.Context(), policy); err != nil {
		errors.WriteSimpleAPIError(w, http.StatusInternalServerError, "INTERNAL", "failed to update policy")
		return
	}
	writeJSON(w, http.StatusOK, policy)
}

func (h *Handler) deleteAuthMethodPolicy(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID, idStr string) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		errors.WriteSimpleAPIError(w, http.StatusBadRequest, "INVALID_ID", "invalid policy ID")
		return
	}

	if err := h.authMethodPolicyRepo.Delete(r.Context(), id, tenantID); err != nil {
		errors.WriteSimpleAPIError(w, http.StatusInternalServerError, "INTERNAL", "failed to delete policy")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
