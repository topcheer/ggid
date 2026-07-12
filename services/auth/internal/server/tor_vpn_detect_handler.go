package server

import (
	"encoding/json"
	"net/http"
)

type DetectedConnection struct {
	IPAddress string  `json:"ip_address"`
	Type      string  `json:"type"`
	Confidence float64 `json:"confidence"`
	FirstSeen string  `json:"first_seen"`
	UserID    string  `json:"user_id"`
}

type CountryStat struct {
	Country string `json:"country"`
	Count   int    `json:"count"`
}

type TorVPNDetectResult struct {
	DetectedConnections []DetectedConnection `json:"detected_connections"`
	ExitNodeList        []string             `json:"exit_node_list"`
	BlocklistRules      []string             `json:"blocklist_rules"`
	PerCountryStats     []CountryStat        `json:"per_country_stats"`
	TotalDetected       int                  `json:"total_detected"`
}

func (h *Handler) handleTorVPNDetect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	result := TorVPNDetectResult{
		DetectedConnections: []DetectedConnection{
			{IPAddress: "203.0.113.50", Type: "tor_exit", Confidence: 0.95, FirstSeen: "2025-01-15T03:00:00Z", UserID: "u-0342"},
			{IPAddress: "198.51.100.12", Type: "vpn", Confidence: 0.82, FirstSeen: "2025-01-15T03:15:00Z", UserID: "u-0517"},
			{IPAddress: "192.0.2.88", Type: "tor_relay", Confidence: 0.78, FirstSeen: "2025-01-14T22:00:00Z", UserID: "u-0891"},
			{IPAddress: "203.0.113.99", Type: "proxy", Confidence: 0.71, FirstSeen: "2025-01-14T18:00:00Z", UserID: "u-0420"},
		},
		ExitNodeList: []string{"203.0.113.50", "192.0.2.88", "198.51.100.45", "203.0.113.77"},
		BlocklistRules: []string{
			"block all known tor exit nodes",
			"challenge vpn connections with captcha",
			"flag proxy connections for review",
		},
		PerCountryStats: []CountryStat{
			{Country: "Unknown", Count: 15},
			{Country: "Russia", Count: 8},
			{Country: "China", Count: 5},
			{Country: "Netherlands", Count: 4},
		},
		TotalDetected: 32,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
