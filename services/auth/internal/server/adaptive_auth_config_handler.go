package server

import (
	"encoding/json"
	"net/http"
)

type AdaptiveAuthConfig struct {
	RiskThresholdMatrix map[string]string `json:"risk_threshold_matrix"`
	SignalWeights       map[string]float64 `json:"signal_weights"`
	StepUpTriggers      []string           `json:"step_up_triggers"`
	OverridePerRole     map[string]string  `json:"override_per_role"`
}

func (h *Handler) handleAdaptiveAuthConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		result := AdaptiveAuthConfig{
			RiskThresholdMatrix: map[string]string{
				"low":      "password",
				"medium":   "otp",
				"high":     "webauthn",
				"critical": "deny",
			},
			SignalWeights: map[string]float64{
				"geo_velocity":       0.25,
				"ip_reputation":      0.20,
				"device_familiarity": 0.20,
				"time_anomaly":       0.15,
				"failed_attempts":    0.10,
				"impossible_travel":  0.10,
			},
			StepUpTriggers: []string{
				"new_ip_address",
				"new_device",
				"sensitive_action: admin_console",
				"sensitive_action: delete_user",
				"sensitive_action: modify_policy",
				"impossible_travel_detected",
			},
			OverridePerRole: map[string]string{
				"admin": "high_threshold: webauthn",
				"auditor": "medium_threshold: otp",
				"service": "always_deny_if_medium",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	case http.MethodPut:
		var req AdaptiveAuthConfig
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
