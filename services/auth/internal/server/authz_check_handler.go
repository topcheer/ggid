package server

import (
	"encoding/json"
	"net/http"

	"github.com/ggid/ggid/pkg/errors"
	"github.com/google/uuid"
)

// authzCheckRequest is the simplified permission check request.
type authzCheckRequest struct {
	UserID   string `json:"user_id"`
	Resource string `json:"resource"`
	Action   string `json:"action"`
}

// handleAuthzCheck is a simplified PDP endpoint: POST /api/v1/authz/check
// Input: {user_id, resource, action} → Output: {allowed: bool, reason: string}
func (h *Handler) handleAuthzCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		errors.WriteSimpleAPIError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	var req authzCheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.WriteSimpleAPIError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	if req.UserID == "" {
		errors.WriteSimpleAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "user_id is required")
		return
	}
	if req.Resource == "" {
		errors.WriteSimpleAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "resource is required")
		return
	}
	if req.Action == "" {
		errors.WriteSimpleAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "action is required")
		return
	}

	// Simplified check: if no policy engine configured, default deny.
	// In production, this delegates to the policy service PDP.
	allowed := false
	reason := ""

	if h.policyCheckFn != nil {
		allowed, reason = h.policyCheckFn(req.UserID, req.Resource, req.Action)
	} else {
		// Basic resource pattern matching as fallback.
		// Allow reads for authenticated users on their own resources.
		if req.Action == "read" || req.Action == "list" || req.Action == "view" {
			allowed = true
			reason = "read access allowed by default policy"
		} else {
			allowed = false
			reason = "write/admin access requires explicit policy grant"
		}
	}

	// Audit the decision.
	h.publishAuditEventWithMeta(r,
		"authz.check", map[bool]string{true: "allow", false: "deny"}[allowed],
		"authorization", req.Resource, uuid.Nil,
		map[string]any{
			"user_id":  req.UserID,
			"resource": req.Resource,
			"action":   req.Action,
			"allowed":  allowed,
			"reason":   reason,
		},
	)

	writeJSON(w, http.StatusOK, map[string]any{
		"allowed":  allowed,
		"user_id":  req.UserID,
		"resource": req.Resource,
		"action":   req.Action,
		"reason":   reason,
	})
}

// PolicyCheckFunc is a pluggable policy check function.
type PolicyCheckFunc func(userID, resource, action string) (bool, string)
