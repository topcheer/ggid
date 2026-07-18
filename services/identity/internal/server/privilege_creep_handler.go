package server

import (
	"encoding/json"
	"net/http"
	"strings"

	ggidtenant "github.com/ggid/ggid/pkg/tenant"
	"github.com/google/uuid"
)

// handlePrivilegeCreep routes privilege creep API requests.
// GET  /api/v1/identity/privilege-creep/alerts        list alerts
// GET  /api/v1/identity/privilege-creep/diff/:user_id  user diff
// POST /api/v1/identity/privilege-creep/scan          trigger scan
func (h *HTTPHandler) handlePrivilegeCreep(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/identity/privilege-creep")
	path = strings.TrimPrefix(path, "/")

	if path == "" || path == "alerts" {
		if r.Method == http.MethodGet {
			h.pcListAlerts(w, r)
			return
		}
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if path == "scan" {
		if r.Method == http.MethodPost {
			h.pcTriggerScan(w, r)
			return
		}
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if strings.HasPrefix(path, "diff/") {
		if r.Method == http.MethodGet {
			h.pcUserDiff(w, r, strings.TrimPrefix(path, "diff/"))
			return
		}
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	writeJSONError(w, http.StatusNotFound, "not found")
}

func (h *HTTPHandler) pcListAlerts(w http.ResponseWriter, r *http.Request) {
	if h.pcRepo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "privilege creep not configured")
		return
	}
	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "X-Tenant-ID header required")
		return
	}
	alerts, err := h.pcRepo.ListAlerts(r.Context(), tc.TenantID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to list alerts")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"alerts": alerts,
		"count":  len(alerts),
	})
}

func (h *HTTPHandler) pcUserDiff(w http.ResponseWriter, r *http.Request, userIDStr string) {
	if h.pcRepo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "privilege creep not configured")
		return
	}
	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "X-Tenant-ID header required")
		return
	}
	if _, err := uuid.Parse(userIDStr); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid user_id")
		return
	}

	// In a full implementation, fetch user's actual permissions and roles
	// from the identity service. For now, return a placeholder diff.
	// The ComputeDiff function is the core logic, callable with real data.
	roles := r.URL.Query()["role"]
	expected := r.URL.Query()["expected"]
	actual := r.URL.Query()["actual"]

	if len(roles) == 0 && len(expected) == 0 && len(actual) == 0 {
		// No query params: return empty diff for this user.
		writeJSON(w, http.StatusOK, &PrivilegeDiff{
			UserID: userIDStr,
		})
		return
	}

	diff := ComputeDiff(userIDStr, roles, expected, actual)
	_ = tc // tenant resolved for future DB-backed diff
	writeJSON(w, http.StatusOK, diff)
}

func (h *HTTPHandler) pcTriggerScan(w http.ResponseWriter, r *http.Request) {
	if h.pcRepo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "privilege creep not configured")
		return
	}
	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "X-Tenant-ID header required")
		return
	}

	// In production: fetch all users, their roles, and effective permissions
	// from the identity service, then run the scan.
	// For manual trigger, accept a JSON body with the data for testing.
	var req struct {
		UserPermissions map[string][]string `json:"user_permissions"`
		UserRoles       map[string][]string `json:"user_roles"`
		ActiveUserIDs   []string            `json:"active_user_ids"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	if len(req.ActiveUserIDs) == 0 {
		writeJSON(w, http.StatusOK, map[string]any{
			"status":  "completed",
			"alerts":  0,
			"message": "no users to scan",
		})
		return
	}

	count, err := h.pcRepo.RunScan(r.Context(), tc.TenantID, req.UserPermissions, req.UserRoles, req.ActiveUserIDs)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "scan failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":    "completed",
		"alerts":    count,
		"users_scanned": len(req.ActiveUserIDs),
	})
}

// handlePrivilegeBaseline sets role permission baselines.
// PUT /api/v1/policy/roles/:id/baseline
func (h *HTTPHandler) handlePrivilegeBaseline(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if h.pcRepo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "privilege creep not configured")
		return
	}
	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "X-Tenant-ID header required")
		return
	}

	// Extract role_id from path: /api/v1/policy/roles/:id/baseline
	roleID := extractRoleIDFromPath(r.URL.Path)
	if roleID == "" {
		writeJSONError(w, http.StatusBadRequest, "role_id required in path")
		return
	}

	var req struct {
		StandardPermissions []string `json:"standard_permissions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if len(req.StandardPermissions) == 0 {
		writeJSONError(w, http.StatusBadRequest, "standard_permissions required")
		return
	}

	baseline := &PrivilegeBaseline{
		RoleID:              roleID,
		TenantID:            tc.TenantID.String(),
		StandardPermissions: req.StandardPermissions,
	}
	if err := h.pcRepo.SetBaseline(r.Context(), baseline); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to set baseline")
		return
	}
	writeJSON(w, http.StatusOK, baseline)
}

// extractRoleIDFromPath parses /api/v1/policy/roles/:id/baseline → :id
func extractRoleIDFromPath(path string) string {
	if !strings.Contains(path, "/baseline") {
		return ""
	}
	segments := strings.Split(path, "/")
	for i, s := range segments {
		if s == "roles" && i+1 < len(segments) {
			return segments[i+1]
		}
	}
	return ""
}
