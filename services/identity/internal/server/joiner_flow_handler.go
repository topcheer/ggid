package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type PreboardingTask struct {
	TaskID     string `json:"task_id"`
	Title      string `json:"title"`
	Category   string `json:"category"`
	Required   bool   `json:"required"`
	Completed  bool   `json:"completed"`
}

type JoinerFlow struct {
	FlowID              string           `json:"flow_id"`
	EmployeeID          string           `json:"employee_id"`
	Name                string           `json:"name"`
	StartDate           string           `json:"start_date"`
	Department          string           `json:"department"`
	RoleTemplates       []string         `json:"role_templates"`
	AutoProvisionApps   []string         `json:"auto_provision_apps"`
	PreboardingTasks     []PreboardingTask `json:"preboarding_tasks"`
	Status              string           `json:"status"`
	ProgressPct         float64          `json:"progress_pct"`
	CreatedAt           string           `json:"created_at"`
}

type JoinerFlowRequest struct {
	EmployeeID        string   `json:"employee_id"`
	Name              string   `json:"name"`
	StartDate         string   `json:"start_date"`
	Department        string   `json:"department"`
	RoleTemplates     []string `json:"role_templates"`
	AutoProvisionApps []string `json:"auto_provision_apps"`
}

var (
	joinerFlowsStore sync.Map
)

func init() {
	joinerFlowsStore.Store("seed-1", JoinerFlow{
		FlowID:    "jf-001",
		EmployeeID: "emp-001",
		Name:      "John Smith",
		StartDate: "2025-02-01",
		Department: "Engineering",
		RoleTemplates: []string{"engineer-base", "github-access"},
		AutoProvisionApps: []string{"slack", "github", "jira", "gsuite"},
		PreboardingTasks: []PreboardingTask{
			{TaskID: "t-1", Title: "Create AD account", Category: "identity", Required: true, Completed: true},
			{TaskID: "t-2", Title: "Provision laptop", Category: "hardware", Required: true, Completed: true},
			{TaskID: "t-3", Title: "Grant repo access", Category: "access", Required: true, Completed: false},
			{TaskID: "t-4", Title: "Schedule orientation", Category: "onboarding", Required: false, Completed: false},
		},
		Status:    "in_progress",
		ProgressPct: 50.0,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	})
}

func (h *HTTPHandler) handleJoinerFlow(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		var flows []JoinerFlow
		joinerFlowsStore.Range(func(_, v any) bool {
			flows = append(flows, v.(JoinerFlow))
			return true
		})
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"flows": flows, "count": len(flows)})
	case http.MethodPost:
		var req JoinerFlowRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.EmployeeID == "" {
			req.EmployeeID = fmt.Sprintf("emp-%d", time.Now().UnixNano()%10000)
		}
		if len(req.RoleTemplates) == 0 {
			req.RoleTemplates = []string{"base-employee"}
		}
		if len(req.AutoProvisionApps) == 0 {
			req.AutoProvisionApps = []string{"email", "slack"}
		}
		tasks := []PreboardingTask{
			{TaskID: "t-1", Title: "Create identity account", Category: "identity", Required: true, Completed: false},
			{TaskID: "t-2", Title: "Provision hardware", Category: "hardware", Required: true, Completed: false},
			{TaskID: "t-3", Title: "Grant app access", Category: "access", Required: true, Completed: false},
			{TaskID: "t-4", Title: "Schedule orientation", Category: "onboarding", Required: false, Completed: false},
		}
		flow := JoinerFlow{
			FlowID:            fmt.Sprintf("jf-%d", time.Now().UnixNano()%100000),
			EmployeeID:        req.EmployeeID,
			Name:              req.Name,
			StartDate:         req.StartDate,
			Department:        req.Department,
			RoleTemplates:     req.RoleTemplates,
			AutoProvisionApps: req.AutoProvisionApps,
			PreboardingTasks:   tasks,
			Status:            "initiated",
			ProgressPct:       0,
			CreatedAt:         time.Now().UTC().Format(time.RFC3339),
		}
		joinerFlowsStore.Store(flow.FlowID, flow)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(flow)
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
