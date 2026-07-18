package server

import (
	"encoding/json"
	"net/http"
	"strings"
)

type BehavioralAlert struct {
	AlertID   string  `json:"alert_id"`
	Type      string  `json:"type"`
	Severity  string  `json:"severity"`
	Timestamp string  `json:"timestamp"`
	Detail    string  `json:"detail"`
}

type AgentLifecycleResult struct {
	AgentID              string            `json:"agent_id"`
	AgentName            string            `json:"agent_name"`
	Status               string            `json:"status"`
	CreatedAt            string            `json:"created_at"`
	LastActive           string            `json:"last_active"`
	Permissions          []string          `json:"permissions"`
	CredentialRotationDue bool             `json:"credential_rotation_due"`
	RotationDueAt        string            `json:"rotation_due_at"`
	BehavioralAlerts     []BehavioralAlert `json:"behavioral_alerts"`
	RequestRate24h       int               `json:"request_rate_24h"`
}

func handleAgentLifecycle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	parts := strings.Split(r.URL.Path, "/")
	agentID := "unknown"
	if len(parts) >= 5 {
		agentID = parts[4]
	}

	result := AgentLifecycleResult{
		AgentID:              agentID,
		AgentName:            "service-agent-" + agentID,
		Status:               "active",
		CreatedAt:            "2024-12-01T00:00:00Z",
		LastActive:           "2025-01-15T09:45:00Z",
		Permissions:          []string{"read:users", "write:audit", "read:policies"},
		CredentialRotationDue: false,
		RotationDueAt:        "2025-03-01T00:00:00Z",
		BehavioralAlerts: []BehavioralAlert{
			{AlertID: "ba-001", Type: "unusual_scope_request", Severity: "medium", Timestamp: "2025-01-14T22:00:00Z", Detail: "Requested scope admin:write outside normal pattern"},
			{AlertID: "ba-002", Type: "off_hours_activity", Severity: "low", Timestamp: "2025-01-13T03:00:00Z", Detail: "API calls at 3AM UTC, outside expected hours"},
		},
		RequestRate24h: 4820,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
