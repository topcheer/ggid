package middleware

// This file defines reusable OpenAPI schema definitions and enhanced operation
// builders for the top 50 GGID core endpoints (auth, identity, oauth, policy).
//
// The schema definitions use a lightweight JSON-schema-compatible representation
// that serializes directly via encoding/json.

// ---------------------------------------------------------------------------
// Schema types
// ---------------------------------------------------------------------------

// SchemaRef is either an inline schema or a $ref string.
type SchemaRef struct {
	Ref         string             `json:"$ref,omitempty"`
	Type        string             `json:"type,omitempty"`
	Format      string             `json:"format,omitempty"`
	Description string             `json:"description,omitempty"`
	Properties  map[string]SchemaRef `json:"properties,omitempty"`
	Items       *SchemaRef         `json:"items,omitempty"`
	Required    []string           `json:"required,omitempty"`
	Example     any                `json:"example,omitempty"`
}

// RequestBody describes the body of a POST/PUT/PATCH request.
type RequestBody struct {
	Description string                     `json:"description"`
	Required    bool                       `json:"required"`
	Content     map[string]MediaTypeObject `json:"content"`
}

// MediaTypeObject describes a single media type (e.g. application/json).
type MediaTypeObject struct {
	Schema SchemaRef `json:"schema"`
}

// Parameter describes a path/query/header parameter.
type Parameter struct {
	Name        string    `json:"name"`
	In          string    `json:"in"`
	Description string    `json:"description,omitempty"`
	Required    bool      `json:"required"`
	Schema      SchemaRef `json:"schema"`
}

// EnhancedResponse extends OpenAPIResponse with content schemas.
type EnhancedResponse struct {
	Description string                     `json:"description"`
	Content     map[string]MediaTypeObject `json:"content,omitempty"`
}

// EnhancedOperation extends OpenAPIOperation with request bodies, parameters,
// and schema-rich responses. Only used for top 50 endpoints that need schemas.
type EnhancedOperation struct {
	Tags        []string               `json:"tags"`
	Summary     string                 `json:"summary"`
	Description string                 `json:"description,omitempty"`
	Security    []map[string][]string  `json:"security,omitempty"`
	Parameters  []Parameter            `json:"parameters,omitempty"`
	RequestBody *RequestBody           `json:"requestBody,omitempty"`
	Responses   map[string]EnhancedResponse `json:"responses"`
}

// ---------------------------------------------------------------------------
// Schema definitions for core entities
// ---------------------------------------------------------------------------

// Schema definitions are keyed by name in the components.schemas section.
func coreSchemas() map[string]SchemaRef {
	return map[string]SchemaRef{
		// ---- Auth ----
		"LoginRequest": {
			Type: "object",
			Properties: map[string]SchemaRef{
				"username":   {Type: "string", Description: "Username or email", Example: "admin"},
				"password":   {Type: "string", Format: "password", Description: "User password", Example: "secret123"},
				"tenant_id":  {Type: "string", Format: "uuid", Description: "Tenant UUID (optional if header provided)"},
				"tenant_slug": {Type: "string", Description: "Tenant slug for tenant resolution"},
			},
			Required: []string{"username", "password"},
		},
		"TokenResponse": {
			Type: "object",
			Properties: map[string]SchemaRef{
				"access_token":      {Type: "string", Description: "JWT access token"},
				"refresh_token":     {Type: "string", Description: "JWT refresh token"},
				"token_type":        {Type: "string", Example: "Bearer"},
				"expires_in":        {Type: "integer", Description: "Access token TTL in seconds", Example: 3600},
				"refresh_expires_in": {Type: "integer", Description: "Refresh token TTL in seconds", Example: 86400},
			},
			Required: []string{"access_token", "token_type", "expires_in"},
		},
		"RefreshRequest": {
			Type: "object",
			Properties: map[string]SchemaRef{
				"refresh_token": {Type: "string", Description: "Valid refresh token"},
			},
			Required: []string{"refresh_token"},
		},
		"LogoutRequest": {
			Type: "object",
			Properties: map[string]SchemaRef{
				"refresh_token": {Type: "string", Description: "Refresh token to invalidate"},
			},
		},
		"RegisterRequest": {
			Type: "object",
			Properties: map[string]SchemaRef{
				"username":  {Type: "string", Example: "alice"},
				"email":     {Type: "string", Format: "email", Example: "alice@example.com"},
				"password":  {Type: "string", Format: "password", Example: "s3cret!"},
				"tenant_id": {Type: "string", Format: "uuid"},
			},
			Required: []string{"username", "email", "password"},
		},
		"ForgotPasswordRequest": {
			Type: "object",
			Properties: map[string]SchemaRef{
				"email": {Type: "string", Format: "email", Example: "alice@example.com"},
			},
			Required: []string{"email"},
		},
		"ChangePasswordRequest": {
			Type: "object",
			Properties: map[string]SchemaRef{
				"current_password": {Type: "string", Format: "password"},
				"new_password":     {Type: "string", Format: "password"},
			},
			Required: []string{"current_password", "new_password"},
		},
		"MFAEnrollRequest": {
			Type: "object",
			Properties: map[string]SchemaRef{
				"method": {Type: "string", Example: "totp", Description: "MFA method: totp, sms, email"},
			},
			Required: []string{"method"},
		},
		"MFAVerifyRequest": {
			Type: "object",
			Properties: map[string]SchemaRef{
				"code":   {Type: "string", Description: "6-digit MFA code", Example: "123456"},
				"method": {Type: "string", Example: "totp"},
			},
			Required: []string{"code"},
		},
		"MFAEnrollResponse": {
			Type: "object",
			Properties: map[string]SchemaRef{
				"secret":     {Type: "string", Description: "TOTP shared secret (base32)"},
				"qr_url":     {Type: "string", Format: "uri", Description: "OTPAuth URL for QR code"},
				"backup_codes": {Type: "array", Items: &SchemaRef{Type: "string"}},
			},
		},
		"SessionResponse": {
			Type: "object",
			Properties: map[string]SchemaRef{
				"user_id":     {Type: "string", Format: "uuid"},
				"username":    {Type: "string"},
				"email":       {Type: "string"},
				"roles":       {Type: "array", Items: &SchemaRef{Type: "string"}},
				"permissions": {Type: "array", Items: &SchemaRef{Type: "string"}},
				"expires_at":  {Type: "string", Format: "date-time"},
			},
		},

		// ---- Identity ----
		"CreateUserRequest": {
			Type: "object",
			Properties: map[string]SchemaRef{
				"username":     {Type: "string", Example: "alice"},
				"email":        {Type: "string", Format: "email", Example: "alice@example.com"},
				"password":     {Type: "string", Format: "password", Example: "s3cret!"},
				"phone":        {Type: "string", Example: "+1-555-0100"},
				"display_name": {Type: "string", Example: "Alice"},
				"locale":       {Type: "string", Example: "en-US"},
				"timezone":     {Type: "string", Example: "America/New_York"},
			},
			Required: []string{"username", "email", "password"},
		},
		"UpdateUserRequest": {
			Type: "object",
			Properties: map[string]SchemaRef{
				"phone":        {Type: "string"},
				"display_name": {Type: "string"},
				"locale":       {Type: "string"},
				"timezone":     {Type: "string"},
			},
		},
		"UserResponse": {
			Type: "object",
			Properties: map[string]SchemaRef{
				"id":             {Type: "string", Format: "uuid"},
				"tenant_id":      {Type: "string", Format: "uuid"},
				"username":       {Type: "string"},
				"email":          {Type: "string", Format: "email"},
				"phone":          {Type: "string"},
				"status":         {Type: "string", Example: "active"},
				"email_verified": {Type: "boolean"},
				"display_name":   {Type: "string"},
				"locale":         {Type: "string"},
				"timezone":       {Type: "string"},
				"created_at":     {Type: "string", Format: "date-time"},
				"updated_at":     {Type: "string", Format: "date-time"},
			},
		},
		"ListUsersResponse": {
			Type: "object",
			Properties: map[string]SchemaRef{
				"users":       {Type: "array", Items: &SchemaRef{Ref: "#/components/schemas/UserResponse"}},
				"total":       {Type: "integer"},
				"next_offset": {Type: "integer"},
			},
		},
		"AssignRoleRequest": {
			Type: "object",
			Properties: map[string]SchemaRef{
				"role_id":   {Type: "string", Format: "uuid"},
				"role_name": {Type: "string"},
			},
			Required: []string{"role_id"},
		},
		"CreateGroupRequest": {
			Type: "object",
			Properties: map[string]SchemaRef{
				"displayName": {Type: "string", Example: "Engineering"},
				"members": {
					Type: "array",
					Items: &SchemaRef{
						Type: "object",
						Properties: map[string]SchemaRef{
							"value": {Type: "string", Description: "User UUID"},
							"display": {Type: "string"},
						},
					},
				},
			},
			Required: []string{"displayName"},
		},
		"SCIMGroupResponse": {
			Type: "object",
			Properties: map[string]SchemaRef{
				"id":          {Type: "string", Format: "uuid"},
				"displayName": {Type: "string"},
				"members":     {Type: "array", Items: &SchemaRef{Type: "object"}},
				"meta":        {Type: "object"},
			},
		},
		"CreateSCIMTokenRequest": {
			Type: "object",
			Properties: map[string]SchemaRef{
				"name": {Type: "string", Example: " Okta provisioning"},
			},
			Required: []string{"name"},
		},
		"CreateSCIMTokenResponse": {
			Type: "object",
			Properties: map[string]SchemaRef{
				"id":         {Type: "string", Format: "uuid"},
				"name":       {Type: "string"},
				"token":      {Type: "string", Description: "Bearer token (shown only once)"},
				"scopes":     {Type: "array", Items: &SchemaRef{Type: "string"}},
				"created_at": {Type: "string", Format: "date-time"},
			},
		},

		// ---- OAuth ----
		"OAuthAuthorizeRequest": {
			Type: "object",
			Properties: map[string]SchemaRef{
				"response_type": {Type: "string", Example: "code"},
				"client_id":     {Type: "string"},
				"redirect_uri":  {Type: "string", Format: "uri"},
				"scope":         {Type: "string", Example: "openid profile email"},
				"state":         {Type: "string"},
			},
			Required: []string{"response_type", "client_id", "redirect_uri"},
		},
		"OAuthTokenRequest": {
			Type: "object",
			Properties: map[string]SchemaRef{
				"grant_type":    {Type: "string", Example: "authorization_code"},
				"code":          {Type: "string"},
				"redirect_uri":  {Type: "string", Format: "uri"},
				"client_id":     {Type: "string"},
				"client_secret": {Type: "string"},
				"refresh_token": {Type: "string"},
			},
			Required: []string{"grant_type"},
		},
		"OAuthTokenResponse": {
			Type: "object",
			Properties: map[string]SchemaRef{
				"access_token":  {Type: "string"},
				"token_type":    {Type: "string", Example: "Bearer"},
				"expires_in":    {Type: "integer", Example: 3600},
				"refresh_token": {Type: "string"},
				"id_token":      {Type: "string", Description: "OIDC ID token (if openid scope)"},
				"scope":         {Type: "string"},
			},
		},
		"OAuthClientResponse": {
			Type: "object",
			Properties: map[string]SchemaRef{
				"id":            {Type: "string", Format: "uuid"},
				"client_id":     {Type: "string"},
				"client_name":   {Type: "string"},
				"redirect_uris": {Type: "array", Items: &SchemaRef{Type: "string"}},
				"grant_types":   {Type: "array", Items: &SchemaRef{Type: "string"}},
				"scopes":        {Type: "array", Items: &SchemaRef{Type: "string"}},
				"created_at":    {Type: "string", Format: "date-time"},
			},
		},

		// ---- Policy ----
		"CreatePolicyRequest": {
			Type: "object",
			Properties: map[string]SchemaRef{
				"name":        {Type: "string", Example: "require-mfa"},
				"description": {Type: "string"},
				"effect":      {Type: "string", Example: "deny"},
				"conditions":  {Type: "object"},
				"actions":     {Type: "array", Items: &SchemaRef{Type: "string"}},
			},
			Required: []string{"name", "effect"},
		},
		"PolicyResponse": {
			Type: "object",
			Properties: map[string]SchemaRef{
				"id":          {Type: "string", Format: "uuid"},
				"name":        {Type: "string"},
				"description": {Type: "string"},
				"effect":      {Type: "string"},
				"conditions":  {Type: "object"},
				"actions":     {Type: "array", Items: &SchemaRef{Type: "string"}},
				"created_at":  {Type: "string", Format: "date-time"},
			},
		},

		// ---- Common ----
		"ErrorResponse": {
			Type: "object",
			Properties: map[string]SchemaRef{
				"error":   {Type: "string", Description: "Error message"},
				"code":    {Type: "string", Description: "Machine-readable error code"},
				"details": {Type: "object"},
			},
			Required: []string{"error"},
		},
		"OKResponse": {
			Type: "object",
			Properties: map[string]SchemaRef{
				"message": {Type: "string", Example: "OK"},
				"success": {Type: "boolean", Example: true},
			},
		},
		"PaginatedResponse": {
			Type: "object",
			Properties: map[string]SchemaRef{
				"total":  {Type: "integer"},
				"limit":  {Type: "integer"},
				"offset": {Type: "integer"},
			},
		},
	}
}

// ---------------------------------------------------------------------------
// Helper builders for enhanced operations
// ---------------------------------------------------------------------------

// jsonContent creates a media type map for application/json.
func jsonContent(schemaName string) map[string]MediaTypeObject {
	return map[string]MediaTypeObject{
		"application/json": {Schema: SchemaRef{Ref: "#/components/schemas/" + schemaName}},
	}
}

// jsonBody creates a RequestBody with the given schema name.
func jsonBody(schemaName, desc string, required bool) *RequestBody {
	return &RequestBody{
		Description: desc,
		Required:    required,
		Content:     jsonContent(schemaName),
	}
}

// okResp creates a 200 response referencing the given schema.
func okResp(schemaName, desc string) EnhancedResponse {
	r := EnhancedResponse{Description: desc}
	if schemaName != "" {
		r.Content = jsonContent(schemaName)
	}
	return r
}

// errResp creates a standard error response.
func errResp(code, desc string) (string, EnhancedResponse) {
	return code, EnhancedResponse{
		Description: desc,
		Content:     jsonContent("ErrorResponse"),
	}
}

// enhancedOp creates an EnhancedOperation with standard responses.
func enhancedOp(tags []string, summary, desc string) *EnhancedOperation {
	resp400, _ := errResp("400", "Bad Request")
	resp401, _ := errResp("401", "Unauthorized")
	return &EnhancedOperation{
		Tags:        tags,
		Summary:     summary,
		Description: desc,
		Responses: map[string]EnhancedResponse{
			"200": okResp("", "OK"),
			"400": resp400,
			"401": resp401,
		},
	}
}

// tenantHeader returns the standard X-Tenant-ID parameter.
func tenantHeader() Parameter {
	return Parameter{
		Name: "X-Tenant-ID", In: "header", Required: true,
		Description: "Tenant identifier",
		Schema:      SchemaRef{Type: "string", Format: "uuid"},
	}
}
