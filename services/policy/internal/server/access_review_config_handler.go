package httpserver

import (
	"encoding/json"
	"net/http"
)

type AccessReviewConfig struct {
	ReviewCycle         string            `json:"review_cycle"`
	CustomCycleDays     int               `json:"custom_cycle_days"`
	ReviewerAssignment  string            `json:"reviewer_assignment"`
	NotificationSettings map[string]any   `json:"notification_settings"`
	AutoRevokeOnFail    bool              `json:"auto_revoke_on_fail"`
	GracePeriodDays     int               `json:"grace_period_days"`
	ScopeFilters        []string          `json:"scope_filters"`
}

var globalAccessReviewConfig = &AccessReviewConfig{
	ReviewCycle:        "quarterly",
	CustomCycleDays:    90,
	ReviewerAssignment: "manager",
	NotificationSettings: map[string]any{
		"advance_notice_days": 7,
		"reminder_interval_days": 3,
		"escalation_after_days": 14,
	},
	AutoRevokeOnFail: false,
	GracePeriodDays:  30,
	ScopeFilters:     []string{"active_only", "high_risk_roles", "privileged_access"},
}

func (s *HTTPServer) handleAccessReviewConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(globalAccessReviewConfig)
	case http.MethodPut:
		var cfg AccessReviewConfig
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
			return
		}
		globalAccessReviewConfig = &cfg
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(cfg)
	default:
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
	}
}
