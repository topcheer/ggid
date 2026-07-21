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
	ClientName               string   `json:"client_name"`
	GrantTypes               []string `json:"grant_types"`
	ResponseTypes            []string `json:"response_types"`
	TokenEndpointAuthMethod  string   `json:"token_endpoint_auth_method"`
	Scope                    string   `json:"scope"`
}

// DCRResponse represents a Dynamic Client Registration response.
type DCRResponse struct {
	ClientID                 string   `json:"client_id"`
	ClientSecret             string   `json:"client_secret"`
	ClientIDIssuedAt         int64    `json:"client_id_issued_at"`
	ClientSecretExpiresAt    int64    `json:"client_secret_expires_at"`
	ClientName               string   `json:"client_name"`
	GrantTypes               []string `json:"grant_types"`
	ResponseTypes            []string `json:"response_types"`
	TokenEndpointAuthMethod  string   `json:"token_endpoint_auth_method"`
	Scope                    string   `json:"scope"`
}

// TokenResponse represents an OAuth2 token response.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

// RegisterViaDCR registers the CLI as an OAuth client using Dynamic Client
// Registration (RFC 7591) in the console tenant. Returns the client ID and secret.
func RegisterViaDCR(baseURL, tenantID, clientName string) (*DCRResponse, error) {
	c := New(baseURL, tenantID, "")
	req := &DCRRequest{
		ClientName:              clientName,
		GrantTypes:              []string{"client_credentials", "refresh_token"},
		ResponseTypes:           []string{},
		TokenEndpointAuthMethod: "client_secret_basic",
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

// GetClientCredentialsToken exchanges client credentials for an access token
// using the client_credentials grant type.
func GetClientCredentialsToken(baseURL, tenantID, clientID, clientSecret string) (*TokenResponse, error) {
	c := New(baseURL, tenantID, "")
	form := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {clientID},
		"client_secret": {clientSecret},
	}
	var resp TokenResponse
	if err := c.PostForm("/api/v1/oauth/token", form, &resp); err != nil {
		return nil, fmt.Errorf("token exchange failed: %w", err)
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
	// Consider token expired 30 seconds before actual expiry to avoid race.
	return time.Now().Unix() >= (expiresAt - 30)
}
