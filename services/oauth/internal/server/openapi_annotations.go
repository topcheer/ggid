// Package server provides OpenAPI (Swagger) annotations for the GGID OAuth 2.0 /
// OpenID Connect Identity Provider service.
//
// @title GGID OAuth 2.0 / OIDC API
// @version 1.0
// @description OpenAPI annotations for the GGID OAuth/OIDC Identity Provider.
// @host localhost:8080
// @BasePath /
package server

// ----------------------------------------------------------------------------
// 1. Authorize — GET/POST /oauth/authorize
// ----------------------------------------------------------------------------

// @Summary      Authorize
// @Description  OAuth 2.0 authorization endpoint (RFC 6749 §3.1). Authenticates the
// @Description  end-user and issues an authorization code that is delivered to the
// @Description  client via a front-channel redirect. Supports PKCE (RFC 7636),
// @Description  OIDC scopes, and Rich Authorization Requests (RFC 9396).
// @Tags         OAuth2 / Authorization
// @Accept       json
// @Produce      json
// @Param        response_type         query    string  true  "Must be \"code\""
// @Param        client_id             query    string  true  "Registered client identifier"
// @Param        redirect_uri          query    string  true  "Client redirect URI registered for this client"
// @Param        scope                 query    string  false "Space-delimited scopes (e.g. \"openid profile email\")"
// @Param        state                 query    string  false "Opaque value returned to the client to prevent CSRF"
// @Param        nonce                 query    string  false "OIDC nonce value to mitigate replay attacks"
// @Param        code_challenge        query    string  false "PKCE code challenge (RFC 7636)"
// @Param        code_challenge_method query    string  false "PKCE method: \"S256\" or \"plain\""
// @Param        acr_values            query    string  false "Requested Authentication Context Class Reference values"
// @Param        user_id               query    string  false "Authenticated user UUID (X-User-ID header alternative)"
// @Param        X-Tenant-ID           header   string  true  "Tenant identifier"
// @Param        authorization_details query    string  false "RAR authorization details JSON array (RFC 9396)"
// @Success      302                   {string} string  "Redirect to client redirect_uri with code and state"
// @Success      200                   {object} map[string]any "Consent required response"
// @Failure      400                   {object} map[string]string "invalid_request / unsupported_response_type"
// @Failure      403                   {object} map[string]string "untrusted_federation_client"
// @Router       /oauth/authorize [get]
// @Router       /oauth/authorize [post]

// ----------------------------------------------------------------------------
// 2. Token — POST /oauth/token
// ----------------------------------------------------------------------------

// @Summary      Token
// @Description  OAuth 2.0 token endpoint (RFC 6749 §3.2). Exchanges an
// @Description  authorization code, refreshes tokens, or issues tokens via
// @Description  client-credentials, device-code (RFC 8628), JWT-bearer
// @Description  (RFC 7523), or token-exchange (RFC 8693) grants. Supports DPoP
// @Description  proof-of-possession (RFC 9449).
// @Tags         OAuth2 / Token
// @Accept       application/x-www-form-urlencoded
// @Produce      json
// @Param        grant_type     formData string  true  "Grant type (authorization_code, refresh_token, client_credentials, urn:ietf:params:oauth:grant-type:device_code, urn:ietf:params:oauth:grant-type:jwt-bearer, urn:ietf:params:oauth:grant-type:token-exchange)"
// @Param        client_id      formData string  true  "Client identifier"
// @Param        client_secret  formData string  false "Client secret (confidential clients)"
// @Param        code           formData string  false "Authorization code (authorization_code grant)"
// @Param        redirect_uri   formData string  false "Redirect URI (authorization_code grant)"
// @Param        code_verifier  formData string  false "PKCE code verifier (authorization_code grant)"
// @Param        refresh_token  formData string  false "Refresh token (refresh_token grant)"
// @Param        scope          formData string  false "Requested scopes"
// @Param        assertion      formData string  false "JWT assertion (jwt-bearer grant)"
// @Param        subject_token       formData string false "Subject token (token-exchange grant)"
// @Param        subject_token_type  formData string false "Subject token type (token-exchange grant)"
// @Param        actor_token        formData string false "Actor token (token-exchange grant)"
// @Param        actor_token_type   formData string false "Actor token type (token-exchange grant)"
// @Param        resource      formData string false "Resource indicator (RFC 8707)"
// @Param        X-Tenant-ID   header   string true  "Tenant identifier"
// @Param        DPoP          header   string false "DPoP proof JWT (RFC 9449)"
// @Success      200           {object} service.TokenResponse "Token response"
// @Failure      400           {object} map[string]string "invalid_client / unsupported_grant_type / invalid_grant"
// @Router       /oauth/token [post]

// ----------------------------------------------------------------------------
// 3. Revoke — POST /oauth/revoke
// ----------------------------------------------------------------------------

// @Summary      Revoke token
// @Description  Revokes an access or refresh token (RFC 7009). The endpoint always
// @Description  returns HTTP 200 regardless of whether the token was valid, as per
// @Description  the specification.
// @Tags         OAuth2 / Token
// @Accept       application/x-www-form-urlencoded
// @Produce      json
// @Param        token           formData string  true  "The token to revoke"
// @Param        token_type_hint formData string  false "Hint: \"access_token\" or \"refresh_token\""
// @Param        X-Tenant-ID     header  string  false "Tenant identifier"
// @Success      200             "Token revoked (always returns 200 per RFC 7009)"
// @Failure      405             {object} map[string]string "method_not_allowed"
// @Router       /oauth/revoke [post]

// ----------------------------------------------------------------------------
// 4. Introspect — POST /oauth/introspect
// ----------------------------------------------------------------------------

// @Summary      Introspect token
// @Description  Validates a token and returns its metadata (RFC 7662). Requires
// @Description  client authentication via HTTP Basic, form-encoded credentials, or
// @Description  Bearer token. Returns {"active": false} for invalid or expired
// @Description  tokens.
// @Tags         OAuth2 / Introspection
// @Accept       application/x-www-form-urlencoded
// @Produce      json
// @Param        token          formData string  true  "The token to introspect"
// @Param        token_type_hint formData string false "Hint: \"access_token\" or \"refresh_token\""
// @Param        Authorization  header  string  true  "Client credentials (Basic/Bearer)"
// @Success      200            {object} service.IntrospectionResponse "Introspection result"
// @Failure      401            {object} map[string]string "invalid_client"
// @Failure      405            {object} map[string]string "method_not_allowed"
// @Router       /oauth/introspect [post]

// ----------------------------------------------------------------------------
// 5. UserInfo — GET/POST /oauth/userinfo
// ----------------------------------------------------------------------------

// @Summary      UserInfo
// @Description  OIDC UserInfo endpoint (RFC 6749 §5.3). Returns claims about the
// @Description  authenticated end-user derived from the supplied Bearer access
// @Description  token.
// @Tags         OIDC / UserInfo
// @Accept       json
// @Produce      json
// @Param        Authorization header string true "Bearer access_token"
// @Success      200 {object} service.UserInfoResponse "User claims"
// @Failure      401 {object} map[string]string "invalid_token"
// @Failure      405 {object} map[string]string "method_not_allowed"
// @Router       /oauth/userinfo [get]
// @Router       /oauth/userinfo [post]

// ----------------------------------------------------------------------------
// 6. JWKS — GET /.well-known/jwks.json
// ----------------------------------------------------------------------------

// @Summary      JSON Web Key Set
// @Description  Returns the public signing keys used by the authorization server
// @Description  to sign tokens, enabling clients to validate JWT signatures
// @Description  (OIDC Discovery / JWK Set, RFC 7517).
// @Tags         OIDC / Discovery
// @Produce      json
// @Success      200 {object} object "JSON Web Key Set"
// @Router       /.well-known/jwks.json [get]

// ----------------------------------------------------------------------------
// 7. Register — POST /oauth/register (Dynamic Client Registration)
// ----------------------------------------------------------------------------

// @Summary      Dynamic Client Registration
// @Description  Registers a new OAuth 2.0 client dynamically (RFC 7591). Accepts
// @Description  client metadata and returns a generated client_id and
// @Description  client_secret.
// @Tags         OAuth2 / Client Registration
// @Accept       json
// @Produce      json
// @Param        body                 body   service.DynamicRegistrationRequest true "Client metadata"
// @Param        X-Tenant-ID          header string true "Tenant identifier"
// @Success      201                  {object} service.DynamicRegistrationResponse "Registered client"
// @Failure      400                  {object} map[string]string "invalid_request"
// @Failure      405                  {object} map[string]string "method_not_allowed"
// @Router       /oauth/register [post]

// ----------------------------------------------------------------------------
// 8. Consent — GET/POST /oauth/consent
// ----------------------------------------------------------------------------

// @Summary      Consent
// @Description  OAuth consent screen endpoint. GET returns the consent prompt with
// @Description  requested scopes; POST records the user's decision (approve/deny)
// @Description  and redirects back to the authorization flow.
// @Tags         OAuth2 / Consent
// @Accept       json
// @Produce      json
// @Param        client_id    query string true "Client requesting consent"
// @Param        scope        query string true "Requested scopes"
// @Param        redirect_uri query string true "Client redirect URI"
// @Param        state        query string false "OAuth state"
// @Param        decision     formData string false "Consent decision: \"approve\" or \"deny\" (POST)"
// @Success      200          {object} map[string]any "Consent prompt or decision result"
// @Failure      405          {object} map[string]string "method_not_allowed"
// @Router       /oauth/consent [get]
// @Router       /oauth/consent [post]

// ----------------------------------------------------------------------------
// 9. Backchannel Logout — POST /oauth/backchannel-logout
// ----------------------------------------------------------------------------

// @Summary      Back-Channel Logout
// @Description  OIDC Back-Channel Logout endpoint (OIDC Back-Channel Logout 1.0).
// @Description  Accepts a logout_token JWT and revokes all tokens associated with
// @Description  the subject (sub) and/or session (sid).
// @Tags         OIDC / Session Management
// @Accept       application/x-www-form-urlencoded
// @Produce      json
// @Param        logout_token formData string true "Logout token JWT"
// @Success      200           {object} map[string]string "status: logged_out"
// @Failure      400           {object} map[string]string "invalid_logout_token"
// @Failure      405           {object} map[string]string "method_not_allowed"
// @Router       /oauth/backchannel-logout [post]

// ----------------------------------------------------------------------------
// 10. PAR (Pushed Authorization Request) — POST /oauth/par
// ----------------------------------------------------------------------------

// @Summary      Pushed Authorization Request
// @Description  Pushed Authorization Request endpoint (RFC 9126). Accepts
// @Description  authorization request parameters, stores them server-side, and
// @Description  returns a request_uri that the client uses in the subsequent
// @Description  authorize redirect.
// @Tags         OAuth2 / PAR
// @Accept       application/x-www-form-urlencoded
// @Produce      json
// @Param        client_id             formData string true  "Client identifier"
// @Param        client_secret         formData string false "Client secret (confidential clients)"
// @Param        redirect_uri          formData string true  "Client redirect URI"
// @Param        response_type         formData string true  "Must be \"code\""
// @Param        scope                 formData string false "Requested scopes"
// @Param        state                 formData string false "OAuth state"
// @Param        nonce                 formData string false "OIDC nonce"
// @Param        code_challenge        formData string false "PKCE code challenge"
// @Param        code_challenge_method formData string false "PKCE method: \"S256\" or \"plain\""
// @Success      201                   {object} service.PushedAuthorizationResponse "request_uri and expiry"
// @Failure      400                   {object} map[string]string "invalid_request"
// @Failure      405                   {object} map[string]string "method_not_allowed"
// @Router       /oauth/par [post]
