package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

// hasScope checks if a specific scope exists in the X-Scopes header using
// exact comma-delimited matching (not substring) to avoid false positives
// like "notplatform:admin" matching "platform:admin".
func hasScope(scopesHeader, target string) bool {
	for _, s := range strings.Split(scopesHeader, ",") {
		if strings.TrimSpace(s) == target {
			return true
		}
	}
	return false
}

// handleSuspendTenant suspends a tenant, blocking all logins and API access.
// POST /api/v1/org/tenants/suspend
// Body: {"tenant_id": "<uuid>", "reason": "optional"}
func (s *HTTPServer) handleSuspendTenant(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if s.tenantSvc == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "tenant service not configured")
		return
	}

	// SECURITY (P0): suspending/activating a tenant is a platform-admin
	// operation. The gateway strips and re-derives X-Scopes from verified
	// JWT claims, so it is trustworthy here. Previously ANY caller could
	// DoS an entire tenant.
	if !hasScope(r.Header.Get("X-Scopes"), "platform:admin") {
		writeJSONError(w, http.StatusForbidden, "platform:admin scope required")
		return
	}

	var req struct {
		TenantID string `json:"tenant_id"`
		Reason   string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.TenantID == "" {
		writeJSONError(w, http.StatusBadRequest, "tenant_id is required")
		return
	}

	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid tenant_id")
		return
	}

	if err := s.tenantSvc.Suspend(r.Context(), tenantID); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "suspend failed")
		return
	}

	actorID, _ := uuid.Parse(r.Header.Get("X-User-ID"))
	s.publishAuditEvent("tenant.suspend", "success", "tenants", tenantID, actorID)

	writeJSON(w, http.StatusOK, map[string]any{
		"tenant_id": req.TenantID,
		"status":    "suspended",
		"reason":    req.Reason,
	})
}

// handleActivateTenant reactivates a previously suspended tenant.
// POST /api/v1/org/tenants/activate
// Body: {"tenant_id": "<uuid>"}
func (s *HTTPServer) handleActivateTenant(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if s.tenantSvc == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "tenant service not configured")
		return
	}

	// SECURITY (P0): platform-admin only — see handleSuspendTenant.
	if !hasScope(r.Header.Get("X-Scopes"), "platform:admin") {
		writeJSONError(w, http.StatusForbidden, "platform:admin scope required")
		return
	}

	var req struct {
		TenantID string `json:"tenant_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.TenantID == "" {
		writeJSONError(w, http.StatusBadRequest, "tenant_id is required")
		return
	}

	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid tenant_id")
		return
	}

	if err := s.tenantSvc.Activate(r.Context(), tenantID); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "activate failed")
		return
	}

	actorID2, _ := uuid.Parse(r.Header.Get("X-User-ID"))
	s.publishAuditEvent("tenant.activate", "success", "tenants", tenantID, actorID2)

	writeJSON(w, http.StatusOK, map[string]any{
		"tenant_id": req.TenantID,
		"status":    "active",
	})
}
