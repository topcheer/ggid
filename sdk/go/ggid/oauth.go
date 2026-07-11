package ggid

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// --- OIDC Discovery ---

// DiscoveryConfig represents the OpenID Connect discovery document.
type DiscoveryConfig struct {
	Issuer                 string   `json:"issuer"`
	AuthorizationEndpoint  string   `json:"authorization_endpoint"`
	TokenEndpoint          string   `json:"token_endpoint"`
	UserInfoEndpoint       string   `json:"userinfo_endpoint"`
	JwksURI                string   `json:"jwks_uri"`
	IntrospectionEndpoint  string   `json:"introspection_endpoint"`
	RevocationEndpoint     string   `json:"revocation_endpoint"`
	EndSessionEndpoint     string   `json:"end_session_endpoint"`
	RegistrationEndpoint   string   `json:"registration_endpoint"`
	PAREndpoint            string   `json:"pushed_authorization_request_endpoint"`
	DeviceAuthEndpoint     string   `json:"device_authorization_endpoint"`
	ScopesSupported        []string `json:"scopes_supported"`
	ResponseTypesSupported []string `json:"response_types_supported"`
	GrantTypesSupported    []string `json:"grant_types_supported"`
	SubjectTypesSupported  []string `json:"subject_types_supported"`
	IDTokenSigningAlgs     []string `json:"id_token_signing_alg_values_supported"`
	CodeChallengeMethods   []string `json:"code_challenge_methods_supported"`
	ClaimsSupported        []string `json:"claims_supported"`
}

// GetOIDCDiscovery fetches the OpenID Connect discovery document.
func (c *Client) GetOIDCDiscovery(ctx context.Context) (*DiscoveryConfig, error) {
	data, err := c.do(ctx, http.MethodGet, "/.well-known/openid-configuration", nil, "")
	if err != nil {
		return nil, err
	}
	var cfg DiscoveryConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal discovery config: %w", err)
	}
	return &cfg, nil
}

// --- JWKS ---

// JWKS represents a JSON Web Key Set.
type JWKS struct {
	Keys []JWK `json:"keys"`
}

// JWK represents a single JSON Web Key.
type JWK struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	N   string `json:"n,omitempty"`
	E   string `json:"e,omitempty"`
	X   string `json:"x,omitempty"`
	Y   string `json:"y,omitempty"`
	Crv string `json:"crv,omitempty"`
}

// GetJWKS fetches the JSON Web Key Set for token verification.
func (c *Client) GetJWKS(ctx context.Context) (*JWKS, error) {
	data, err := c.do(ctx, http.MethodGet, "/oauth/jwks", nil, "")
	if err != nil {
		return nil, err
	}
	var jwks JWKS
	if err := json.Unmarshal(data, &jwks); err != nil {
		return nil, fmt.Errorf("unmarshal JWKS: %w", err)
	}
	return &jwks, nil
}

// --- OAuth Client Management (RFC 7591 / 7592) ---

// OAuthClient represents an OAuth 2.0 client application.
type OAuthClient struct {
	ClientID            string   `json:"client_id"`
	ClientSecret        string   `json:"client_secret,omitempty"`
	ClientName          string   `json:"client_name"`
	RedirectURIs        []string `json:"redirect_uris"`
	GrantTypes          []string `json:"grant_types"`
	ResponseTypes       []string `json:"response_types"`
	Scope               string   `json:"scope"`
	TokenEndpointMethod string   `json:"token_endpoint_auth_method"`
	LogoURI             string   `json:"logo_uri,omitempty"`
	ClientURI           string   `json:"client_uri,omitempty"`
	PolicyURI           string   `json:"policy_uri,omitempty"`
	TosURI              string   `json:"tos_uri,omitempty"`
}

// RegisterOAuthClient registers a new OAuth client via RFC 7591 dynamic registration.
func (c *Client) RegisterOAuthClient(ctx context.Context, client OAuthClient) (*OAuthClient, error) {
	data, err := c.do(ctx, http.MethodPost, "/api/v1/oauth/register", client, "")
	if err != nil {
		return nil, err
	}
	var result OAuthClient
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("unmarshal oauth client: %w", err)
	}
	return &result, nil
}

// ListOAuthClients lists all OAuth clients for the current tenant.
func (c *Client) ListOAuthClients(ctx context.Context, token string) ([]OAuthClient, error) {
	data, err := c.do(ctx, http.MethodGet, "/api/v1/oauth/clients", nil, token)
	if err != nil {
		return nil, err
	}
	var clients []OAuthClient
	if err := json.Unmarshal(data, &clients); err != nil {
		return nil, fmt.Errorf("unmarshal oauth clients: %w", err)
	}
	return clients, nil
}

// GetOAuthClient retrieves a single OAuth client by ID.
func (c *Client) GetOAuthClient(ctx context.Context, token, clientID string) (*OAuthClient, error) {
	data, err := c.do(ctx, http.MethodGet, "/api/v1/oauth/clients/"+clientID, nil, token)
	if err != nil {
		return nil, err
	}
	var client OAuthClient
	if err := json.Unmarshal(data, &client); err != nil {
		return nil, fmt.Errorf("unmarshal oauth client: %w", err)
	}
	return &client, nil
}

// DeleteOAuthClient removes an OAuth client.
func (c *Client) DeleteOAuthClient(ctx context.Context, token, clientID string) error {
	_, err := c.do(ctx, http.MethodDelete, "/api/v1/oauth/clients/"+clientID, nil, token)
	return err
}

// --- Device Authorization Flow (RFC 8628) ---

// DeviceAuthResponse represents the response from a device authorization request.
type DeviceAuthResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete,omitempty"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

// DeviceAuthorization initiates the device authorization flow (RFC 8628).
func (c *Client) DeviceAuthorization(ctx context.Context, clientID, scope string) (*DeviceAuthResponse, error) {
	form := url.Values{}
	form.Set("client_id", clientID)
	form.Set("scope", scope)

	req, err := http.NewRequestWithContext(ensureContext(ctx), http.MethodPost,
		c.gatewayURL+"/api/v1/oauth/device_authorization", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create device auth request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Tenant-ID", c.tenantID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("device auth request: %w", err)
	}
	defer resp.Body.Close()

	var result DeviceAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode device auth response: %w", err)
	}
	return &result, nil
}

// ApproveDeviceFlow approves a pending device authorization request.
func (c *Client) ApproveDeviceFlow(ctx context.Context, token, userCode string) error {
	body := map[string]string{"user_code": userCode}
	_, err := c.do(ctx, http.MethodPost, "/api/v1/oauth/device/approve", body, token)
	return err
}

// --- Pushed Authorization Request (PAR) ---

// PARResponse represents the response from a pushed authorization request.
type PARResponse struct {
	RequestURI string `json:"request_uri"`
	ExpiresIn  int    `json:"expires_in"`
}

// PushedAuthorizationRequest submits a PAR (RFC 9126) to the server.
func (c *Client) PushedAuthorizationRequest(ctx context.Context, params url.Values) (*PARResponse, error) {
	req, err := http.NewRequestWithContext(ensureContext(ctx), http.MethodPost,
		c.gatewayURL+"/oauth/par", strings.NewReader(params.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create PAR request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Tenant-ID", c.tenantID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("PAR request: %w", err)
	}
	defer resp.Body.Close()

	var result PARResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode PAR response: %w", err)
	}
	return &result, nil
}

// --- UserInfo ---

// UserInfo represents the OpenID Connect UserInfo response.
type UserInfo struct {
	Sub               string `json:"sub"`
	Name              string `json:"name,omitempty"`
	Email             string `json:"email,omitempty"`
	EmailVerified     bool   `json:"email_verified,omitempty"`
	PreferredUsername string `json:"preferred_username,omitempty"`
	Picture           string `json:"picture,omitempty"`
	Zoneinfo          string `json:"zoneinfo,omitempty"`
	Locale            string `json:"locale,omitempty"`
	UpdatedAt         int64  `json:"updated_at,omitempty"`
}

// GetUserInfo fetches the UserInfo for the given access token.
func (c *Client) GetUserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
	data, err := c.do(ctx, http.MethodGet, "/oauth/userinfo", nil, accessToken)
	if err != nil {
		return nil, err
	}
	var info UserInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("unmarshal userinfo: %w", err)
	}
	return &info, nil
}

// --- Token Revocation ---

// RevokeToken revokes an access or refresh token (RFC 7009).
func (c *Client) RevokeToken(ctx context.Context, token string) error {
	_, err := c.do(ctx, http.MethodPost, "/api/v1/oauth/revoke",
		map[string]string{"token": token}, "")
	return err
}

// --- SAML ---

// GetSAMLMetadata fetches the SAML service provider metadata XML.
func (c *Client) GetSAMLMetadata(ctx context.Context) ([]byte, error) {
	data, err := c.do(ctx, http.MethodGet, "/saml/metadata", nil, "")
	if err != nil {
		return nil, err
	}
	return data, nil
}

// --- Introspection Config ---

// IntrospectionConfig represents the introspection cache configuration.
type IntrospectionConfig struct {
	CacheTTL int `json:"cache_ttl_seconds"`
	CacheEnabled bool `json:"cache_enabled"`
}

// GetIntrospectionConfig retrieves the current introspection cache config.
func (c *Client) GetIntrospectionConfig(ctx context.Context, token string) (*IntrospectionConfig, error) {
	data, err := c.do(ctx, http.MethodGet, "/api/v1/oauth/introspection/config", nil, token)
	if err != nil {
		return nil, err
	}
	var cfg IntrospectionConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal introspection config: %w", err)
	}
	return &cfg, nil
}

// UpdateIntrospectionConfig updates the introspection cache config.
func (c *Client) UpdateIntrospectionConfig(ctx context.Context, token string, cfg IntrospectionConfig) error {
	_, err := c.do(ctx, http.MethodPut, "/api/v1/oauth/introspection/config", cfg, token)
	return err
}

// --- Authorize URL Builder ---

// AuthorizeURLOptions configures the authorization URL builder.
type AuthorizeURLOptions struct {
	ClientID            string
	RedirectURI         string
	ResponseType        string // "code" or "token"
	Scope               string // e.g. "openid profile email"
	State               string
	Nonce               string
	CodeChallenge       string
	CodeChallengeMethod string // "S256" or "plain"
}

// GenerateAuthorizeURL builds a full authorization endpoint URL with the given options.
// Use this to redirect users to the GGID login page for interactive OAuth flows.
func (c *Client) GenerateAuthorizeURL(opts AuthorizeURLOptions) string {
	q := url.Values{}
	q.Set("client_id", opts.ClientID)
	q.Set("redirect_uri", opts.RedirectURI)
	q.Set("response_type", opts.ResponseType)
	if opts.Scope != "" {
		q.Set("scope", opts.Scope)
	}
	if opts.State != "" {
		q.Set("state", opts.State)
	}
	if opts.Nonce != "" {
		q.Set("nonce", opts.Nonce)
	}
	if opts.CodeChallenge != "" {
		q.Set("code_challenge", opts.CodeChallenge)
		q.Set("code_challenge_method", opts.CodeChallengeMethod)
	}
	return c.gatewayURL + "/oauth/authorize?" + q.Encode()
}
