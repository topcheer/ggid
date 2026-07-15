package httpserver

import (
	"net/http"
	"time"

	"github.com/ggid/ggid/services/audit/internal/domain"
)

// GET /api/v1/security/dashboard
// Returns aggregate security metrics for the security center dashboard.
// Routed via gateway /api/v1/security prefix → audit service.
func (s *HTTPServer) handleSecurityDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Return structured response with typed empty arrays for frontend compatibility.
	// Security analytics data will be aggregated from auth events as the pipeline matures.
	resp := map[string]any{
		"total_active_sessions": 0,
		"failed_logins_24h":     0,
		"mfa_coverage_pct":      0,
		"blocked_ips":           0,
		"mfa_enrolled":          0,
		"mfa_not_enrolled":      0,
		"mfa_methods":           []map[string]any{},
		"session_locations":     []map[string]any{},
		"failed_login_chart":    []map[string]any{},
		"risky_ips":             []map[string]any{},
		"webauthn_devices":      []map[string]any{},
	}

	// Aggregate from audit events if service available
	if s.svc != nil {
		ctx := r.Context()
		// Count failed logins from audit events
		since := time.Now().Add(-24 * time.Hour)
		filter := domain.ListFilter{Action: "login", Result: "failure", StartTime: &since}
		if _, count, err := s.svc.ListEvents(ctx, filter, 1, 1); err == nil {
			resp["failed_logins_24h"] = count
		}
	}

	writeJSON(w, http.StatusOK, resp)
}
