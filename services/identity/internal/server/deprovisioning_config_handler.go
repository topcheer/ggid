package server

import (
	"encoding/json"
	"net/http"
)

type DeprovisioningWorkflowStep string

const (
	DeprovisioningStepNotify          DeprovisioningWorkflowStep = "notify"
	DeprovisioningStepDisableSessions DeprovisioningWorkflowStep = "disable_sessions"
	DeprovisioningStepRevokeTokens    DeprovisioningWorkflowStep = "revoke_tokens"
	DeprovisioningStepArchiveData     DeprovisioningWorkflowStep = "archive_data"
	DeprovisioningStepNotifyManagers  DeprovisioningWorkflowStep = "notify_managers"
)

type DeprovisioningConfig struct {
	WorkflowSteps           []DeprovisioningWorkflowStep `json:"workflow_steps"`
	GracePeriodDays         int                          `json:"grace_period_days"`
	CascadeToLinkedAccounts bool                         `json:"cascade_to_linked_accounts"`
	DataRetentionPolicy     string                       `json:"data_retention_policy"`
	NotificationTemplates   map[string]string            `json:"notification_templates"`
	DryRun                  bool                         `json:"dry_run"`
}

var globalDeprovisioningConfig = &DeprovisioningConfig{
	WorkflowSteps: []DeprovisioningWorkflowStep{
		DeprovisioningStepNotify,
		DeprovisioningStepDisableSessions,
		DeprovisioningStepRevokeTokens,
		DeprovisioningStepArchiveData,
		DeprovisioningStepNotifyManagers,
	},
	GracePeriodDays:         7,
	CascadeToLinkedAccounts: false,
	DataRetentionPolicy:     "retain_90_days",
	NotificationTemplates: map[string]string{
		"user":    "deprovisioning_user_notification",
		"manager": "deprovisioning_manager_notification",
	},
	DryRun: false,
}

func (h *HTTPHandler) handleDeprovisioningConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(globalDeprovisioningConfig)
	case http.MethodPut:
		var cfg DeprovisioningConfig
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
			return
		}
		if cfg.GracePeriodDays < 0 {
			http.Error(w, `{"error":"grace_period_days must be non-negative"}`, http.StatusBadRequest)
			return
		}
		globalDeprovisioningConfig = &cfg
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(cfg)
	default:
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
	}
}
