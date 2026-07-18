package server

import (
	"encoding/json"
	"net/http"
)

type IdPEndpoint struct {
	Name        string  `json:"name"`
	Status      string  `json:"status"`
	LatencyMs   float64 `json:"latency_ms"`
	HealthScore float64 `json:"health_score"`
}

type FailoverEvent struct {
	Timestamp string `json:"timestamp"`
	From      string `json:"from"`
	To        string `json:"to"`
	Trigger   string `json:"trigger"`
	Duration  string `json:"duration"`
}

type FailoverConfigResult struct {
	Primary    IdPEndpoint   `json:"primary"`
	Secondary  IdPEndpoint   `json:"secondary"`
	FailoverRules []struct {
		Trigger string `json:"trigger"`
		Action  string `json:"action"`
	} `json:"failover_rules"`
	FailoverHistory []FailoverEvent `json:"failover_history"`
	AutoFallback    bool            `json:"auto_fallback"`
	CurrentActive   string          `json:"current_active"`
}

func (h *HTTPHandler) handleIdPFailoverConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		result := FailoverConfigResult{
			Primary:   IdPEndpoint{Name: "okta-prod", Status: "healthy", LatencyMs: 45.2, HealthScore: 0.98},
			Secondary: IdPEndpoint{Name: "azure-ad-backup", Status: "standby", LatencyMs: 120.5, HealthScore: 0.95},
			FailoverRules: []struct {
				Trigger string `json:"trigger"`
				Action  string `json:"action"`
			}{
				{Trigger: "primary_latency > 500ms", Action: "switch_to_secondary"},
				{Trigger: "primary_error_rate > 5%", Action: "switch_to_secondary"},
				{Trigger: "primary_health < 0.5", Action: "switch_to_secondary"},
			},
			FailoverHistory: []FailoverEvent{
				{Timestamp: "2025-01-10T14:00:00Z", From: "okta-prod", To: "azure-ad-backup", Trigger: "primary_latency > 500ms", Duration: "2h30m"},
				{Timestamp: "2024-12-20T09:00:00Z", From: "okta-prod", To: "azure-ad-backup", Trigger: "scheduled_maintenance", Duration: "1h"},
			},
			AutoFallback:  true,
			CurrentActive: "okta-prod",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	case http.MethodPut:
		var req struct {
			AutoFallback *bool  `json:"auto_fallback"`
			ManualSwitch string `json:"manual_switch"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"status": "updated", "active": "okta-prod"})
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
