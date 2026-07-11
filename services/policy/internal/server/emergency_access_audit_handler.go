package httpserver

import (
	"net/http"
)

// GET /api/v1/policies/emergency-access/audit
func (s *HTTPServer) handleEmergencyAccessAudit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed"); return
	}
	records := []map[string]any{
		{"id": "bg-001", "requester": "admin", "reason": "production incident - unable to access auth service", "scope": []string{"admin.users", "admin.policies"}, "activated_at": "2026-07-12T03:47:00Z", "deactivated_at": "2026-07-12T05:12:00Z", "duration": "1h25m", "actions_taken": []string{"disabled MFA for u-045", "reset password for u-067"}, "approved_by": "cto"},
		{"id": "bg-002", "requester": "devops", "reason": "emergency deployment rollback", "scope": []string{"prod.deploy"}, "activated_at": "2026-07-10T18:00:00Z", "deactivated_at": "2026-07-10T18:45:00Z", "duration": "45m", "actions_taken": []string{"rolled back deployment v3.2.1"}, "approved_by": "vp-eng"},
	}
	writeJSON(w, http.StatusOK, map[string]any{"break_glass_events": records, "total": len(records), "total_duration": "2h10m"})
}
