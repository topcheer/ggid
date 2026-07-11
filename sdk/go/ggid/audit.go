// Package ggid provides audit, compliance, and alerting helpers for the GGID SDK.
//
// This file adds support for the Audit Service API, enabling Go applications
// to query audit events, generate compliance reports, manage alert rules,
// and configure data retention policies through the existing Client.do() helper.
//
// Quick start:
//
//	client := ggid.NewClient("https://iam.example.com",
//		ggid.WithTenantID("00000000-0000-0000-0000-000000000001"))
//	events, _ := client.ListAuditEvents(ctx, accessToken, ggid.AuditEventFilter{})

package ggid

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"
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
	Timestamp    time.Time              `json:"timestamp"`
	IP           string                 `json:"ip,omitempty"`
	UserAgent    string                 `json:"user_agent,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// AuditEventFilter holds query parameters for listing audit events.
type AuditEventFilter struct {
	EventType    string
	ActorID      string
	ResourceType string
	ResourceID   string
	StartDate    time.Time
	EndDate      time.Time
	Limit        int
	Offset       int
}

// ListAuditEvents retrieves audit events with optional filtering.
func (c *Client) ListAuditEvents(ctx context.Context, accessToken string, filter AuditEventFilter) ([]AuditEvent, error) {
	params := url.Values{}
	if filter.EventType != "" {
		params.Set("event_type", filter.EventType)
	}
	if filter.ActorID != "" {
		params.Set("actor_id", filter.ActorID)
	}
	if filter.ResourceType != "" {
		params.Set("resource_type", filter.ResourceType)
	}
	if filter.ResourceID != "" {
		params.Set("resource_id", filter.ResourceID)
	}
	if !filter.StartDate.IsZero() {
		params.Set("start_date", filter.StartDate.Format(time.RFC3339))
	}
	if !filter.EndDate.IsZero() {
		params.Set("end_date", filter.EndDate.Format(time.RFC3339))
	}
	if filter.Limit > 0 {
		params.Set("limit", strconv.Itoa(filter.Limit))
	}
	if filter.Offset > 0 {
		params.Set("offset", strconv.Itoa(filter.Offset))
	}

	path := "/api/v1/audit/events"
	if encoded := params.Encode(); encoded != "" {
		path += "?" + encoded
	}

	resp, err := c.do(ctx, "GET", path, nil, accessToken)
	if err != nil {
		return nil, err
	}

	var events []AuditEvent
	if err := json.Unmarshal(resp, &events); err != nil {
		// Try wrapped structure
		var wrapped struct {
			Events []AuditEvent `json:"events"`
		}
		if err2 := json.Unmarshal(resp, &wrapped); err2 == nil && len(wrapped.Events) > 0 {
			return wrapped.Events, nil
		}
		return nil, fmt.Errorf("parse audit events: %w", err)
	}
	return events, nil
}

// ComplianceReport represents a generated compliance report.
type ComplianceReport struct {
	Type      string                 `json:"type"`
	Period    map[string]string      `json:"period"`
	Summary   map[string]interface{} `json:"summary"`
	Controls  []ComplianceControl    `json:"controls"`
	Generated time.Time              `json:"generated"`
}

// ComplianceControl represents a single control in a compliance report.
type ComplianceControl struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Status      string `json:"status"`
	Description string `json:"description,omitempty"`
}

// GetComplianceReport generates a compliance report (soc2, hipaa, or gdpr).
func (c *Client) GetComplianceReport(ctx context.Context, accessToken, reportType string, startDate, endDate time.Time) (*ComplianceReport, error) {
	params := url.Values{}
	params.Set("type", reportType)
	if !startDate.IsZero() {
		params.Set("start_date", startDate.Format(time.RFC3339))
	}
	if !endDate.IsZero() {
		params.Set("end_date", endDate.Format(time.RFC3339))
	}

	path := "/api/v1/audit/compliance-report?" + params.Encode()
	resp, err := c.do(ctx, "GET", path, nil, accessToken)
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

// GetAlertRules retrieves the current alerting configuration.
func (c *Client) GetAlertRules(ctx context.Context, accessToken string) ([]AlertRule, error) {
	resp, err := c.do(ctx, "GET", "/api/v1/audit/alerts/config", nil, accessToken)
	if err != nil {
		return nil, err
	}

	var result struct {
		Rules []AlertRule `json:"rules"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		// Try flat array
		var rules []AlertRule
		if err2 := json.Unmarshal(resp, &rules); err2 == nil {
			return rules, nil
		}
		return nil, fmt.Errorf("parse alert rules: %w", err)
	}
	return result.Rules, nil
}

// UpsertAlertRule creates or updates an alert rule.
func (c *Client) UpsertAlertRule(ctx context.Context, accessToken string, rule AlertRule) error {
	_, err := c.do(ctx, "PUT", "/api/v1/audit/alerts/config", rule, accessToken)
	return err
}

// TestAlert sends a test notification for the alerting system.
func (c *Client) TestAlert(ctx context.Context, accessToken string) error {
	_, err := c.do(ctx, "POST", "/api/v1/audit/alerts/test", nil, accessToken)
	return err
}

// RetentionPolicy defines how long audit events are retained.
type RetentionPolicy struct {
	MaxAge   string `json:"max_age,omitempty"`
	MaxCount int64  `json:"max_count,omitempty"`
}

// GetRetentionPolicy retrieves the current data retention policy.
func (c *Client) GetRetentionPolicy(ctx context.Context, accessToken string) (*RetentionPolicy, error) {
	resp, err := c.do(ctx, "GET", "/api/v1/audit/retention", nil, accessToken)
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
func (c *Client) UpdateRetentionPolicy(ctx context.Context, accessToken string, policy RetentionPolicy) error {
	_, err := c.do(ctx, "PUT", "/api/v1/audit/retention", policy, accessToken)
	return err
}

// VerifyAuditIntegrity verifies the hash chain integrity of audit events.
func (c *Client) VerifyAuditIntegrity(ctx context.Context, accessToken string) (bool, error) {
	resp, err := c.do(ctx, "POST", "/api/v1/audit/verify-integrity", nil, accessToken)
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
func (c *Client) ExportAuditEvents(ctx context.Context, accessToken, format string) ([]byte, error) {
	path := "/api/v1/audit/export?format=" + url.QueryEscape(format)
	resp, err := c.do(ctx, "GET", path, nil, accessToken)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
