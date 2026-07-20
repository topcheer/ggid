package httpserver

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
)

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
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.publishAuditEvent("tenant.suspend", "success", "tenants", tenantID, uuid.Nil)

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
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.publishAuditEvent("tenant.activate", "success", "tenants", tenantID, uuid.Nil)

	writeJSON(w, http.StatusOK, map[string]any{
		"tenant_id": req.TenantID,
		"status":    "active",
	})
}
