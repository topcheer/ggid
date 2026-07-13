package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// HTTPIdentityClient implements IdentityClient via REST calls to the Identity Service.
type HTTPIdentityClient struct {
	baseURL string
	client  *http.Client
}

// NewHTTPIdentityClient creates a new HTTP-based identity client.
func NewHTTPIdentityClient(baseURL string) *HTTPIdentityClient {
	return &HTTPIdentityClient{
		baseURL: baseURL,
		client:  &http.Client{},
	}
}

func (c *HTTPIdentityClient) GetUser(ctx context.Context, tenantID uuid.UUID, identifier string) (*UserInfo, error) {
	url := fmt.Sprintf("%s/api/v1/users?username=%s", c.baseURL, identifier)
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	req.Header.Set("X-Tenant-ID", tenantID.String())
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("identity service unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("identity service returned %d", resp.StatusCode)
	}
	var u struct {
		ID          string `json:"id"`
		Username    string `json:"username"`
		Email       string `json:"email"`
		Status      string `json:"status"`
		DisplayName string `json:"display_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
		return nil, fmt.Errorf("failed to decode user: %w", err)
	}
	uid, err := uuid.Parse(u.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}
	return &UserInfo{
		ID:          uid,
		TenantID:    tenantID,
		Username:    u.Username,
		Email:       u.Email,
		Status:      u.Status,
		DisplayName: u.DisplayName,
	}, nil
}

func (c *HTTPIdentityClient) GetUserByID(ctx context.Context, tenantID, userID uuid.UUID) (*UserInfo, error) {
	url := fmt.Sprintf("%s/api/v1/users/%s", c.baseURL, userID)
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	req.Header.Set("X-Tenant-ID", tenantID.String())
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("identity service unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("identity service returned %d", resp.StatusCode)
	}
	var u struct {
		ID          string `json:"id"`
		Username    string `json:"username"`
		Email       string `json:"email"`
		Status      string `json:"status"`
		DisplayName string `json:"display_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
		return nil, fmt.Errorf("failed to decode user: %w", err)
	}
	uid, _ := uuid.Parse(u.ID)
	return &UserInfo{
		ID: uid, TenantID: tenantID, Username: u.Username,
		Email: u.Email, Status: u.Status, DisplayName: u.DisplayName,
	}, nil
}

func (c *HTTPIdentityClient) FindExternalIdentity(ctx context.Context, tenantID uuid.UUID, provider, externalID string) (*ExternalIdentityLink, error) {
	// Not implemented in identity service REST API yet
	return nil, nil
}

func (c *HTTPIdentityClient) LinkExternalIdentity(ctx context.Context, tenantID, userID uuid.UUID, provider, externalID string, metadata map[string]any) error {
	// Not implemented yet
	return nil
}

func (c *HTTPIdentityClient) CreateUserFromSocial(ctx context.Context, tenantID uuid.UUID, username, email, displayName, provider, externalID string, metadata map[string]any) (*UserInfo, error) {
	// Generate a random password — LDAP users authenticate via LDAP, not local password
	randomPass := fmt.Sprintf("ldap-%s-%d", externalID[:8], time.Now().UnixNano())
	body, _ := json.Marshal(map[string]string{
		"username":      username,
		"email":         email,
		"password":      randomPass,
		"display_name":  displayName,
	})
	url := fmt.Sprintf("%s/api/v1/users", c.baseURL)
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", tenantID.String())
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("identity service unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("identity service returned %d", resp.StatusCode)
	}
	var u struct {
		ID          string `json:"id"`
		Username    string `json:"username"`
		Email       string `json:"email"`
		Status      string `json:"status"`
		DisplayName string `json:"display_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
		return nil, fmt.Errorf("failed to decode created user: %w", err)
	}
	uid, err := uuid.Parse(u.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID in response: %w", err)
	}
	return &UserInfo{
		ID: uid, TenantID: tenantID, Username: u.Username,
		Email: u.Email, Status: u.Status, DisplayName: u.DisplayName,
	}, nil
}
