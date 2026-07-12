package httpserver

import (
	"encoding/json"
	"net/http"
)

type DecisionLogStatsResult struct {
	TotalDecisions  int            `json:"total_decisions"`
	AllowCount      int            `json:"allow_count"`
	DenyCount       int            `json:"deny_count"`
	ByPolicy        map[string]int `json:"by_policy"`
	ByResourceType  map[string]int `json:"by_resource_type"`
	AvgEvalTimeMs   float64        `json:"avg_eval_time_ms"`
	CacheHitRate    float64        `json:"cache_hit_rate"`
	TopDeniedActions []string      `json:"top_denied_actions"`
	MismatchCount   int            `json:"mismatch_count"`
}

func (s *HTTPServer) handleDecisionLogStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	result := DecisionLogStatsResult{
		TotalDecisions: 45800,
		AllowCount:     42100,
		DenyCount:      3700,
		ByPolicy: map[string]int{
			"pol-admin-access": 8200,
			"pol-read-only":    15000,
			"pol-api-gateway":  12600,
			"pol-data-access":  10000,
		},
		ByResourceType: map[string]int{
			"users":    18500,
			"policies": 8200,
			"audit":    6800,
			"orgs":     5200,
			"api":      7100,
		},
		AvgEvalTimeMs:    2.3,
		CacheHitRate:     0.87,
		TopDeniedActions: []string{"admin:write", "audit:delete", "policy:modify", "org:delete"},
		MismatchCount:    12,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
