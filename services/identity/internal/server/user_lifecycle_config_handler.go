package server

import (
	"encoding/json"
	"net/http"
)

type UserLifecycleConfig struct {
	AutoDeactivateAfterDays int               `json:"auto_deactivate_after_days"`
	DormantDetectionRules   []string          `json:"dormant_detection_rules"`
	StageTransitions        map[string]string `json:"stage_transitions"`
	NotificationBeforeDays  int               `json:"notification_before_days"`
	PerRoleOverride         map[string]int    `json:"per_role_override"`
}

func (h *HTTPHandler) handleUserLifecycleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		result := UserLifecycleConfig{
			AutoDeactivateAfterDays: 180,
			DormantDetectionRules:   []string{"no_login_90d", "no_api_calls_90d", "no_session_90d"},
			StageTransitions: map[string]string{
				"active→dormant":     "no_activity_90d",
				"dormant→active":     "any_login",
				"dormant→suspended":  "no_activity_180d",
				"suspended→deactivated": "admin_confirm_30d",
				"pending→active":     "email_verified",
			},
			NotificationBeforeDays: 14,
			PerRoleOverride: map[string]int{"admin": 90, "service": 365, "viewer": 120},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	case http.MethodPut:
		var req UserLifecycleConfig
		json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"status": "updated", "config": req})
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
