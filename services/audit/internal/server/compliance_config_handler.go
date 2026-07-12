package httpserver

import (
	"encoding/json"
	"net/http"
)

type ComplianceConfig struct {
	ComplianceFrameworks      []string               `json:"compliance_frameworks"`
	DataClassificationRules   map[string]string      `json:"data_classification_rules"`
	RetentionPerCategory      map[string]int         `json:"retention_per_category"`
	LegalHoldConfig           map[string]any         `json:"legal_hold_config"`
	AutomatedDeletionSchedule map[string]any         `json:"automated_deletion_schedule"`
	CrossBorderTransferLog    bool                   `json:"cross_border_transfer_log"`
}

var globalComplianceConfig = &ComplianceConfig{
	ComplianceFrameworks: []string{"soc2", "gdpr", "hipaa", "iso27001", "pci"},
	DataClassificationRules: map[string]string{
		"pii":    "restricted",
		"phi":    "restricted",
		"financial": "confidential",
		"public": "public",
	},
	RetentionPerCategory: map[string]int{
		"restricted":   2555,
		"confidential": 1825,
		"internal":     1095,
		"public":       365,
	},
	LegalHoldConfig: map[string]any{
		"enabled":              true,
		"bypass_retention":     true,
		"require_approval":     true,
		"approver_role":        "legal_admin",
	},
	AutomatedDeletionSchedule: map[string]any{
		"enabled":          true,
		"frequency":        "daily",
		"batch_size":       1000,
		"dry_run_default":  true,
	},
	CrossBorderTransferLog: true,
}

func (s *HTTPServer) handleComplianceConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(globalComplianceConfig)
	case http.MethodPut:
		var cfg ComplianceConfig
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
			return
		}
		globalComplianceConfig = &cfg
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(cfg)
	default:
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
	}
}
