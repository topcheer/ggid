package server

import (
	"encoding/json"
	"net/http"
)

type StuffingIPStat struct {
	IPAddress string `json:"ip_address"`
	Attempts  int    `json:"attempts"`
	Country   string `json:"country"`
}

type StuffingUAStat struct {
	UserAgent string `json:"user_agent"`
	Attempts  int    `json:"attempts"`
}

type CredentialStuffingStats struct {
	TotalAttempts          int             `json:"total_attempts"`
	BlockedByRateLimit     int             `json:"blocked_by_rate_limit"`
	BlockedByCaptcha       int             `json:"blocked_by_captcha"`
	UniqueTargetedAccounts int             `json:"unique_targeted_accounts"`
	TopSourceIPs           []StuffingIPStat `json:"top_source_ips"`
	TopUserAgents          []StuffingUAStat `json:"top_user_agents"`
	AttackPattern          string           `json:"attack_pattern"`
	PeakTime               string           `json:"peak_time"`
	GeneratedAt            string          `json:"generated_at"`
}

func (h *Handler) handleCredentialStuffingStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	result := CredentialStuffingStats{
		TotalAttempts:          18540,
		BlockedByRateLimit:     14200,
		BlockedByCaptcha:       2340,
		UniqueTargetedAccounts: 3420,
		TopSourceIPs: []StuffingIPStat{
			{IPAddress: "203.0.113.50", Attempts: 4200, Country: "Unknown"},
			{IPAddress: "198.51.100.12", Attempts: 3100, Country: "Russia"},
			{IPAddress: "192.0.2.88", Attempts: 2800, Country: "China"},
			{IPAddress: "203.0.113.99", Attempts: 1900, Country: "Brazil"},
			{IPAddress: "198.51.100.45", Attempts: 1540, Country: "Nigeria"},
		},
		TopUserAgents: []StuffingUAStat{
			{UserAgent: "python-requests/2.28.0", Attempts: 8200},
			{UserAgent: "curl/7.88.0", Attempts: 4100},
			{UserAgent: "Mozilla/5.0 (automated)", Attempts: 3240},
		},
		AttackPattern: "distributed",
		PeakTime:      "2025-01-15T03:00:00Z",
		GeneratedAt:   "2025-01-15T10:00:00Z",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
