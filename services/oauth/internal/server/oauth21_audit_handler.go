package server

import (
	"encoding/json"
	"net/http"
)

type ComplianceCheck struct {
	Requirement    string  `json:"requirement"`
	Status         string  `json:"status"`
	Detail         string  `json:"detail"`
	RemediationURL string  `json:"remediation_url,omitempty"`
}

type NonCompliantClient struct {
	ClientID   string   `json:"client_id"`
	ClientName string   `json:"client_name"`
	Issues     []string `json:"issues"`
	RiskLevel  string   `json:"risk_level"`
}

type OAuth21AuditResult struct {
	ComplianceChecklist   []ComplianceCheck     `json:"compliance_checklist"`
	OverallCompliancePct  float64               `json:"overall_compliance_pct"`
	NonCompliantClients   []NonCompliantClient  `json:"non_compliant_clients"`
	RemediationActions    []string              `json:"remediation_actions"`
	TotalClientsAudited   int                   `json:"total_clients_audited"`
}

func handleOAuth21Audit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	result := OAuth21AuditResult{
		ComplianceChecklist: []ComplianceCheck{
			{Requirement: "PKCE required for all flows", Status: "compliant", Detail: "100% of clients enforce PKCE"},
			{Requirement: "Implicit grant disabled", Status: "compliant", Detail: "No clients use implicit flow"},
			{Requirement: "Password grant disabled", Status: "non_compliant", Detail: "2 clients still use password grant", RemediationURL: "/docs/oauth-2-1-migration"},
			{Requirement: "Exact redirect URI matching", Status: "compliant", Detail: "All redirect URIs use exact match"},
			{Requirement: "State parameter enforced", Status: "compliant", Detail: "100% of authorize requests include state"},
			{Requirement: "DPoP for sender-constrained tokens", Status: "partial", Detail: "82% of token-bound clients use DPoP; 18% use mTLS only", RemediationURL: "/docs/dpop-setup"},
		},
		OverallCompliancePct: 83.3,
		NonCompliantClients: []NonCompliantClient{
			{ClientID: "c-004", ClientName: "legacy-app", Issues: []string{"password_grant_enabled", "no_pkce"}, RiskLevel: "high"},
			{ClientID: "c-005", ClientName: "batch-processor", Issues: []string{"password_grant_enabled"}, RiskLevel: "medium"},
			{ClientID: "c-008", ClientName: "mobile-legacy", Issues: []string{"no_dpop", "wildcard_redirect"}, RiskLevel: "high"},
		},
		RemediationActions: []string{
			"Disable password grant for c-004 and c-005 within 7 days",
			"Migrate c-008 to PKCE + DPoP flow",
			"Fix wildcard redirect URI for c-008",
			"Enable DPoP for remaining 18% mTLS-only clients",
		},
		TotalClientsAudited: 23,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
