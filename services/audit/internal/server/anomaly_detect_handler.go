package httpserver

import (
	"encoding/json"
	"net/http"
	"time"
)

type Anomaly struct {
	Type          string   `json:"type"`
	Confidence    float64  `json:"confidence"`
	Description   string   `json:"description"`
	RelatedEvents []string `json:"related_events"`
	DetectedAt    string   `json:"detected_at"`
}

// POST /api/v1/audit/anomalies/detect
func (s *HTTPServer) handleAnomalyDetect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		TimeWindow string `json:"time_window"` // e.g. "24h"
		TenantID   string `json:"tenant_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeJSONError(w, http.StatusBadRequest, "invalid request body"); return }

	// Simulated anomaly detection based on baseline deviations
	anomalies := []Anomaly{
		{
			Type: "unusual_login_time", Confidence: 0.87,
			Description: "User logged in at 3:47 AM, outside baseline hours (8AM-7PM)",
			RelatedEvents: []string{"evt-2026-07-12-0347", "evt-2026-07-12-0348"},
			DetectedAt:    time.Now().UTC().Format(time.RFC3339),
		},
		{
			Type: "mass_permission_changes", Confidence: 0.94,
			Description: "42 permission grants in 10 minutes (baseline: 2-3/day)",
			RelatedEvents: []string{"evt-perm-batch-001", "evt-perm-batch-002"},
			DetectedAt:    time.Now().UTC().Format(time.RFC3339),
		},
		{
			Type: "bulk_data_export", Confidence: 0.91,
			Description: "Exported 15,000 user records (baseline: <100/day)",
			RelatedEvents: []string{"evt-export-large-001"},
			DetectedAt:    time.Now().UTC().Format(time.RFC3339),
		},
		{
			Type: "impossible_travel", Confidence: 0.79,
			Description: "Login from Tokyo 5 min after login from London (impossible distance)",
			RelatedEvents: []string{"evt-login-london", "evt-login-tokyo"},
			DetectedAt:    time.Now().UTC().Format(time.RFC3339),
		},
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"tenant_id":       req.TenantID,
		"time_window":     req.TimeWindow,
		"anomalies":       anomalies,
		"total_anomalies": len(anomalies),
		"high_confidence": countHighConfidence(anomalies),
		"analyzed_at":     time.Now().UTC().Format(time.RFC3339),
	})
}

func countHighConfidence(anomalies []Anomaly) int {
	count := 0
	for _, a := range anomalies {
		if a.Confidence >= 0.9 {
			count++
		}
	}
	return count
}
