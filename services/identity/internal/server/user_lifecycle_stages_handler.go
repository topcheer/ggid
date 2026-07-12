package server

import (
	"encoding/json"
	"net/http"
)

type LifecycleStage struct {
	Stage  string `json:"stage"`
	Count  int    `json:"count"`
	AvgDaysInStage float64 `json:"avg_time_in_stage_days"`
}

type UserLifecycleResult struct {
	Stages              []LifecycleStage `json:"stages"`
	TotalUsers          int              `json:"total_users"`
	TransitionRules     []string         `json:"transition_rules"`
	AutoDeactivateRules []string         `json:"auto_deactivate_rules"`
}

func (h *HTTPHandler) handleUserLifecycleStages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	result := UserLifecycleResult{
		Stages: []LifecycleStage{
			{Stage: "active", Count: 8420, AvgDaysInStage: 420},
			{Stage: "dormant", Count: 580, AvgDaysInStage: 90},
			{Stage: "suspended", Count: 42, AvgDaysInStage: 7},
			{Stage: "deactivated", Count: 180, AvgDaysInStage: 0},
			{Stage: "pending", Count: 35, AvgDaysInStage: 2},
		},
		TotalUsers: 9257,
		TransitionRules: []string{
			"active → dormant: no_login_90d",
			"dormant → active: any_login",
			"dormant → suspended: no_login_180d",
			"suspended → deactivated: admin_action_30d_no_appeal",
			"pending → active: email_verified",
		},
		AutoDeactivateRules: []string{
			"suspended users auto-deactivated after 30 days",
			"dormant users flagged for review after 180 days",
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
