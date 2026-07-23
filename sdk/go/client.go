// Package ggid provides a Go SDK for integrating with the GGID IAM platform.
//
// It offers both server-side management operations (user, role, org CRUD)
// and client-side authentication helpers (JWT verification, middleware).
package ggid

// Version is the current SDK version.
const Version = "1.0.0"

import (
	"bytes"
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Client is the GGID SDK client.
type Client struct {
	baseURL    string
	httpClient *http.Client
	apiKey     string
	jwksURL    string
	useDiscovery bool

	// JWKS cache for JWT signature verification.
	jwksMu      sync.RWMutex
	jwks        map[string]*rsa.PublicKey
	jwksExpiry  time.Time
	jwksTTL     time.Duration
}

// Option configures the Client.
type Option func(*Client)

// WithAPIKey sets the server-side API key for management operations.
func WithAPIKey(key string) Option {
	return func(c *Client) { c.apiKey = key }
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) { c.httpClient = hc }
}

// WithJWKS enables JWT signature verification using the JWKS endpoint.
// The keys are cached for the given TTL (recommended 15m).
func WithJWKS(ttl time.Duration) Option {
	return func(c *Client) {
		c.jwksURL = "/.well-known/jwks.json"
		c.jwksTTL = ttl
	}
}

// OIDCDiscovery holds the OpenID Connect discovery document.
type OIDCDiscovery struct {
	Issuer                 string   `json:"issuer"`
	AuthorizationEndpoint  string   `json:"authorization_endpoint"`
	TokenEndpoint          string   `json:"token_endpoint"`
	UserInfoEndpoint       string   `json:"userinfo_endpoint"`
	JwksURI                string   `json:"jwks_uri"`
	IntrospectionEndpoint  string   `json:"introspection_endpoint"`
	DeviceAuthEndpoint     string   `json:"device_authorization_endpoint,omitempty"`
	GrantTypesSupported    []string `json:"grant_types_supported"`
}

// GetDiscovery fetches the OIDC discovery document.
func (c *Client) GetDiscovery(ctx context.Context) (*OIDCDiscovery, error) {
	var d OIDCDiscovery
	if err := c.get(ctx, "/.well-known/openid-configuration", nil, &d); err != nil {
		return nil, err
	}
	return &d, nil
}

// WithDiscovery auto-configures from OIDC discovery — no hardcoded URLs needed.
// The SDK fetches /.well-known/openid-configuration on first use to get
// jwks_uri, issuer, token_endpoint, etc. Just provide baseURL.
func WithDiscovery() Option {
	return func(c *Client) {
		c.useDiscovery = true
	}
}

// New creates a new GGID SDK client.
func New(baseURL string, opts ...Option) *Client {
	c := &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				// Disable automatic gzip decompression — gateway returns
				// compressed JWKS responses that cause unexpected EOF.
				DisableCompression: true,
			},
		},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

// UserInfo represents the authenticated user information extracted from a JWT.
type UserInfo struct {
	UserID      string         `json:"user_id"`
	TenantID    string         `json:"tenant_id"`
	Username    string         `json:"username"`
	Email       string         `json:"email"`
	Roles       []string       `json:"roles"`
	Scopes      []string       `json:"scopes"`       // OAuth scopes (openid, profile, email)
	Permissions []string       `json:"permissions"` // Fine-grained permissions (inventory:read)
	Claims      map[string]any `json:"claims,omitempty"`
}

// CreateUserRequest holds parameters for creating a user.
type CreateUserRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Phone    string `json:"phone,omitempty"`
}

// UpdateUserRequest holds optional fields for updating a user.
type UpdateUserRequest struct {
	Email  *string `json:"email,omitempty"`
	Phone  *string `json:"phone,omitempty"`
	Status *string `json:"status,omitempty"`
}

// User represents a user in GGID.
type User struct {
	ID            string    `json:"id"`
	TenantID      string    `json:"tenant_id"`
	Username      string    `json:"username"`
	Email         string    `json:"email"`
	Phone         string    `json:"phone"`
	Status        string    `json:"status"`
	EmailVerified bool      `json:"email_verified"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// Role represents a role with associated permissions.
type Role struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	Key         string    `json:"key"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// CreateRoleRequest holds parameters for creating a role.
type CreateRoleRequest struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// Organization represents an organizational unit.
type Organization struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	Name        string    `json:"name"`
	ParentID    string    `json:"parent_id,omitempty"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// CreateOrgRequest holds parameters for creating an organization.
type CreateOrgRequest struct {
	Name        string `json:"name"`
	ParentID    string `json:"parent_id,omitempty"`
	Description string `json:"description,omitempty"`
}

// PageResult holds a paginated result set.
type PageResult[T any] struct {
	Items      []T `json:"items"`
	TotalCount int `json:"total_count"`
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
}

// ListOptions controls pagination and filtering.
type ListOptions struct {
	Page     int    `json:"page,omitempty"`
	PageSize int    `json:"page_size,omitempty"`
	Search   string `json:"search,omitempty"`
	Status   string `json:"status,omitempty"`
}

// TokenSet holds a token response.
type TokenSet struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token,omitempty"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope,omitempty"`
}

// LoginRequest holds credentials for password login.
type LoginRequest struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	ClientID   string `json:"client_id,omitempty"`
	TenantID   string `json:"tenant_id,omitempty"`
	TenantSlug string `json:"tenant_slug,omitempty"`}

// APIError represents a structured error returned by the GGID API.
type APIError struct {
	StatusCode int
	Code       string `json:"code,omitempty"`
	Message    string `json:"message,omitempty"`
	Detail     any    `json:"detail,omitempty"`
}

func (e *APIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("ggid: %s (status %d): %s", e.Code, e.StatusCode, e.Message)
	}
	return fmt.Sprintf("ggid: API error (status %d): %s", e.StatusCode, e.Message)
}

// IsNotFound returns true if the error is a 404 Not Found.
func (e *APIError) IsNotFound() bool { return e.StatusCode == http.StatusNotFound }

// IsUnauthorized returns true if the error is a 401.
func (e *APIError) IsUnauthorized() bool { return e.StatusCode == http.StatusUnauthorized }

// IsForbidden returns true if the error is a 403.
func (e *APIError) IsForbidden() bool { return e.StatusCode == http.StatusForbidden }

// IsConflict returns true if the error is a 409.
func (e *APIError) IsConflict() bool { return e.StatusCode == http.StatusConflict }

// IsRateLimited returns true if the error is a 429.
func (e *APIError) IsRateLimited() bool { return e.StatusCode == http.StatusTooManyRequests }

// ---------------------------------------------------------------------------
// Auth operations
// ---------------------------------------------------------------------------

// Login authenticates via OAuth2 password grant.
func (c *Client) Login(ctx context.Context, req *LoginRequest) (*TokenSet, error) {
	form := url.Values{
		"grant_type": {"password"},
		"username":  {req.Username},
		"password":  {req.Password},
		"client_id": {req.ClientID},
	}
	postReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/api/v1/oauth/token", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if req.TenantID != "" {
		postReq.Header.Set("X-Tenant-ID", req.TenantID)
	}
	var ts TokenSet
	if err := c.do(postReq, &ts); err != nil {
		return nil, err
	}
	return &ts, nil
}

// Logout invalidates the given access token.
func (c *Client) Logout(ctx context.Context, accessToken string) error {
	return c.post(ctx, "/api/v1/auth/logout", map[string]string{"access_token": accessToken}, nil)
}

// ExchangeAgentToken exchanges a subject token for an agent token (RFC 8693).
func (c *Client) ExchangeAgentToken(ctx context.Context, subjectToken, grantType, audience string) (*TokenSet, error) {
	form := url.Values{
		"grant_type":    {grantType},
		"subject_token": {subjectToken},
	}
	if audience != "" {
		form.Set("audience", audience)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/api/v1/oauth/token", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	var ts TokenSet
	if err := c.do(req, &ts); err != nil {
		return nil, err
	}
	return &ts, nil
}

// ExchangeSAMLToken exchanges a SAML assertion for an access token (RFC 7522).
func (c *Client) ExchangeSAMLToken(ctx context.Context, samlResponse, clientID string) (*TokenSet, error) {
	form := url.Values{
		"grant_type": {"urn:ietf:params:oauth:grant-type:saml2-bearer"},
		"assertion": {samlResponse},
	}
	if clientID != "" {
		form.Set("client_id", clientID)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/api/v1/oauth/token", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	var ts TokenSet
	if err := c.do(req, &ts); err != nil {
		return nil, err
	}
	return &ts, nil
}

// ClientCredentials exchanges client credentials for an access token (RFC 6749 §4.4).
// Used for machine-to-machine (M2M) authentication.
func (c *Client) ClientCredentials(ctx context.Context, clientID, clientSecret, tenantID string, scopes []string) (*TokenSet, error) {
	form := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {clientID},
		"client_secret": {clientSecret},
	}
	if tenantID != "" {
		form.Set("X-Tenant-ID", tenantID)
	}
	if len(scopes) > 0 {
		form.Set("scope", strings.Join(scopes, " "))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/api/v1/oauth/token", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if tenantID != "" {
		req.Header.Set("X-Tenant-ID", tenantID)
	}

	var ts TokenSet
	if err := c.do(req, &ts); err != nil {
		return nil, err
	}
	return &ts, nil
}

// GetAuthorizeURL builds the OAuth2 authorization endpoint URL with PKCE.
// The client redirects the user's browser to this URL to begin the auth flow.
// After login, GGID redirects back to redirectURI with an authorization code.
func (c *Client) GetAuthorizeURL(clientID, redirectURI, tenantID string, opts ...AuthorizeOpt) string {
	o := &AuthorizeConfig{
		Scope:          "openid profile email",
		ResponseType:   "code",
		CodeChallengeMethod: "S256",
	}
	for _, opt := range opts {
		opt(o)
	}

	params := url.Values{
		"response_type":         {o.ResponseType},
		"client_id":             {clientID},
		"redirect_uri":          {redirectURI},
		"scope":                 {o.Scope},
		"state":                 {o.State},
		"code_challenge_method": {o.CodeChallengeMethod},
		"tenant_id":             {tenantID},
	}
	if o.CodeChallenge != "" {
		params.Set("code_challenge", o.CodeChallenge)
	}
	return c.baseURL + "/api/v1/oauth/authorize?" + params.Encode()
}

// ExchangeCode exchanges an authorization code for a token set (OAuth2 §4.1).
// Used after the redirect from GetAuthorizeURL. Pass the PKCE code_verifier
// that corresponds to the code_challenge used in GetAuthorizeURL.
func (c *Client) ExchangeCode(ctx context.Context, code, redirectURI, clientID, codeVerifier, tenantID string) (*TokenSet, error) {
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"client_id":     {clientID},
	}
	if codeVerifier != "" {
		form.Set("code_verifier", codeVerifier)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/api/v1/oauth/token", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if tenantID != "" {
		req.Header.Set("X-Tenant-ID", tenantID)
	}

	var ts TokenSet
	if err := c.do(req, &ts); err != nil {
		return nil, err
	}
	return &ts, nil
}

// AuthorizeConfig holds optional parameters for the authorize URL.
type AuthorizeConfig struct {
	Scope              string
	State              string
	ResponseType       string
	CodeChallenge      string
	CodeChallengeMethod string
}

// AuthorizeOpt configures optional authorize parameters.
type AuthorizeOpt func(*AuthorizeConfig)

// WithScope sets the OAuth2 scope.
func WithScope(scope string) AuthorizeOpt {
	return func(a *AuthorizeConfig) { a.Scope = scope }
}

// WithState sets the OAuth2 state parameter (CSRF protection).
func WithState(state string) AuthorizeOpt {
	return func(a *AuthorizeConfig) { a.State = state }
}

// WithCodeChallenge sets the PKCE code_challenge (S256 hashed verifier).
func WithCodeChallenge(challenge string) AuthorizeOpt {
	return func(a *AuthorizeConfig) { a.CodeChallenge = challenge }
}

// RefreshToken refreshes an access token using a refresh token.
func (c *Client) RefreshToken(ctx context.Context, refreshToken string) (*TokenSet, error) {
	body := map[string]string{"refresh_token": refreshToken}
	var ts TokenSet
	if err := c.post(ctx, "/api/v1/auth/refresh", body, &ts); err != nil {
		return nil, err
	}
	return &ts, nil
}

// VerifyToken validates an access token and returns user info.
// Requires JWKS to be configured via WithJWKS(). Without JWKS, signature
// verification is impossible and the call returns an error.
func (c *Client) VerifyToken(ctx context.Context, accessToken string) (*UserInfo, error) {
	// Auto-configure from OIDC discovery if enabled
	if c.useDiscovery && c.jwksURL == "" {
		disc, err := c.GetDiscovery(ctx)
		if err != nil {
			return nil, fmt.Errorf("OIDC discovery failed: %w", err)
		}
		if disc.JwksURI != "" {
			// Convert absolute URL to relative path for c.get()
			if strings.HasPrefix(disc.JwksURI, c.baseURL) {
				c.jwksURL = strings.TrimPrefix(disc.JwksURI, c.baseURL)
			} else {
				c.jwksURL = disc.JwksURI
			}
		}
		if c.jwksTTL == 0 {
			c.jwksTTL = 15 * time.Minute
		}
	}
	if c.jwksURL == "" {
		return nil, fmt.Errorf("JWKS not configured: call WithJWKS() or WithDiscovery() to enable")
	}
	return c.verifyTokenOnline(ctx, accessToken)
}

func (c *Client) verifyTokenOnline(ctx context.Context, accessToken string) (*UserInfo, error) {
	// Parse header to get kid
	parser := jwt.NewParser()
	unverified, _, err := parser.ParseUnverified(accessToken, jwt.MapClaims{})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token header: %w", err)
	}

	kid := ""
	if unverified.Header != nil {
		if v, ok := unverified.Header["kid"].(string); ok {
			kid = v
		}
	}

	pubKey, err := c.getJWKSPublicKey(ctx, kid)
	if err != nil {
		return nil, fmt.Errorf("failed to get signing key: %w", err)
	}

	token, err := jwt.Parse(accessToken, func(t *jwt.Token) (any, error) {
		if !IsSupportedAlg(t.Method.Alg()) {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return pubKey, nil
	})
	if err != nil {
		return nil, fmt.Errorf("token verification failed: %w", err)
	}
	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claimsToUserInfo(token)
}

// ---------------------------------------------------------------------------
// User management
// ---------------------------------------------------------------------------

// CreateUser creates a new user (requires API key).
func (c *Client) CreateUser(ctx context.Context, req *CreateUserRequest) (*User, error) {
	var u User
	if err := c.post(ctx, "/api/v1/users", req, &u); err != nil {
		return nil, err
	}
	return &u, nil
}

// GetUser retrieves a user by ID.
func (c *Client) GetUser(ctx context.Context, userID string) (*User, error) {
	var u User
	if err := c.get(ctx, "/api/v1/users/"+userID, nil, &u); err != nil {
		return nil, err
	}
	return &u, nil
}

// UpdateUser updates mutable fields of a user.
func (c *Client) UpdateUser(ctx context.Context, userID string, req *UpdateUserRequest) (*User, error) {
	var u User
	if err := c.patch(ctx, "/api/v1/users/"+userID, req, &u); err != nil {
		return nil, err
	}
	return &u, nil
}

// DeleteUser deletes a user by ID.
func (c *Client) DeleteUser(ctx context.Context, userID string) error {
	return c.del(ctx, "/api/v1/users/"+userID)
}

// ListUsers lists users with pagination.
func (c *Client) ListUsers(ctx context.Context, opts *ListOptions) (*PageResult[User], error) {
	var result PageResult[User]
	if err := c.get(ctx, "/api/v1/users", optsToParams(opts), &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// AssignRole assigns a role to a user.
func (c *Client) AssignRole(ctx context.Context, userID, roleID string) error {
	return c.post(ctx, fmt.Sprintf("/api/v1/users/%s/roles", userID), map[string]string{"role_id": roleID}, nil)
}

// RemoveRole removes a role from a user.
func (c *Client) RemoveRole(ctx context.Context, userID, roleID string) error {
	return c.del(ctx, fmt.Sprintf("/api/v1/users/%s/roles/%s", userID, roleID))
}

// ---------------------------------------------------------------------------
// Role management
// ---------------------------------------------------------------------------

// CreateRole creates a new role (requires API key).
func (c *Client) CreateRole(ctx context.Context, req *CreateRoleRequest) (*Role, error) {
	var r Role
	if err := c.post(ctx, "/api/v1/roles", req, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

// ListRoles lists roles with pagination.
func (c *Client) ListRoles(ctx context.Context, opts *ListOptions) (*PageResult[Role], error) {
	var result PageResult[Role]
	if err := c.get(ctx, "/api/v1/roles", optsToParams(opts), &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CheckPermission checks if a user has permission for an action on a resource.
func (c *Client) CheckPermission(ctx context.Context, userID, resource, action string) (bool, error) {
	body := map[string]string{
		"user_id":  userID,
		"resource": resource,
		"action":   action,
	}
	var resp struct {
		Allowed bool `json:"allowed"`
	}
	if err := c.post(ctx, "/api/v1/policies/check", body, &resp); err != nil {
		return false, err
	}
	return resp.Allowed, nil
}

// ---------------------------------------------------------------------------
// Organization management
// ---------------------------------------------------------------------------

// CreateOrg creates a new organization.
func (c *Client) CreateOrg(ctx context.Context, req *CreateOrgRequest) (*Organization, error) {
	var o Organization
	if err := c.post(ctx, "/api/v1/organizations", req, &o); err != nil {
		return nil, err
	}
	return &o, nil
}

// ListOrgs lists organizations with pagination.
func (c *Client) ListOrgs(ctx context.Context, opts *ListOptions) (*PageResult[Organization], error) {
	var result PageResult[Organization]
	if err := c.get(ctx, "/api/v1/organizations", optsToParams(opts), &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ---------------------------------------------------------------------------
// JWKS cache
// ---------------------------------------------------------------------------

type jwksResponse struct {
	Keys []struct {
		Kid string `json:"kid"`
		Kty string `json:"kty"`
		N   string `json:"n"`
		E   string `json:"e"`
	} `json:"keys"`
}

func (c *Client) getJWKSPublicKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	c.jwksMu.RLock()
	if c.jwks != nil && time.Now().Before(c.jwksExpiry) {
		if key, ok := c.jwks[kid]; ok {
			c.jwksMu.RUnlock()
			return key, nil
		}
	}
	c.jwksMu.RUnlock()

	// Fetch fresh JWKS.
	var resp jwksResponse
	if err := c.get(ctx, c.jwksURL, nil, &resp); err != nil {
		return nil, err
	}

	c.jwksMu.Lock()
	defer c.jwksMu.Unlock()

	c.jwks = make(map[string]*rsa.PublicKey, len(resp.Keys))
	for _, key := range resp.Keys {
		if key.Kty != "RSA" {
			continue
		}
		pub, err := jwkToRSAPublicKey(key.N, key.E)
		if err != nil {
			continue
		}
		c.jwks[key.Kid] = pub
	}
	c.jwksExpiry = time.Now().Add(c.jwksTTL)

	if key, ok := c.jwks[kid]; ok {
		return key, nil
	}
	return nil, fmt.Errorf("key ID %q not found in JWKS", kid)
}

func jwkToRSAPublicKey(nB64, eB64 string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(nB64)
	if err != nil {
		return nil, fmt.Errorf("decode n: %w", err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(eB64)
	if err != nil {
		return nil, fmt.Errorf("decode e: %w", err)
	}

	exponent := 0
	for _, b := range eBytes {
		exponent = exponent<<8 + int(b)
	}

	modulus := new(big.Int).SetBytes(nBytes)

	return &rsa.PublicKey{
		N: modulus,
		E: exponent,
	}, nil
}

// ---------------------------------------------------------------------------
// HTTP helpers
// ---------------------------------------------------------------------------

func (c *Client) get(ctx context.Context, path string, params map[string]string, out any) error {
	u := c.baseURL + path
	if len(params) > 0 {
		v := url.Values{}
		for k, val := range params {
			v.Set(k, val)
		}
		u += "?" + v.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	return c.do(req, out)
}

func (c *Client) post(ctx context.Context, path string, body any, out any) error {
	return c.sendWithBody(ctx, http.MethodPost, path, body, out)
}

func (c *Client) patch(ctx context.Context, path string, body any, out any) error {
	return c.sendWithBody(ctx, http.MethodPatch, path, body, out)
}

func (c *Client) put(ctx context.Context, path string, body any, out any) error {
	return c.sendWithBody(ctx, http.MethodPut, path, body, out)
}

func (c *Client) del(ctx context.Context, path string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	return c.do(req, nil)
}

func (c *Client) sendWithBody(ctx context.Context, method, path string, body any, out any) error {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return c.do(req, out)
}

func (c *Client) do(req *http.Request, out any) error {
	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		apiErr := &APIError{StatusCode: resp.StatusCode}
		// Try to parse structured error.
		_ = json.Unmarshal(raw, apiErr)
		if apiErr.Message == "" {
			apiErr.Message = string(raw)
		}
		return apiErr
	}

	if out != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			return fmt.Errorf("unmarshal response: %w", err)
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func optsToParams(opts *ListOptions) map[string]string {
	if opts == nil {
		return nil
	}
	params := map[string]string{}
	if opts.Page > 0 {
		params["page"] = fmt.Sprintf("%d", opts.Page)
	}
	if opts.PageSize > 0 {
		params["page_size"] = fmt.Sprintf("%d", opts.PageSize)
	}
	if opts.Search != "" {
		params["search"] = opts.Search
	}
	if opts.Status != "" {
		params["status"] = opts.Status
	}
	return params
}

func claimsToUserInfo(token *jwt.Token) (*UserInfo, error) {
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid claims type")
	}

	info := &UserInfo{
		UserID:   getString(claims, "sub"),
		Username: getString(claims, "username"),
		Email:    getString(claims, "email"),
		Claims:   claims,
	}
	if tid, ok := claims["tenant_id"]; ok {
		info.TenantID = fmt.Sprintf("%v", tid)
	}
	if roles, ok := claims["roles"].([]any); ok {
		for _, r := range roles {
			info.Roles = append(info.Roles, fmt.Sprintf("%v", r))
		}
	}
	if scopes, ok := claims["scope"].(string); ok {
		info.Scopes = strings.Split(scopes, " ")
	}
	// Extract fine-grained permissions claim
	if perms, ok := claims["permissions"].([]any); ok {
		for _, p := range perms {
			info.Permissions = append(info.Permissions, fmt.Sprintf("%v", p))
		}
	}
	return info, nil
}

func getString(claims jwt.MapClaims, key string) string {
	if v, ok := claims[key]; ok {
		return fmt.Sprintf("%v", v)
	}
	return ""
}
