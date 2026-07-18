package server

import (
	"encoding/json"
	"net/http"
	"time"
)

// POST /api/v1/auth/risk-notify
func (h *Handler) handleRiskNotify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed"); return
	}
	var req struct {
		UserID   string `json:"user_id"`
		EventType string `json:"event_type"` // new_device, impossible_travel, brute_force
		Channel  string `json:"channel"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeError(w, http.StatusBadRequest, "invalid JSON"); return }
	if req.UserID == "" || req.EventType == "" { writeError(w, http.StatusBadRequest, "user_id and event_type required"); return }
	channel := req.Channel
	if channel == "" { channel = "email" }
	var severity, message string
	switch req.EventType {
	case "new_device": severity = "medium"; message = "New device sign-in detected"
	case "impossible_travel": severity = "high"; message = "Impossible travel detected — possible account compromise"
	case "brute_force": severity = "critical"; message = "Multiple failed login attempts detected"
	default: severity = "low"; message = "Security event detected"
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "sent", "user_id": req.UserID, "event_type": req.EventType, "channel": channel, "severity": severity, "message": message, "timestamp": time.Now().UTC().Format(time.RFC3339)})
}
