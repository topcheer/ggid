package client

import (
	"fmt"
	"net/url"
	"time"
)

// ConsoleTenantID is the default console tenant UUID used for DCR.
const ConsoleTenantID = "00000000-0000-0000-0000-000000000001"

// DCRRequest represents a Dynamic Client Registration request (RFC 7591).
type DCRRequest struct {
	ClientName              string   `json:"client_name"`
	GrantTypes              []string `json:"grant_types"`
	ResponseTypes           []string `json:"response_types"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
	Scope                   string   `json:"scope"`
}

// DCRResponse represents a Dynamic Client Registration response.
type DCRResponse struct {
	ClientID                string   `json:"client_id"`
	ClientSecret            string   `json:"client_secret"`
	ClientIDIssuedAt        int64    `json:"client_id_issued_at"`
	ClientSecretExpiresAt   int64    `json:"client_secret_expires_at"`
	ClientName              string   `json:"client_name"`
	GrantTypes              []string `json:"grant_types"`
	ResponseTypes           []string `json:"response_types"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
	Scope                   string   `json:"scope"`
}

// TokenResponse represents an OAuth2 token response.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

// DeviceAuthResponse represents a device authorization response (RFC 8628).
type DeviceAuthResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete,omitempty"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

// DeviceTokenPollResult represents the result of polling the device token endpoint.
type DeviceTokenPollResult struct {
	Token     *TokenResponse
	Pending   bool   // true if authorization is still pending
	SlowDown  bool   // true if polling too fast
	Error     string // error message if not pending and no token
}

// RegisterViaDCR registers the CLI as a public OAuth client (no secret) using
// Dynamic Client Registration (RFC 7591) in the console tenant.
// The client is configured for device_code grant — a user-interactive flow that
// produces tokens with real user identity (sub, scopes, session).
func RegisterViaDCR(baseURL, tenantID, clientName string) (*DCRResponse, error) {
	c := New(baseURL, tenantID, "")
	req := &DCRRequest{
		ClientName:              clientName,
		GrantTypes:              []string{"urn:ietf:params:oauth:grant-type:device_code", "refresh_token"},
		ResponseTypes:           []string{},
		TokenEndpointAuthMethod: "none", // public client — no secret needed
		Scope:                   "openid profile email users:read users:write roles:read roles:write orgs:read orgs:write audit:read policies:read policies:write oauth:read oauth:write settings:read settings:write tenants:read tenants:write webhooks:read webhooks:write apikeys:read apikeys:write security:read security:write governance:read provisioning:read provisioning:write identity:read identity:write",
	}

	var resp DCRResponse
	if err := c.Post("/api/v1/oauth/register", req, &resp); err != nil {
		return nil, fmt.Errorf("DCR registration failed: %w", err)
	}
	if resp.ClientID == "" {
		return nil, fmt.Errorf("DCR registration returned empty client_id")
	}
	return &resp, nil
}

// RequestDeviceAuthorization initiates a device authorization flow (RFC 8628).
// The user must visit verification_uri and enter the user_code to authorize.
func RequestDeviceAuthorization(baseURL, tenantID, clientID string, scopes string) (*DeviceAuthResponse, error) {
	c := New(baseURL, tenantID, "")
	form := url.Values{
		"client_id": {clientID},
		"scope":     {scopes},
	}
	var resp DeviceAuthResponse
	if err := c.PostForm("/api/v1/oauth/device_authorization", form, &resp); err != nil {
		return nil, fmt.Errorf("device authorization request failed: %w", err)
	}
	if resp.DeviceCode == "" {
		return nil, fmt.Errorf("device authorization returned empty device_code")
	}
	return &resp, nil
}

// PollDeviceToken polls the token endpoint for a device code grant.
// Returns the token if the user has authorized, or a pending/slow_down status.
func PollDeviceToken(baseURL, tenantID, clientID, deviceCode string) (*DeviceTokenPollResult, error) {
	c := New(baseURL, tenantID, "")
	form := url.Values{
		"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
		"client_id":   {clientID},
		"device_code": {deviceCode},
	}

	var tokenResp TokenResponse
	err := c.PostForm("/api/v1/oauth/token", form, &tokenResp)
	if err == nil && tokenResp.AccessToken != "" {
		return &DeviceTokenPollResult{Token: &tokenResp}, nil
	}

	// Parse error from the API response.
	if apiErr, ok := err.(*APIError); ok {
		switch {
		case contains(apiErr.Body, "authorization_pending"):
			return &DeviceTokenPollResult{Pending: true}, nil
		case contains(apiErr.Body, "slow_down"):
			return &DeviceTokenPollResult{SlowDown: true}, nil
		case contains(apiErr.Body, "expired_token"):
			return &DeviceTokenPollResult{Error: "device code expired"}, nil
		case contains(apiErr.Body, "access_denied"):
			return &DeviceTokenPollResult{Error: "user denied authorization"}, nil
		default:
			return &DeviceTokenPollResult{Error: apiErr.Message}, nil
		}
	}
	if err != nil {
		return nil, fmt.Errorf("token poll failed: %w", err)
	}
	return &DeviceTokenPollResult{Error: "unexpected empty response"}, nil
}

// RefreshToken exchanges a refresh token for a new access token.
func RefreshToken(baseURL, tenantID, clientID, refreshToken string) (*TokenResponse, error) {
	c := New(baseURL, tenantID, "")
	form := url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {clientID},
		"refresh_token": {refreshToken},
	}
	var resp TokenResponse
	if err := c.PostForm("/api/v1/oauth/token", form, &resp); err != nil {
		return nil, fmt.Errorf("token refresh failed: %w", err)
	}
	if resp.AccessToken == "" {
		return nil, fmt.Errorf("token endpoint returned empty access_token")
	}
	return &resp, nil
}

// IsTokenExpired returns true if the token expiry has passed.
func IsTokenExpired(expiresAt int64) bool {
	if expiresAt == 0 {
		return true
	}
	// Consider token expired 60 seconds before actual expiry to allow refresh.
	return time.Now().Unix() >= (expiresAt - 60)
}

// contains is a simple substring check (avoids importing strings in client package).
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsFold(s, substr))
}

func containsFold(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			a, b := s[i+j], substr[j]
			if a >= 'A' && a <= 'Z' {
				a += 32
			}
			if b >= 'A' && b <= 'Z' {
				b += 32
			}
			if a != b {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
