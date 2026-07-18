package middleware

// This file augments the top 50 core endpoints with full request/response
// schemas. It replaces the basic `op()` calls for key endpoints in
// addAuthPaths, addIdentityPaths, addOAuthPaths, and addPolicyPaths.
//
// The EnhancedOperation type has the same JSON structure as OpenAPIOperation
// but adds requestBody, parameters, and schema-rich response content.

// enhanceAuthPaths upgrades auth core endpoints with request/response schemas.
// Called after addAuthPaths to overwrite the basic entries.
func enhanceAuthPaths(m map[string]OpenAPIPath) {
	// POST /api/v1/auth/login
	m["/api/v1/auth/login"] = OpenAPIPath{
		Post: enhancedOp([]string{"Auth"}, "User login",
			"Authenticate with username/password. Returns JWT access + refresh tokens.").
			WithBody("LoginRequest", "Login credentials", true).
			WithOK("TokenResponse", "Authentication successful").
			With401().
			With429().
			Done(),
	}

	// POST /api/v1/auth/logout
	m["/api/v1/auth/logout"] = OpenAPIPath{
		Post: enhancedOp([]string{"Auth"}, "Logout",
			"Invalidate the current session and refresh token.").
			WithBody("LogoutRequest", "Refresh token to invalidate", false).
			WithOK("OKResponse", "Logout successful").
			Done(),
	}

	// POST /api/v1/auth/refresh
	m["/api/v1/auth/refresh"] = OpenAPIPath{
		Post: enhancedOp([]string{"Auth"}, "Refresh access token",
			"Exchange a valid refresh token for a new access token pair.").
			WithBody("RefreshRequest", "Refresh token", true).
			WithOK("TokenResponse", "New token pair").
			With401().
			Done(),
	}

	// POST /api/v1/auth/register
	m["/api/v1/auth/register"] = OpenAPIPath{
		Post: enhancedOp([]string{"Auth"}, "Register new user",
			"Create a new user account. Requires X-Tenant-ID header.").
			WithParam(tenantHeader()).
			WithBody("RegisterRequest", "User registration details", true).
			With201("UserResponse", "User created").
			With400().
			With409().
			Done(),
	}

	// POST /api/v1/auth/forgot-password
	m["/api/v1/auth/forgot-password"] = OpenAPIPath{
		Post: enhancedOp([]string{"Auth"}, "Request password reset",
			"Send a password reset email to the given address.").
			WithBody("ForgotPasswordRequest", "Email address", true).
			WithOK("OKResponse", "Reset email sent (if account exists)").
			Done(),
	}

	// POST /api/v1/auth/password/change
	m["/api/v1/auth/password/change"] = OpenAPIPath{
		Post: enhancedOp([]string{"Auth"}, "Change password",
			"Change the current user's password. Requires authentication.").
			WithBody("ChangePasswordRequest", "Current and new password", true).
			WithOK("OKResponse", "Password changed").
			With400().
			With401().
			Done(),
	}

	// GET /api/v1/auth/me
	m["/api/v1/auth/me"] = OpenAPIPath{
		Get: enhancedOp([]string{"Auth"}, "Get current session",
			"Returns the authenticated user's session info including roles and permissions.").
			WithOK("SessionResponse", "Current session details").
			With401().
			Done(),
	}

	// POST /api/v1/auth/mfa/enroll
	m["/api/v1/auth/mfa/enroll"] = OpenAPIPath{
		Post: enhancedOp([]string{"MFA"}, "Enroll MFA",
			"Enroll a new MFA factor (TOTP, SMS, or email). Returns secret and backup codes.").
			WithBody("MFAEnrollRequest", "MFA method selection", true).
			WithOK("MFAEnrollResponse", "MFA enrollment details").
			With401().
			Done(),
	}

	// POST /api/v1/auth/mfa/verify
	m["/api/v1/auth/mfa/verify"] = OpenAPIPath{
		Post: enhancedOp([]string{"MFA"}, "Verify MFA code",
			"Verify a 6-digit MFA code to complete login or enrollment.").
			WithBody("MFAVerifyRequest", "MFA verification code", true).
			WithOK("TokenResponse", "MFA verified, tokens issued").
			With401().
			Done(),
	}

	// POST /api/v1/auth/mfa/disable
	m["/api/v1/auth/mfa/disable"] = OpenAPIPath{
		Post: enhancedOp([]string{"MFA"}, "Disable MFA",
			"Disable MFA for the current user. Requires current password verification.").
			WithOK("OKResponse", "MFA disabled").
			With401().
			Done(),
	}

	// GET /api/v1/auth/mfa/backup-codes
	m["/api/v1/auth/mfa/backup-codes"] = OpenAPIPath{
		Get: enhancedOp([]string{"MFA"}, "List backup codes",
			"Returns remaining MFA backup codes for the current user.").
			WithOK("", "List of backup codes").
			With401().
			Done(),
	}

	// POST /api/v1/auth/sessions
	m["/api/v1/auth/sessions"] = OpenAPIPath{
		Get: enhancedOp([]string{"Auth"}, "List active sessions",
			"Returns all active sessions for the authenticated user.").
			WithOK("", "List of active sessions").
			With401().
			Done(),
		Delete: enhancedOp([]string{"Auth"}, "Revoke all sessions",
			"Revoke all sessions except the current one.").
			WithOK("OKResponse", "Sessions revoked").
			With401().
			Done(),
	}

	// POST /api/v1/auth/validate
	m["/api/v1/auth/validate"] = OpenAPIPath{
		Post: enhancedOp([]string{"Auth"}, "Validate token",
			"Validate a JWT token and return its claims. Used internally by gateway.").
			WithOK("", "Token validation result").
			With401().
			Done(),
	}

	// POST /api/v1/auth/impersonate
	m["/api/v1/auth/impersonate"] = OpenAPIPath{
		Post: enhancedOp([]string{"Auth"}, "Impersonate user",
			"Start an impersonation session. Requires admin privileges.").
			WithBody("", "Target user ID", true).
			WithOK("TokenResponse", "Impersonation tokens").
			With401().
			With403().
			Done(),
	}
}

// enhanceIdentityPaths upgrades identity core endpoints with schemas.
func enhanceIdentityPaths(m map[string]OpenAPIPath) {
	// GET /api/v1/users
	m["/api/v1/users"] = OpenAPIPath{
		Get: enhancedOp([]string{"Users"}, "List users",
			"Returns a paginated list of users with optional filtering and sorting.").
			WithParam(tenantHeader()).
			WithQueryParam("search", "Free-text search term", false).
			WithQueryParam("page_size", "Page size (default 50)", false).
			WithQueryParam("status", "Filter by user status", false).
			WithOK("ListUsersResponse", "Paginated user list").
			With401().
			Done(),
		Post: enhancedOp([]string{"Users"}, "Create user",
			"Create a new user account scoped to the tenant.").
			WithParam(tenantHeader()).
			WithBody("CreateUserRequest", "User creation payload", true).
			With201("UserResponse", "User created").
			With400().
			With401().
			With409().
			Done(),
	}

	// GET/PUT/DELETE /api/v1/users/{id}
	m["/api/v1/users/{id}"] = OpenAPIPath{
		Get: enhancedOp([]string{"Users"}, "Get user by ID",
			"Returns the full profile of the user identified by the path ID.").
			WithParam(tenantHeader()).
			WithPathParam("id", "User UUID").
			WithOK("UserResponse", "User profile").
			With401().
			With404().
			Done(),
		Put: enhancedOp([]string{"Users"}, "Update user",
			"Update mutable profile fields for the given user.").
			WithParam(tenantHeader()).
			WithPathParam("id", "User UUID").
			WithBody("UpdateUserRequest", "Fields to update", true).
			WithOK("UserResponse", "Updated user").
			With400().
			With401().
			With404().
			Done(),
		Delete: enhancedOp([]string{"Users"}, "Delete user",
			"Permanently delete the user identified by the path ID.").
			WithParam(tenantHeader()).
			WithPathParam("id", "User UUID").
			WithOK("OKResponse", "User deleted").
			With401().
			With404().
			Done(),
	}

	// GET /api/v1/users/search
	m["/api/v1/users/search"] = OpenAPIPath{
		Get: enhancedOp([]string{"Users"}, "Search users",
			"Performs a filtered search across users with pagination.").
			WithQueryParam("q", "Search query", false).
			WithQueryParam("status", "User status filter", false).
			WithQueryParam("limit", "Max results (default 20)", false).
			WithQueryParam("offset", "Result offset", false).
			WithOK("", "Search results").
			With401().
			Done(),
	}

	// GET /api/v1/users/me
	m["/api/v1/users/me"] = OpenAPIPath{
		Get: enhancedOp([]string{"Users"}, "Get current user profile",
			"Returns the profile of the authenticated user.").
			WithOK("UserResponse", "Current user profile").
			With401().
			Done(),
	}

	// POST /api/v1/users/{id}/roles
	m["/api/v1/users/{id}/roles"] = OpenAPIPath{
		Get: enhancedOp([]string{"Roles"}, "List user roles",
			"Returns all roles assigned to the user.").
			WithParam(tenantHeader()).
			WithPathParam("id", "User UUID").
			WithOK("", "List of role assignments").
			With401().
			Done(),
		Post: enhancedOp([]string{"Roles"}, "Assign role to user",
			"Assign the given role to the user. Duplicate assignments are rejected.").
			WithParam(tenantHeader()).
			WithPathParam("id", "User UUID").
			WithBody("AssignRoleRequest", "Role to assign", true).
			With201("", "Role assigned").
			With400().
			With401().
			With409().
			Done(),
	}

	// POST /api/v1/users/{id}/lock
	m["/api/v1/users/{id}/lock"] = OpenAPIPath{
		Post: enhancedOp([]string{"Users"}, "Lock user",
			"Lock the user account, preventing authentication.").
			WithParam(tenantHeader()).
			WithPathParam("id", "User UUID").
			WithOK("UserResponse", "Locked user").
			With401().
			With404().
			Done(),
	}

	// POST /api/v1/users/{id}/deactivate
	m["/api/v1/users/{id}/deactivate"] = OpenAPIPath{
		Post: enhancedOp([]string{"Users"}, "Deactivate user",
			"Deactivate the user account, marking it inactive.").
			WithParam(tenantHeader()).
			WithPathParam("id", "User UUID").
			WithOK("UserResponse", "Deactivated user").
			With401().
			With404().
			Done(),
	}

	// SCIM Groups
	m["/api/v1/scim/Groups"] = OpenAPIPath{
		Get: enhancedOp([]string{"Groups"}, "List groups (SCIM)",
			"Returns a paginated list of SCIM Group resources.").
			WithParam(tenantHeader()).
			WithQueryParam("count", "Page size (default 100)", false).
			WithQueryParam("filter", "SCIM filter expression", false).
			WithOK("SCIMGroupResponse", "List of groups").
			With401().
			Done(),
		Post: enhancedOp([]string{"Groups"}, "Create group (SCIM)",
			"Creates a new SCIM Group resource. Groups map to GGID roles.").
			WithParam(tenantHeader()).
			WithBody("CreateGroupRequest", "SCIM Group payload", true).
			With201("SCIMGroupResponse", "Group created").
			With400().
			With401().
			Done(),
	}

	// SCIM Tokens
	m["/api/v1/identity/scim/tokens"] = OpenAPIPath{
		Get: enhancedOp([]string{"SCIM"}, "List SCIM tokens",
			"Returns all SCIM bearer tokens for the tenant.").
			WithParam(tenantHeader()).
			WithOK("", "List of tokens").
			With401().
			Done(),
		Post: enhancedOp([]string{"SCIM"}, "Create SCIM token",
			"Mint a new SCIM bearer token. Plaintext token is returned only once.").
			WithParam(tenantHeader()).
			WithBody("CreateSCIMTokenRequest", "Token name", true).
			With201("CreateSCIMTokenResponse", "Token created").
			With400().
			With401().
			Done(),
	}
}

// enhanceOAuthPaths upgrades OAuth core endpoints with schemas.
func enhanceOAuthPaths(m map[string]OpenAPIPath) {
	// GET /oauth/authorize
	m["/oauth/authorize"] = OpenAPIPath{
		Get: enhancedOp([]string{"OAuth"}, "Authorization endpoint",
			"OAuth 2.1 authorization endpoint. Redirects to login or consent UI.").
			WithQueryParam("response_type", "Response type (code, token)", true).
			WithQueryParam("client_id", "OAuth client ID", true).
			WithQueryParam("redirect_uri", "Redirect URI", true).
			WithQueryParam("scope", "Requested scopes", false).
			WithQueryParam("state", "CSRF state token", false).
			With302("Redirect to callback URL").
			With400().
			Done(),
	}

	// POST /oauth/token
	m["/oauth/token"] = OpenAPIPath{
		Post: enhancedOp([]string{"OAuth"}, "Token endpoint",
			"OAuth 2.1 token endpoint. Supports authorization_code, refresh_token, and client_credentials grants.").
			WithBody("OAuthTokenRequest", "Token request parameters", true).
			WithOK("OAuthTokenResponse", "Token response").
			With400().
			With401().
			Done(),
	}

	// GET /oauth/clients
	m["/oauth/clients"] = OpenAPIPath{
		Get: enhancedOp([]string{"OAuth"}, "List OAuth clients",
			"Returns all registered OAuth clients for the tenant.").
			WithParam(tenantHeader()).
			WithOK("OAuthClientResponse", "List of clients").
			With401().
			Done(),
		Post: enhancedOp([]string{"OAuth"}, "Register OAuth client",
			"Register a new OAuth client application.").
			WithParam(tenantHeader()).
			WithBody("OAuthClientResponse", "Client registration details", true).
			With201("OAuthClientResponse", "Client created").
			With400().
			With401().
			Done(),
	}

	// GET /oauth/clients/{id}
	m["/oauth/clients/{id}"] = OpenAPIPath{
		Get: enhancedOp([]string{"OAuth"}, "Get OAuth client",
			"Returns the OAuth client identified by the path ID.").
			WithPathParam("id", "Client UUID").
			WithOK("OAuthClientResponse", "Client details").
			With401().
			With404().
			Done(),
		Delete: enhancedOp([]string{"OAuth"}, "Delete OAuth client",
			"Permanently delete the OAuth client.").
			WithPathParam("id", "Client UUID").
			WithOK("OKResponse", "Client deleted").
			With401().
			With404().
			Done(),
	}

	// GET /.well-known/openid-configuration
	m["/.well-known/openid-configuration"] = OpenAPIPath{
		Get: enhancedOp([]string{"OAuth"}, "OIDC discovery",
			"OpenID Connect discovery endpoint. Returns provider metadata.").
			WithOK("", "OIDC provider metadata").
			Done(),
	}

	// GET /.well-known/jwks.json
	m["/.well-known/jwks.json"] = OpenAPIPath{
		Get: enhancedOp([]string{"OAuth"}, "JWKS endpoint",
			"JSON Web Key Set endpoint. Returns public signing keys for JWT verification.").
			WithOK("", "JWKS key set").
			Done(),
	}

	// POST /oauth/revoke
	m["/oauth/revoke"] = OpenAPIPath{
		Post: enhancedOp([]string{"OAuth"}, "Revoke token",
			"OAuth 2.1 token revocation endpoint (RFC 7009).").
			WithBody("RefreshRequest", "Token to revoke", true).
			WithOK("OKResponse", "Token revoked").
			Done(),
	}

	// POST /oauth/introspect
	m["/oauth/introspect"] = OpenAPIPath{
		Post: enhancedOp([]string{"OAuth"}, "Introspect token",
			"OAuth 2.1 token introspection endpoint (RFC 7662).").
			WithBody("RefreshRequest", "Token to introspect", true).
			WithOK("", "Introspection result").
			With401().
			Done(),
	}
}

// enhancePolicyPaths upgrades policy core endpoints with schemas.
func enhancePolicyPaths(m map[string]OpenAPIPath) {
	// GET /api/v1/policies
	m["/api/v1/policies"] = OpenAPIPath{
		Get: enhancedOp([]string{"Policy"}, "List policies",
			"Returns all access policies for the tenant.").
			WithParam(tenantHeader()).
			WithOK("PolicyResponse", "List of policies").
			With401().
			Done(),
		Post: enhancedOp([]string{"Policy"}, "Create policy",
			"Create a new access policy.").
			WithParam(tenantHeader()).
			WithBody("CreatePolicyRequest", "Policy definition", true).
			With201("PolicyResponse", "Policy created").
			With400().
			With401().
			Done(),
	}

	// GET /api/v1/policies/{id}
	m["/api/v1/policies/{id}"] = OpenAPIPath{
		Get: enhancedOp([]string{"Policy"}, "Get policy",
			"Returns the policy identified by the path ID.").
			WithParam(tenantHeader()).
			WithPathParam("id", "Policy UUID").
			WithOK("PolicyResponse", "Policy details").
			With401().
			With404().
			Done(),
		Delete: enhancedOp([]string{"Policy"}, "Delete policy",
			"Delete the policy identified by the path ID.").
			WithParam(tenantHeader()).
			WithPathParam("id", "Policy UUID").
			WithOK("OKResponse", "Policy deleted").
			With401().
			With404().
			Done(),
	}

	// GET /api/v1/roles
	m["/api/v1/roles"] = OpenAPIPath{
		Get: enhancedOp([]string{"Roles"}, "List roles",
			"Returns all roles for the tenant.").
			WithParam(tenantHeader()).
			WithOK("", "List of roles").
			With401().
			Done(),
		Post: enhancedOp([]string{"Roles"}, "Create role",
			"Create a new role.").
			WithParam(tenantHeader()).
			WithOK("", "Role created").
			With400().
			With401().
			Done(),
	}

	// GET /api/v1/roles/{id}
	m["/api/v1/roles/{id}"] = OpenAPIPath{
		Get: enhancedOp([]string{"Roles"}, "Get role",
			"Returns the role identified by the path ID.").
			WithParam(tenantHeader()).
			WithPathParam("id", "Role UUID").
			WithOK("", "Role details").
			With401().
			With404().
			Done(),
		Delete: enhancedOp([]string{"Roles"}, "Delete role",
			"Delete the role identified by the path ID.").
			WithParam(tenantHeader()).
			WithPathParam("id", "Role UUID").
			WithOK("OKResponse", "Role deleted").
			With401().
			With404().
			Done(),
	}
}

// registerEnhancedPaths applies schema-rich definitions to the top 50 endpoints.
func registerEnhancedPaths(m map[string]OpenAPIPath) {
	enhanceAuthPaths(m)
	enhanceIdentityPaths(m)
	enhanceOAuthPaths(m)
	enhancePolicyPaths(m)
}

// ---------------------------------------------------------------------------
// Fluent builder methods on EnhancedOperation
// ---------------------------------------------------------------------------

// WithBody adds a request body schema reference.
func (o *EnhancedOperation) WithBody(schemaName, desc string, required bool) *EnhancedOperation {
	o.RequestBody = jsonBody(schemaName, desc, required)
	return o
}

// WithParam adds a parameter (header, query, or path).
func (o *EnhancedOperation) WithParam(p Parameter) *EnhancedOperation {
	o.Parameters = append(o.Parameters, p)
	return o
}

// WithHeaderParam adds a required header parameter.
func (o *EnhancedOperation) WithHeaderParam(name, desc string) *EnhancedOperation {
	o.Parameters = append(o.Parameters, Parameter{
		Name: name, In: "header", Required: true, Description: desc,
		Schema: SchemaRef{Type: "string"},
	})
	return o
}

// WithQueryParam adds an optional query parameter.
func (o *EnhancedOperation) WithQueryParam(name, desc string, required bool) *EnhancedOperation {
	o.Parameters = append(o.Parameters, Parameter{
		Name: name, In: "query", Required: required, Description: desc,
		Schema: SchemaRef{Type: "string"},
	})
	return o
}

// WithPathParam adds a required path parameter.
func (o *EnhancedOperation) WithPathParam(name, desc string) *EnhancedOperation {
	o.Parameters = append(o.Parameters, Parameter{
		Name: name, In: "path", Required: true, Description: desc,
		Schema: SchemaRef{Type: "string", Format: "uuid"},
	})
	return o
}

// WithOK sets the 200 response with an optional schema.
func (o *EnhancedOperation) WithOK(schemaName, desc string) *EnhancedOperation {
	o.Responses["200"] = okResp(schemaName, desc)
	return o
}

// With201 sets the 201 response with an optional schema.
func (o *EnhancedOperation) With201(schemaName, desc string) *EnhancedOperation {
	o.Responses["201"] = okResp(schemaName, desc)
	return o
}

// With302 sets the 302 redirect response.
func (o *EnhancedOperation) With302(desc string) *EnhancedOperation {
	o.Responses["302"] = EnhancedResponse{Description: desc}
	return o
}

// With400 adds a 400 Bad Request response.
func (o *EnhancedOperation) With400() *EnhancedOperation {
	code, resp := errResp("400", "Bad Request")
	o.Responses[code] = resp
	return o
}

// With401 adds a 401 Unauthorized response.
func (o *EnhancedOperation) With401() *EnhancedOperation {
	code, resp := errResp("401", "Unauthorized")
	o.Responses[code] = resp
	return o
}

// With403 adds a 403 Forbidden response.
func (o *EnhancedOperation) With403() *EnhancedOperation {
	code, resp := errResp("403", "Forbidden")
	o.Responses[code] = resp
	return o
}

// With404 adds a 404 Not Found response.
func (o *EnhancedOperation) With404() *EnhancedOperation {
	code, resp := errResp("404", "Not Found")
	o.Responses[code] = resp
	return o
}

// With409 adds a 409 Conflict response.
func (o *EnhancedOperation) With409() *EnhancedOperation {
	code, resp := errResp("409", "Conflict")
	o.Responses[code] = resp
	return o
}

// With429 adds a 429 Too Many Requests response.
func (o *EnhancedOperation) With429() *EnhancedOperation {
	o.Responses["429"] = EnhancedResponse{
		Description: "Too Many Requests — rate limit exceeded",
	}
	return o
}

// Done returns the completed EnhancedOperation pointer.
func (o *EnhancedOperation) Done() *EnhancedOperation {
	return o
}
