// Package ggid provides a Go SDK for integrating with the GGID IAM platform.
//
// It offers both server-side management operations (user, role, org CRUD)
// and client-side authentication helpers (JWT verification, middleware).
package ggid

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

// New creates a new GGID SDK client.
func New(baseURL string, opts ...Option) *Client {
	c := &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{Timeout: 30 * time.Second},
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
	UserID   string         `json:"user_id"`
	TenantID string         `json:"tenant_id"`
	Username string         `json:"username"`
	Email    string         `json:"email"`
	Roles    []string       `json:"roles"`
	Scopes   []string       `json:"scopes"`
	Claims   map[string]any `json:"claims,omitempty"`
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
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// LoginRequest holds credentials for password login.
type LoginRequest struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	TenantID   string `json:"tenant_id,omitempty"`
	TenantSlug string `json:"tenant_slug,omitempty"`
}

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

// Login authenticates a user with username/password and returns tokens.
func (c *Client) Login(ctx context.Context, req *LoginRequest) (*TokenSet, error) {
	var ts TokenSet
	if err := c.post(ctx, "/api/v1/auth/login", req, &ts); err != nil {
		return nil, err
	}
	return &ts, nil
}

// Logout invalidates the given access token.
func (c *Client) Logout(ctx context.Context, accessToken string) error {
	return c.post(ctx, "/api/v1/auth/logout", map[string]string{"access_token": accessToken}, nil)
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
// When JWKS is configured (WithJWKS), the JWT signature is verified;
// otherwise claims are parsed without signature verification (offline mode).
func (c *Client) VerifyToken(ctx context.Context, accessToken string) (*UserInfo, error) {
	if c.jwksURL != "" {
		return c.verifyTokenOnline(ctx, accessToken)
	}
	return c.verifyTokenOffline(accessToken)
}

func (c *Client) verifyTokenOffline(accessToken string) (*UserInfo, error) {
	parser := jwt.NewParser()
	token, _, err := parser.ParseUnverified(accessToken, jwt.MapClaims{})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}
	return claimsToUserInfo(token)
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
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
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
	return info, nil
}

func getString(claims jwt.MapClaims, key string) string {
	if v, ok := claims[key]; ok {
		return fmt.Sprintf("%v", v)
	}
	return ""
}
