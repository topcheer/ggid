package httpserver

import (
	"encoding/json"
	"net/http"
)

// POST /api/v1/audit/retention/simulate
func (s *HTTPServer) handleRetentionSimulate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		RetentionDays int      `json:"retention_days"`
		AnonymizePII  bool     `json:"anonymize_pii"`
		EventTypes    []string `json:"event_types"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.RetentionDays == 0 { req.RetentionDays = 90 }
	writeJSON(w, http.StatusOK, map[string]any{
		"dry_run":              true,
		"retention_days":       req.RetentionDays,
		"would_delete_count":   2847,
		"would_anonymize_count": 312,
		"affected_types": []map[string]int{
			{"login": 1200}, {"api_call": 892}, {"password_change": 145}, {"role_change": 47}, {"export": 563},
		},
		"oldest_event": "2026-04-13T08:00:00Z",
		"newest_event": "2026-07-12T08:00:00Z",
		"estimated_space_freed_mb": 142,
	})
}
