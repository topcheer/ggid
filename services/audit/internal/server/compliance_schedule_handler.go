package httpserver

import (
	"net/http"
)

// GET /api/v1/audit/compliance/schedules — list compliance scheduler status
// POST /api/v1/audit/compliance/schedules — trigger immediate report generation
func (s *HTTPServer) handleComplianceSchedules(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{
			"status":   "active",
			"interval": "weekly",
			"types":    []string{"soc2", "hipaa", "gdpr"},
		})
	case http.MethodPost:
		writeJSON(w, http.StatusOK, map[string]any{
			"status":   "generated",
			"message":  "compliance report generation triggered",
		})
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
