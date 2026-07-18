package httpserver

import (
	"encoding/json"
	"net/http"
)

type DepartmentAnalyticsItem struct {
	DepartmentID        string  `json:"department_id"`
	Name                string  `json:"name"`
	Headcount           int     `json:"headcount"`
	AvgTenureDays       int     `json:"avg_tenure_days"`
	GrowthRate30d       float64 `json:"growth_rate_30d"`
	BudgetUtilizationPct float64 `json:"budget_utilization_pct"`
	OpenPositions       int     `json:"open_positions"`
	AttritionRate       float64 `json:"attrition_rate"`
}

type DepartmentAnalyticsResult struct {
	Departments   []DepartmentAnalyticsItem `json:"departments"`
	TotalHeadcount int                      `json:"total_headcount"`
	AvgGrowthRate  float64                   `json:"avg_growth_rate_30d"`
	AvgAttrition   float64                   `json:"avg_attrition_rate"`
	GeneratedAt    string                    `json:"generated_at"`
}

func (s *HTTPServer) handleDepartmentAnalytics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	result := DepartmentAnalyticsResult{
		Departments: []DepartmentAnalyticsItem{
			{DepartmentID: "d-001", Name: "Engineering", Headcount: 45, AvgTenureDays: 540, GrowthRate30d: 0.12, BudgetUtilizationPct: 83.3, OpenPositions: 3, AttritionRate: 0.08},
			{DepartmentID: "d-002", Name: "Sales", Headcount: 20, AvgTenureDays: 365, GrowthRate30d: 0.15, BudgetUtilizationPct: 90.0, OpenPositions: 2, AttritionRate: 0.15},
			{DepartmentID: "d-003", Name: "Marketing", Headcount: 12, AvgTenureDays: 420, GrowthRate30d: 0.0, BudgetUtilizationPct: 46.7, OpenPositions: 1, AttritionRate: 0.05},
			{DepartmentID: "d-004", Name: "Operations", Headcount: 18, AvgTenureDays: 680, GrowthRate30d: -0.05, BudgetUtilizationPct: 75.6, OpenPositions: 0, AttritionRate: 0.11},
			{DepartmentID: "d-005", Name: "IT", Headcount: 8, AvgTenureDays: 720, GrowthRate30d: 0.0, BudgetUtilizationPct: 38.0, OpenPositions: 1, AttritionRate: 0.03},
		},
		TotalHeadcount: 103,
		AvgGrowthRate:  0.044,
		AvgAttrition:   0.084,
		GeneratedAt:    "2025-01-15T10:00:00Z",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
