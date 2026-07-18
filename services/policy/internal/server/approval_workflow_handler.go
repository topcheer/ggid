package httpserver

import (
	"encoding/json"
	"net/http"
)

type ApprovalWorkflowResult struct {
	Pipeline            []string                    `json:"pipeline"`
	ReviewerAssignment  map[string]string           `json:"reviewer_assignment"`
	ChangeFreezeWindows []struct {
		Name      string `json:"name"`
		Start     string `json:"start"`
		End       string `json:"end"`
	} `json:"change_freeze_windows"`
	EmergencyBypass struct {
		Enabled         bool   `json:"enabled"`
		RequiredApprover string `json:"required_approver"`
		CoolDownMins    int    `json:"cooldown_minutes"`
	} `json:"emergency_bypass"`
	SoDEnforcement bool `json:"sod_enforcement"`
}

func (s *HTTPServer) handleApprovalWorkflow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	result := ApprovalWorkflowResult{
		Pipeline:           []string{"draft", "review", "approve", "activate"},
		ReviewerAssignment: map[string]string{"access": "security-team", "data": "dpo", "infra": "devops-lead", "compliance": "compliance-officer"},
		ChangeFreezeWindows: []struct {
			Name  string `json:"name"`
			Start string `json:"start"`
			End   string `json:"end"`
		}{
			{Name: "holiday_freeze", Start: "2025-12-20T00:00:00Z", End: "2026-01-05T00:00:00Z"},
			{Name: "month_end_freeze", Start: "2025-01-28T17:00:00Z", End: "2025-02-01T09:00:00Z"},
		},
	}
	result.EmergencyBypass.Enabled = true
	result.EmergencyBypass.RequiredApprover = "ciso"
	result.EmergencyBypass.CoolDownMins = 60
	result.SoDEnforcement = true
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
