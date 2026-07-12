package server

import (
	"encoding/json"
	"net/http"
)

type TokenRevocationStatsResult struct {
	TotalRevocations int            `json:"total_revocations"`
	ByReason         map[string]int `json:"by_reason"`
	ByClient         []struct {
		ClientID   string `json:"client_id"`
		Count      int    `json:"count"`
		TopReason  string `json:"top_reason"`
	} `json:"by_client"`
	Trend30d  []int `json:"trend_30d"`
	PeakHour  int   `json:"peak_hour"`
}

func handleTokenRevocationStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	result := TokenRevocationStatsResult{
		TotalRevocations: 1840,
		ByReason: map[string]int{
			"user_initiated":   820,
			"admin":            350,
			"expired":          480,
			"security_event":   120,
			"refresh_rotation": 70,
		},
		ByClient: []struct {
			ClientID  string `json:"client_id"`
			Count     int    `json:"count"`
			TopReason string `json:"top_reason"`
		}{
			{ClientID: "web-console", Count: 820, TopReason: "user_initiated"},
			{ClientID: "mobile-app", Count: 480, TopReason: "expired"},
			{ClientID: "service-agent-01", Count: 290, TopReason: "admin"},
		},
		Trend30d: []int{45, 52, 38, 61, 72, 68, 55, 49, 63, 58, 71, 80, 65, 59, 62, 73, 88, 76, 69, 54, 61, 67, 82, 91, 78, 73, 85, 79, 94, 96},
		PeakHour: 9,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
