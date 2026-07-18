package server

import (
	"encoding/json"
	"net/http"
)

type MFAChallengeConfig struct {
	MethodPriority       []string         `json:"method_priority"`
	RequireStepUpFor     []string         `json:"require_step_up_for"`
	ChallengeFrequency   string           `json:"challenge_frequency"`
	FallbackMethod       string           `json:"fallback_method"`
	GracePeriodMinutes   int              `json:"grace_period_minutes"`
}

func (h *Handler) handleMFAChallengeConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		result := MFAChallengeConfig{
			MethodPriority:     []string{"webauthn", "totp", "sms", "email"},
			RequireStepUpFor:   []string{"admin:delete_user", "admin:modify_policy", "admin:reset_password", "billing:change_plan", "security:rotate_keys"},
			ChallengeFrequency: "threshold_minutes",
			FallbackMethod:     "totp",
			GracePeriodMinutes: 30,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	case http.MethodPut:
		var req MFAChallengeConfig
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"status": "updated", "config": req})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
