package httpserver

import (
	"encoding/json"
	"net/http"
)

type AlertRule struct {
	Name      string `json:"name"`
	Condition string `json:"condition"`
	Severity  string `json:"severity"`
	Channel   string `json:"channel"`
}

type AlertEvaluationConfig struct {
	AlertRules         []AlertRule `json:"alert_rules"`
	EvaluationInterval int         `json:"evaluation_interval_seconds"`
	CorrelationWindow  int         `json:"correlation_window_minutes"`
	DedupStrategy      string      `json:"dedup_strategy"`
	EscalationRules    []string    `json:"escalation_rules"`
	SuppressAfterCount int         `json:"suppress_after_count"`
	NotifyChannels     []string    `json:"notify_channels"`
}

var globalAlertEvaluationConfig = &AlertEvaluationConfig{
	AlertRules: []AlertRule{
		{Name: "brute_force", Condition: "failed_logins > 10 in 5m", Severity: "high", Channel: "siem"},
		{Name: "privilege_escalation", Condition: "role_change_without_approval", Severity: "critical", Channel: "webhook"},
	},
	EvaluationInterval: 60,
	CorrelationWindow:  30,
	DedupStrategy:      "fingerprint",
	EscalationRules:    []string{"notify_manager_after_3", "page_oncall_after_5"},
	SuppressAfterCount: 10,
	NotifyChannels:     []string{"webhook", "email", "siem"},
}

func (s *HTTPServer) handleAlertEvaluationConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(globalAlertEvaluationConfig)
	case http.MethodPut:
		var cfg AlertEvaluationConfig
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		globalAlertEvaluationConfig = &cfg
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(cfg)
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}