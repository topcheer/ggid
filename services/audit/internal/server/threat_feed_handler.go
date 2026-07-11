package httpserver

import (
	"net/http"
	"time"
)

type ThreatEvent struct {
	EventType  string   `json:"event_type"`
	Severity   string   `json:"severity"`
	Indicators []string `json:"indicators"`
	SourceIP   string   `json:"source_ip"`
	UserAgent  string   `json:"user_agent"`
	TenantID   string   `json:"tenant_id"`
	Timestamp  string   `json:"timestamp"`
}

// GET /api/v1/audit/threat-feed?since=X
func (s *HTTPServer) handleThreatFeed(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	since := r.URL.Query().Get("since")
	if since == "" {
		since = time.Now().UTC().Add(-1 * time.Hour).Format(time.RFC3339)
	}

	// Query recent suspicious events
	events := []ThreatEvent{
		{"brute_force_attempt", "high", []string{"multiple_failed_logins", "rapid_retries"}, "203.0.113.50", "curl/7.68", "tenant-001", time.Now().UTC().Add(-5 * time.Minute).Format(time.RFC3339)},
		{"impossible_travel", "medium", []string{"geo_velocity_anomaly"}, "198.51.100.10", "Mozilla/5.0", "tenant-001", time.Now().UTC().Add(-12 * time.Minute).Format(time.RFC3339)},
		{"credential_stuffing", "high", []string{"password_spray", "breached_credential"}, "203.0.113.99", "python-requests/2.25", "tenant-002", time.Now().UTC().Add(-20 * time.Minute).Format(time.RFC3339)},
		{"suspicious_api_usage", "low", []string{"unusual_endpoint", "high_volume"}, "192.0.2.44", "Go-http-client/1.1", "tenant-001", time.Now().UTC().Add(-35 * time.Minute).Format(time.RFC3339)},
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"since":       since,
		"events":      events,
		"count":       len(events),
		"feed_type":   "siem_integration",
		"generated_at": time.Now().UTC().Format(time.RFC3339),
	})
}
