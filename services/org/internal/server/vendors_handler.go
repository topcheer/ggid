package httpserver

import (
	"encoding/json"
	"net/http"
)

type Vendor struct {
	VendorID         string  `json:"vendor_id"`
	VendorName       string  `json:"vendor_name"`
	ServiceType      string  `json:"service_type"`
	 DataAccessScope  string  `json:"data_access_scope"`
	RiskRating       string  `json:"risk_rating"`
	ContractExpiry   string  `json:"contract_expiry"`
	ComplianceStatus string  `json:"compliance_status"`
	LastAssessment   string  `json:"last_assessment"`
}

type VendorListResult struct {
	Vendors     []Vendor `json:"vendors"`
	TotalCount  int      `json:"total_count"`
	HighRisk    int      `json:"high_risk_count"`
	ExpiringSoon int     `json:"expiring_within_30d"`
}

func (s *HTTPServer) handleVendors(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	result := VendorListResult{
		Vendors: []Vendor{
			{VendorID: "v-001", VendorName: "AWS", ServiceType: "cloud_infrastructure", DataAccessScope: "compute, storage, network", RiskRating: "low", ContractExpiry: "2025-12-31", ComplianceStatus: "SOC2_TypeII", LastAssessment: "2024-11-15"},
			{VendorID: "v-002", VendorName: "Datadog", ServiceType: "monitoring", DataAccessScope: "metrics, logs (read-only)", RiskRating: "low", ContractExpiry: "2025-06-30", ComplianceStatus: "SOC2_TypeII", LastAssessment: "2024-10-20"},
			{VendorID: "v-003", VendorName: "Stripe", ServiceType: "payments", DataAccessScope: "billing, PII (encrypted)", RiskRating: "medium", ContractExpiry: "2025-03-15", ComplianceStatus: "PCI_DSS", LastAssessment: "2024-12-01"},
			{VendorID: "v-004", VendorName: "Slack", ServiceType: "communication", DataAccessScope: "messages, files", RiskRating: "low", ContractExpiry: "2025-09-30", ComplianceStatus: "SOC2_TypeII", LastAssessment: "2024-09-10"},
			{VendorID: "v-005", VendorName: "ZoomInfo", ServiceType: "sales_intelligence", DataAccessScope: "contact data enrichment", RiskRating: "high", ContractExpiry: "2025-02-01", ComplianceStatus: "GDPR_Review_Pending", LastAssessment: "2024-08-05"},
		},
		TotalCount:   5,
		HighRisk:     1,
		ExpiringSoon: 2,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
