package server

import (
	"encoding/json"
	"net/http"
)

type BruteForceConfig struct {
	MaxFailedAttempts        int               `json:"max_failed_attempts"`
	LockoutDurationMinutes   int               `json:"lockout_duration_minutes"`
	ProgressiveBackoff       bool              `json:"progressive_backoff"`
	PerEndpointOverrides     map[string]int    `json:"per_endpoint_overrides"`
	IPAllowlist              []string          `json:"ip_allowlist"`
	CaptchaTriggerAfter      int               `json:"captcha_trigger_after"`
	AutoUnlockAfterMinutes   int               `json:"auto_unlock_after"`
}

func (h *Handler) handleBruteForceConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		result := BruteForceConfig{
			MaxFailedAttempts:      5,
			LockoutDurationMinutes: 30,
			ProgressiveBackoff:     true,
			PerEndpointOverrides: map[string]int{
				"/api/v1/auth/login":    5,
				"/api/v1/auth/mfa/verify": 3,
				"/api/v1/oauth/token":  10,
			},
			IPAllowlist:            []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"},
			CaptchaTriggerAfter:    3,
			AutoUnlockAfterMinutes: 30,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	case http.MethodPut:
		var req BruteForceConfig
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"status": "updated", "config": req})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
