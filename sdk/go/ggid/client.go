// Package ggid provides a Go SDK for the GGID IAM Platform.
//
// It offers JWT verification, user management, RBAC permission checking,
// and HTTP middleware for protecting your Go applications.
//
// Quick start:
//
//	client := ggid.NewClient("https://iam.example.com",
//		ggid.WithTenantID("00000000-0000-0000-0000-000000000001"))
//	tokens, err := client.Login(ctx, "admin", "Admin@123456")
//	users, err := client.ListUsers(ctx, tokens.AccessToken)
package ggid

import (
	"context"
	"errors"
	"net/http"
)

// Version is the SDK version.
const Version = "1.0.0"

// Client is the main GGID API client.
type Client struct {
	gatewayURL string
	tenantID   string
	httpClient *http.Client

	// Optional: JWT verifier for token validation
	verifier *JWTVerifier

	// Credentials for auto-auth (optional)
	username string
	password string
	tokens   *TokenSet
}

// Option configures a Client.
type Option func(*Client)

// WithTenantID sets the tenant ID for all requests.
func WithTenantID(id string) Option {
	return func(c *Client) { c.tenantID = id }
}

// WithHTTPClient sets a custom *http.Client.
func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) { c.httpClient = h }
}

// WithJWKS enables JWT verification using the given JWKS URL.
func WithJWKS(jwksURL string) Option {
	return func(c *Client) {
		c.verifier = NewJWTVerifier(jwksURL)
	}
}

// WithCredentials sets username/password for automatic re-authentication.
func WithCredentials(username, password string) Option {
	return func(c *Client) {
		c.username = username
		c.password = password
	}
}

// NewClient creates a new GGID API client.
func NewClient(gatewayURL string, opts ...Option) *Client {
	c := &Client{
		gatewayURL: gatewayURL,
		tenantID:   "00000000-0000-0000-0000-000000000001",
		httpClient: &http.Client{Timeout: 30000000000}, // 30s
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Errors
var (
	ErrNotAuthenticated = errors.New("not authenticated")
	ErrTokenExpired     = errors.New("token expired")
)

// TokenSet represents the JWT token response from login.
type TokenSet struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	IDToken      string `json:"id_token,omitempty"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

// User represents a GGID user.
type User struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	Email       string `json:"email"`
	Status      string `json:"status"`
	DisplayName string `json:"display_name,omitempty"`
	CreatedAt   string `json:"created_at,omitempty"`
}

// Role represents a GGID role.
type Role struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Key         string `json:"key"`
	Description string `json:"description,omitempty"`
	SystemRole  bool   `json:"system_role"`
}

// PolicyResult represents the result of a permission check.
type PolicyResult struct {
	Allowed bool   `json:"allowed"`
	Reason  string `json:"reason,omitempty"`
}

// ensureContext returns the given context or context.Background if nil.
func ensureContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}
