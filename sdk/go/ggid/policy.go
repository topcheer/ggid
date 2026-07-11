package ggid

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// --- Policy & RBAC Service Client ---

// Permission represents a permission entry.
type Permission struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Resource    string `json:"resource"`
	Action     string `json:"action"`
	Description string `json:"description,omitempty"`
}

// PolicyCheckRequest is the ABAC policy evaluation request.
type PolicyCheckRequest struct {
	Subject    string            `json:"subject"`
	Resource   string            `json:"resource"`
	Action     string            `json:"action"`
	Context    map[string]string `json:"context,omitempty"`
}

// CreateRole creates a new role in the policy service.
func (c *Client) CreateRole(ctx context.Context, token, name, key, description string) (*Role, error) {
	body := map[string]string{
		"name":        name,
		"key":         key,
		"description": description,
	}
	resp, err := c.do(ctx, http.MethodPost, "/api/v1/roles", body, token)
	if err != nil {
		return nil, err
	}
	var role Role
	if err := json.Unmarshal(resp, &role); err != nil {
		return nil, fmt.Errorf("parse role response: %w", err)
	}
	return &role, nil
}

// GetRole retrieves a single role by ID.
func (c *Client) GetRole(ctx context.Context, token, roleID string) (*Role, error) {
	resp, err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/roles/%s", roleID), nil, token)
	if err != nil {
		return nil, err
	}
	var role Role
	if err := json.Unmarshal(resp, &role); err != nil {
		return nil, fmt.Errorf("parse role response: %w", err)
	}
	return &role, nil
}

// DeleteRole removes a role by ID.
func (c *Client) DeleteRole(ctx context.Context, token, roleID string) error {
	_, err := c.do(ctx, http.MethodDelete, fmt.Sprintf("/api/v1/roles/%s", roleID), nil, token)
	return err
}

// AssignRole assigns a role to a user.
func (c *Client) AssignRole(ctx context.Context, token, userID, roleID string) error {
	body := map[string]string{
		"user_id": userID,
		"role_id": roleID,
	}
	_, err := c.do(ctx, http.MethodPost, "/api/v1/roles/assign", body, token)
	return err
}

// RevokeRole revokes a role from a user.
func (c *Client) RevokeRole(ctx context.Context, token, userID, roleID string) error {
	body := map[string]string{
		"user_id": userID,
		"role_id": roleID,
	}
	_, err := c.do(ctx, http.MethodDelete, "/api/v1/roles/revoke", body, token)
	return err
}

// GetUserRoles lists all roles assigned to a user.
func (c *Client) GetUserRoles(ctx context.Context, token, userID string) ([]Role, error) {
	resp, err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/users/%s/roles", userID), nil, token)
	if err != nil {
		return nil, err
	}
	var roles []Role
	if err := json.Unmarshal(resp, &roles); err != nil {
		return nil, fmt.Errorf("parse user roles: %w", err)
	}
	return roles, nil
}

// ListPermissions retrieves all available permissions.
func (c *Client) ListPermissions(ctx context.Context, token string) ([]Permission, error) {
	resp, err := c.do(ctx, http.MethodGet, "/api/v1/permissions", nil, token)
	if err != nil {
		return nil, err
	}
	var perms []Permission
	if err := json.Unmarshal(resp, &perms); err != nil {
		return nil, fmt.Errorf("parse permissions: %w", err)
	}
	return perms, nil
}

// CheckPolicy evaluates an ABAC policy with context attributes.
// Unlike CheckPermission (basic allow/deny), CheckPolicy supports
// additional context attributes for attribute-based evaluation.
func (c *Client) CheckPolicy(ctx context.Context, token string, req *PolicyCheckRequest) (*PolicyResult, error) {
	resp, err := c.do(ctx, http.MethodPost, "/api/v1/policies/check", req, token)
	if err != nil {
		return nil, err
	}
	var result PolicyResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("parse policy result: %w", err)
	}
	return &result, nil
}
