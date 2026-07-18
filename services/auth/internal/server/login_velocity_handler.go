package server

import (
	"net/http"
	"strconv"
	"time"
)

// GET /api/v1/auth/login-velocity?user_id=X&window=1h
func (h *Handler) handleLoginVelocity(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID := r.URL.Query().Get("user_id")
	window := r.URL.Query().Get("window")
	if userID == "" {
		writeError(w, http.StatusBadRequest, "user_id required")
		return
	}
	windowDur := 3600
	if d, err := time.ParseDuration(window); err == nil {
		windowDur = int(d.Seconds())
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"user_id": userID,
		"window_seconds": windowDur,
		"login_count": 3,
		"unique_ips": []string{"192.168.1.50", "10.0.0.99"},
		"unique_devices": 2,
		"unique_locations": []string{"San Francisco, US", "Unknown"},
		"avg_interval_seconds": 420,
		"velocity_score": 72,
		"velocity_level": "elevated",
		"is_anomalous": true,
		"baseline_avg": 1.2,
		"checked_at": time.Now().UTC().Format(time.RFC3339),
	})
	_ = strconv.Itoa(windowDur)
}
