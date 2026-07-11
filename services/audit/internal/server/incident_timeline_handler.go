package httpserver

import (
	"net/http"
	"strings"
	"time"
)

// GET /api/v1/audit/incidents/{id}/timeline
func (s *HTTPServer) handleIncidentTimeline(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	incidentID := strings.TrimPrefix(r.URL.Path, "/api/v1/audit/incidents/")
	incidentID = strings.TrimSuffix(incidentID, "/timeline")
	if incidentID == "" {
		writeJSONError(w, http.StatusBadRequest, "incident_id required")
		return
	}

	timeline := []map[string]any{
		{"timestamp": "2026-07-12T03:47:00Z", "phase": "detection", "event": "anomaly detected — unusual login time", "source": "anomaly_engine", "severity": "medium"},
		{"timestamp": "2026-07-12T03:48:00Z", "phase": "detection", "event": "correlated with impossible travel alert", "source": "correlation_engine", "severity": "high"},
		{"timestamp": "2026-07-12T03:52:00Z", "phase": "escalation", "event": "incident created, severity escalated to high", "source": "soc_analyst", "severity": "high"},
		{"timestamp": "2026-07-12T03:55:00Z", "phase": "response", "event": "user sessions revoked", "source": "auto_response", "severity": "high"},
		{"timestamp": "2026-07-12T04:01:00Z", "phase": "response", "event": "source IP blocked across all services", "source": "auto_response", "severity": "high"},
		{"timestamp": "2026-07-12T04:15:00Z", "phase": "response", "event": "user notified via email + push", "source": "notification_service", "severity": "medium"},
		{"timestamp": "2026-07-12T06:30:00Z", "phase": "resolution", "event": "user confirmed legitimate access from new location", "source": "user_confirmation", "severity": "low"},
		{"timestamp": "2026-07-12T06:35:00Z", "phase": "resolution", "event": "incident closed — verified as false positive", "source": "soc_analyst", "severity": "low"},
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"incident_id":   incidentID,
		"timeline":      timeline,
		"total_events":  len(timeline),
		"current_phase": "resolution",
		"status":        "closed",
		"duration":      "2h48m",
		"generated_at":  time.Now().UTC().Format(time.RFC3339),
	})
}
