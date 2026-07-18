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
		// Auth
		"/api/v1/auth/login":            {Post: op([]string{"Auth"}, "User login")},
		"/api/v1/auth/register":         {Post: op([]string{"Auth"}, "Self-service registration")},
		"/api/v1/auth/verify-email":     {Get: op([]string{"Auth"}, "Verify email address")},
		"/api/v1/auth/forgot-password":  {Post: op([]string{"Auth"}, "Request password reset")},
		"/api/v1/auth/reset-password":   {Post: op([]string{"Auth"}, "Reset password")},
		"/api/v1/auth/profile":          {Put: op([]string{"Auth"}, "Update own profile")},
		"/api/v1/auth/password-policy":  {Get: op([]string{"Auth"}, "Get password policy")},
		"/api/v1/auth/sessions":         {Get: op([]string{"Auth"}, "List sessions")},
		"/api/v1/auth/mfa/enroll":       {Post: op([]string{"Auth"}, "Enroll MFA")},
		"/api/v1/auth/mfa/verify":       {Post: op([]string{"Auth"}, "Verify MFA")},
		"/api/v1/auth/webauthn/begin":   {Post: op([]string{"Auth"}, "Begin WebAuthn registration")},
		// Identity
		"/api/v1/identity/users":            {Get: op([]string{"Identity"}, "List users"), Post: op([]string{"Identity"}, "Create user")},
		"/api/v1/identity/groups":           {Get: op([]string{"Identity"}, "List groups"), Post: op([]string{"Identity"}, "Create group")},
		"/api/v1/identity/roles":            {Get: op([]string{"Identity"}, "List roles")},
		"/api/v1/identity/dashboard/stats":  {Get: op([]string{"Identity"}, "Dashboard statistics")},
		"/api/v1/identity/consent/registry":  {Get: op([]string{"Identity"}, "Consent registry"), Post: op([]string{"Identity"}, "Grant consent")},
		// OAuth
		"/api/v1/oauth/token":              {Post: op([]string{"OAuth"}, "Issue token")},
		"/api/v1/oauth/authorize":          {Post: op([]string{"OAuth"}, "Authorize")},
		"/api/v1/oauth/clients":            {Get: op([]string{"OAuth"}, "List clients")},
		"/api/v1/oauth/introspect":         {Post: op([]string{"OAuth"}, "Token introspection")},
		"/api/v1/oauth/revoke":             {Post: op([]string{"OAuth"}, "Revoke token")},
		"/.well-known/openid-configuration": {Get: op([]string{"OAuth"}, "OIDC discovery")},
		// Policy
		"/api/v1/policy/authorize":     {Post: op([]string{"Policy"}, "Unified PDP authorize")},
		"/api/v1/policy/decisions":     {Get: op([]string{"Policy"}, "Decision audit log")},
		"/api/v1/risk/evaluate":        {Post: op([]string{"Risk"}, "Evaluate risk score")},
		"/api/v1/risk/scores/{user_id}": {Get: op([]string{"Risk"}, "Get risk score")},
		"/api/v1/risk/signals":         {Get: op([]string{"Risk"}, "List risk signals")},
		// Audit
		"/api/v1/audit/events":         {Get: op([]string{"Audit"}, "List audit events")},
		"/api/v1/audit/incidents":      {Get: op([]string{"Audit"}, "List incidents")},
		"/api/v1/soar/playbooks":       {Get: op([]string{"SOAR"}, "List playbooks"), Post: op([]string{"SOAR"}, "Create playbook")},
		// Admin
		"/api/v1/admin/backups":        {Get: op([]string{"Admin"}, "List backups")},
		"/api/v1/admin/backups/trigger": {Post: op([]string{"Admin"}, "Trigger backup")},
		"/api/v1/admin/secrets":        {Get: op([]string{"Admin"}, "List secret references")},
		"/api/v1/admin/keys":           {Get: op([]string{"Admin"}, "List active keys")},
		"/api/v1/admin/email/config":   {Get: op([]string{"Admin"}, "Email config"), Put: op([]string{"Admin"}, "Update email config")},
		"/api/v1/admin/rls/status":     {Get: op([]string{"Admin"}, "RLS status")},
		"/api/v1/quotas/{tenant_id}":   {Get: op([]string{"Admin"}, "Get tenant quota"), Put: op([]string{"Admin"}, "Update quota")},
		// MDM
		"/api/v1/mdm/connectors":       {Get: op([]string{"MDM"}, "List connectors"), Post: op([]string{"MDM"}, "Add connector")},
		"/api/v1/mdm/devices":          {Get: op([]string{"MDM"}, "List MDM devices")},
		// HR
		"/api/v1/hr/connectors":        {Get: op([]string{"HR"}, "List HR connectors"), Post: op([]string{"HR"}, "Add connector")},
		"/api/v1/hr/sync":              {Post: op([]string{"HR"}, "Trigger HR sync")},
		"/api/v1/hr/dormant":           {Get: op([]string{"HR"}, "Dormant accounts")},
		// Plugins
		"/api/v1/plugins":              {Get: op([]string{"Plugins"}, "List plugins")},
		"/api/v1/plugins/upload":       {Post: op([]string{"Plugins"}, "Upload plugin")},
		// Notifications
		"/api/v1/notifications/rules":  {Get: op([]string{"Notifications"}, "List rules"), Post: op([]string{"Notifications"}, "Create rule")},
		"/api/v1/notifications/log":   {Get: op([]string{"Notifications"}, "Notification log")},
		// GraphQL + Observability
		"/graphql":                     {Post: op([]string{"GraphQL"}, "GraphQL endpoint")},
		"/api/v1/observability/health": {Get: op([]string{"Observability"}, "Exporter health")},
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
<title>GGID API Documentation</title>
<link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
</head>
<body>
<div id="swagger-ui"></div>
<script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
<script>
window.onload = function() {
	window.ui = SwaggerUIBundle({
		url: '/swagger.json',
		dom_id: '#swagger-ui',
		deepLinking: true,
		presets: [SwaggerUIBundle.presets.apis],
		layout: 'BaseLayout',
	});
};
</script>
</body>
</html>`
