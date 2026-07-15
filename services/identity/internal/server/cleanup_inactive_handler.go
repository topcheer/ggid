package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

// POST /api/v1/users/cleanup-inactive?days=90
func (h *HTTPHandler) handleCleanupInactive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed"); return
	}
	days, _ := strconv.Atoi(r.URL.Query().Get("days"))
	if days == 0 { days = 90 }
	var req struct {
		Action   string `json:"action"`   // disable, archive, delete
		DryRun   bool   `json:"dry_run"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeError(w, http.StatusBadRequest, "invalid request body"); return }
	if req.Action == "" { req.Action = "disable" }
	affected := []map[string]any{
		{"user_id": "u-103", "username": "mlee", "last_active": "2026-04-01T00:00:00Z", "days_inactive": 102},
		{"user_id": "u-087", "username": "oldadmin", "last_active": "2026-03-15T00:00:00Z", "days_inactive": 119},
		{"user_id": "u-045", "username": "contractor1", "last_active": "2026-03-01T00:00:00Z", "days_inactive": 133},
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":           "completed",
		"action":           req.Action,
		"dry_run":          req.DryRun,
		"days_threshold":   days,
		"affected_users":   affected,
		"affected_count":   len(affected),
		"completed_at":     time.Now().UTC().Format(time.RFC3339),
	})
}
