package server

import (
	"net/http"
)

// CredentialStuffingStats holds credential stuffing detection statistics.
// These would be populated from real detection data when available.
type CredentialStuffingStats struct {
	TotalAttempts          int              `json:"total_attempts"`
	BlockedByRateLimit     int              `json:"blocked_by_rate_limit"`
	BlockedByCaptcha       int              `json:"blocked_by_captcha"`
	UniqueTargetedAccounts int              `json:"unique_targeted_accounts"`
	TopSourceIPs           []StuffingIPStat `json:"top_source_ips"`
	TopUserAgents          []StuffingUAStat `json:"top_user_agents"`
	AttackPattern          string           `json:"attack_pattern"`
	PeakTime               string           `json:"peak_time"`
	GeneratedAt            string           `json:"generated_at"`
}

type StuffingIPStat struct {
	IPAddress string `json:"ip_address"`
	Attempts  int    `json:"attempts"`
	Country   string `json:"country"`
}

type StuffingUAStat struct {
	UserAgent string `json:"user_agent"`
	Attempts  int    `json:"attempts"`
}

func (h *Handler) handleCredentialStuffingStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	// Return empty real stats — no hardcoded mock data.
	// Real data comes from ITDR detections (brute_force/credential_stuffing rules).
	result := CredentialStuffingStats{
		TopSourceIPs:  []StuffingIPStat{},
		TopUserAgents: []StuffingUAStat{},
	}
	writeJSON(w, http.StatusOK, result)
}
