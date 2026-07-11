package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
)

type DeptBudget struct {
	DeptID       string  `json:"dept_id"`
	Budget       float64 `json:"budget"`
	Allocated    float64 `json:"allocated"`
	Headcount    int     `json:"headcount"`
	HeadcountCost float64 `json:"headcount_cost"`
}

var (
	deptBudgetMu sync.RWMutex
	deptBudgets  = make(map[string]*DeptBudget) // key: orgID/deptID
)

func budgetKey(orgID, deptID string) string {
	return orgID + "/" + deptID
}

// Routes: /api/v1/organizations/{id}/budget-summary
//         /api/v1/organizations/{id}/departments/{dept_id}/budget
func (s *HTTPServer) handleOrgBudget(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	if strings.HasSuffix(path, "/budget-summary") {
		if r.Method != http.MethodGet {
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		// Extract orgID: /api/v1/organizations/{id}/budget-summary
		parts := strings.Split(path, "/")
		if len(parts) < 5 {
			writeJSONError(w, http.StatusBadRequest, "invalid path")
			return
		}
		orgID := parts[4]

		deptBudgetMu.RLock()
		var depts []*DeptBudget
		var totalBudget, totalAllocated, totalHCCost float64
		totalHC := 0
		for k, db := range deptBudgets {
			if strings.HasPrefix(k, orgID+"/") {
				depts = append(depts, db)
				totalBudget += db.Budget
				totalAllocated += db.Allocated
				totalHC += db.Headcount
				totalHCCost += db.HeadcountCost
			}
		}
		deptBudgetMu.RUnlock()

		writeJSON(w, http.StatusOK, map[string]any{
			"org_id":          orgID,
			"total_budget":    totalBudget,
			"allocated":       totalAllocated,
			"remaining":       totalBudget - totalAllocated,
			"headcount":       totalHC,
			"headcount_cost":  totalHCCost,
			"departments":     depts,
		})
		return
	}

	// PUT /api/v1/organizations/{id}/departments/{dept_id}/budget
	if strings.Contains(path, "/departments/") && strings.HasSuffix(path, "/budget") {
		parts := strings.Split(path, "/")
		if len(parts) < 7 {
			writeJSONError(w, http.StatusBadRequest, "invalid path")
			return
		}
		orgID := parts[4]
		deptID := parts[6]

		if r.Method == http.MethodPut || r.Method == http.MethodPost {
			var req struct {
				Budget        float64 `json:"budget"`
				Allocated     float64 `json:"allocated"`
				Headcount     int     `json:"headcount"`
				HeadcountCost float64 `json:"headcount_cost"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeJSONError(w, http.StatusBadRequest, "invalid JSON")
				return
			}
			key := budgetKey(orgID, deptID)
			db := &DeptBudget{
				DeptID: deptID, Budget: req.Budget, Allocated: req.Allocated,
				Headcount: req.Headcount, HeadcountCost: req.HeadcountCost,
			}
			deptBudgetMu.Lock()
			deptBudgets[key] = db
			deptBudgetMu.Unlock()
			writeJSON(w, http.StatusOK, map[string]any{
				"status": "updated", "org_id": orgID, "department": db,
			})
			return
		}
	}

	writeJSONError(w, http.StatusNotFound, "not found")
}
