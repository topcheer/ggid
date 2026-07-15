package ggid

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// --- Authentication ---

// Login authenticates with username/password and returns tokens.
func (c *Client) Login(ctx context.Context, username, password string) (*TokenSet, error) {
	return c.LoginWithTenant(ctx, username, password, "", "")
}

// LoginWithTenant authenticates with optional tenant_id or tenant_slug.
// If both are empty, the server uses the X-Tenant-ID header or default tenant.
func (c *Client) LoginWithTenant(ctx context.Context, username, password, tenantID, tenantSlug string) (*TokenSet, error) {
	body := map[string]string{"username": username, "password": password}
	if tenantID != "" {
		body["tenant_id"] = tenantID
	}
	if tenantSlug != "" {
		body["tenant_slug"] = tenantSlug
	}
	resp, err := c.do(ctx, http.MethodPost, "/api/v1/auth/login", body, "")
	if err != nil {
		return nil, err
	}
	var ts TokenSet
	if err := json.Unmarshal(resp, &ts); err != nil {
		return nil, fmt.Errorf("parse login response: %w", err)
	}
	c.tokens = &ts
	return &ts, nil
}

// Register creates a new user account.
func (c *Client) Register(ctx context.Context, username, email, password, name string) (string, error) {
	body := map[string]string{"username": username, "email": email, "password": password, "name": name}
	resp, err := c.do(ctx, http.MethodPost, "/api/v1/auth/register", body, "")
	if err != nil {
		return "", err
	}
	var result struct {
		UserID string `json:"user_id"`
	}
	_ = json.Unmarshal(resp, &result)
	return result.UserID, nil
}

// Refresh exchanges a refresh token for a new token set.
func (c *Client) Refresh(ctx context.Context, refreshToken string) (*TokenSet, error) {
	body := map[string]string{"refresh_token": refreshToken}
	resp, err := c.do(ctx, http.MethodPost, "/api/v1/auth/refresh", body, "")
	if err != nil {
		return nil, err
	}
	var ts TokenSet
	if err := json.Unmarshal(resp, &ts); err != nil {
		return nil, fmt.Errorf("parse refresh response: %w", err)
	}
	return &ts, nil
}

// --- User Management ---

// ListUsers returns users in the tenant.
func (c *Client) ListUsers(ctx context.Context, token string) ([]User, error) {
	resp, err := c.do(ctx, http.MethodGet, "/api/v1/users", nil, token)
	if err != nil {
		return nil, err
	}
	var result struct {
		Users []User `json:"users"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		// Try flat array
		var users []User
		if err2 := json.Unmarshal(resp, &users); err2 == nil {
			return users, nil
		}
		return nil, fmt.Errorf("parse users response: %w", err)
	}
	return result.Users, nil
}

// GetUser returns a single user by ID.
func (c *Client) GetUser(ctx context.Context, token, userID string) (*User, error) {
	resp, err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/users/%s", userID), nil, token)
	if err != nil {
		return nil, err
	}
	var u User
	if err := json.Unmarshal(resp, &u); err != nil {
		return nil, fmt.Errorf("parse user response: %w", err)
	}
	return &u, nil
}

// DeleteUser removes a user by ID.
func (c *Client) DeleteUser(ctx context.Context, token, userID string) error {
	_, err := c.do(ctx, http.MethodDelete, fmt.Sprintf("/api/v1/users/%s", userID), nil, token)
	return err
}

// --- RBAC ---

// ListRoles returns all roles in the tenant.
func (c *Client) ListRoles(ctx context.Context, token string) ([]Role, error) {
	resp, err := c.do(ctx, http.MethodGet, "/api/v1/roles", nil, token)
	if err != nil {
		return nil, err
	}
	var result struct {
		Roles []Role `json:"roles"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		var roles []Role
		if err2 := json.Unmarshal(resp, &roles); err2 == nil {
			return roles, nil
		}
		return nil, fmt.Errorf("parse roles response: %w", err)
	}
	return result.Roles, nil
}

// CheckPermission checks if a user can perform an action on a resource.
func (c *Client) CheckPermission(ctx context.Context, token, resource, action string) (*PolicyResult, error) {
	body := map[string]string{"resource": resource, "action": action}
	resp, err := c.do(ctx, http.MethodPost, "/api/v1/policies/check", body, token)
	if err != nil {
		return nil, err
	}
	var result PolicyResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("parse policy response: %w", err)
	}
	return &result, nil
}

// --- JWT Verification ---

// VerifyToken verifies a JWT and returns claims.
func (c *Client) VerifyToken(ctx context.Context, token string) (map[string]interface{}, error) {
	if c.verifier == nil {
		return nil, fmt.Errorf("no JWKS URL configured")
	}
	return c.verifier.Verify(ctx, token)
}

// --- Internal HTTP ---

func (c *Client) do(ctx context.Context, method, path string, body interface{}, token string) ([]byte, error) {
	ctx = ensureContext(ctx)

	var reqBody *strings.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		reqBody = strings.NewReader(string(data))
	}

	url := c.gatewayURL + path
	var req *http.Request
	var err error
	if reqBody != nil {
		req, err = http.NewRequestWithContext(ctx, method, url, reqBody)
	} else {
		req, err = http.NewRequestWithContext(ctx, method, url, nil)
	}
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("X-Tenant-ID", c.tenantID)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		// Try to parse structured error for better message
		msg := string(respBody)
		var structured struct {
			Error   string `json:"error"`
			Title   string `json:"title"`
			Detail  string `json:"detail"`
			Message string `json:"message"`
		}
		if json.Unmarshal(respBody, &structured) == nil {
			if structured.Detail != "" {
				msg = structured.Detail
			} else if structured.Message != "" {
				msg = structured.Message
			} else if structured.Title != "" {
				msg = structured.Title
			} else if structured.Error != "" {
				msg = structured.Error
			}
		}
		return nil, NewAPIError(resp.StatusCode, msg)
	}

	return io.ReadAll(resp.Body)
}
