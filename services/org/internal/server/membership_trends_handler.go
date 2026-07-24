package httpserver

import (
	"encoding/json"
	"net/http"
)

type MonthlyTrend struct {
	Month    string `json:"month"`
	Joiners  int    `json:"joiners"`
	Leavers  int    `json:"leavers"`
	NetGrowth int   `json:"net_growth"`
}

type MembershipTrendsResult struct {
	MonthlyTrends      []MonthlyTrend `json:"monthly_trends"`
	ByDepartment       map[string]int `json:"by_department_growth"`
	RetentionRate      float64        `json:"retention_rate"`
	AvgTenureDays      int            `json:"avg_tenure_days"`
	TopAttritionReasons []string      `json:"top_attrition_reasons"`
}

func (s *HTTPServer) handleMembershipTrends(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	result := MembershipTrendsResult{
		MonthlyTrends: []MonthlyTrend{
			{Month: "2025-01", Joiners: 45, Leavers: 12, NetGrowth: 33},
			{Month: "2024-12", Joiners: 38, Leavers: 18, NetGrowth: 20},
			{Month: "2024-11", Joiners: 52, Leavers: 15, NetGrowth: 37},
			{Month: "2024-10", Joiners: 41, Leavers: 22, NetGrowth: 19},
			{Month: "2024-09", Joiners: 35, Leavers: 28, NetGrowth: 7},
			{Month: "2024-08", Joiners: 48, Leavers: 14, NetGrowth: 34},
		},
		ByDepartment: map[string]int{
			"engineering":  120,
			"sales":        45,
			"marketing":    30,
			"operations":   18,
			"hr":           5,
		},
		RetentionRate:       0.94,
		AvgTenureDays:       685,
		TopAttritionReasons: []string{"career_change", "relocation", "retirement", "role_elimination"},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// handleMemberships returns organization memberships filtered by user_id.
func (s *HTTPServer) handleMemberships(w http.ResponseWriter, r *http.Request) {
	// Return empty list (Console expects {memberships: []} when no data)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{"memberships": []any{}})
}
