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

// capRequest is the DTO for create/update operations.
type capRequest struct {
	Name       string                  `json:"name"`
	Conditions repository.Conditions   `json:"conditions"`
	Action     string                  `json:"action"`
	Priority   int                     `json:"priority"`
	Enabled    bool                    `json:"enabled"`
}

// capEvaluateRequest is the DTO for the evaluate endpoint.
type capEvaluateRequest struct {
	DevicePosture int    `json:"device_posture"`
	RiskScore     int    `json:"risk_score"`
	GeoCountry    string `json:"geo_country"`
	AuthMethod    string `json:"auth_method"`
	IPAddress     string `json:"ip_address"`
}

func (h *Handler) SetConditionalAccessRepo(repo *repository.ConditionalAccessRepository) {
	h.capRepo = repo
}

func (h *Handler) handleConditionalAccess(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/auth/conditional-access")

	switch {
	case path == "/evaluate" && r.Method == http.MethodPost:
		h.capEvaluate(w, r)
	case path == "/policies" && (r.Method == http.MethodGet || r.Method == http.MethodPost):
		h.capPolicies(w, r)
	case strings.HasPrefix(path, "/policies/") && (r.Method == http.MethodPut || r.Method == http.MethodDelete):
		h.capPolicyByID(w, r, strings.TrimPrefix(path, "/policies/"))
	default:
		errors.WriteSimpleAPIError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
	}
}

func (h *Handler) capPolicies(w http.ResponseWriter, r *http.Request) {
	tc, err := tenant.FromContext(r.Context())
	if err != nil {
		errors.WriteSimpleAPIError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.capList(w, r, tc.TenantID)
	case http.MethodPost:
		h.capCreate(w, r, tc.TenantID)
	}
}

func (h *Handler) capList(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	if h.capRepo == nil {
		writeJSON(w, http.StatusOK, []interface{}{})
		return
	}
	policies, err := h.capRepo.ListByTenant(r.Context(), tenantID)
	if err != nil {
		errors.WriteSimpleAPIError(w, http.StatusInternalServerError, "INTERNAL", "failed to list policies")
		return
	}
	writeJSON(w, http.StatusOK, policies)
}

func (h *Handler) capCreate(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	var req capRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.WriteSimpleAPIError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	if req.Name == "" {
		errors.WriteSimpleAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "name is required")
		return
	}
	if err := repository.ValidateAction(req.Action); err != nil {
		errors.WriteSimpleAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	policy := &repository.ConditionalAccessPolicy{
		ID:         uuid.New(),
		TenantID:   tenantID,
		Name:       req.Name,
		Conditions: req.Conditions,
		Action:     req.Action,
		Priority:   req.Priority,
		Enabled:    req.Enabled,
	}
	if policy.Action == "" {
		policy.Action = repository.ActionBlock
	}

	if h.capRepo != nil {
		if err := h.capRepo.Create(r.Context(), policy); err != nil {
			errors.WriteSimpleAPIError(w, http.StatusInternalServerError, "INTERNAL", "failed to create policy")
			return
		}
	}

	h.publishAuditEventWithMeta(r,
		"conditional_access.policy.create", "success",
		"conditional_access", policy.Name, policy.ID,
		map[string]any{"action": policy.Action, "priority": policy.Priority},
	)

	writeJSON(w, http.StatusCreated, policy)
}

func (h *Handler) capPolicyByID(w http.ResponseWriter, r *http.Request, idStr string) {
	tc, err := tenant.FromContext(r.Context())
	if err != nil {
		errors.WriteSimpleAPIError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context required")
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		errors.WriteSimpleAPIError(w, http.StatusBadRequest, "INVALID_ID", "invalid policy ID")
		return
	}

	switch r.Method {
	case http.MethodPut:
		h.capUpdate(w, r, tc.TenantID, id)
	case http.MethodDelete:
		h.capDelete(w, r, tc.TenantID, id)
	}
}

func (h *Handler) capUpdate(w http.ResponseWriter, r *http.Request, tenantID, id uuid.UUID) {
	var req capRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.WriteSimpleAPIError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	if err := repository.ValidateAction(req.Action); err != nil {
		errors.WriteSimpleAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	policy := &repository.ConditionalAccessPolicy{
		ID:         id,
		TenantID:   tenantID,
		Name:       req.Name,
		Conditions: req.Conditions,
		Action:     req.Action,
		Priority:   req.Priority,
		Enabled:    req.Enabled,
	}

	if h.capRepo != nil {
		if err := h.capRepo.Update(r.Context(), policy); err != nil {
			errors.WriteSimpleAPIError(w, http.StatusInternalServerError, "INTERNAL", "failed to update policy")
			return
		}
	}

	h.publishAuditEventWithMeta(r,
		"conditional_access.policy.update", "success",
		"conditional_access", policy.Name, id,
		map[string]any{"action": policy.Action, "priority": policy.Priority},
	)

	writeJSON(w, http.StatusOK, policy)
}

func (h *Handler) capDelete(w http.ResponseWriter, r *http.Request, tenantID, id uuid.UUID) {
	if h.capRepo != nil {
		if err := h.capRepo.Delete(r.Context(), id, tenantID); err != nil {
			errors.WriteSimpleAPIError(w, http.StatusInternalServerError, "INTERNAL", "failed to delete policy")
			return
		}
	}

	h.publishAuditEventWithMeta(r,
		"conditional_access.policy.delete", "success",
		"conditional_access", "", id,
		nil,
	)

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) capEvaluate(w http.ResponseWriter, r *http.Request) {
	tc, err := tenant.FromContext(r.Context())
	if err != nil {
		errors.WriteSimpleAPIError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context required")
		return
	}

	var req capEvaluateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.WriteSimpleAPIError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	if h.capRepo == nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"action":       repository.ActionAllow,
			"description":  repository.ActionDescription(repository.ActionAllow),
			"policy_name":  "",
			"matched":      false,
		})
		return
	}

	evalCtx := repository.EvalContext{
		DevicePosture: req.DevicePosture,
		RiskScore:     req.RiskScore,
		GeoCountry:    req.GeoCountry,
		AuthMethod:    req.AuthMethod,
		IPAddress:     req.IPAddress,
	}

	action, policy := h.capRepo.Evaluate(r.Context(), tc.TenantID, evalCtx)

	resp := map[string]any{
		"action":      action,
		"description": repository.ActionDescription(action),
		"matched":     policy != nil,
	}
	if policy != nil {
		resp["policy_id"] = policy.ID
		resp["policy_name"] = policy.Name
		resp["priority"] = policy.Priority
	}

	// Audit evaluation.
	h.publishAuditEventWithMeta(r,
		"conditional_access.evaluate", action,
		"conditional_access", "", uuid.Nil,
		map[string]any{
			"device_posture": req.DevicePosture,
			"risk_score":     req.RiskScore,
			"geo_country":    req.GeoCountry,
			"action":         action,
		},
	)

	writeJSON(w, http.StatusOK, resp)
}
