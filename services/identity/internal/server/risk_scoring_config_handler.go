package server

import (
	"encoding/json"
	"net/http"
)

type IdentityRiskScoringConfig struct {
	RiskFactors []string          `json:"risk_factors"`
	Weights     map[string]float64 `json:"weights"`
	Thresholds  map[string]float64 `json:"thresholds"`
	ActionMapping map[string]string `json:"action_mapping"`
	AdaptiveLearning bool         `json:"adaptive_learning"`
}

func (h *HTTPHandler) handleIdentityRiskScoringConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		result := IdentityRiskScoringConfig{
			RiskFactors: []string{"login_location", "device_fingerprint", "time_anomaly", "failed_attempts"},
			Weights: map[string]float64{"login_location": 0.25, "device_fingerprint": 0.30, "time_anomaly": 0.20, "failed_attempts": 0.25},
			Thresholds: map[string]float64{"low": 0.3, "medium": 0.5, "high": 0.7, "critical": 0.9},
			ActionMapping: map[string]string{"low": "challenge", "medium": "challenge", "high": "block", "critical": "block+notify"},
			AdaptiveLearning: true,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	case http.MethodPut:
		var req IdentityRiskScoringConfig
		json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"status": "updated", "config": req})
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
