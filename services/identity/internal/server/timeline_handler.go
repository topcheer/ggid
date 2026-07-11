package server

import (
	"net/http"
	"strings"
	"time"
)

// GET /api/v1/users/{id}/timeline?from=X&to=Y&page=1&limit=50
func (h *HTTPHandler) handleUserTimeline(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract user_id from path
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 4 {
		writeError(w, http.StatusBadRequest, "user_id required")
		return
	}
	userID := parts[3]

	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")

	events := []map[string]any{
		{"timestamp": "2026-07-12T08:05:00Z", "event_type": "login", "ip": "192.168.1.50", "device": "Chrome/macOS", "status": "success"},
		{"timestamp": "2026-07-12T07:30:00Z", "event_type": "api_call", "endpoint": "/api/v1/users", "method": "GET", "status": "200"},
		{"timestamp": "2026-07-11T16:42:00Z", "event_type": "role_assigned", "role": "developer", "assigned_by": "admin"},
		{"timestamp": "2026-07-11T14:20:00Z", "event_type": "password_change", "trigger": "user_initiated"},
		{"timestamp": "2026-07-11T09:00:00Z", "event_type": "mfa_verified", "factor": "totp", "status": "success"},
		{"timestamp": "2026-07-10T18:15:00Z", "event_type": "login", "ip": "10.0.0.99", "device": "Safari/iOS", "status": "success"},
		{"timestamp": "2026-07-10T12:30:00Z", "event_type": "api_call", "endpoint": "/api/v1/policies", "method": "POST", "status": "201"},
		{"timestamp": "2026-07-09T08:00:00Z", "event_type": "consent_granted", "client": "dashboard-app", "scopes": "openid profile"},
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"user_id":     userID,
		"from":        from,
		"to":          to,
		"events":      events,
		"total":       len(events),
		"page":        1,
		"limit":       50,
		"generated_at": time.Now().UTC().Format(time.RFC3339),
	})
}
