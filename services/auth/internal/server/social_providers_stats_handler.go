package server

import (
	"encoding/json"
	"net/http"
)

type ProviderStats struct {
	Provider      string  `json:"provider"`
	UserCount     int     `json:"user_count"`
	LoginCount30d int     `json:"login_count_30d"`
	SuccessRate   float64 `json:"success_rate"`
	AvgLatencyMs  float64 `json:"avg_latency_ms"`
	NewUsers30d   int     `json:"new_users_30d"`
	TopErrors     []string `json:"top_errors"`
}

type SocialProvidersResult struct {
	PerProvider []ProviderStats `json:"per_provider"`
	TotalLogins int             `json:"total_logins_30d"`
}

func (h *Handler) handleSocialProvidersStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	result := SocialProvidersResult{
		PerProvider: []ProviderStats{
			{Provider: "google", UserCount: 5200, LoginCount30d: 18500, SuccessRate: 98.5, AvgLatencyMs: 120, NewUsers30d: 340, TopErrors: []string{"invalid_state", "email_not_verified"}},
			{Provider: "github", UserCount: 3100, LoginCount30d: 8200, SuccessRate: 97.2, AvgLatencyMs: 180, NewUsers30d: 120, TopErrors: []string{"scope_denied"}},
			{Provider: "microsoft", UserCount: 1800, LoginCount30d: 5400, SuccessRate: 96.8, AvgLatencyMs: 150, NewUsers30d: 85, TopErrors: []string{"tenant_mismatch"}},
			{Provider: "apple", UserCount: 420, LoginCount30d: 890, SuccessRate: 94.1, AvgLatencyMs: 220, NewUsers30d: 30, TopErrors: []string{"private_email_relay_timeout"}},
		},
		TotalLogins: 32990,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
