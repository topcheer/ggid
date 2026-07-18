package server

import (
	"net/http"
	"sync"
	"time"
)

// searchStat tracks search index statistics.
type searchStat struct {
	TotalUsers       int              `json:"total_users"`
	IndexedCount     int              `json:"indexed_count"`
	IndexHealth      string           `json:"index_health"`
	LastReindex      string           `json:"last_reindex"`
	AvgSearchTimeMs  float64          `json:"avg_search_time_ms"`
	PopularQueries   []map[string]any `json:"popular_queries"`
	IndexSize        string           `json:"index_size"`
}

var searchStatStore = struct {
	sync.RWMutex
	stats searchStat
}{stats: searchStat{
	TotalUsers: 15420, IndexedCount: 15398, IndexHealth: "healthy",
	LastReindex: time.Now().UTC().Add(-6 * time.Hour).Format(time.RFC3339),
	AvgSearchTimeMs: 12.5, IndexSize: "245MB",
	PopularQueries: []map[string]any{
		{"query": "admin", "count": 3420, "avg_results": 15},
		{"query": "engineering", "count": 2180, "avg_results": 5200},
		{"query": "@gmail.com", "count": 1850, "avg_results": 320},
		{"query": "alice", "count": 1240, "avg_results": 8},
		{"query": "role:viewer", "count": 980, "avg_results": 3200},
		{"query": "status:active", "count": 820, "avg_results": 14200},
		{"query": "department:Sales", "count": 650, "avg_results": 3100},
		{"query": "no_mfa", "count": 420, "avg_results": 5800},
		{"query": "dormant", "count": 310, "avg_results": 180},
		{"query": "recently_active", "count": 280, "avg_results": 9800},
	},
}}

// GET /api/v1/users/search-stats
func (h *HTTPHandler) handleSearchStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	searchStatStore.RLock()
	stats := searchStatStore.stats
	searchStatStore.RUnlock()

	coverage := 0.0
	if stats.TotalUsers > 0 {
		coverage = float64(stats.IndexedCount) / float64(stats.TotalUsers) * 100
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"total_users":       stats.TotalUsers,
		"indexed_count":     stats.IndexedCount,
		"unindexed_count":   stats.TotalUsers - stats.IndexedCount,
		"index_coverage_pct": coverage,
		"index_health":      stats.IndexHealth,
		"last_reindex":      stats.LastReindex,
		"avg_search_time_ms": stats.AvgSearchTimeMs,
		"index_size":        stats.IndexSize,
		"popular_queries":   stats.PopularQueries,
		"checked_at":        time.Now().UTC().Format(time.RFC3339),
	})
}
