package server

import (
	"encoding/json"
	"net/http"
)

type DataCategory struct {
	Category           string `json:"category"`
	SensitivityLevel   string `json:"sensitivity_level"`
	LawfulBasis        string `json:"lawful_basis"`
	RetentionPeriod    string `json:"retention_period"`
	CrossBorderStatus  string `json:"cross_border_status"`
	DataSubjectsCount  int    `json:"data_subjects_count"`
}

type PIPLDataInventoryResult struct {
	DataCategories   []DataCategory `json:"data_categories"`
	TotalSubjects    int            `json:"total_data_subjects"`
	CrossBorderCount int            `json:"cross_border_categories"`
	CompliancePct    float64        `json:"compliance_pct"`
	GeneratedAt      string         `json:"generated_at"`
}

func (h *HTTPHandler) handlePIPLDataInventory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	result := PIPLDataInventoryResult{
		DataCategories: []DataCategory{
			{Category: "basic_info", SensitivityLevel: "general", LawfulBasis: "contract_performance", RetentionPeriod: "5_years", CrossBorderStatus: "approved", DataSubjectsCount: 12450},
			{Category: "biometric", SensitivityLevel: "sensitive", LawfulBasis: "separate_consent", RetentionPeriod: "1_year", CrossBorderStatus: "restricted", DataSubjectsCount: 320},
			{Category: "location", SensitivityLevel: "sensitive", LawfulBasis: "legitimate_interest", RetentionPeriod: "90_days", CrossBorderStatus: "restricted", DataSubjectsCount: 8900},
			{Category: "financial", SensitivityLevel: "sensitive", LawfulBasis: "legal_obligation", RetentionPeriod: "10_years", CrossBorderStatus: "approved", DataSubjectsCount: 5600},
			{Category: "identity_docs", SensitivityLevel: "sensitive", LawfulBasis: "legal_obligation", RetentionPeriod: "5_years", CrossBorderStatus: "restricted", DataSubjectsCount: 12450},
			{Category: "health_records", SensitivityLevel: "sensitive", LawfulBasis: "separate_consent", RetentionPeriod: "30_years", CrossBorderStatus: "prohibited", DataSubjectsCount: 0},
		},
		TotalSubjects:    12450,
		CrossBorderCount: 4,
		CompliancePct:    96.8,
		GeneratedAt:      "2025-01-15T10:00:00Z",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
