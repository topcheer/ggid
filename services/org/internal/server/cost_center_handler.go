package httpserver

import (
	"encoding/json"
	"net/http"
	"sync"
)

type CostCenterDepartment struct {
	DepartmentID   string  `json:"department_id"`
	Name           string  `json:"name"`
	CostCenter     string  `json:"cost_center"`
	MemberCount    int     `json:"member_count"`
	ResourceUsage  float64 `json:"resource_usage"`
	BudgetLimit    float64 `json:"budget_limit"`
	UtilizationPct float64 `json:"utilization_pct"`
}

type BudgetAlert struct {
	DepartmentID string  `json:"department_id"`
	AlertType    string  `json:"alert_type"`
	Threshold    float64 `json:"threshold"`
	CurrentValue float64 `json:"current_value"`
	Severity     string  `json:"severity"`
}

type AllocationSummary struct {
	TotalBudget    float64 `json:"total_budget"`
	TotalUsage     float64 `json:"total_usage"`
	OverallUtilPct float64 `json:"overall_utilization_pct"`
	DepartmentCount int   `json:"department_count"`
}

type CostCenterResult struct {
	Departments       []CostCenterDepartment `json:"departments"`
	BudgetAlerts      []BudgetAlert          `json:"budget_alerts"`
	AllocationSummary AllocationSummary      `json:"allocation_summary"`
}

var costCenterStore sync.Map
var costCenterOnce sync.Once

func initCostCenterData() {
	costCenterOnce.Do(func() {
		data := CostCenterResult{
			Departments: []CostCenterDepartment{
				{DepartmentID: "d-001", Name: "Engineering", CostCenter: "CC-ENG-001", MemberCount: 45, ResourceUsage: 125000, BudgetLimit: 150000, UtilizationPct: 83.3},
				{DepartmentID: "d-002", Name: "Sales", CostCenter: "CC-SAL-001", MemberCount: 20, ResourceUsage: 78000, BudgetLimit: 80000, UtilizationPct: 97.5},
				{DepartmentID: "d-003", Name: "Marketing", CostCenter: "CC-MKT-001", MemberCount: 12, ResourceUsage: 35000, BudgetLimit: 60000, UtilizationPct: 58.3},
				{DepartmentID: "d-004", Name: "Operations", CostCenter: "CC-OPS-001", MemberCount: 18, ResourceUsage: 92000, BudgetLimit: 90000, UtilizationPct: 102.2},
			},
			BudgetAlerts: []BudgetAlert{
				{DepartmentID: "d-002", AlertType: "budget_warning", Threshold: 90.0, CurrentValue: 97.5, Severity: "warning"},
				{DepartmentID: "d-004", AlertType: "budget_exceeded", Threshold: 100.0, CurrentValue: 102.2, Severity: "critical"},
			},
			AllocationSummary: AllocationSummary{
				TotalBudget: 380000, TotalUsage: 330000, OverallUtilPct: 86.8, DepartmentCount: 4,
			},
		}
		costCenterStore.Store("latest", data)
	})
}

func (s *HTTPServer) handleCostCenters(w http.ResponseWriter, r *http.Request) {
	initCostCenterData()
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	val, ok := costCenterStore.Load("latest")
	if !ok {
		http.Error(w, "no data", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(val)
}
