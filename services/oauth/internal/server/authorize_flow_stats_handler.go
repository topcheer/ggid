package server

import (
	"encoding/json"
	"net/http"
)

type AbandonmentStep struct {
	Step       string  `json:"step"`
	Count      int     `json:"count"`
	Pct        float64 `json:"pct"`
}

type AuthorizeClientStat struct {
	ClientID     string  `json:"client_id"`
	ClientName   string  `json:"client_name"`
	Attempts     int     `json:"attempts"`
	ConsentRate  float64 `json:"consent_rate"`
	AvgDurationMs float64 `json:"avg_duration_ms"`
}

type AuthorizeFlowStats struct {
	TotalAttempts        int                   `json:"total_attempts"`
	ConsentRate          float64               `json:"consent_rate"`
	AbandonmentAtStep    []AbandonmentStep     `json:"abandonment_at_step"`
	AvgDurationMs        float64               `json:"avg_duration_ms"`
	TopClients           []AuthorizeClientStat  `json:"top_clients"`
	RedirectURIErrors    int                   `json:"redirect_uri_errors"`
	PKCEAdoptionPct      float64               `json:"pkce_adoption_pct"`
	GeneratedAt          string                `json:"generated_at"`
}

func handleAuthorizeFlowStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	result := AuthorizeFlowStats{
		TotalAttempts: 12450,
		ConsentRate:   0.782,
		AbandonmentAtStep: []AbandonmentStep{
			{Step: "login", Count: 820, Pct: 6.6},
			{Step: "consent_screen", Count: 1450, Pct: 11.6},
			{Step: "redirect", Count: 440, Pct: 3.5},
		},
		AvgDurationMs: 3200,
		TopClients: []AuthorizeClientStat{
			{ClientID: "c-001", ClientName: "web-app", Attempts: 5200, ConsentRate: 0.85, AvgDurationMs: 2800},
			{ClientID: "c-002", ClientName: "mobile-app", Attempts: 3800, ConsentRate: 0.72, AvgDurationMs: 3500},
			{ClientID: "c-003", ClientName: "api-gateway", Attempts: 2100, ConsentRate: 0.80, AvgDurationMs: 3000},
		},
		RedirectURIErrors: 87,
		PKCEAdoptionPct:   0.91,
		GeneratedAt:       "2025-01-15T10:00:00Z",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
