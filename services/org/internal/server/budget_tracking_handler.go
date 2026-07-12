package httpserver

import (
	"encoding/json"
	"net/http"
	"sync"
)

type BudgetDepartment struct {
	DepartmentID    string  `json:"department_id"`
	Name            string  `json:"name"`
	BudgetAmount    float64 `json:"budget_amount"`
	SpendToDate     float64 `json:"spend_to_date"`
	BurnRateMonthly float64 `json:"burn_rate_monthly"`
	ProjectedEOY    float64 `json:"projected_eoy"`
	OverBudget      bool    `json:"over_budget"`
	CostPerUser     float64 `json:"cost_per_user"`
}

type BudgetTrackingAlert struct {
	DepartmentID string  `json:"department_id"`
	AlertType    string  `json:"alert_type"`
	Severity     string  `json:"severity"`
	CurrentSpend float64 `json:"current_spend"`
	BudgetLimit  float64 `json:"budget_limit"`
}

type BudgetTrackingResult struct {
	Departments     []BudgetDepartment `json:"departments"`
	OverBudgetAlerts []BudgetTrackingAlert     `json:"over_budget_alerts"`
	TotalBudget     float64            `json:"total_budget"`
	TotalSpend      float64            `json:"total_spend"`
	ProjectedTotal  float64            `json:"projected_total_eoy"`
	AvgCostPerUser  float64            `json:"avg_cost_per_user"`
}

var budgetTrackingOnce sync.Once
var budgetTrackingData BudgetTrackingResult

func initBudgetData() {
	budgetTrackingOnce.Do(func() {
		budgetTrackingData = BudgetTrackingResult{
			Departments: []BudgetDepartment{
				{DepartmentID: "d-001", Name: "Engineering", BudgetAmount: 150000, SpendToDate: 87500, BurnRateMonthly: 12500, ProjectedEOY: 150000, OverBudget: false, CostPerUser: 1944},
				{DepartmentID: "d-002", Name: "Sales", BudgetAmount: 80000, SpendToDate: 72000, BurnRateMonthly: 12000, ProjectedEOY: 108000, OverBudget: true, CostPerUser: 3600},
				{DepartmentID: "d-003", Name: "Marketing", BudgetAmount: 60000, SpendToDate: 28000, BurnRateMonthly: 5600, ProjectedEOY: 67200, OverBudget: false, CostPerUser: 2333},
				{DepartmentID: "d-004", Name: "Operations", BudgetAmount: 90000, SpendToDate: 68000, BurnRateMonthly: 11500, ProjectedEOY: 138000, OverBudget: true, CostPerUser: 3778},
				{DepartmentID: "d-005", Name: "IT", BudgetAmount: 50000, SpendToDate: 19000, BurnRateMonthly: 3200, ProjectedEOY: 38400, OverBudget: false, CostPerUser: 1583},
			},
			OverBudgetAlerts: []BudgetTrackingAlert{
				{DepartmentID: "d-002", AlertType: "projected_over_budget", Severity: "critical", CurrentSpend: 72000, BudgetLimit: 80000},
				{DepartmentID: "d-004", AlertType: "projected_over_budget", Severity: "critical", CurrentSpend: 68000, BudgetLimit: 90000},
			},
			TotalBudget:    430000,
			TotalSpend:     274500,
			ProjectedTotal: 501600,
			AvgCostPerUser: 2412,
		}
	})
}

func (s *HTTPServer) handleBudgetTracking(w http.ResponseWriter, r *http.Request) {
	initBudgetData()
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(budgetTrackingData)
}
