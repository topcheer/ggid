package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
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
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *HTTPIdentityClient) GetUser(ctx context.Context, tenantID uuid.UUID, identifier string) (*UserInfo, error) {
	// First try username search
	url := fmt.Sprintf("%s/api/v1/users?username=%s", c.baseURL, identifier)
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	req.Header.Set("X-Tenant-ID", tenantID.String())
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("identity service unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		var u struct {
			ID          string `json:"id"`
			Username    string `json:"username"`
			Email       string `json:"email"`
			Status      string `json:"status"`
			DisplayName string `json:"display_name"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&u); err == nil && u.ID != "" {
			uid, err := uuid.Parse(u.ID)
			if err == nil {
				return &UserInfo{
					ID: uid, TenantID: tenantID, Username: u.Username,
					Email: u.Email, Status: u.Status, DisplayName: u.DisplayName,
				}, nil
			}
		}
	}

	// Fallback: list all users and search by email or username
	listURL := fmt.Sprintf("%s/api/v1/users", c.baseURL)
	req2, _ := http.NewRequestWithContext(ctx, "GET", listURL, nil)
	req2.Header.Set("X-Tenant-ID", tenantID.String())
	resp2, err := c.client.Do(req2)
	if err != nil {
		return nil, fmt.Errorf("identity service unreachable: %w", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		return nil, nil
	}
	var list struct {
		Users []struct {
			ID          string `json:"id"`
			Username    string `json:"username"`
			Email       string `json:"email"`
			Status      string `json:"status"`
			DisplayName string `json:"display_name"`
		} `json:"users"`
	}
	if err := json.NewDecoder(resp2.Body).Decode(&list); err != nil {
		return nil, nil
	}
	for _, u := range list.Users {
		if u.Email == identifier || u.Username == identifier {
			uid, err := uuid.Parse(u.ID)
			if err != nil {
				continue
			}
			return &UserInfo{
				ID: uid, TenantID: tenantID, Username: u.Username,
				Email: u.Email, Status: u.Status, DisplayName: u.DisplayName,
			}, nil
		}
	}
	return nil, nil
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

// GetUserRoles fetches the user's assigned role keys from the Identity Service.
// Calls GET /api/v1/users/{id}/roles and extracts role keys.
// Falls back to ["user"] on any error to ensure users always get basic access.
func (c *HTTPIdentityClient) GetUserRoles(ctx context.Context, tenantID, userID uuid.UUID) ([]string, error) {
	url := fmt.Sprintf("%s/api/v1/users/%s/roles", c.baseURL, userID)
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	req.Header.Set("X-Tenant-ID", tenantID.String())
	resp, err := c.client.Do(req)
	if err != nil {
		return []string{"user"}, nil // degraded mode — give basic access
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return []string{"user"}, nil
	}
	var result struct {
		Roles []struct {
			RoleID   string `json:"role_id"`
			RoleName string `json:"role_name"`
		} `json:"roles"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return []string{"user"}, nil
	}
	if len(result.Roles) == 0 {
		return []string{"user"}, nil
	}
	scopes := make([]string, 0, len(result.Roles)+1)

	// Platform-reserved scope strings that must NEVER be derived from
	// tenant-controlled role names. A tenant admin can create a role named
	// "platform:admin" — if that string leaks into the JWT scope claim, it
	// grants platform-level access. Only the platform tenant (ID checked
	// by the caller via X-Tenant-ID) may legitimately hold these roles.
	platformReservedScopes := map[string]bool{
		"platform:admin":     true,
		"platform administrator": true,
		"tenant:admin":       true,
		"tenant administrator":   true,
	}

	// The platform tenant ID — the only tenant whose roles may legitimately
	// produce platform:admin / tenant:admin scopes.
	// In production this comes from config; the env override allows testing.
	platformTenantID := os.Getenv("GGID_PLATFORM_TENANT_ID")
	if platformTenantID == "" {
		platformTenantID = os.Getenv("GGID_TENANT_ID")
	}
	isPlatformTenant := platformTenantID != "" && tenantID.String() == platformTenantID

	for _, r := range result.Roles {
		key := r.RoleName
		if key == "" {
			key = r.RoleID
		}
		if key == "" {
			continue
		}
		// Drop reserved platform scopes from non-platform tenants.
		if !isPlatformTenant && platformReservedScopes[strings.ToLower(key)] {
			slog.Warn("scope generation: dropping platform-reserved scope from non-platform tenant role",
				"scope", key, "tenant_id", tenantID.String(), "role_id", r.RoleID)
			continue
		}
		scopes = append(scopes, key)
	}
	if len(scopes) == 0 {
		return []string{"user"}, nil
	}
	return scopes, nil
}

// GetUserPermissions fetches the fine-grained permissions for all roles
// assigned to a user. Queries the role_permissions table via identity service
// batch endpoint: GET /api/v1/users/{id}/permissions
func (c *HTTPIdentityClient) GetUserPermissions(ctx context.Context, tenantID, userID uuid.UUID) ([]string, error) {
	url := fmt.Sprintf("%s/api/v1/users/%s/permissions", c.baseURL, userID)
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	req.Header.Set("X-Tenant-ID", tenantID.String())
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, nil // degraded mode
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, nil
	}
	var result struct {
		Permissions []struct {
			Key string `json:"key"`
		} `json:"permissions"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, nil
	}
	perms := make([]string, 0, len(result.Permissions))
	for _, p := range result.Permissions {
		if p.Key != "" {
			perms = append(perms, p.Key)
		}
	}
	return perms, nil
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

// ResolveTenantBySlug calls the identity service's public tenant resolve endpoint.
func (c *HTTPIdentityClient) ResolveTenantBySlug(ctx context.Context, slug string) (uuid.UUID, error) {
	url := fmt.Sprintf("%s/api/v1/tenants/resolve?slug=%s", c.baseURL, slug)
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	resp, err := c.client.Do(req)
	if err != nil {
		return uuid.Nil, fmt.Errorf("identity service unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return uuid.Nil, fmt.Errorf("tenant not found for slug %q (status %d)", slug, resp.StatusCode)
	}
	var t struct {
		TenantID string `json:"tenant_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&t); err != nil {
		return uuid.Nil, fmt.Errorf("failed to decode tenant resolve response: %w", err)
	}
	return uuid.Parse(t.TenantID)
}
