package httpserver

import (
	"net/http"
	"time"
)

// GET /api/v1/policies/standing-access
func (s *HTTPServer) handleStandingAccess(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed"); return
	}
	access := []map[string]any{
		{"user_id": "u-001", "username": "admin", "resource": "/secrets/vault", "access_type": "standing", "duration": "permanent", "recommendation": "convert_to_jit", "max_duration": "4h", "approval_required": true},
		{"user_id": "u-004", "username": "bwang", "resource": "/policies", "access_type": "standing", "duration": "permanent", "recommendation": "convert_to_jit", "max_duration": "8h", "approval_required": true},
		{"user_id": "u-012", "username": "devops", "resource": "/prod/deploy", "access_type": "standing", "duration": "permanent", "recommendation": "convert_to_jit", "max_duration": "2h", "approval_required": false},
	}
	writeJSON(w, http.StatusOK, map[string]any{"standing_access": access, "total": len(access), "jit_recommended": len(access), "generated_at": time.Now().UTC().Format(time.RFC3339)})
}
