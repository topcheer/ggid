package server

import (
	"encoding/json"
	"net/http"
)

type DPoPConfig struct {
	RequireDPoP          bool            `json:"require_dpop"`
	PerClientOverride    map[string]bool `json:"per_client_override"`
	ProofMaxAgeSeconds   int             `json:"proof_max_age_seconds"`
	KeyBindingAlgorithm  string          `json:"key_binding_algorithm"`
	DPoPStats            struct {
		TotalProofs    int     `json:"total_proofs"`
		ValidProofs    int     `json:"valid_proofs"`
		ReplayDetected int     `json:"replay_detected"`
		ValidationRate float64 `json:"validation_rate"`
	} `json:"dpop_stats"`
	ExemptedClients []string `json:"exempted_clients"`
}

func handleDPoPConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		result := DPoPConfig{
			RequireDPoP:        true,
			PerClientOverride:  map[string]bool{"legacy-service": false, "mobile-app": true},
			ProofMaxAgeSeconds: 60,
			KeyBindingAlgorithm: "ES256",
		}
		result.DPoPStats.TotalProofs = 45200
		result.DPoPStats.ValidProofs = 44980
		result.DPoPStats.ReplayDetected = 3
		result.DPoPStats.ValidationRate = 99.5
		result.ExemptedClients = []string{"legacy-service", "internal-healthcheck"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	case http.MethodPut:
		var req DPoPConfig
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
