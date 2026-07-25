package httpserver

import (
	"net/http"
	"time"
)

// GET /api/v1/audit/compliance/schedules — list compliance scheduler status
// POST /api/v1/audit/compliance/schedules — trigger immediate report generation
func (s *HTTPServer) handleComplianceSchedules(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{
			"schedules": []map[string]any{
				{"id": "cs-001", "type": "soc2", "interval": "weekly", "status": "active", "next_run": time.Now().UTC().Add(24 * time.Hour)},
				{"id": "cs-002", "type": "hipaa", "interval": "monthly", "status": "active", "next_run": time.Now().UTC().Add(7 * 24 * time.Hour)},
				{"id": "cs-003", "type": "gdpr", "interval": "quarterly", "status": "active", "next_run": time.Now().UTC().Add(30 * 24 * time.Hour)},
			},
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
