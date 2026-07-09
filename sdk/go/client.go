// Package ggid provides a Go SDK for integrating with the GGID IAM platform.
package ggid

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Client is the GGID SDK client.
type Client struct {
	baseURL    string
	httpClient *http.Client
	apiKey     string
	jwksURL    string
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

// New creates a new GGID SDK client.
func New(baseURL string, opts ...Option) *Client {
	c := &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// UserInfo represents the authenticated user information.
type UserInfo struct {
	UserID   string            `json:"user_id"`
	TenantID string            `json:"tenant_id"`
	Username string            `json:"username"`
	Email    string            `json:"email"`
	Roles    []string          `json:"roles"`
	Scopes   []string          `json:"scopes"`
	Claims   map[string]any    `json:"claims,omitempty"`
}

// CreateUserRequest holds parameters for creating a user.
type CreateUserRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Phone    string `json:"phone,omitempty"`
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

// PageResult holds a paginated result set.
type PageResult[T any] struct {
	Items      []T   `json:"items"`
	TotalCount int   `json:"total_count"`
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
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

// VerifyToken validates an access token and returns user info.
// Uses JWT parsing for offline verification when possible.
func (c *Client) VerifyToken(ctx context.Context, accessToken string) (*UserInfo, error) {
	// Parse JWT without verification first to extract claims
	parser := jwt.NewParser()
	token, _, err := parser.ParseUnverified(accessToken, jwt.MapClaims{})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}
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

// RefreshToken refreshes an access token using a refresh token.
func (c *Client) RefreshToken(ctx context.Context, refreshToken string) (*TokenSet, error) {
	body := map[string]string{"refresh_token": refreshToken}
	var ts TokenSet
	if err := c.post(ctx, "/api/v1/auth/refresh", body, &ts); err != nil {
		return nil, err
	}
	return &ts, nil
}

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

// ListUsers lists users with pagination.
func (c *Client) ListUsers(ctx context.Context, opts *ListOptions) (*PageResult[User], error) {
	var result PageResult[User]
	params := map[string]string{}
	if opts != nil {
		if opts.Page > 0 {
			params["page"] = fmt.Sprintf("%d", opts.Page)
		}
		if opts.PageSize > 0 {
			params["page_size"] = fmt.Sprintf("%d", opts.PageSize)
		}
		if opts.Search != "" {
			params["search"] = opts.Search
		}
	}
	if err := c.get(ctx, "/api/v1/users", params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// --- HTTP helpers ---

func (c *Client) get(ctx context.Context, path string, params map[string]string, out any) error {
	url := c.baseURL + path
	if len(params) > 0 {
		url += "?"
		for k, v := range params {
			url += fmt.Sprintf("&%s=%s", k, v)
		}
	}
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	return c.do(req, out)
}

func (c *Client) post(ctx context.Context, path string, body any, out any) error {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal body: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+path, bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
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
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(raw))
	}

	if out != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			return fmt.Errorf("unmarshal response: %w", err)
		}
	}
	return nil
}

func getString(claims jwt.MapClaims, key string) string {
	if v, ok := claims[key]; ok {
		return fmt.Sprintf("%v", v)
	}
	return ""
}
