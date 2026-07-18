package server

import (
	"encoding/json"
	"net/http"
)

type SensitiveAttribute struct {
	Name            string `json:"name"`
	PIIClass        string `json:"pii_classification"`
	AccessFreq      int    `json:"access_frequency_30d"`
	LastAccessedBy  string `json:"last_accessed_by"`
	LastAccessedAt  string `json:"last_accessed_at"`
	MaskRule        string `json:"mask_rule"`
	RetentionDays   int    `json:"retention_days"`
}

type AttributeGovernanceResult struct {
	SensitiveAttributes []SensitiveAttribute `json:"sensitive_attributes"`
	TotalAttributes     int                   `json:"total_attributes"`
	HighRiskCount       int                   `json:"high_risk_count"`
	PolicyCompliancePct float64               `json:"policy_compliance_pct"`
}

func (h *HTTPHandler) handleAttributeGovernance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	result := AttributeGovernanceResult{
		SensitiveAttributes: []SensitiveAttribute{
			{Name: "ssn", PIIClass: "high", AccessFreq: 12, LastAccessedBy: "admin@ggid.dev", LastAccessedAt: "2025-01-14T15:30:00Z", MaskRule: "full_mask", RetentionDays: 365},
			{Name: "email", PIIClass: "medium", AccessFreq: 8420, LastAccessedBy: "system@example.com", LastAccessedAt: "2025-01-15T09:00:00Z", MaskRule: "partial_mask", RetentionDays: 1095},
			{Name: "phone", PIIClass: "medium", AccessFreq: 1240, LastAccessedBy: "admin@ggid.dev", LastAccessedAt: "2025-01-14T11:20:00Z", MaskRule: "partial_mask", RetentionDays: 1095},
			{Name: "date_of_birth", PIIClass: "high", AccessFreq: 45, LastAccessedBy: "hr@ggid.dev", LastAccessedAt: "2025-01-10T14:00:00Z", MaskRule: "full_mask", RetentionDays: 2555},
			{Name: "home_address", PIIClass: "medium", AccessFreq: 89, LastAccessedBy: "hr@ggid.dev", LastAccessedAt: "2025-01-08T10:15:00Z", MaskRule: "partial_mask", RetentionDays: 1825},
			{Name: "salary", PIIClass: "high", AccessFreq: 23, LastAccessedBy: "finance@ggid.dev", LastAccessedAt: "2025-01-12T16:45:00Z", MaskRule: "full_mask", RetentionDays: 2555},
		},
		TotalAttributes:     42,
		HighRiskCount:       3,
		PolicyCompliancePct: 94.5,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
