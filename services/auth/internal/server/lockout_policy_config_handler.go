package server

import (
	"encoding/json"
	"net/http"
)

type LockoutPolicyConfig struct {
	MaxFailedAttempts     int               `json:"max_failed_attempts"`
	LockoutDurationMins   int               `json:"lockout_duration_minutes"`
	ProgressiveBackoff    bool              `json:"progressive_backoff"`
	PerEndpointConfig     map[string]int    `json:"per_endpoint_config"`
	CaptchaTriggerAfter   int               `json:"captcha_trigger_after"`
	AutoUnlockAfterMins   int               `json:"auto_unlock_after"`
}

func (h *Handler) handleLockoutPolicyConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		result := LockoutPolicyConfig{
			MaxFailedAttempts:   5,
			LockoutDurationMins: 30,
			ProgressiveBackoff:  true,
			PerEndpointConfig: map[string]int{
				"/api/v1/auth/login":     5,
				"/api/v1/auth/mfa/verify": 3,
				"/api/v1/oauth/token":    10,
			},
			CaptchaTriggerAfter: 3,
			AutoUnlockAfterMins: 30,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	case http.MethodPut:
		var req LockoutPolicyConfig
		json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"status": "updated", "config": req})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
