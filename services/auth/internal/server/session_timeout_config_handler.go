package server

import (
	"encoding/json"
	"net/http"
)

type SessionTimeoutConfig struct {
	IdleTimeoutMinutes      int               `json:"idle_timeout_minutes"`
	AbsoluteTimeoutHours    int               `json:"absolute_timeout_hours"`
	WarningBeforeMinutes    int               `json:"warning_before_minutes"`
	PerRoleOverride         map[string]int    `json:"per_role_override"`
	GracePeriodOnMobile     int               `json:"grace_period_on_mobile_minutes"`
	EnforceOnMobile         bool              `json:"enforce_on_mobile"`
}

func (h *Handler) handleSessionTimeoutConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		result := SessionTimeoutConfig{
			IdleTimeoutMinutes:   30,
			AbsoluteTimeoutHours: 8,
			WarningBeforeMinutes: 5,
			PerRoleOverride: map[string]int{
				"admin":    15,
				"viewer":   120,
				"service":  480,
			},
			GracePeriodOnMobile: 60,
			EnforceOnMobile:     false,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	case http.MethodPut:
		var req SessionTimeoutConfig
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"status": "updated", "config": req})
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
