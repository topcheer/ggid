package server

import (
	"encoding/json"
	"net/http"
)

type HijackEvent struct {
	Timestamp    string  `json:"timestamp"`
	EventType    string  `json:"event_type"`
	Detail       string  `json:"detail"`
	Severity     string  `json:"severity"`
	Location    string  `json:"location,omitempty"`
	IPAddress   string  `json:"ip_address,omitempty"`
}

type HijackTimelineResult struct {
	UserID           string         `json:"user_id"`
	SuspiciousLogins []HijackEvent  `json:"suspicious_logins"`
	GeoJumps         []HijackEvent  `json:"geo_jumps"`
		DeviceChanges    []HijackEvent  `json:"device_changes"`
	IPChanges        []HijackEvent  `json:"ip_changes"`
	ConfidenceScore  float64        `json:"confidence_score"`
	RecommendedActions []string     `json:"recommended_actions"`
}

func (h *Handler) handleHijackTimeline(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		userID = "u-001"
	}

	result := HijackTimelineResult{
		UserID: userID,
		SuspiciousLogins: []HijackEvent{
			{Timestamp: "2025-01-15T03:22:00Z", EventType: "login", Detail: "Login from unrecognized device", Severity: "high", Location: "Unknown", IPAddress: "203.0.113.50"},
			{Timestamp: "2025-01-15T03:45:00Z", EventType: "login", Detail: "Login outside normal hours", Severity: "medium", Location: "Singapore", IPAddress: "203.0.113.51"},
		},
		GeoJumps: []HijackEvent{
			{Timestamp: "2025-01-15T03:30:00Z", EventType: "geo_jump", Detail: "Impossible travel: SF → Singapore in 8min", Severity: "critical", Location: "Singapore", IPAddress: "203.0.113.51"},
		},
		DeviceChanges: []HijackEvent{
			{Timestamp: "2025-01-15T03:22:00Z", EventType: "device_change", Detail: "New device fingerprint: unknown-chrome-win", Severity: "high"},
		},
		IPChanges: []HijackEvent{
			{Timestamp: "2025-01-15T03:20:00Z", EventType: "ip_change", Detail: "IP changed from 10.0.0.5 to 203.0.113.50", Severity: "medium", IPAddress: "203.0.113.50"},
			{Timestamp: "2025-01-15T03:40:00Z", EventType: "ip_change", Detail: "IP changed from 203.0.113.50 to 203.0.113.51", Severity: "medium", IPAddress: "203.0.113.51"},
		},
		ConfidenceScore: 0.87,
		RecommendedActions: []string{
			"Force immediate password reset",
			"Revoke all active sessions",
			"Enable MFA enrollment",
			"Block source IPs 203.0.113.50/31",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
