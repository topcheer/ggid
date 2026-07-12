package server

import (
	"encoding/json"
	"net/http"
)

type PARConfig struct {
	RequirePAR         bool              `json:"require_par"`
	PARLifetimeSeconds int               `json:"par_lifetime_seconds"`
	MaxRequestSizeKB   int               `json:"max_request_size_kb"`
	PerClientOverride  map[string]bool   `json:"per_client_override"`
	ExemptedClients    []string          `json:"exempted_clients"`
	PARUsageStats      struct {
		TotalAuthzReqs int     `json:"total_authz_requests"`
		PARRequests    int     `json:"par_requests"`
		AdoptionRate   float64 `json:"adoption_rate"`
	} `json:"par_usage_stats"`
}

func handlePARConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		result := PARConfig{
			RequirePAR:         true,
			PARLifetimeSeconds: 60,
			MaxRequestSizeKB:   16,
			PerClientOverride:  map[string]bool{"legacy-app": false},
			ExemptedClients:    []string{"legacy-app", "internal-healthcheck"},
		}
		result.PARUsageStats.TotalAuthzReqs = 32100
		result.PARUsageStats.PARRequests = 29800
		result.PARUsageStats.AdoptionRate = 92.8
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	case http.MethodPut:
		var req PARConfig
		json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"status": "updated", "config": req})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
