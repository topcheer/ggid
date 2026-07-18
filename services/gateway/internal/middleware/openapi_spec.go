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
