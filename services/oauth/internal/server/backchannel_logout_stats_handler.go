package server

import (
	"encoding/json"
	"net/http"
)

type FailureReason struct {
	Reason string `json:"reason"`
	Count  int    `json:"count"`
}

type IDPProviderStat struct {
	Provider    string  `json:"provider"`
	Requests    int     `json:"requests"`
	SuccessPct  float64 `json:"success_pct"`
	AvgLatencyMs float64 `json:"avg_latency_ms"`
}

type BackchannelLogoutStats struct {
	TotalLogoutRequests  int               `json:"total_logout_requests"`
	SuccessfulLogoutPct   float64           `json:"successful_logout_pct"`
	FailedLogoutCount     int               `json:"failed_logout_count"`
	TopFailureReasons     []FailureReason   `json:"top_failure_reasons"`
	AvgLatencyMs          float64           `json:"avg_latency_ms"`
	ByIDPProvider         []IDPProviderStat `json:"by_idp_provider"`
	GeneratedAt           string            `json:"generated_at"`
}

func handleBackchannelLogoutStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	result := BackchannelLogoutStats{
		TotalLogoutRequests: 3420,
		SuccessfulLogoutPct:  0.934,
		FailedLogoutCount:    225,
		TopFailureReasons: []FailureReason{
			{Reason: "token_expired", Count: 98},
			{Reason: "invalid_signature", Count: 67},
			{Reason: "idp_unreachable", Count: 42},
			{Reason: "sub_claim_mismatch", Count: 18},
		},
		AvgLatencyMs: 145,
		ByIDPProvider: []IDPProviderStat{
			{Provider: "google", Requests: 1800, SuccessPct: 0.96, AvgLatencyMs: 120},
			{Provider: "azure_ad", Requests: 920, SuccessPct: 0.92, AvgLatencyMs: 180},
			{Provider: "okta", Requests: 520, SuccessPct: 0.89, AvgLatencyMs: 155},
			{Provider: "onelogin", Requests: 180, SuccessPct: 0.87, AvgLatencyMs: 200},
		},
		GeneratedAt: "2025-01-15T10:00:00Z",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
