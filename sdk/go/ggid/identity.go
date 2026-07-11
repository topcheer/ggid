package ggid

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// --- Identity Service Client ---

// CreateUserRequest holds the parameters for creating a new user.
type CreateUserRequest struct {
	Username    string `json:"username"`
	Email      string `json:"email"`
	Password   string `json:"password"`
	DisplayName string `json:"display_name,omitempty"`
	Status     string `json:"status,omitempty"`
}

// UpdateUserRequest holds the parameters for updating a user.
type UpdateUserRequest struct {
	Email       *string `json:"email,omitempty"`
	DisplayName *string `json:"display_name,omitempty"`
	Status     *string `json:"status,omitempty"`
}

// CreateUser creates a new user via the identity service.
func (c *Client) CreateUser(ctx context.Context, token string, req *CreateUserRequest) (*User, error) {
	resp, err := c.do(ctx, http.MethodPost, "/api/v1/users", req, token)
	if err != nil {
		return nil, err
	}
	var user User
	if err := json.Unmarshal(resp, &user); err != nil {
		return nil, fmt.Errorf("parse user response: %w", err)
	}
	return &user, nil
}

// UpdateUser updates an existing user's attributes.
func (c *Client) UpdateUser(ctx context.Context, token, userID string, req *UpdateUserRequest) (*User, error) {
	resp, err := c.do(ctx, http.MethodPut, fmt.Sprintf("/api/v1/users/%s", userID), req, token)
	if err != nil {
		return nil, err
	}
	var user User
	if err := json.Unmarshal(resp, &user); err != nil {
		return nil, fmt.Errorf("parse user response: %w", err)
	}
	return &user, nil
}

// LockUser locks a user account, preventing login.
func (c *Client) LockUser(ctx context.Context, token, userID string) error {
	_, err := c.do(ctx, http.MethodPost, fmt.Sprintf("/api/v1/users/%s/lock", userID), nil, token)
	return err
}

// UnlockUser unlocks a previously locked user account.
func (c *Client) UnlockUser(ctx context.Context, token, userID string) error {
	_, err := c.do(ctx, http.MethodPost, fmt.Sprintf("/api/v1/users/%s/unlock", userID), nil, token)
	return err
}

// SearchUsers searches for users by query string.
func (c *Client) SearchUsers(ctx context.Context, token, query string) ([]User, error) {
	path := fmt.Sprintf("/api/v1/users?q=%s", query)
	resp, err := c.do(ctx, http.MethodGet, path, nil, token)
	if err != nil {
		return nil, err
	}
	var users []User
	if err := json.Unmarshal(resp, &users); err != nil {
		return nil, fmt.Errorf("parse users response: %w", err)
	}
	return users, nil
}
