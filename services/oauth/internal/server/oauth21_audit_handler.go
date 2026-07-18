package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/ggid/ggid/services/oauth/internal/domain"
	"github.com/ggid/ggid/services/oauth/internal/service"
)

type ComplianceCheck struct {
	Requirement    string `json:"requirement"`
	Status         string `json:"status"`
	Detail         string `json:"detail"`
	RemediationURL string `json:"remediation_url,omitempty"`
}

type NonCompliantClient struct {
	ClientID   string   `json:"client_id"`
	ClientName string   `json:"client_name"`
	Issues     []string `json:"issues"`
	RiskLevel  string   `json:"risk_level"`
}

type OAuth21AuditResult struct {
	ComplianceChecklist  []ComplianceCheck    `json:"compliance_checklist"`
	OverallCompliancePct float64             `json:"overall_compliance_pct"`
	NonCompliantClients  []NonCompliantClient `json:"non_compliant_clients"`
	RemediationActions   []string             `json:"remediation_actions"`
	TotalClientsAudited  int                   `json:"total_clients_audited"`
}

func handleOAuth21Audit(oauthSvc *service.OAuthService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	clients, _, err := oauthSvc.ListClients(r.Context(), 1000, 0)
	if err != nil {
		writeInternalError(w, "OAuth21Audit", err)
		return
	}

	var nonCompliantClients []NonCompliantClient
	var remediationActions []string
	checkResults := map[string]bool{
		"pkce":          true,
		"implicit":      true,
		"password":      true,
		"redirect_uris": true,
		"auth_method":   true,
	}

	for _, c := range clients {
		issues := auditClient(c)
		if len(issues) == 0 {
			continue
		}

		// Update global check results based on issue types
		for _, issue := range issues {
			switch issue {
			case "public_client_without_pkce":
				checkResults["pkce"] = false
			case "implicit_grant_enabled":
				checkResults["implicit"] = false
			case "password_grant_enabled":
				checkResults["password"] = false
			case "insecure_redirect_uri", "non_https_redirect_uri":
				checkResults["redirect_uris"] = false
			case "invalid_token_endpoint_auth_method":
				checkResults["auth_method"] = false
			}
		}

		riskLevel := "medium"
		for _, issue := range issues {
			if issue == "password_grant_enabled" || issue == "implicit_grant_enabled" || issue == "non_https_redirect_uri" {
				riskLevel = "high"
				break
			}
		}

		nonCompliantClients = append(nonCompliantClients, NonCompliantClient{
			ClientID:   c.ClientID,
			ClientName: c.Name,
			Issues:     issues,
			RiskLevel:  riskLevel,
		})

		remediationActions = append(remediationActions, buildRemediation(c, issues)...)
	}

	complianceChecklist := []ComplianceCheck{
		{
			Requirement: "PKCE required for all public clients",
			Status:      boolStatus(checkResults["pkce"]),
			Detail:      pkceDetail(checkResults["pkce"], len(nonCompliantClients)),
			RemediationURL: "/docs/oauth-2-1-migration",
		},
		{
			Requirement: "Implicit grant disabled",
			Status:      boolStatus(checkResults["implicit"]),
			Detail:      conditionalDetail(checkResults["implicit"], "No clients use implicit flow", "One or more clients use implicit flow"),
			RemediationURL: "/docs/oauth-2-1-migration",
		},
		{
			Requirement: "Password grant disabled",
			Status:      boolStatus(checkResults["password"]),
			Detail:      conditionalDetail(checkResults["password"], "No clients use password grant", "One or more clients use password grant"),
			RemediationURL: "/docs/oauth-2-1-migration",
		},
		{
			Requirement: "Exact redirect URI matching with HTTPS",
			Status:      boolStatus(checkResults["redirect_uris"]),
			Detail:      conditionalDetail(checkResults["redirect_uris"], "All redirect URIs use HTTPS", "One or more redirect URIs are not HTTPS"),
			RemediationURL: "/docs/oauth-2-1-migration",
		},
		{
			Requirement: "Token endpoint auth method allowed",
			Status:      boolStatus(checkResults["auth_method"]),
			Detail:      conditionalDetail(checkResults["auth_method"], "All clients use allowed auth methods", "One or more clients use unsupported auth methods"),
			RemediationURL: "/docs/oauth-2-1-migration",
		},
		{
			Requirement: "DPoP for sender-constrained tokens",
			Status:      "partial",
			Detail:      "DPoP enforcement requires client configuration review outside this audit scope",
			RemediationURL: "/docs/dpop-setup",
		},
	}

	overallPct := 100.0
	if len(clients) > 0 {
		compliant := len(clients) - len(nonCompliantClients)
		overallPct = float64(compliant) / float64(len(clients)) * 100
	}

	result := OAuth21AuditResult{
		ComplianceChecklist:  complianceChecklist,
		OverallCompliancePct: overallPct,
		NonCompliantClients:  nonCompliantClients,
		RemediationActions:   remediationActions,
		TotalClientsAudited:  len(clients),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
	}
}

func auditClient(c *domain.OAuthClient) []string {
	var issues []string

	for _, gt := range c.GrantTypes {
		switch gt {
		case "implicit":
			issues = append(issues, "implicit_grant_enabled")
		case "password":
			issues = append(issues, "password_grant_enabled")
		}
	}

	for _, uri := range c.RedirectURIs {
		if !strings.HasPrefix(uri, "https://") {
			issues = append(issues, "non_https_redirect_uri")
		}
		if strings.Contains(uri, "*") {
			issues = append(issues, "wildcard_redirect_uri")
		}
	}

	if c.IsPublic() && !c.RequirePKCE {
		issues = append(issues, "public_client_without_pkce")
	}

	allowedAuthMethods := map[string]bool{
		"client_secret_post":   true,
		"client_secret_basic":  true,
		"private_key_jwt":      true,
	}
	if c.TokenEndpointAuthMethod != "" && !allowedAuthMethods[c.TokenEndpointAuthMethod] {
		issues = append(issues, "invalid_token_endpoint_auth_method")
	}

	return issues
}

func buildRemediation(c *domain.OAuthClient, issues []string) []string {
	var actions []string
	for _, issue := range issues {
		switch issue {
		case "implicit_grant_enabled":
			actions = append(actions, "Disable implicit grant for client "+c.ClientID+" ("+c.Name+")")
		case "password_grant_enabled":
			actions = append(actions, "Disable password grant for client "+c.ClientID+" ("+c.Name+")")
		case "non_https_redirect_uri", "wildcard_redirect_uri":
			actions = append(actions, "Use HTTPS exact-match redirect URIs for client "+c.ClientID+" ("+c.Name+")")
		case "public_client_without_pkce":
			actions = append(actions, "Enable PKCE (S256) for client "+c.ClientID+" ("+c.Name+")")
		case "invalid_token_endpoint_auth_method":
			actions = append(actions, "Use client_secret_post, client_secret_basic, or private_key_jwt for client "+c.ClientID+" ("+c.Name+")")
		}
	}
	return actions
}

func boolStatus(ok bool) string {
	if ok {
		return "compliant"
	}
	return "non_compliant"
}

func conditionalDetail(ok bool, okText, failText string) string {
	if ok {
		return okText
	}
	return failText
}

func pkceDetail(ok bool, nonCompliantCount int) string {
	if ok {
		return "All public clients enforce PKCE"
	}
	if nonCompliantCount == 1 {
		return "1 client is missing PKCE enforcement"
	}
	return "Multiple clients are missing PKCE enforcement"
}
