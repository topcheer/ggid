package httpserver

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/ggid/ggid/services/audit/internal/repository"
)

// ComplianceControlMapping is the enriched mapping with framework metadata,
// GGID feature cross-reference, and CCM control linkage.
type ComplianceControlMapping struct {
	ControlID     string `json:"control_id"`
	ControlName   string `json:"control_name"`
	TrustCategory string `json:"trust_category"`
	GGIDFeature   string `json:"ggid_feature"`
	Status        string `json:"status"`
	EvidenceQuery string `json:"evidence_query"`
	CCMControlID  string `json:"ccm_control_id"`
	Description   string `json:"description"`
}

// frameworkTrustPrinciples maps framework to their trust principle categories.
var frameworkTrustPrinciples = map[string][]string{
	"soc2": {
		"Security", "Availability", "Processing Integrity", "Confidentiality", "Privacy",
	},
	"iso27001": {
		"A.5", "A.6", "A.8", "A.9", "A.12", "A.16",
	},
}

// enhancedFrameworkMappings provides full SOC2 Type II (5 trust principles),
// ISO 27001 Annex A (key 10), and CCM cross-references.
var enhancedFrameworkMappings = map[string][]ComplianceControlMapping{
	"soc2": {
		// Security (CC6.x)
		{"CC6.1", "Logical and Physical Access Controls", "Security", "Auth + MFA + Password Policy", "covered", "SELECT count(*) FROM users WHERE mfa_enabled = true", "password_policy_compliance", "GGID enforces MFA, password complexity, and RBAC"},
		{"CC6.2", "User Authentication Credentials", "Security", "Auth + Password History + Breach Check", "covered", "SELECT count(*) FROM password_history WHERE created_at > now() - interval '7 days'", "password_history_check", "Password rotation, breach detection, pepper rotation"},
		{"CC6.3", "Authorization Controls for Access", "Security", "RBAC + ABAC + Policy Engine", "covered", "SELECT count(*) FROM user_roles", "rbac_coverage", "Role-based + attribute-based access control via unified PDP"},
		{"CC6.6", "Logical Access Security Measures", "Security", "JWT + Session Timeout + Revocation", "covered", "SELECT count(*) FROM sessions WHERE expires_at > now()", "session_timeout_compliance", "JWT sessions with configurable timeout and instant revocation"},
		// Availability (A1.x)
		{"A1.1", "System Monitoring and Health", "Availability", "Health Check + K8s Probes + Backup", "covered", "SELECT count(*) FROM health_checks WHERE status = 'healthy' AND checked_at > now() - interval '1 hour'", "system_availability", "Kubernetes liveness/readiness probes, PostgreSQL backups, Redis HA"},
		{"A1.2", "Environmental Protections", "Availability", "Rate Limiting + DDoS Protection", "partial", "SELECT count(*) FROM rate_limit_events WHERE blocked = true", "rate_limit_active", "Per-tenant rate limiting, sliding window throttle, IP-based blocking"},
		// Processing Integrity (PI1.x)
		{"PI1.1", "Audit Chain Integrity", "Processing Integrity", "Hash Chain Audit Logging", "covered", "SELECT count(*) FROM audit_events WHERE hash_verified = true", "audit_integrity", "Tamper-evident SHA-256 hash chain for all audit events"},
		{"PI1.2", "Error Handling and Reporting", "Processing Integrity", "Error Tracking + SIEM Integration", "partial", "SELECT count(*) FROM error_logs WHERE resolved = true", "error_handling", "Structured error logging with SIEM forwarding and alerting"},
		// Confidentiality (C1.x)
		{"C1.1", "Data Confidentiality Measures", "Confidentiality", "RLS + Field-Level Encryption", "covered", "SELECT count(*) FROM rls_policies WHERE enabled = true", "encryption_at_rest", "Row-level security, PII vault encryption, TLS 1.3 in transit"},
		{"C1.2", "Data Transmission and Disposal", "Confidentiality", "TLS 1.3 + JWT Key Rotation", "covered", "SELECT count(*) FROM jwt_key_rotations WHERE rotated_at > now() - interval '90 days'", "key_rotation", "Automatic JWT key rotation every 90 days, Argon2id password hashing"},
		// Privacy (P1.x, P2.x)
		{"P1.1", "Privacy Notice and Consent", "Privacy", "GDPR Consent + Data Subject Rights", "partial", "SELECT count(*) FROM dsr_requests WHERE status = 'completed'", "gdpr_compliance", "GDPR data subject rights API, consent tracking, data minimization"},
		{"P2.1", "Data Retention and Disposal", "Privacy", "Retention Policies + Auto-Deletion", "partial", "SELECT count(*) FROM retention_policies WHERE active = true", "data_retention", "Configurable retention periods with automated data deletion"},
		// Monitoring (CC7.x, CC8.x)
		{"CC7.1", "System Operations Monitoring", "Security", "Prometheus + Grafana + 4 Dashboards", "covered", "SELECT count(*) FROM prometheus_metrics WHERE scrape_ok = true", "monitoring_coverage", "Full Prometheus metrics, Grafana dashboards (overview/auth/perf/security), alerting rules"},
		{"CC7.2", "Anomaly Detection", "Security", "ITDR + UEBA + SIEM Feed", "covered", "SELECT count(*) FROM itdr_detections WHERE severity IN ('high','critical')", "anomaly_detection", "AI-driven threat detection, behavioral analytics, brute-force detection"},
		{"CC8.1", "Change Management", "Security", "GitOps + Migration Tools + CI/CD", "partial", "SELECT count(*) FROM schema_migrations ORDER BY version DESC LIMIT 1", "change_management", "Versioned DB migrations, CI/CD pipeline, immutable audit trail for config changes"},
	},
	"iso27001": {
		// A.5 Organizational
		{"A.5.1", "Policies for Information Security", "A.5", "Security Policy + Hardening Docs", "covered", "SELECT count(*) FROM security_policies WHERE active = true", "policy_docs", "Documented information security policies, hardening guides, production deployment docs"},
		// A.6 People
		{"A.6.1.1", "Information Security Roles and Responsibilities", "A.6", "RBAC + Admin Roles + SoD", "covered", "SELECT count(*) FROM user_roles ur JOIN roles r ON r.id = ur.role_id WHERE r.key LIKE '%admin%'", "admin_access", "Role-based admin access with separation of duties checks"},
		// A.8 Asset Management
		{"A.8.1.1", "Inventory of Assets", "A.8", "NHI Registry + Device Mgmt", "partial", "SELECT count(*) FROM non_human_identities WHERE active = true", "asset_inventory", "Non-human identity registry with risk scoring, device registration and tracking"},
		{"A.8.2.1", "Classification of Information", "A.8", "Data Classification + PII Tags", "partial", "SELECT count(*) FROM pii_fields WHERE classified = true", "data_classification", "PII field tagging, data classification metadata, field-level encryption"},
		// A.9 Access Control
		{"A.9.1.1", "Access Control Policy", "A.9", "Policy Engine + ABAC + PDP", "covered", "SELECT count(*) FROM access_policies WHERE enabled = true", "access_policy", "Attribute-based access control policies with unified policy decision point"},
		{"A.9.2.3", "Management of Privileged Access Rights", "A.9", "PAM + Session Recording + Creep Detection", "partial", "SELECT count(*) FROM privileged_sessions WHERE active = true", "privileged_access", "Privileged access management with session recording and privilege creep detection"},
		{"A.9.4.1", "Information Access Restriction", "A.9", "RLS + Column-Level Security", "covered", "SELECT count(*) FROM rls_policies WHERE enabled = true", "row_level_security", "PostgreSQL row-level security with strict tenant isolation"},
		// A.12 Operations Security
		{"A.12.4.1", "Event Logging", "A.12", "Audit Chain + SIEM Integration", "covered", "SELECT count(*) FROM audit_events WHERE created_at > now() - interval '1 hour'", "audit_logging", "Tamper-evident audit logging with hash chain verification and SIEM forwarding"},
		{"A.12.6.1", "Management of Technical Vulnerabilities", "A.12", "Dependency Scanning + Secret Detection", "partial", "SELECT count(*) FROM dependency_scans WHERE critical = 0", "vulnerability_management", "Automated dependency scanning, ggshield secret detection, patch management"},
		// A.16 Incident Management
		{"A.16.1.1", "Incident Management", "A.16", "ITDR + SoAR + Webhook Alerting", "covered", "SELECT count(*) FROM itdr_detections WHERE status = 'handled'", "incident_response", "Automated incident detection, SoAR playbook execution, webhook alerting to Slack/PagerDuty"},
	},
}

// GET /api/v1/audit/compliance-mapping?framework=soc2
// Returns framework control mappings with GGID feature cross-references and
// CCM control IDs. Falls back to PG repo when available, otherwise uses
// static enhanced mappings.
func (s *HTTPServer) handleComplianceMapping(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	framework := r.URL.Query().Get("framework")

	// Try PG repo first
	if s.complianceMappingRepo != nil && framework != "" {
		mappings, err := s.complianceMappingRepo.ListByFramework(r.Context(), defaultTenantID(), framework)
		if err == nil && len(mappings) > 0 {
			writeComplianceMappingResponse(w, framework, mappings)
			return
		}
	}

	// Fall back to static enhanced mappings
	if framework != "" {
		mappings, ok := enhancedFrameworkMappings[framework]
		if !ok {
			writeJSONError(w, http.StatusBadRequest, "unsupported framework: "+framework)
			return
		}
		writeEnhancedComplianceResponse(w, framework, mappings)
		return
	}

	// No framework filter — return all frameworks
	allMappings := make(map[string][]ComplianceControlMapping)
	for fw, ms := range enhancedFrameworkMappings {
		allMappings[fw] = ms
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"frameworks":     allMappings,
		"framework_list": frameworkList(),
	})
}

func writeEnhancedComplianceResponse(w http.ResponseWriter, framework string, mappings []ComplianceControlMapping) {
	covered, partial, gaps := 0, 0, 0
	trustCategories := make(map[string]int)
	for _, m := range mappings {
		switch m.Status {
		case "covered":
			covered++
		case "partial":
			partial++
		case "gap":
			gaps++
		}
		trustCategories[m.TrustCategory]++
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"framework":       framework,
		"trust_principles": frameworkTrustPrinciples[framework],
		"controls":        mappings,
		"summary": map[string]int{
			"total":   len(mappings),
			"covered": covered,
			"partial": partial,
			"gap":     gaps,
		},
		"trust_category_counts": trustCategories,
	})
}

func writeComplianceMappingResponse(w http.ResponseWriter, framework string, pgMappings []repository.ComplianceMapping) {
	// Convert PG records to the response format
	mappings := make([]ComplianceControlMapping, len(pgMappings))
	covered, partial, gaps := 0, 0, 0
	trustCategories := make(map[string]int)
	for i, m := range pgMappings {
		mappings[i] = ComplianceControlMapping{
			ControlID:     m.ControlID,
			ControlName:   m.ControlName,
			TrustCategory: m.TrustCategory,
			GGIDFeature:   m.GGIDFeature,
			Status:        m.Status,
			EvidenceQuery: m.EvidenceQuery,
			CCMControlID:  m.CCMControlID,
			Description:   m.Description,
		}
		switch m.Status {
		case "covered":
			covered++
		case "partial":
			partial++
		case "gap":
			gaps++
		}
		trustCategories[m.TrustCategory]++
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"framework":        framework,
		"trust_principles": frameworkTrustPrinciples[framework],
		"controls":         mappings,
		"summary": map[string]int{
			"total":   len(mappings),
			"covered": covered,
			"partial": partial,
			"gap":     gaps,
		},
		"trust_category_counts": trustCategories,
		"source":                "database",
	})
}

func frameworkList() []string {
	return []string{"soc2", "iso27001", "gdpr", "hipaa", "ccm"}
}

func defaultTenantID() uuid.UUID {
	return uuid.Nil
}
