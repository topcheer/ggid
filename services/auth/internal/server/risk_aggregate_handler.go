package server

import (
	"net/http"
	"sync"
	"time"
)

// riskAggregateUser holds per-user aggregated risk data.
type riskAggregateUser struct {
	UserID    string  `json:"user_id"`
	Username  string  `json:"username"`
	OrgUnit   string  `json:"org_unit"`
	RiskScore int     `json:"risk_score"`
	Factors   int     `json:"factors"`
}

var riskAggregateStore = struct {
	sync.RWMutex
	users []riskAggregateUser
}{users: []riskAggregateUser{
	{UserID: "user-001", Username: "alice.admin", OrgUnit: "Security", RiskScore: 85, Factors: 4},
	{UserID: "user-002", Username: "bob.dormant", OrgUnit: "Engineering", RiskScore: 72, Factors: 3},
	{UserID: "user-003", Username: "carol.nopass", OrgUnit: "Sales", RiskScore: 65, Factors: 2},
	{UserID: "user-004", Username: "dave.priv", OrgUnit: "Engineering", RiskScore: 58, Factors: 2},
	{UserID: "user-005", Username: "eve.exposed", OrgUnit: "Marketing", RiskScore: 90, Factors: 5},
	{UserID: "user-006", Username: "frank.low", OrgUnit: "Sales", RiskScore: 12, Factors: 0},
	{UserID: "user-007", Username: "grace.medium", OrgUnit: "Engineering", RiskScore: 35, Factors: 1},
	{UserID: "user-008", Username: "henry.high", OrgUnit: "Security", RiskScore: 78, Factors: 3},
}}

// GET /api/v1/auth/risk/aggregate?group_by=user|org&org=X
func (h *Handler) handleRiskAggregate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	groupBy := r.URL.Query().Get("group_by")
	if groupBy == "" {
		groupBy = "user"
	}
	orgFilter := r.URL.Query().Get("org")

	riskAggregateStore.RLock()
	defer riskAggregateStore.RUnlock()

	// Filter by org if specified
	filtered := []riskAggregateUser{}
	for _, u := range riskAggregateStore.users {
		if orgFilter != "" && u.OrgUnit != orgFilter {
			continue
		}
		filtered = append(filtered, u)
	}

	// Compute aggregate stats
	totalScore := 0
	highRisk := 0
	for _, u := range filtered {
		totalScore += u.RiskScore
		if u.RiskScore >= 70 {
			highRisk++
		}
	}
	avgScore := 0
	if len(filtered) > 0 {
		avgScore = totalScore / len(filtered)
	}

	// 7-day trend (simulated)
	now := time.Now().UTC()
	trend7d := []map[string]any{}
	for i := 6; i >= 0; i-- {
		day := now.AddDate(0, 0, -i)
		baseAvg := avgScore - (i * 2)
		if baseAvg < 0 {
			baseAvg = 0
		}
		trend7d = append(trend7d, map[string]any{
			"date":         day.Format("2006-01-02"),
			"avg_score":    baseAvg,
			"high_risk_count": highRisk - (i / 3),
		})
	}

	// Group by org if requested
	byOrg := map[string]map[string]int{}
	if groupBy == "org" {
		for _, u := range filtered {
			if _, ok := byOrg[u.OrgUnit]; !ok {
				byOrg[u.OrgUnit] = map[string]int{"count": 0, "total_score": 0, "high_risk": 0}
			}
			byOrg[u.OrgUnit]["count"]++
			byOrg[u.OrgUnit]["total_score"] += u.RiskScore
			if u.RiskScore >= 70 {
				byOrg[u.OrgUnit]["high_risk"]++
			}
		}
	}

	result := map[string]any{
		"avg_score":       avgScore,
		"high_risk_users": highRisk,
		"total_users":     len(filtered),
		"trends_7d":       trend7d,
		"checked_at":      now.Format(time.RFC3339),
	}

	if groupBy == "org" {
		orgSummary := []map[string]any{}
		for org, stats := range byOrg {
			avg := 0
			if stats["count"] > 0 {
				avg = stats["total_score"] / stats["count"]
			}
			orgSummary = append(orgSummary, map[string]any{
				"org_unit":    org,
				"user_count":  stats["count"],
				"avg_score":   avg,
				"high_risk":   stats["high_risk"],
			})
		}
		result["by_org"] = orgSummary
	} else {
		result["users"] = filtered
	}

	writeJSON(w, http.StatusOK, result)
}
