package server

import (
	"encoding/json"
	"net/http"
)

type RiskScoringWeights struct {
	GeoVelocity      float64 `json:"geo_velocity"`
	IPReputation     float64 `json:"ip_reputation"`
	DeviceFamiliarity float64 `json:"device_familiarity"`
	TimeAnomaly      float64 `json:"time_anomaly"`
	FailedAttempts   float64 `json:"failed_attempts"`
}

type RiskThreshold struct {
	Level   string  `json:"level"`
	MinScore float64 `json:"min_score"`
	MaxScore float64 `json:"max_score"`
	Action  string  `json:"action"`
}

type RiskScoringConfig struct {
	Weights            RiskScoringWeights `json:"weights"`
	Thresholds         []RiskThreshold    `json:"thresholds"`
	ActionsPerLevel    map[string]string  `json:"actions_per_level"`
	AdaptiveMFATrigger float64            `json:"adaptive_mfa_trigger"`
	Enabled            bool               `json:"enabled"`
	ModelVersion       string             `json:"model_version"`
}

func (h *Handler) handleRiskScoringConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	result := RiskScoringConfig{
		Weights: RiskScoringWeights{
			GeoVelocity:       0.30,
			IPReputation:      0.25,
			DeviceFamiliarity: 0.20,
			TimeAnomaly:       0.15,
			FailedAttempts:    0.10,
		},
		Thresholds: []RiskThreshold{
			{Level: "low", MinScore: 0.0, MaxScore: 0.3, Action: "allow"},
			{Level: "medium", MinScore: 0.3, MaxScore: 0.6, Action: "require_mfa"},
			{Level: "high", MinScore: 0.6, MaxScore: 0.85, Action: "step_up_auth"},
			{Level: "critical", MinScore: 0.85, MaxScore: 1.0, Action: "block"},
		},
		ActionsPerLevel: map[string]string{
			"low":      "allow",
			"medium":   "require_mfa",
			"high":     "step_up_auth",
			"critical": "block_and_alert",
		},
		AdaptiveMFATrigger: 0.35,
		Enabled:            true,
		ModelVersion:       "v2.1.0",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
