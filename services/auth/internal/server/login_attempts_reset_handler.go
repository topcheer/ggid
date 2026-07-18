package server

import (
	"net/http"
	"strings"

	ggidtenant "github.com/ggid/ggid/pkg/tenant"
	"github.com/google/uuid"
)

// handleLoginAttemptsReset clears brute force counters for a specific user.
// DELETE /api/v1/auth/login-attempts/:username
//
// Admin-only endpoint. Clears Redis lockout key so the user can immediately
// retry login without waiting for the lockout window to expire.
func (h *Handler) handleLoginAttemptsReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract username from path.
	username := strings.TrimPrefix(r.URL.Path, "/api/v1/auth/login-attempts/")
	if username == "" || strings.Contains(username, "/") {
		writeError(w, http.StatusBadRequest, "username required in path")
		return
	}

	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "valid X-Tenant-ID header required")
		return
	}

	// Reset failed login counter in Redis.
	h.authSvc.ResetFailedLogins(r.Context(), tc.TenantID, username)

	// Audit the admin action.
	adminID, _ := uuid.Parse(r.Header.Get("X-User-ID"))
	h.publishAuditEvent("auth.login_attempts.reset", "success", tc.TenantID, adminID)

	writeJSON(w, http.StatusOK, map[string]any{
		"status":   "cleared",
		"username": username,
		"message":  "Login attempt counter reset. User can retry immediately.",
	})
}
