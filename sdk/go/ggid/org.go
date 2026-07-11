package ggid

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// --- Organization Service Client ---

// Organization represents a GGID organization.
type Organization struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	ParentID    string `json:"parent_id,omitempty"`
	Status     string `json:"status"`
	CreatedAt   string `json:"created_at,omitempty"`
}

// Department represents a department within an organization.
type Department struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	OrgID       string `json:"org_id"`
	ParentID    string `json:"parent_id,omitempty"`
	Description string `json:"description,omitempty"`
}

// Membership represents a user's membership in an organization.
type Membership struct {
	UserID string `json:"user_id"`
	OrgID  string `json:"org_id"`
	Role  string `json:"role"`
}

// ListOrganizations retrieves all organizations.
func (c *Client) ListOrganizations(ctx context.Context, token string) ([]Organization, error) {
	resp, err := c.do(ctx, http.MethodGet, "/api/v1/orgs", nil, token)
	if err != nil {
		return nil, err
	}
	var orgs []Organization
	if err := json.Unmarshal(resp, &orgs); err != nil {
		return nil, fmt.Errorf("parse organizations response: %w", err)
	}
	return orgs, nil
}

// GetOrganization retrieves a single organization by ID.
func (c *Client) GetOrganization(ctx context.Context, token, orgID string) (*Organization, error) {
	resp, err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/orgs/%s", orgID), nil, token)
	if err != nil {
		return nil, err
	}
	var org Organization
	if err := json.Unmarshal(resp, &org); err != nil {
		return nil, fmt.Errorf("parse organization response: %w", err)
	}
	return &org, nil
}

// CreateOrganization creates a new organization.
func (c *Client) CreateOrganization(ctx context.Context, token, name, description string) (*Organization, error) {
	body := map[string]string{
		"name":        name,
		"description": description,
	}
	resp, err := c.do(ctx, http.MethodPost, "/api/v1/orgs", body, token)
	if err != nil {
		return nil, err
	}
	var org Organization
	if err := json.Unmarshal(resp, &org); err != nil {
		return nil, fmt.Errorf("parse organization response: %w", err)
	}
	return &org, nil
}

// DeleteOrganization removes an organization by ID.
func (c *Client) DeleteOrganization(ctx context.Context, token, orgID string) error {
	_, err := c.do(ctx, http.MethodDelete, fmt.Sprintf("/api/v1/orgs/%s", orgID), nil, token)
	return err
}

// ListDepartments retrieves all departments within an organization.
func (c *Client) ListDepartments(ctx context.Context, token, orgID string) ([]Department, error) {
	resp, err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/orgs/%s/departments", orgID), nil, token)
	if err != nil {
		return nil, err
	}
	var depts []Department
	if err := json.Unmarshal(resp, &depts); err != nil {
		return nil, fmt.Errorf("parse departments response: %w", err)
	}
	return depts, nil
}

// CreateDepartment creates a new department within an organization.
func (c *Client) CreateDepartment(ctx context.Context, token, orgID, name, description string) (*Department, error) {
	body := map[string]string{
		"name":        name,
		"description": description,
	}
	resp, err := c.do(ctx, http.MethodPost, fmt.Sprintf("/api/v1/orgs/%s/departments", orgID), body, token)
	if err != nil {
		return nil, err
	}
	var dept Department
	if err := json.Unmarshal(resp, &dept); err != nil {
		return nil, fmt.Errorf("parse department response: %w", err)
	}
	return &dept, nil
}

// AddMember adds a user to an organization with a specified role.
func (c *Client) AddMember(ctx context.Context, token, orgID, userID, role string) error {
	body := map[string]string{
		"user_id": userID,
		"role":    role,
	}
	_, err := c.do(ctx, http.MethodPost, fmt.Sprintf("/api/v1/orgs/%s/members", orgID), body, token)
	return err
}

// RemoveMember removes a user from an organization.
func (c *Client) RemoveMember(ctx context.Context, token, orgID, userID string) error {
	_, err := c.do(ctx, http.MethodDelete, fmt.Sprintf("/api/v1/orgs/%s/members/%s", orgID, userID), nil, token)
	return err
}
