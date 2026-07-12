package server

import (
	"net/http"
	"sync"
	"time"
)

// tokenLifetimeRecord tracks token lifetime metrics per client.
type tokenLifetimeRecord struct {
	ClientID       string  `json:"client_id"`
	ClientName     string  `json:"client_name"`
	TokenType      string  `json:"token_type"`
	AvgLifetime    float64 `json:"avg_lifetime_seconds"`
	MedianLifetime float64 `json:"median_lifetime_seconds"`
	MinLifetime    float64 `json:"min_lifetime_seconds"`
	MaxLifetime    float64 `json:"max_lifetime_seconds"`
	ShortLivedPct  float64 `json:"short_lived_pct"`  // % of tokens < 1h
	LongLivedPct   float64 `json:"long_lived_pct"`   // % of tokens > 24h
	TotalTokens    int     `json:"total_tokens"`
	ActiveTokens   int     `json:"active_tokens"`
}

var tokenLifetimeStore = struct {
	sync.RWMutex
	data map[string]*tokenLifetimeRecord
}{data: map[string]*tokenLifetimeRecord{
	"web-app": {
		ClientID: "web-app", ClientName: "Web Application", TokenType: "access",
		AvgLifetime: 3600, MedianLifetime: 3600, MinLifetime: 1800, MaxLifetime: 7200,
		ShortLivedPct: 92.3, LongLivedPct: 1.2, TotalTokens: 15420, ActiveTokens: 320,
	},
	"mobile-ios": {
		ClientID: "mobile-ios", ClientName: "iOS Mobile App", TokenType: "access",
		AvgLifetime: 86400, MedianLifetime: 86400, MinLifetime: 3600, MaxLifetime: 259200,
		ShortLivedPct: 15.0, LongLivedPct: 45.5, TotalTokens: 8920, ActiveTokens: 180,
	},
	"admin-cli": {
		ClientID: "admin-cli", ClientName: "Admin CLI", TokenType: "access",
		AvgLifetime: 1800, MedianLifetime: 1800, MinLifetime: 900, MaxLifetime: 3600,
		ShortLivedPct: 98.5, LongLivedPct: 0.0, TotalTokens: 1240, ActiveTokens: 12,
	},
	"service-backend": {
		ClientID: "service-backend", ClientName: "Backend Service", TokenType: "client_credentials",
		AvgLifetime: 3600, MedianLifetime: 3600, MinLifetime: 3600, MaxLifetime: 3600,
		ShortLivedPct: 100.0, LongLivedPct: 0.0, TotalTokens: 45000, ActiveTokens: 5,
	},
}}

// GET /api/v1/oauth/token-lifetime/analytics?group_by=client
// Returns token lifetime statistics broken down by client.
func handleTokenLifetimeAnalytics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	tokenLifetimeStore.RLock()
	defer tokenLifetimeStore.RUnlock()

	records := make([]*tokenLifetimeRecord, 0, len(tokenLifetimeStore.data))
	totalTokens := 0
	totalActive := 0
	allLifetimes := []float64{}

	for _, rec := range tokenLifetimeStore.data {
		records = append(records, rec)
		totalTokens += rec.TotalTokens
		totalActive += rec.ActiveTokens
		allLifetimes = append(allLifetimes, rec.AvgLifetime)
	}

	// Compute overall stats
	overallAvg := 0.0
	overallMedian := 0.0
	if len(allLifetimes) > 0 {
		sum := 0.0
		for _, v := range allLifetimes {
			sum += v
		}
		overallAvg = sum / float64(len(allLifetimes))
		// Simple median
		mid := len(allLifetimes) / 2
		overallMedian = allLifetimes[mid]
		if len(allLifetimes)%2 == 0 {
			overallMedian = (allLifetimes[mid-1] + allLifetimes[mid]) / 2
		}
	}

	// Find shortest and longest lived clients
	var shortestClient, longestClient string
	shortestAvg := 1e12
	longestAvg := 0.0
	for _, rec := range records {
		if rec.AvgLifetime < shortestAvg {
			shortestAvg = rec.AvgLifetime
			shortestClient = rec.ClientID
		}
		if rec.AvgLifetime > longestAvg {
			longestAvg = rec.AvgLifetime
			longestClient = rec.ClientID
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"records":           records,
		"total_clients":     len(records),
		"total_tokens":      totalTokens,
		"active_tokens":     totalActive,
		"overall_avg":       overallAvg,
		"overall_median":    overallMedian,
		"shortest_lived_client": map[string]any{
			"client_id":      shortestClient,
			"avg_lifetime_s": shortestAvg,
		},
		"longest_lived_client": map[string]any{
			"client_id":      longestClient,
			"avg_lifetime_s": longestAvg,
		},
		"generated_at": time.Now().UTC().Format(time.RFC3339),
	})
}
