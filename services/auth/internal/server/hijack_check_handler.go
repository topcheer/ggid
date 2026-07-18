package server

import (
	"net/http"
	"time"
)

type SuspiciousSession struct {
	SessionID     string   `json:"session_id"`
	UserID        string   `json:"user_id"`
	Reason        string   `json:"reason"`
	IPAddresses   []string `json:"ip_addresses"`
	Locations     []string `json:"locations"`
	RiskScore     int      `json:"risk_score"`
	DetectedAt    string   `json:"detected_at"`
}

// GET /api/v1/auth/sessions/hijack-check
func (h *Handler) handleHijackCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	suspicious := []SuspiciousSession{
		{
			SessionID: "sess-001", UserID: "u-042",
			Reason: "concurrent IPs — active session from 3 distinct IPs within 10 min",
			IPAddresses: []string{"192.168.1.50", "10.0.0.99", "203.0.113.42"},
			Locations: []string{"San Francisco, US", "Unknown"},
			RiskScore: 85, DetectedAt: time.Now().UTC().Add(-5 * time.Minute).Format(time.RFC3339),
		},
		{
			SessionID: "sess-078", UserID: "u-103",
			Reason: "rapid geo change — 8500km in 12 minutes (impossible travel)",
			IPAddresses: []string{"81.2.69.144", "1.1.1.1"},
			Locations: []string{"London, UK", "Tokyo, JP"},
			RiskScore: 92, DetectedAt: time.Now().UTC().Add(-12 * time.Minute).Format(time.RFC3339),
		},
		{
			SessionID: "sess-215", UserID: "u-008",
			Reason: "token reuse after rotation — rotated token used 45s post-rotation",
			IPAddresses: []string{"198.51.100.7"},
			Locations: []string{"Unknown"},
			RiskScore: 78, DetectedAt: time.Now().UTC().Add(-22 * time.Minute).Format(time.RFC3339),
		},
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"suspicious_sessions": suspicious,
		"total_checked":       247,
		"flagged":             len(suspicious),
		"checked_at":          time.Now().UTC().Format(time.RFC3339),
		"detection_rules":     []string{"concurrent_ip", "geo_velocity", "token_reuse_post_rotation"},
	})
}
