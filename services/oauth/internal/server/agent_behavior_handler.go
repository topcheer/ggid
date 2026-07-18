package server

import (
	"encoding/json"
	"net/http"
	"strings"
)

type BehaviorBaseline struct {
	RequestRate int      `json:"request_rate"`
	APIPatterns []string `json:"api_patterns"`
	AccessHours string   `json:"access_hours"`
}

type AnomalyAlert struct {
	AlertID   string  `json:"alert_id"`
	Type      string  `json:"type"`
	Severity  string  `json:"severity"`
	Detail    string  `json:"detail"`
	Score     float64 `json:"score"`
}

type AgentBehaviorResult struct {
	AgentID              string           `json:"agent_id"`
	Baseline             BehaviorBaseline `json:"baseline"`
	Current              BehaviorBaseline `json:"current"`
	DeviationScore       float64          `json:"deviation_score"`
	AnomalyAlerts        []AnomalyAlert   `json:"anomaly_alerts"`
	AutoSuspendThreshold float64          `json:"auto_suspend_threshold"`
	SuspendRecommended   bool             `json:"suspend_recommended"`
}

func handleAgentBehavior(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	parts := strings.Split(r.URL.Path, "/")
	agentID := "unknown"
	if len(parts) >= 5 {
		agentID = parts[4]
	}

	result := AgentBehaviorResult{
		AgentID: agentID,
		Baseline: BehaviorBaseline{
			RequestRate: 200, APIPatterns: []string{"GET /api/v1/users", "POST /api/v1/audit"}, AccessHours: "08:00-18:00 UTC",
		},
		Current: BehaviorBaseline{
			RequestRate: 1850, APIPatterns: []string{"GET /api/v1/users", "DELETE /api/v1/users", "POST /api/v1/admin"}, AccessHours: "02:00-04:00 UTC",
		},
		DeviationScore: 0.78,
		AnomalyAlerts: []AnomalyAlert{
			{AlertID: "aa-001", Type: "request_rate_spike", Severity: "critical", Detail: "9.2x normal request rate", Score: 0.91},
			{AlertID: "aa-002", Type: "new_api_pattern", Severity: "high", Detail: "DELETE /api/v1/users not in baseline", Score: 0.82},
			{AlertID: "aa-003", Type: "off_hours_access", Severity: "high", Detail: "Active at 2-4AM UTC, baseline is 8AM-6PM", Score: 0.75},
			{AlertID: "aa-004", Type: "admin_endpoint_access", Severity: "critical", Detail: "POST /api/v1/admin outside normal scope", Score: 0.88},
		},
		AutoSuspendThreshold: 0.70,
		SuspendRecommended:   true,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
