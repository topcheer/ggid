package server

import (
	"encoding/json"
	"net/http"
)

type ProvisioningLogEntry struct {
	Timestamp    string `json:"timestamp"`
	UserID       string `json:"user_id"`
	Source       string `json:"source"`
	Action       string `json:"action"`
	TargetApp    string `json:"target_app"`
	Status       string `json:"status"`
	ErrorDetails string `json:"error_details,omitempty"`
	Retry        int    `json:"retry"`
}

type ProvisioningLogResult struct {
	Events      []ProvisioningLogEntry `json:"events"`
	TotalEvents int                 `json:"total_events"`
	SuccessCount int                `json:"success_count"`
	FailedCount  int                `json:"failed_count"`
	PendingCount int                `json:"pending_count"`
}

func (h *HTTPHandler) handleProvisioningLog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	result := ProvisioningLogResult{
		Events: []ProvisioningLogEntry{
			{Timestamp: "2025-01-15T09:30:00Z", UserID: "u-0342", Source: "SCIM", Action: "create", TargetApp: "slack", Status: "success", Retry: 0},
			{Timestamp: "2025-01-15T09:25:00Z", UserID: "u-0517", Source: "JIT", Action: "update", TargetApp: "google_workspace", Status: "success", Retry: 0},
			{Timestamp: "2025-01-15T09:00:00Z", UserID: "u-0891", Source: "SCIM", Action: "disable", TargetApp: "zoom", Status: "failed", ErrorDetails: "connection_timeout", Retry: 2},
			{Timestamp: "2025-01-15T08:45:00Z", UserID: "u-0420", Source: "manual", Action: "create", TargetApp: "github", Status: "pending", Retry: 0},
			{Timestamp: "2025-01-15T08:30:00Z", UserID: "u-0342", Source: "SCIM", Action: "create", TargetApp: "google_workspace", Status: "success", Retry: 0},
			{Timestamp: "2025-01-14T22:00:00Z", UserID: "u-0517", Source: "SCIM", Action: "update", TargetApp: "slack", Status: "failed", ErrorDetails: "rate_limit_exceeded", Retry: 3},
		},
		TotalEvents:  6,
		SuccessCount: 3,
		FailedCount:  2,
		PendingCount: 1,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
