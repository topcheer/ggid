package server

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// passwordSprayAttempt records a single failed login attempt for spray detection.
type passwordSprayAttempt struct {
	Username  string    `json:"username"`
	Timestamp time.Time `json:"timestamp"`
	IPAddress string    `json:"ip_address"`
}

var sprayTracker = struct {
	sync.RWMutex
	attempts []passwordSprayAttempt
}{attempts: []passwordSprayAttempt{}}

// POST /api/v1/auth/detect-password-spray
// Body: {"password": "...", "time_window_minutes": 15, "threshold": 5}
// Detects single_password → multiple_users pattern in a short time window.
func (h *Handler) handleDetectPasswordSpray(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		Password          string `json:"password"`
		TimeWindowMinutes int    `json:"time_window_minutes"`
		Threshold         int    `json:"threshold"`
		IPAddress         string `json:"ip_address"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.TimeWindowMinutes <= 0 {
		req.TimeWindowMinutes = 15
	}
	if req.Threshold <= 0 {
		req.Threshold = 5
	}

	// Record this check attempt as a data point (simulating real failed login tracking)
	if req.Password != "" {
		sprayTracker.Lock()
		sprayTracker.attempts = append(sprayTracker.attempts, passwordSprayAttempt{
			Username:  "spray-check",
			Timestamp: time.Now().UTC(),
			IPAddress: req.IPAddress,
		})
		// Keep only last 1000 entries
		if len(sprayTracker.attempts) > 1000 {
			sprayTracker.attempts = sprayTracker.attempts[len(sprayTracker.attempts)-1000:]
		}
		sprayTracker.Unlock()
	}

	// Analyze recent attempts within the time window
	cutoff := time.Now().UTC().Add(-time.Duration(req.TimeWindowMinutes) * time.Minute)
	affectedUsers := map[string]bool{}
	uniqueIPs := map[string]bool{}

	sprayTracker.RLock()
	for _, a := range sprayTracker.attempts {
		if a.Timestamp.After(cutoff) && a.Username != "spray-check" {
			affectedUsers[a.Username] = true
			if a.IPAddress != "" {
				uniqueIPs[a.IPAddress] = true
			}
		}
	}
	sprayTracker.RUnlock()

	userCount := len(affectedUsers)
	isDetected := userCount >= req.Threshold

	// Confidence calculation based on affected user count
	confidence := 0.0
	if userCount > 0 {
		confidence = float64(userCount) / float64(userCount+req.Threshold) * 100
		if isDetected {
			confidence = 90.0 + float64(userCount-req.Threshold)*2
			if confidence > 99.9 {
				confidence = 99.9
			}
		}
	}

	affectedList := make([]string, 0, len(affectedUsers))
	for u := range affectedUsers {
		affectedList = append(affectedList, u)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"is_detected":          isDetected,
		"confidence":           confidence,
		"affected_users":       affectedList,
		"affected_user_count":  userCount,
		"unique_ip_count":      len(uniqueIPs),
		"time_window_minutes":  req.TimeWindowMinutes,
		"threshold":            req.Threshold,
		"recommended_action": func() string {
			if isDetected {
				return "lock_accounts_and_notify"
			}
			if userCount > 0 {
				return "monitor"
			}
			return "none"
		}(),
		"checked_at": time.Now().UTC().Format(time.RFC3339),
	})
}
