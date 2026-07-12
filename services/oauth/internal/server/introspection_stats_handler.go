package server

import (
	"encoding/json"
	"net/http"
	"sync"
)

type IntrospectionClientStat struct {
	ClientID    string `json:"client_id"`
	ClientName  string `json:"client_name"`
	Requests    int    `json:"requests"`
	AvgLatencyMs float64 `json:"avg_latency_ms"`
}

type IntrospectionStats struct {
	TotalRequests  int                       `json:"total_requests"`
	UniqueClients  int                       `json:"unique_clients"`
	AvgLatencyMs   float64                   `json:"avg_latency_ms"`
	CacheHitRate   float64                   `json:"cache_hit_rate"`
	RateLimitHits  int                       `json:"rate_limit_hits"`
	TopClients     []IntrospectionClientStat `json:"top_clients"`
	GeneratedAt    string                    `json:"generated_at"`
}

var introspectionStatsStore sync.Map
var introspectionStatsOnce sync.Once

func initIntrospectionStats() {
	introspectionStatsOnce.Do(func() {
		data := IntrospectionStats{
			TotalRequests: 15420,
			UniqueClients: 23,
			AvgLatencyMs:  8.5,
			CacheHitRate:  0.78,
			RateLimitHits: 12,
			TopClients: []IntrospectionClientStat{
				{ClientID: "c-001", ClientName: "web-app", Requests: 5200, AvgLatencyMs: 6.2},
				{ClientID: "c-002", ClientName: "mobile-app", Requests: 3800, AvgLatencyMs: 9.1},
				{ClientID: "c-003", ClientName: "api-gateway", Requests: 2900, AvgLatencyMs: 5.8},
				{ClientID: "c-004", ClientName: "cli-tool", Requests: 1500, AvgLatencyMs: 12.4},
				{ClientID: "c-005", ClientName: "batch-processor", Requests: 1200, AvgLatencyMs: 15.0},
			},
			GeneratedAt: "2025-01-15T10:00:00Z",
		}
		introspectionStatsStore.Store("latest", data)
	})
}

func handleIntrospectionStats(w http.ResponseWriter, r *http.Request) {
	initIntrospectionStats()
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	val, ok := introspectionStatsStore.Load("latest")
	if !ok {
		http.Error(w, "no data", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(val)
}
