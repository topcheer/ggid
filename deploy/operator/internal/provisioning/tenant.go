package provisioning

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// TenantProvisioner handles shared-mode tenant lifecycle.
// It calls the GGID Org service API to create/manage tenants.
type TenantProvisioner struct {
	gatewayURL string
	httpClient *http.Client
}

// NewTenantProvisioner creates a provisioner for shared-mode tenants.
func NewTenantProvisioner(gatewayURL string) *TenantProvisioner {
	return &TenantProvisioner{
		gatewayURL: gatewayURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// CreateTenantRequest is the payload sent to the Org service.
type CreateTenantRequest struct {
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	Plan     string `json:"plan"`
	Status   string `json:"status"`
	MaxUsers int32  `json:"max_users"`
}

// CreateTenantResponse is the response from the Org service.
type CreateTenantResponse struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Slug   string `json:"slug"`
	Plan   string `json:"plan"`
	Status string `json:"status"`
}

// Provision creates a new tenant in the shared GGID instance.
func (p *TenantProvisioner) Provision(req *CreateTenantRequest) (*CreateTenantResponse, error) {
	if p.gatewayURL == "" {
		return nil, fmt.Errorf("gateway URL not configured (set GGID_GATEWAY_URL)")
	}

	body, _ := json.Marshal(req)
	resp, err := p.httpClient.Post(
		p.gatewayURL+"/api/v1/org/tenants",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to call org service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		var errResp map[string]any
		json.NewDecoder(resp.Body).Decode(&errResp)
		return nil, fmt.Errorf("org service returned %d: %v", resp.StatusCode, errResp)
	}

	var result CreateTenantResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &result, nil
}

// GatewayURL returns the configured gateway URL.
func (p *TenantProvisioner) GatewayURL() string { return p.gatewayURL }

// Deprovision removes a tenant (soft delete).
func (p *TenantProvisioner) Deprovision(tenantID string) error {
	if p.gatewayURL == "" {
		return fmt.Errorf("gateway URL not configured")
	}
	req, _ := http.NewRequest(http.MethodDelete, p.gatewayURL+"/api/v1/org/tenants/"+tenantID, nil)
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call org service: %w", err)
	}
	resp.Body.Close()
	return nil
}

// SeedDefaultRoles creates default admin and viewer roles for a tenant.
func (p *TenantProvisioner) SeedDefaultRoles(tenantID string) error {
	if p.gatewayURL == "" {
		return fmt.Errorf("gateway URL not configured")
	}
	roles := []map[string]string{
		{"key": "admin", "name": "Administrator", "description": "Full access"},
		{"key": "viewer", "name": "Viewer", "description": "Read-only access"},
	}
	for _, role := range roles {
		body, _ := json.Marshal(role)
		resp, err := p.httpClient.Post(
			p.gatewayURL+"/api/v1/policy/roles?tenant_id="+tenantID,
			"application/json",
			bytes.NewReader(body),
		)
		if err != nil {
			return fmt.Errorf("failed to seed role %s: %w", role["key"], err)
		}
		resp.Body.Close()
	}
	return nil
}

// CreateAdminUser creates the initial admin user for a tenant.
func (p *TenantProvisioner) CreateAdminUser(tenantID, email, password string) error {
	if p.gatewayURL == "" {
		return fmt.Errorf("gateway URL not configured")
	}
	body, _ := json.Marshal(map[string]string{
		"username": email,
		"password": password,
		"email":    email,
	})
	resp, err := p.httpClient.Post(
		p.gatewayURL+"/api/v1/auth/register",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("failed to create admin user: %w", err)
	}
	resp.Body.Close()
	return nil
}
