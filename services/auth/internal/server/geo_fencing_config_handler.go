package server

import (
	"encoding/json"
	"net/http"
)

type GeoFencingConfig struct {
	Enabled       bool                `json:"enabled"`
	Rules         []GeoFencingRule    `json:"rules"`
	WhitelistIPs  []string            `json:"whitelist_ips"`
}

type GeoFencingRule struct {
	Country  string `json:"country"`
	Region   string `json:"region,omitempty"`
	CIDRs    []string `json:"cidr_ranges"`
	Action   string `json:"action"`
}

func (h *Handler) handleGeoFencingConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		result := GeoFencingConfig{
			Enabled: true,
			Rules: []GeoFencingRule{
				{Country: "US", CIDRs: []string{}, Action: "allow"},
				{Country: "CN", CIDRs: []string{}, Action: "deny"},
				{Country: "RU", CIDRs: []string{}, Action: "challenge"},
				{Country: "*", Region: "TOR_EXIT_NODES", CIDRs: []string{"203.0.113.0/24", "198.51.100.0/24"}, Action: "deny"},
			},
			WhitelistIPs: []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	case http.MethodPut:
		var req GeoFencingConfig
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
