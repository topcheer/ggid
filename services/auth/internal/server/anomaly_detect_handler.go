package server

import (
	"encoding/json"
	"net/http"
)

type AnomalyEvent struct {
	EventID    string  `json:"event_id"`
	Type       string  `json:"type"`
	Severity   string  `json:"severity"`
	UserID     string  `json:"user_id"`
	Timestamp  string  `json:"timestamp"`
	Confidence float64 `json:"confidence"`
}

type AnomalyDetectResult struct {
	AnomalyEvents     []AnomalyEvent `json:"anomaly_events"`
	DetectedPatterns  []string       `json:"detected_patterns"`
	AutoActionsTaken  []string       `json:"auto_actions_taken"`
	TotalDetected     int            `json:"total_detected"`
	CriticalCount     int            `json:"critical_count"`
}

func (h *Handler) handleAnomalyDetect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	result := AnomalyDetectResult{
		AnomalyEvents: []AnomalyEvent{
			{EventID: "ae-001", Type: "off_hours_login", Severity: "medium", UserID: "u-0342", Timestamp: "2025-01-15T03:00:00Z", Confidence: 0.72},
			{EventID: "ae-002", Type: "impossible_travel", Severity: "high", UserID: "u-0517", Timestamp: "2025-01-15T03:15:00Z", Confidence: 0.89},
			{EventID: "ae-003", Type: "new_device", Severity: "low", UserID: "u-0891", Timestamp: "2025-01-15T08:00:00Z", Confidence: 0.55},
			{EventID: "ae-004", Type: "unusual_resource_access", Severity: "high", UserID: "u-0342", Timestamp: "2025-01-15T03:20:00Z", Confidence: 0.84},
			{EventID: "ae-005", Type: "credential_stuffing_burst", Severity: "critical", UserID: "u-0420", Timestamp: "2025-01-15T02:45:00Z", Confidence: 0.95},
		},
		DetectedPatterns: []string{"off_hours_login", "impossible_travel", "new_device", "unusual_resource_access", "credential_stuffing_burst"},
		AutoActionsTaken: []string{"forced_mfa_challenge: u-0342", "session_terminated: u-0517", "rate_limit_applied: u-0420"},
		TotalDetected:    5,
		CriticalCount:    1,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
