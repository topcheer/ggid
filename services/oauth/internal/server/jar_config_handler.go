package server

import (
	"encoding/json"
	"net/http"
)

type JARConfig struct {
	RequireJAR           bool              `json:"require_jar"`
	JARLifetimeSeconds   int               `json:"jar_lifetime_seconds"`
	SigningAlg           string            `json:"signing_alg"`
	PerClientOverride    map[string]string `json:"per_client_override"`
	EncryptionOptional   bool              `json:"encryption_optional"`
	UsageStats           struct {
		TotalRequests int `json:"total_requests"`
		JARRequests   int `json:"jar_requests"`
		AdoptionRate  float64 `json:"adoption_rate"`
	} `json:"usage_stats"`
}

func handleJARConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		result := JARConfig{
			RequireJAR:         false,
			JARLifetimeSeconds: 60,
			SigningAlg:         "RS256",
			PerClientOverride:  map[string]string{"secure-service": "ES256"},
			EncryptionOptional: true,
		}
		result.UsageStats.TotalRequests = 45200
		result.UsageStats.JARRequests = 12800
		result.UsageStats.AdoptionRate = 28.3
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	case http.MethodPut:
		var req JARConfig
		json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"status": "updated", "config": req})
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
