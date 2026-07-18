package middleware

import (
	"encoding/json"
	"net/http"
)

// GGIDOpenAPISpec is the unified OpenAPI 3.1 spec for all services.
// Generated from swag annotations + manual curation.
type GGIDOpenAPISpec struct {
	OpenAPI    string                    `json:"openapi"`
	Info       OpenAPIInfo               `json:"info"`
	Servers    []OpenAPIServer           `json:"servers"`
	Paths      map[string]OpenAPIPath    `json:"paths"`
	Components OpenAPIComponents         `json:"components"`
}


type OpenAPIServer struct {
	URL         string `json:"url"`
	Description string `json:"description"`
}

type OpenAPIPath struct {
	Get    *OpenAPIOperation `json:"get,omitempty"`
	Post   *OpenAPIOperation `json:"post,omitempty"`
	Put    *OpenAPIOperation `json:"put,omitempty"`
	Delete *OpenAPIOperation `json:"delete,omitempty"`
}

type OpenAPIOperation struct {
	Tags       []string            `json:"tags"`
	Summary    string              `json:"summary"`
	Security   []map[string][]string `json:"security,omitempty"`
	Responses  map[string]OpenAPIResponse `json:"responses"`
}

type OpenAPIResponse struct {
	Description string `json:"description"`
}

type OpenAPIComponents struct {
	SecuritySchemes map[string]OpenAPISecurityScheme `json:"securitySchemes"`
}

type OpenAPISecurityScheme struct {
	Type         string `json:"type"`
	Scheme       string `json:"scheme,omitempty"`
	BearerFormat string `json:"bearerFormat,omitempty"`
	In           string `json:"in,omitempty"`
	Name         string `json:"name,omitempty"`
	Description  string `json:"description,omitempty"`
}

// GenerateOpenAPISpec returns the unified GGID OpenAPI spec.
func GenerateOpenAPISpec() *GGIDOpenAPISpec {
	bearer := []map[string][]string{{"bearerAuth": {}}}
	return &GGIDOpenAPISpec{
		OpenAPI: "3.1.0",
		Info: OpenAPIInfo{
			Title:       "GGID Platform API",
			Description: "Unified Identity, Access Management, and Security platform",
			Version:     "1.0.0",
		},
		Servers: []OpenAPIServer{
			{URL: "https://api.ggid.io", Description: "Production"},
			{URL: "http://localhost:8080", Description: "Local dev"},
		},
		Components: OpenAPIComponents{
			SecuritySchemes: map[string]OpenAPISecurityScheme{
				"bearerAuth": {Type: "http", Scheme: "bearer", BearerFormat: "JWT", Description: "JWT Bearer token"},
				"dpop":      {Type: "http", Scheme: "DPoP", Description: "DPoP proof"},
				"apiKey":    {Type: "apiKey", In: "header", Name: "X-API-Key", Description: "API key auth"},
				"mtls":      {Type: "mutualTLS", Description: "mTLS client cert"},
			},
		},
		Paths: generatePaths(bearer),
	}
}

func generatePaths(sec []map[string][]string) map[string]OpenAPIPath {
	op := func(tags []string, summary string) *OpenAPIOperation {
		return &OpenAPIOperation{
			Tags: tags, Summary: summary, Security: sec,
			Responses: map[string]OpenAPIResponse{
				"200": {Description: "Success"},
				"400": {Description: "Bad request"},
				"401": {Description: "Unauthorized"},
				"403": {Description: "Forbidden"},
			},
		}
	}
	return map[string]OpenAPIPath{
		// Auth — Authentication
		"/api/v1/auth/login":               {Post: op([]string{"Auth"}, "User login — authenticate with username/password, returns JWT tokens")},
		"/api/v1/auth/register":            {Post: op([]string{"Auth"}, "Self-service user registration")},
		"/api/v1/auth/logout":              {Post: op([]string{"Auth"}, "Logout and invalidate session")},
		"/api/v1/auth/refresh":             {Post: op([]string{"Auth"}, "Refresh access token using refresh token")},
		"/api/v1/auth/profile":             {Get: op([]string{"Auth"}, "Get current user profile"), Put: op([]string{"Auth"}, "Update own profile")},
		"/api/v1/auth/verify-email":        {Get: op([]string{"Auth"}, "Verify email address with token")},
		"/api/v1/auth/forgot-password":     {Post: op([]string{"Auth"}, "Request password reset email")},
		"/api/v1/auth/password/forgot":     {Post: op([]string{"Auth"}, "Request password reset (v2)")},
		"/api/v1/auth/password/reset":      {Post: op([]string{"Auth"}, "Reset password with token")},
		"/api/v1/auth/password/change":     {Post: op([]string{"Auth"}, "Change password (requires current password)")},
		"/api/v1/auth/password/strength":   {Post: op([]string{"Auth"}, "Evaluate password strength (zxcvbn score 0-4)")},
		"/api/v1/auth/password/policy":     {Get: op([]string{"Auth"}, "Get password policy requirements")},
		// Auth — Sessions
		"/api/v1/auth/sessions":            {Get: op([]string{"Sessions"}, "List active sessions for current user")},
		"/api/v1/auth/sessions/{id}":       {Delete: op([]string{"Sessions"}, "Revoke a session by ID")},
		// Auth — MFA
		"/api/v1/auth/mfa/enroll":          {Post: op([]string{"MFA"}, "Enroll MFA (TOTP)")},
		"/api/v1/auth/mfa/verify":          {Post: op([]string{"MFA"}, "Verify MFA code")},
		"/api/v1/auth/mfa/disable":         {Post: op([]string{"MFA"}, "Disable MFA")},
		"/api/v1/auth/mfa/backup-codes":    {Get: op([]string{"MFA"}, "List backup codes"), Post: op([]string{"MFA"}, "Generate new backup codes")},
		// Auth — WebAuthn
		"/api/v1/auth/webauthn/begin":      {Post: op([]string{"WebAuthn"}, "Begin WebAuthn registration")},
		"/api/v1/auth/webauthn/finish":     {Post: op([]string{"WebAuthn"}, "Finish WebAuthn registration")},
		"/api/v1/auth/webauthn/login/begin": {Post: op([]string{"WebAuthn"}, "Begin WebAuthn login")},
		"/api/v1/auth/webauthn/login/finish": {Post: op([]string{"WebAuthn"}, "Finish WebAuthn login")},
		"/api/v1/auth/webauthn/aaguid":     {Get: op([]string{"WebAuthn"}, "List AAGUID allowlist"), Post: op([]string{"WebAuthn"}, "Add AAGUID to allowlist")},
		// Auth — Conditional Access
		"/api/v1/auth/conditional-access/policies": {Get: op([]string{"Conditional Access"}, "List CAP policies"), Post: op([]string{"Conditional Access"}, "Create CAP policy")},
		"/api/v1/auth/conditional-access/evaluate": {Post: op([]string{"Conditional Access"}, "Evaluate conditions against context")},
		// Auth: Simplified authz
		"/api/v1/authz/check":              {Post: op([]string{"Authz"}, "Simplified permission check: {user_id, resource, action} -> {allowed}")},
		// OAuth: UserInfo
		"/oauth/userinfo":                  {Get: op([]string{"OAuth"}, "Enhanced UserInfo: profile + roles + groups + permissions + risk_level")},
		// Auth — TAP
		"/api/v1/auth/tap":                 {Post: op([]string{"TAP"}, "Issue Temporary Access Pass")},
		"/api/v1/auth/tap/batch":           {Post: op([]string{"TAP"}, "Batch issue TAPs")},
		"/api/v1/auth/tap/policy":          {Get: op([]string{"TAP"}, "Get TAP policy"), Put: op([]string{"TAP"}, "Update TAP policy")},
		// Auth — Break Glass
		"/api/v1/auth/break-glass/activate": {Post: op([]string{"Break Glass"}, "Activate break-glass access")},
		"/api/v1/auth/break-glass/history":  {Get: op([]string{"Break Glass"}, "Break-glass activation history")},
		// Identity — Users
		"/api/v1/users":                    {Get: op([]string{"Identity"}, "List users"), Post: op([]string{"Identity"}, "Create user")},
		"/api/v1/users/{id}":               {Get: op([]string{"Identity"}, "Get user by ID"), Put: op([]string{"Identity"}, "Update user"), Delete: op([]string{"Identity"}, "Delete user")},
		"/api/v1/users/import":             {Post: op([]string{"Identity"}, "Import users from CSV")},
		"/api/v1/users/export":             {Get: op([]string{"Identity"}, "Export users to CSV")},
		// Identity — Groups
		"/api/v1/groups":                   {Get: op([]string{"Identity"}, "List groups"), Post: op([]string{"Identity"}, "Create group")},
		"/api/v1/groups/{id}":              {Get: op([]string{"Identity"}, "Get group by ID"), Put: op([]string{"Identity"}, "Update group"), Delete: op([]string{"Identity"}, "Delete group")},
		"/api/v1/groups/{id}/members":      {Get: op([]string{"Identity"}, "List group members"), Post: op([]string{"Identity"}, "Add member to group")},
		// Identity — Organization
		"/api/v1/orgs":                     {Get: op([]string{"Org"}, "List organizations"), Post: op([]string{"Org"}, "Create organization")},
		"/api/v1/orgs/{id}":                {Get: op([]string{"Org"}, "Get organization by ID"), Put: op([]string{"Org"}, "Update organization"), Delete: op([]string{"Org"}, "Delete organization")},
		"/api/v1/departments":              {Get: op([]string{"Org"}, "List departments"), Post: op([]string{"Org"}, "Create department")},
		"/api/v1/teams":                    {Get: op([]string{"Org"}, "List teams"), Post: op([]string{"Org"}, "Create team")},
		// OAuth
		"/api/v1/oauth/token":              {Post: op([]string{"OAuth"}, "Issue token (client_credentials, authorization_code, refresh_token)")},
		"/api/v1/oauth/authorize":          {Get: op([]string{"OAuth"}, "OAuth 2.0 authorize endpoint"), Post: op([]string{"OAuth"}, "OAuth 2.0 authorize (POST)")},
		"/api/v1/oauth/clients":            {Get: op([]string{"OAuth"}, "List OAuth clients"), Post: op([]string{"OAuth"}, "Create OAuth client")},
		"/api/v1/oauth/clients/{id}":       {Get: op([]string{"OAuth"}, "Get OAuth client"), Put: op([]string{"OAuth"}, "Update OAuth client"), Delete: op([]string{"OAuth"}, "Delete OAuth client")},
		"/api/v1/oauth/introspect":         {Post: op([]string{"OAuth"}, "Token introspection (RFC 7662)")},
		"/api/v1/oauth/revoke":             {Post: op([]string{"OAuth"}, "Revoke token (RFC 7009)")},
		"/.well-known/openid-configuration": {Get: op([]string{"OAuth"}, "OIDC discovery document")},
		"/.well-known/jwks.json":           {Get: op([]string{"OAuth"}, "JWKS — public keys for JWT verification")},
		// Policy — Roles & Permissions
		"/api/v1/roles":                    {Get: op([]string{"Policy"}, "List roles"), Post: op([]string{"Policy"}, "Create role")},
		"/api/v1/roles/{id}":               {Get: op([]string{"Policy"}, "Get role by ID"), Put: op([]string{"Policy"}, "Update role"), Delete: op([]string{"Policy"}, "Delete role")},
		"/api/v1/permissions":              {Get: op([]string{"Policy"}, "List permissions"), Post: op([]string{"Policy"}, "Create permission")},
		// Policy — ABAC/RBAC
		"/api/v1/policies":                 {Get: op([]string{"Policy"}, "List policies"), Post: op([]string{"Policy"}, "Create policy")},
		"/api/v1/policies/{id}":            {Get: op([]string{"Policy"}, "Get policy by ID"), Put: op([]string{"Policy"}, "Update policy"), Delete: op([]string{"Policy"}, "Delete policy")},
		"/api/v1/policies/check":           {Post: op([]string{"Policy"}, "Check access (principal, resource, action) → allow/deny")},
		"/api/v1/policies/evaluate":        {Post: op([]string{"Policy"}, "Evaluate all matching policies with decision trail")},
		"/api/v1/policies/sod/check":       {Post: op([]string{"Policy"}, "Check Separation of Duties violations")},
		"/api/v1/policies/sod/violations":  {Get: op([]string{"Policy"}, "List SoD violations")},
		// Audit
		"/api/v1/audit/events":             {Get: op([]string{"Audit"}, "List audit events with filtering")},
		"/api/v1/audit/events/{id}":        {Get: op([]string{"Audit"}, "Get audit event by ID")},
		"/api/v1/audit/stats":              {Get: op([]string{"Audit"}, "Aggregate audit statistics")},
		"/api/v1/audit/export":             {Get: op([]string{"Audit"}, "Export audit events (JSON/CSV)")},
		"/api/v1/audit/ccm/results":        {Get: op([]string{"CCM"}, "Get compliance monitoring results")},
		"/api/v1/audit/ccm/run":            {Post: op([]string{"CCM"}, "Trigger compliance scan")},
		"/api/v1/audit/ccm/history":        {Get: op([]string{"CCM"}, "Get compliance history")},
		// Admin
		"/api/v1/admin/backups":            {Get: op([]string{"Admin"}, "List backups")},
		"/api/v1/admin/backups/trigger":    {Post: op([]string{"Admin"}, "Trigger backup")},
		"/api/v1/admin/secrets":            {Get: op([]string{"Admin"}, "List secret references")},
		"/api/v1/admin/keys":               {Get: op([]string{"Admin"}, "List active signing keys")},
		"/api/v1/quotas/{tenant_id}":       {Get: op([]string{"Admin"}, "Get tenant quota"), Put: op([]string{"Admin"}, "Update quota")},
		// GraphQL + Observability
		"/graphql":                         {Post: op([]string{"GraphQL"}, "GraphQL endpoint")},
		"/api/v1/observability/health":     {Get: op([]string{"Observability"}, "Exporter health")},
		"/healthz":                         {Get: op([]string{"System"}, "Health check")},
		"/readyz":                          {Get: op([]string{"System"}, "Readiness check")},
		// Legacy aliases for backward compatibility
		"/api/v1/identity/users":           {Get: op([]string{"Identity"}, "List users (legacy alias)"), Post: op([]string{"Identity"}, "Create user (legacy alias)")},		"/api/v1/policy/authorize":         {Post: op([]string{"Policy"}, "Unified PDP authorize")},
		"/api/v1/risk/evaluate":            {Post: op([]string{"Risk"}, "Evaluate risk score")},
		"/api/v1/mdm/connectors":           {Get: op([]string{"MDM"}, "List MDM connectors"), Post: op([]string{"MDM"}, "Add MDM connector")},
		"/api/v1/mdm/devices":              {Get: op([]string{"MDM"}, "List MDM devices")},
		"/api/v1/risk/scores/{user_id}":    {Get: op([]string{"Risk"}, "Get risk score for user")},
		"/api/v1/risk/signals":             {Get: op([]string{"Risk"}, "List risk signals")},
		"/api/v1/hr/connectors":            {Get: op([]string{"HR"}, "List HR connectors"), Post: op([]string{"HR"}, "Add HR connector")},
		"/api/v1/hr/sync":                  {Post: op([]string{"HR"}, "Trigger HR sync")},
		"/api/v1/hr/dormant":               {Get: op([]string{"HR"}, "Dormant accounts")},
		"/api/v1/notifications/rules":      {Get: op([]string{"Notifications"}, "List notification rules"), Post: op([]string{"Notifications"}, "Create notification rule")},
		"/api/v1/soar/playbooks":           {Get: op([]string{"SOAR"}, "List SOAR playbooks"), Post: op([]string{"SOAR"}, "Create SOAR playbook")},
		// KB-299: Bulk endpoint coverage expansion.
		"/api/v1/.well-known/federation-configuration": {Get: op([]string{"Other"}, "Federation Configuration"), Post: op([]string{"Other"}, "Federation Configuration")},
		"/api/v1/access-requests": {Get: op([]string{"Access"}, "Access Requests"), Post: op([]string{"Access"}, "Access Requests")},
		"/api/v1/access-requests/": {Get: op([]string{"Access"}, "Access Requests"), Post: op([]string{"Access"}, "Access Requests")},
		"/api/v1/admin/config": {Get: op([]string{"Admin"}, "Config"), Post: op([]string{"Admin"}, "Config")},
		"/api/v1/admin/config/": {Get: op([]string{"Admin"}, "Config"), Post: op([]string{"Admin"}, "Config")},
		"/api/v1/admin/email/config": {Get: op([]string{"Admin"}, "Config"), Post: op([]string{"Admin"}, "Config")},
		"/api/v1/admin/email/test": {Get: op([]string{"Admin"}, "Test"), Post: op([]string{"Admin"}, "Test")},
		"/api/v1/admin/feature-flags": {Get: op([]string{"Admin"}, "Feature Flags"), Post: op([]string{"Admin"}, "Feature Flags")},
		"/api/v1/admin/feature-flags/": {Get: op([]string{"Admin"}, "Feature Flags"), Post: op([]string{"Admin"}, "Feature Flags")},
		"/api/v1/admin/keys/history": {Get: op([]string{"Admin"}, "History"), Post: op([]string{"Admin"}, "History")},
		"/api/v1/admin/migration/config": {Get: op([]string{"Admin"}, "Config"), Post: op([]string{"Admin"}, "Config")},
		"/api/v1/admin/migration/mappings": {Get: op([]string{"Admin"}, "Mappings"), Post: op([]string{"Admin"}, "Mappings")},
		"/api/v1/admin/migration/mappings/": {Get: op([]string{"Admin"}, "Mappings"), Post: op([]string{"Admin"}, "Mappings")},
		"/api/v1/admin/migration/stats": {Get: op([]string{"Admin"}, "Stats"), Post: op([]string{"Admin"}, "Stats")},
		"/api/v1/admin/migration/test": {Get: op([]string{"Admin"}, "Test"), Post: op([]string{"Admin"}, "Test")},
		"/api/v1/admin/rls/status": {Get: op([]string{"Admin"}, "Status"), Post: op([]string{"Admin"}, "Status")},
		"/api/v1/admin/rls/test": {Get: op([]string{"Admin"}, "Test"), Post: op([]string{"Admin"}, "Test")},
		"/api/v1/admin/secrets/health": {Get: op([]string{"Admin"}, "Health"), Post: op([]string{"Admin"}, "Health")},
		"/api/v1/agents": {Get: op([]string{"Other"}, "Agents"), Post: op([]string{"Other"}, "Agents")},
		"/api/v1/agents/": {Get: op([]string{"Other"}, "Agents"), Post: op([]string{"Other"}, "Agents")},
		"/api/v1/agents/drift/report": {Get: op([]string{"Other"}, "Report"), Post: op([]string{"Other"}, "Report")},
		"/api/v1/agents/register": {Get: op([]string{"Other"}, "Register"), Post: op([]string{"Other"}, "Register")},
		"/api/v1/agents/reviews": {Get: op([]string{"Other"}, "Reviews"), Post: op([]string{"Other"}, "Reviews")},
		"/api/v1/agents/reviews/": {Get: op([]string{"Other"}, "Reviews"), Post: op([]string{"Other"}, "Reviews")},
		"/api/v1/agents/shadows": {Get: op([]string{"Other"}, "Shadows"), Post: op([]string{"Other"}, "Shadows")},
		"/api/v1/agents/token": {Get: op([]string{"Other"}, "Token"), Post: op([]string{"Other"}, "Token")},
		"/api/v1/agents/verify": {Get: op([]string{"Other"}, "Verify"), Post: op([]string{"Other"}, "Verify")},
		"/api/v1/alerts": {Get: op([]string{"Other"}, "Alerts"), Post: op([]string{"Other"}, "Alerts")},
		"/api/v1/audit": {Get: op([]string{"Audit"}, "Audit"), Post: op([]string{"Audit"}, "Audit")},
		"/api/v1/audit/access-reviews": {Get: op([]string{"Audit"}, "Access Reviews"), Post: op([]string{"Audit"}, "Access Reviews")},
		"/api/v1/audit/access-reviews/pending": {Get: op([]string{"Audit"}, "Pending"), Post: op([]string{"Audit"}, "Pending")},
		"/api/v1/audit/activity": {Get: op([]string{"Audit"}, "Activity"), Post: op([]string{"Audit"}, "Activity")},
		"/api/v1/audit/aggregations": {Get: op([]string{"Audit"}, "Aggregations"), Post: op([]string{"Audit"}, "Aggregations")},
		"/api/v1/audit/aggregations/daily": {Get: op([]string{"Audit"}, "Daily"), Post: op([]string{"Audit"}, "Daily")},
		"/api/v1/audit/alert-evaluation/config": {Get: op([]string{"Audit"}, "Config"), Post: op([]string{"Audit"}, "Config")},
		"/api/v1/audit/alert-webhooks": {Get: op([]string{"Audit"}, "Alert Webhooks"), Post: op([]string{"Audit"}, "Alert Webhooks")},
		"/api/v1/audit/alerts/config": {Get: op([]string{"Audit"}, "Config"), Post: op([]string{"Audit"}, "Config")},
		"/api/v1/audit/alerts/evaluate": {Get: op([]string{"Audit"}, "Evaluate"), Post: op([]string{"Audit"}, "Evaluate")},
		"/api/v1/audit/alerts/test": {Get: op([]string{"Audit"}, "Test"), Post: op([]string{"Audit"}, "Test")},
		"/api/v1/audit/anomalies/detect": {Get: op([]string{"Audit"}, "Detect"), Post: op([]string{"Audit"}, "Detect")},
		"/api/v1/audit/anomaly-detection": {Get: op([]string{"Audit"}, "Anomaly Detection"), Post: op([]string{"Audit"}, "Anomaly Detection")},
		"/api/v1/audit/anomaly-detection/": {Get: op([]string{"Audit"}, "Anomaly Detection"), Post: op([]string{"Audit"}, "Anomaly Detection")},
		"/api/v1/audit/ccm/summary": {Get: op([]string{"Audit"}, "Summary"), Post: op([]string{"Audit"}, "Summary")},
		"/api/v1/audit/compliance-report": {Get: op([]string{"Audit"}, "Compliance Report"), Post: op([]string{"Audit"}, "Compliance Report")},
		"/api/v1/audit/compliance-schedules": {Get: op([]string{"Audit"}, "Compliance Schedules"), Post: op([]string{"Audit"}, "Compliance Schedules")},
		"/api/v1/audit/compliance/auto-collect": {Get: op([]string{"Audit"}, "Auto Collect"), Post: op([]string{"Audit"}, "Auto Collect")},
		"/api/v1/audit/compliance/auto-score": {Get: op([]string{"Audit"}, "Auto Score"), Post: op([]string{"Audit"}, "Auto Score")},
		"/api/v1/audit/compliance/cert-export": {Get: op([]string{"Audit"}, "Cert Export"), Post: op([]string{"Audit"}, "Cert Export")},
		"/api/v1/audit/compliance/config": {Get: op([]string{"Audit"}, "Config"), Post: op([]string{"Audit"}, "Config")},
		"/api/v1/audit/compliance/dashboard": {Get: op([]string{"Audit"}, "Dashboard"), Post: op([]string{"Audit"}, "Dashboard")},
		"/api/v1/audit/compliance/drift": {Get: op([]string{"Audit"}, "Drift"), Post: op([]string{"Audit"}, "Drift")},
		"/api/v1/audit/compliance/evidence": {Get: op([]string{"Audit"}, "Evidence"), Post: op([]string{"Audit"}, "Evidence")},
		"/api/v1/audit/compliance/evidence-attachments": {Get: op([]string{"Audit"}, "Evidence Attachments"), Post: op([]string{"Audit"}, "Evidence Attachments")},
		"/api/v1/audit/compliance/evidence-expiry": {Get: op([]string{"Audit"}, "Evidence Expiry"), Post: op([]string{"Audit"}, "Evidence Expiry")},
		"/api/v1/audit/compliance/evidence-refresh": {Get: op([]string{"Audit"}, "Evidence Refresh"), Post: op([]string{"Audit"}, "Evidence Refresh")},
		"/api/v1/audit/compliance/evidence/": {Get: op([]string{"Audit"}, "Evidence"), Post: op([]string{"Audit"}, "Evidence")},
		"/api/v1/audit/compliance/evidence/verify-integrity": {Get: op([]string{"Audit"}, "Verify Integrity"), Post: op([]string{"Audit"}, "Verify Integrity")},
		"/api/v1/audit/compliance/gaps": {Get: op([]string{"Audit"}, "Gaps"), Post: op([]string{"Audit"}, "Gaps")},
		"/api/v1/audit/compliance/gaps/": {Get: op([]string{"Audit"}, "Gaps"), Post: op([]string{"Audit"}, "Gaps")},
		"/api/v1/audit/compliance/heatmap": {Get: op([]string{"Audit"}, "Heatmap"), Post: op([]string{"Audit"}, "Heatmap")},
		"/api/v1/audit/compliance/mapping": {Get: op([]string{"Audit"}, "Mapping"), Post: op([]string{"Audit"}, "Mapping")},
		"/api/v1/audit/compliance/remediation-progress": {Get: op([]string{"Audit"}, "Remediation Progress"), Post: op([]string{"Audit"}, "Remediation Progress")},
		"/api/v1/audit/compliance/schedule-collect": {Get: op([]string{"Audit"}, "Schedule Collect"), Post: op([]string{"Audit"}, "Schedule Collect")},
		"/api/v1/audit/compliance/schedules": {Get: op([]string{"Audit"}, "Schedules"), Post: op([]string{"Audit"}, "Schedules")},
		"/api/v1/audit/compliance/score-history": {Get: op([]string{"Audit"}, "Score History"), Post: op([]string{"Audit"}, "Score History")},
		"/api/v1/audit/compliance/widget-data": {Get: op([]string{"Audit"}, "Widget Data"), Post: op([]string{"Audit"}, "Widget Data")},
		"/api/v1/audit/correlate": {Get: op([]string{"Audit"}, "Correlate"), Post: op([]string{"Audit"}, "Correlate")},
		"/api/v1/audit/correlation/rules": {Get: op([]string{"Audit"}, "Rules"), Post: op([]string{"Audit"}, "Rules")},
		"/api/v1/audit/cross-system-correlate": {Get: op([]string{"Audit"}, "Cross System Correlate"), Post: op([]string{"Audit"}, "Cross System Correlate")},
		"/api/v1/audit/dsr": {Get: op([]string{"Audit"}, "Dsr"), Post: op([]string{"Audit"}, "Dsr")},
		"/api/v1/audit/events/deduplicate": {Get: op([]string{"Audit"}, "Deduplicate"), Post: op([]string{"Audit"}, "Deduplicate")},
		"/api/v1/audit/events/subscribe": {Get: op([]string{"Audit"}, "Subscribe"), Post: op([]string{"Audit"}, "Subscribe")},
		"/api/v1/audit/events/subscribe/": {Get: op([]string{"Audit"}, "Subscribe"), Post: op([]string{"Audit"}, "Subscribe")},
		"/api/v1/audit/evidence-collection": {Get: op([]string{"Audit"}, "Evidence Collection"), Post: op([]string{"Audit"}, "Evidence Collection")},
		"/api/v1/audit/evidence-collection/": {Get: op([]string{"Audit"}, "Evidence Collection"), Post: op([]string{"Audit"}, "Evidence Collection")},
		"/api/v1/audit/evidence/chain": {Get: op([]string{"Audit"}, "Chain"), Post: op([]string{"Audit"}, "Chain")},
		"/api/v1/audit/export/schedule-config": {Get: op([]string{"Audit"}, "Schedule Config"), Post: op([]string{"Audit"}, "Schedule Config")},
		"/api/v1/audit/exports": {Get: op([]string{"Audit"}, "Exports"), Post: op([]string{"Audit"}, "Exports")},
		"/api/v1/audit/exports/": {Get: op([]string{"Audit"}, "Exports"), Post: op([]string{"Audit"}, "Exports")},
		"/api/v1/audit/exports/schedule": {Get: op([]string{"Audit"}, "Schedule"), Post: op([]string{"Audit"}, "Schedule")},
		"/api/v1/audit/forensics/timeline": {Get: op([]string{"Audit"}, "Timeline"), Post: op([]string{"Audit"}, "Timeline")},
		"/api/v1/audit/framework-coverage": {Get: op([]string{"Audit"}, "Framework Coverage"), Post: op([]string{"Audit"}, "Framework Coverage")},
		"/api/v1/audit/gdpr-forget": {Get: op([]string{"Audit"}, "Gdpr Forget"), Post: op([]string{"Audit"}, "Gdpr Forget")},
		"/api/v1/audit/gdpr-forget/": {Get: op([]string{"Audit"}, "Gdpr Forget"), Post: op([]string{"Audit"}, "Gdpr Forget")},
		"/api/v1/audit/gdpr/forget": {Get: op([]string{"Audit"}, "Forget"), Post: op([]string{"Audit"}, "Forget")},
		"/api/v1/audit/hash-chain": {Get: op([]string{"Audit"}, "Hash Chain"), Post: op([]string{"Audit"}, "Hash Chain")},
		"/api/v1/audit/hash-chain/config": {Get: op([]string{"Audit"}, "Config"), Post: op([]string{"Audit"}, "Config")},
		"/api/v1/audit/impersonation": {Get: op([]string{"Audit"}, "Impersonation"), Post: op([]string{"Audit"}, "Impersonation")},
		"/api/v1/audit/incidents": {Get: op([]string{"Audit"}, "Incidents"), Post: op([]string{"Audit"}, "Incidents")},
		"/api/v1/audit/incidents/active": {Get: op([]string{"Audit"}, "Active"), Post: op([]string{"Audit"}, "Active")},
		"/api/v1/audit/integrity/sign-pqc": {Get: op([]string{"Audit"}, "Sign Pqc"), Post: op([]string{"Audit"}, "Sign Pqc")},
		"/api/v1/audit/integrity/verify": {Get: op([]string{"Audit"}, "Verify"), Post: op([]string{"Audit"}, "Verify")},
		"/api/v1/audit/integrity/verify-pqc": {Get: op([]string{"Audit"}, "Verify Pqc"), Post: op([]string{"Audit"}, "Verify Pqc")},
		"/api/v1/audit/isolation-check": {Get: op([]string{"Audit"}, "Isolation Check"), Post: op([]string{"Audit"}, "Isolation Check")},
		"/api/v1/audit/itdr/composite-rules": {Get: op([]string{"Audit"}, "Composite Rules"), Post: op([]string{"Audit"}, "Composite Rules")},
		"/api/v1/audit/itdr/composite-rules/": {Get: op([]string{"Audit"}, "Composite Rules"), Post: op([]string{"Audit"}, "Composite Rules")},
		"/api/v1/audit/itdr/detections": {Get: op([]string{"Audit"}, "Detections"), Post: op([]string{"Audit"}, "Detections")},
		"/api/v1/audit/itdr/detections/": {Get: op([]string{"Audit"}, "Detections"), Post: op([]string{"Audit"}, "Detections")},
		"/api/v1/audit/itdr/incidents": {Get: op([]string{"Audit"}, "Incidents"), Post: op([]string{"Audit"}, "Incidents")},
		"/api/v1/audit/itdr/playbooks": {Get: op([]string{"Audit"}, "Playbooks"), Post: op([]string{"Audit"}, "Playbooks")},
		"/api/v1/audit/itdr/rules": {Get: op([]string{"Audit"}, "Rules"), Post: op([]string{"Audit"}, "Rules")},
		"/api/v1/audit/itdr/rules/": {Get: op([]string{"Audit"}, "Rules"), Post: op([]string{"Audit"}, "Rules")},
		"/api/v1/audit/itdr/stats": {Get: op([]string{"Audit"}, "Stats"), Post: op([]string{"Audit"}, "Stats")},
		"/api/v1/audit/itdr/threat-heatmap": {Get: op([]string{"Audit"}, "Threat Heatmap"), Post: op([]string{"Audit"}, "Threat Heatmap")},
		"/api/v1/audit/lineage": {Get: op([]string{"Audit"}, "Lineage"), Post: op([]string{"Audit"}, "Lineage")},
		"/api/v1/audit/metrics": {Get: op([]string{"Audit"}, "Metrics"), Post: op([]string{"Audit"}, "Metrics")},
		"/api/v1/audit/pii-scan": {Get: op([]string{"Audit"}, "Pii Scan"), Post: op([]string{"Audit"}, "Pii Scan")},
		"/api/v1/audit/query-metrics": {Get: op([]string{"Audit"}, "Query Metrics"), Post: op([]string{"Audit"}, "Query Metrics")},
		"/api/v1/audit/regulatory/report": {Get: op([]string{"Audit"}, "Report"), Post: op([]string{"Audit"}, "Report")},
		"/api/v1/audit/reports": {Get: op([]string{"Audit"}, "Reports"), Post: op([]string{"Audit"}, "Reports")},
		"/api/v1/audit/reports/": {Get: op([]string{"Audit"}, "Reports"), Post: op([]string{"Audit"}, "Reports")},
		"/api/v1/audit/reports/custom": {Get: op([]string{"Audit"}, "Custom"), Post: op([]string{"Audit"}, "Custom")},
		"/api/v1/audit/reports/generate": {Get: op([]string{"Audit"}, "Generate"), Post: op([]string{"Audit"}, "Generate")},
		"/api/v1/audit/retention": {Get: op([]string{"Audit"}, "Retention"), Post: op([]string{"Audit"}, "Retention")},
		"/api/v1/audit/retention-policies": {Get: op([]string{"Audit"}, "Retention Policies"), Post: op([]string{"Audit"}, "Retention Policies")},
		"/api/v1/audit/retention/execute": {Get: op([]string{"Audit"}, "Execute"), Post: op([]string{"Audit"}, "Execute")},
		"/api/v1/audit/retention/simulate": {Get: op([]string{"Audit"}, "Simulate"), Post: op([]string{"Audit"}, "Simulate")},
		"/api/v1/audit/risk-score": {Get: op([]string{"Audit"}, "Risk Score"), Post: op([]string{"Audit"}, "Risk Score")},
		"/api/v1/audit/rules": {Get: op([]string{"Audit"}, "Rules"), Post: op([]string{"Audit"}, "Rules")},
		"/api/v1/audit/sbom": {Get: op([]string{"Audit"}, "Sbom"), Post: op([]string{"Audit"}, "Sbom")},
		"/api/v1/audit/sbom/": {Get: op([]string{"Audit"}, "Sbom"), Post: op([]string{"Audit"}, "Sbom")},
		"/api/v1/audit/search": {Get: op([]string{"Audit"}, "Search"), Post: op([]string{"Audit"}, "Search")},
		"/api/v1/audit/security-posture": {Get: op([]string{"Audit"}, "Security Posture"), Post: op([]string{"Audit"}, "Security Posture")},
		"/api/v1/audit/siem/forwarder-config": {Get: op([]string{"Audit"}, "Forwarder Config"), Post: op([]string{"Audit"}, "Forwarder Config")},
		"/api/v1/audit/siem/health": {Get: op([]string{"Audit"}, "Health"), Post: op([]string{"Audit"}, "Health")},
		"/api/v1/audit/siem/health-check": {Get: op([]string{"Audit"}, "Health Check"), Post: op([]string{"Audit"}, "Health Check")},
		"/api/v1/audit/siem/metrics": {Get: op([]string{"Audit"}, "Metrics"), Post: op([]string{"Audit"}, "Metrics")},
		"/api/v1/audit/stream": {Get: op([]string{"Audit"}, "Stream"), Post: op([]string{"Audit"}, "Stream")},
		"/api/v1/audit/tamper-check": {Get: op([]string{"Audit"}, "Tamper Check"), Post: op([]string{"Audit"}, "Tamper Check")},
		"/api/v1/audit/threat-feed": {Get: op([]string{"Audit"}, "Threat Feed"), Post: op([]string{"Audit"}, "Threat Feed")},
		"/api/v1/audit/threat-intel/check": {Get: op([]string{"Audit"}, "Check"), Post: op([]string{"Audit"}, "Check")},
		"/api/v1/audit/threat-intel/indicators": {Get: op([]string{"Audit"}, "Indicators"), Post: op([]string{"Audit"}, "Indicators")},
		"/api/v1/audit/threat-intel/sources": {Get: op([]string{"Audit"}, "Sources"), Post: op([]string{"Audit"}, "Sources")},
		"/api/v1/audit/threat-intel/sources/": {Get: op([]string{"Audit"}, "Sources"), Post: op([]string{"Audit"}, "Sources")},
		"/api/v1/audit/threat-intel/stats": {Get: op([]string{"Audit"}, "Stats"), Post: op([]string{"Audit"}, "Stats")},
		"/api/v1/audit/timeline/reconstruct": {Get: op([]string{"Audit"}, "Reconstruct"), Post: op([]string{"Audit"}, "Reconstruct")},
		"/api/v1/audit/verify-integrity": {Get: op([]string{"Audit"}, "Verify Integrity"), Post: op([]string{"Audit"}, "Verify Integrity")},
		"/api/v1/audit/webhooks": {Get: op([]string{"Audit"}, "Webhooks"), Post: op([]string{"Audit"}, "Webhooks")},
		"/api/v1/audit/webhooks/": {Get: op([]string{"Audit"}, "Webhooks"), Post: op([]string{"Audit"}, "Webhooks")},
		"/api/v1/audit/webhooks/delivery-status": {Get: op([]string{"Audit"}, "Delivery Status"), Post: op([]string{"Audit"}, "Delivery Status")},
		"/api/v1/audit/ws": {Get: op([]string{"Audit"}, "Ws"), Post: op([]string{"Audit"}, "Ws")},
		"/api/v1/auth/access-keys": {Get: op([]string{"API Keys"}, "Access Keys"), Post: op([]string{"API Keys"}, "Access Keys")},
		"/api/v1/auth/access-keys/": {Get: op([]string{"API Keys"}, "Access Keys"), Post: op([]string{"API Keys"}, "Access Keys")},


	}
}

// SwaggerUIHandler serves the interactive Swagger UI at /docs.
func SwaggerUIHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(swaggerUITemplate))
	}
}

// OpenAPISpecHandler serves the raw JSON spec at /swagger.json.
func OpenAPISpecHandler() http.HandlerFunc {
	spec := GenerateOpenAPISpec()
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(spec)
	}
}

const swaggerUITemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>GGID API Documentation</title>
<link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
<style>
  body { margin: 0; }
  .topbar { background: #1a1a2e; padding: 10px 20px; display: flex; align-items: center; gap: 16px; }
  .topbar a { color: #e0e0e0; text-decoration: none; font-size: 14px; }
  .topbar .brand { font-size: 18px; font-weight: bold; color: #fff; }
  .topbar .badge { background: #4CAF50; color: #fff; padding: 2px 8px; border-radius: 4px; font-size: 12px; }
</style>
</head>
<body>
<div class="topbar">
  <span class="brand">GGID Platform API</span>
  <span class="badge">Interactive</span>
  <a href="/swagger.json" target="_blank">swagger.json</a>
  <a href="https://github.com/topcheer/ggid" target="_blank">GitHub</a>
</div>
<div id="swagger-ui"></div>
<script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
<script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-standalone-preset.js"></script>
<script>
window.onload = function() {
  window.ui = SwaggerUIBundle({
    url: '/swagger.json',
    dom_id: '#swagger-ui',
    deepLinking: true,
    presets: [SwaggerUIBundle.presets.apis, SwaggerUIStandalonePreset],
    plugins: [SwaggerUIBundle.plugins.DownloadUrl],
    layout: 'StandaloneLayout',
    docExpansion: 'list',
    filter: true,
    showRequestHeaders: true,
    showCommonExtensions: true,
    tryItOutEnabled: true,
    supportedSubmitMethods: ['get', 'post', 'put', 'delete', 'patch'],
    requestInterceptor: function(req) {
      // Auto-attach tenant header from localStorage if available.
      var token = localStorage.getItem('ggid_token');
      if (token) {
        req.headers['Authorization'] = 'Bearer ' + token;
      }
      var tenant = localStorage.getItem('ggid_tenant');
      if (tenant) {
        req.headers['X-Tenant-ID'] = tenant;
      }
      return req;
    },
    responseInterceptor: function(res) {
      // Auto-capture token from login responses.
      if (res.url.includes('/auth/login') && res.body) {
        try {
          var body = JSON.parse(res.body);
          if (body.access_token) {
            localStorage.setItem('ggid_token', body.access_token);
          }
        } catch(e) {}
      }
      return res;
    }
  });
};
</script>
</body>
</html>`
