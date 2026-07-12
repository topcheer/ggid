package server

import (
	"encoding/json"
	"net/http"
)

type ClientConsentSummary struct {
	ClientName      string   `json:"client_name"`
	UserCount       int      `json:"user_count"`
	Scopes          []string `json:"scopes"`
	LastGranted     string   `json:"last_granted"`
	RevocationTrend float64  `json:"revocation_trend_30d"`
	PendingExpiry   int      `json:"pending_expiry"`
}

type ConsentDashboardResult struct {
	ActiveConsents   []ClientConsentSummary `json:"active_consents"`
	TotalActiveUsers int                    `json:"total_active_users"`
	TotalRevocations int                    `json:"total_revocations_30d"`
	ExpiringSoon     int                    `json:"expiring_within_7d"`
}

func handleConsentDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	result := ConsentDashboardResult{
		ActiveConsents: []ClientConsentSummary{
			{ClientName: "web-console", UserCount: 450, Scopes: []string{"openid", "profile", "read:users"}, LastGranted: "2025-01-15T09:00:00Z", RevocationTrend: 2.1, PendingExpiry: 12},
			{ClientName: "mobile-app", UserCount: 320, Scopes: []string{"openid", "profile"}, LastGranted: "2025-01-14T18:00:00Z", RevocationTrend: 5.3, PendingExpiry: 8},
			{ClientName: "analytics-dashboard", UserCount: 85, Scopes: []string{"read:audit", "read:policies"}, LastGranted: "2025-01-13T10:00:00Z", RevocationTrend: 0.5, PendingExpiry: 3},
			{ClientName: "service-agent-01", UserCount: 1, Scopes: []string{"read:users", "write:audit"}, LastGranted: "2025-01-10T00:00:00Z", RevocationTrend: 0.0, PendingExpiry: 0},
		},
		TotalActiveUsers: 856,
		TotalRevocations: 42,
		ExpiringSoon:     23,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
