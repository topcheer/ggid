// Package ggid provides audit, compliance, and retention helpers for the GGID SDK.
//
// This file adds support for the Audit Service API: querying audit events,
// generating compliance reports, managing alert rules, and configuring data
// retention policies through the GGID Gateway.

package ggid

import (
	"context"
	"encoding/json"
	"fmt"
)

// AuditEvent represents a single audit log entry.
type AuditEvent struct {
	ID           string                 `json:"id"`
	TenantID     string                 `json:"tenant_id"`
	EventType    string                 `json:"event_type"`
	ActorID      string                 `json:"actor_id"`
	ActorType    string                 `json:"actor_type"`
	ActorName    string                 `json:"actor_name,omitempty"`
	ResourceType string                 `json:"resource_type"`
	ResourceID   string                 `json:"resource_id"`
	ResourceName string                 `json:"resource_name,omitempty"`
	Action       string                 `json:"action"`
	IP           string                 `json:"ip,omitempty"`
	UserAgent    string                 `json:"user_agent,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	Timestamp    string                 `json:"timestamp"`
	Hash         string                 `json:"hash,omitempty"`
}

// AuditEventFilter holds query parameters for listing audit events.
type AuditEventFilter struct {
	EventType    string
	ActorID      string
	ResourceType string
	ResourceID   string
	StartTime    string // RFC3339
	EndTime      string // RFC3339
	Limit        int
	Offset       int
}

// ListAuditEvents retrieves audit events with optional filtering.
func (c *Client) ListAuditEvents(ctx context.Context, token string, f AuditEventFilter) ([]AuditEvent, error) {
	path := "/api/v1/audit/events"
	q := buildQueryString(f)
	if q != "" {
		path += "?" + q
	}

	resp, err := c.do(ctx, "GET", path, nil, token)
	if err != nil {
		return nil, err
	}

	var events []AuditEvent
	if err := json.Unmarshal(resp, &events); err != nil {
		return nil, fmt.Errorf("parse audit events: %w", err)
	}
	return events, nil
}

// ComplianceReport holds a generated compliance report.
type ComplianceReport struct {
	Type      string                 `json:"type"`
	Period    map[string]string      `json:"period"`
	Summary   map[string]interface{} `json:"summary"`
	Controls  []map[string]interface{} `json:"controls"`
	Generated string                 `json:"generated,omitempty"`
}

// GetComplianceReport generates a compliance report (soc2, hipaa, or gdpr).
func (c *Client) GetComplianceReport(ctx context.Context, token, reportType, startDate, endDate string) (*ComplianceReport, error) {
	path := "/api/v1/audit/compliance-report?type=" + reportType
	if startDate != "" {
		path += "&start_date=" + startDate
	}
	if endDate != "" {
		path += "&end_date=" + endDate
	}

	resp, err := c.do(ctx, "GET", path, nil, token)
	if err != nil {
		return nil, err
	}

	var report ComplianceReport
	if err := json.Unmarshal(resp, &report); err != nil {
		return nil, fmt.Errorf("parse compliance report: %w", err)
	}
	return &report, nil
}

// AlertRule defines a real-time alerting rule on audit events.
type AlertRule struct {
	ID        string `json:"id,omitempty"`
	Name      string `json:"name"`
	Condition string `json:"condition"`
	Threshold int    `json:"threshold"`
	Window    string `json:"window"`
	Action    string `json:"action"`
	Enabled   bool   `json:"enabled"`
}

// GetAlertRules retrieves all configured alert rules.
func (c *Client) GetAlertRules(ctx context.Context, token string) ([]AlertRule, error) {
	resp, err := c.do(ctx, "GET", "/api/v1/audit/alerts/config", nil, token)
	if err != nil {
		return nil, err
	}

	var result struct {
		Rules []AlertRule `json:"rules"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("parse alert rules: %w", err)
	}
	return result.Rules, nil
}

// UpsertAlertRule creates or updates an alert rule.
func (c *Client) UpsertAlertRule(ctx context.Context, token string, rule AlertRule) error {
	_, err := c.do(ctx, "PUT", "/api/v1/audit/alerts/config", rule, token)
	return err
}

// TestAlert sends a test notification for the alerting system.
func (c *Client) TestAlert(ctx context.Context, token string) error {
	_, err := c.do(ctx, "POST", "/api/v1/audit/alerts/test", nil, token)
	return err
}

// RetentionPolicy defines how long audit events are retained.
type RetentionPolicy struct {
	MaxAgeDays int   `json:"max_age_days"`
	MaxEvents  int64 `json:"max_events,omitempty"`
}

// GetRetentionPolicy retrieves the current data retention policy.
func (c *Client) GetRetentionPolicy(ctx context.Context, token string) (*RetentionPolicy, error) {
	resp, err := c.do(ctx, "GET", "/api/v1/audit/retention", nil, token)
	if err != nil {
		return nil, err
	}

	var policy RetentionPolicy
	if err := json.Unmarshal(resp, &policy); err != nil {
		return nil, fmt.Errorf("parse retention policy: %w", err)
	}
	return &policy, nil
}

// UpdateRetentionPolicy updates the data retention policy.
func (c *Client) UpdateRetentionPolicy(ctx context.Context, token string, policy RetentionPolicy) error {
	_, err := c.do(ctx, "PUT", "/api/v1/audit/retention", policy, token)
	return err
}

// VerifyAuditIntegrity verifies the hash chain integrity of audit events.
func (c *Client) VerifyAuditIntegrity(ctx context.Context, token string) (bool, error) {
	resp, err := c.do(ctx, "POST", "/api/v1/audit/verify-integrity", nil, token)
	if err != nil {
		return false, err
	}

	var result struct {
		Valid   bool   `json:"valid"`
		Message string `json:"message,omitempty"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return false, fmt.Errorf("parse integrity result: %w", err)
	}
	return result.Valid, nil
}

// ExportAuditEvents exports audit events as CSV or JSON.
// format can be "csv" or "json".
func (c *Client) ExportAuditEvents(ctx context.Context, token, format string) ([]byte, error) {
	path := "/api/v1/audit/export?format=" + format
	return c.do(ctx, "GET", path, nil, token)
}

// AccessRequest represents an IGA access request.
type AccessRequest struct {
	ID           string `json:"id,omitempty"`
	Resource     string `json:"resource"`
	Action       string `json:"action"`
	Justification string `json:"justification,omitempty"`
	Duration     string `json:"duration,omitempty"`
	Status       string `json:"status,omitempty"`
	RequesterID  string `json:"requester_id,omitempty"`
	CreatedAt    string `json:"created_at,omitempty"`
}

// ListAccessRequests retrieves access requests with optional status filter.
func (c *Client) ListAccessRequests(ctx context.Context, token, status string) ([]AccessRequest, error) {
	path := "/api/v1/access-requests"
	if status != "" {
		path += "?status=" + status
	}

	resp, err := c.do(ctx, "GET", path, nil, token)
	if err != nil {
		return nil, err
	}

	var requests []AccessRequest
	if err := json.Unmarshal(resp, &requests); err != nil {
		return nil, fmt.Errorf("parse access requests: %w", err)
	}
	return requests, nil
}

// SubmitAccessRequest creates a new access request.
func (c *Client) SubmitAccessRequest(ctx context.Context, token string, req AccessRequest) (*AccessRequest, error) {
	resp, err := c.do(ctx, "POST", "/api/v1/access-requests", req, token)
	if err != nil {
		return nil, err
	}

	var result AccessRequest
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("parse access request response: %w", err)
	}
	return &result, nil
}

// ApproveAccessRequest approves an access request by ID.
func (c *Client) ApproveAccessRequest(ctx context.Context, token, requestID string) error {
	_, err := c.do(ctx, "POST", "/api/v1/access-requests/"+requestID+"/approve", nil, token)
	return err
}

// DenyAccessRequest denies an access request by ID.
func (c *Client) DenyAccessRequest(ctx context.Context, token, requestID string) error {
	_, err := c.do(ctx, "POST", "/api/v1/access-requests/"+requestID+"/deny", nil, token)
	return err
}

// BrandingConfig holds per-tenant branding settings.
type BrandingConfig struct {
	LogoURL      string `json:"logo_url,omitempty"`
	PrimaryColor string `json:"primary_color,omitempty"`
	CustomCSS    string `json:"custom_css,omitempty"`
}

// GetBranding retrieves the branding configuration for a tenant.
func (c *Client) GetBranding(ctx context.Context, token, tenantID string) (*BrandingConfig, error) {
	resp, err := c.do(ctx, "GET", "/api/v1/tenants/"+tenantID+"/branding", nil, token)
	if err != nil {
		return nil, err
	}

	var config BrandingConfig
	if err := json.Unmarshal(resp, &config); err != nil {
		return nil, fmt.Errorf("parse branding config: %w", err)
	}
	return &config, nil
}

// UpdateBranding updates the branding configuration for a tenant.
func (c *Client) UpdateBranding(ctx context.Context, token, tenantID string, config BrandingConfig) error {
	_, err := c.do(ctx, "PUT", "/api/v1/tenants/"+tenantID+"/branding", config, token)
	return err
}

// buildQueryString converts an AuditEventFilter into a URL query string.
func buildQueryString(f AuditEventFilter) string {
	var q string
	add := func(key, val string) {
		if val != "" {
			if q != "" {
				q += "&"
			}
			q += key + "=" + val
		}
	}
	add("event_type", f.EventType)
	add("actor_id", f.ActorID)
	add("resource_type", f.ResourceType)
	add("resource_id", f.ResourceID)
	add("start_time", f.StartTime)
	add("end_time", f.EndTime)
	if f.Limit > 0 {
		add("limit", fmt.Sprintf("%d", f.Limit))
	}
	if f.Offset > 0 {
		add("offset", fmt.Sprintf("%d", f.Offset))
	}
	return q
}
