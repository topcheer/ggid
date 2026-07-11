package ggid

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// --- Login Security ---

// LoginAttemptInfo represents a user's login attempt history and lockout status.
type LoginAttemptInfo struct {
	FailedAttempts  int    `json:"failed_attempts"`
	LockedUntil    string `json:"locked_until,omitempty"`
	LastAttemptIP  string `json:"last_attempt_ip,omitempty"`
	LastAttemptTime string `json:"last_attempt_time,omitempty"`
}

// GetLoginAttempts retrieves login attempt info for a user.
func (c *Client) GetLoginAttempts(ctx context.Context, token, userID string) (*LoginAttemptInfo, error) {
	data, err := c.do(ctx, http.MethodGet, "/api/v1/auth/login-attempts/"+userID, nil, token)
	if err != nil {
		return nil, err
	}
	var info LoginAttemptInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("unmarshal login attempts: %w", err)
	}
	return &info, nil
}

// ResetLoginAttempts resets the failed login attempt counter for a user.
func (c *Client) ResetLoginAttempts(ctx context.Context, token, userID string) error {
	_, err := c.do(ctx, http.MethodPost, "/api/v1/auth/login-attempts/"+userID+"/reset", nil, token)
	return err
}

// PasswordHistoryCheck checks if a password matches any in the user's history.
func (c *Client) PasswordHistoryCheck(ctx context.Context, token, userID, newPassword string) (bool, error) {
	body := map[string]string{"user_id": userID, "new_password": newPassword}
	data, err := c.do(ctx, http.MethodPost, "/api/v1/auth/password-history-check", body, token)
	if err != nil {
		return false, err
	}
	var result struct {
		IsRepeated   bool `json:"is_repeated"`
		HistoryCount int  `json:"history_count"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return false, fmt.Errorf("unmarshal password check: %w", err)
	}
	return result.IsRepeated, nil
}

// --- Account Linking ---

// LinkedAccount represents an external provider linked to a local user.
type LinkedAccount struct {
	Provider      string `json:"provider"`
	ExternalID    string `json:"external_id"`
	ExternalEmail string `json:"external_email,omitempty"`
	LinkedAt      string `json:"linked_at,omitempty"`
}

// LinkAccount links an external provider to a user.
func (c *Client) LinkAccount(ctx context.Context, token, userID string, link LinkedAccount) error {
	_, err := c.do(ctx, http.MethodPost, "/api/v1/users/"+userID+"/link", link, token)
	return err
}

// UnlinkAccount removes an external provider link from a user.
func (c *Client) UnlinkAccount(ctx context.Context, token, userID, provider string) error {
	_, err := c.do(ctx, http.MethodDelete, "/api/v1/users/"+userID+"/link/"+provider, nil, token)
	return err
}

// --- OAuth Consent Management ---

// Consent represents a user's consent granted to an OAuth client.
type Consent struct {
	ID        string   `json:"id"`
	UserID    string   `json:"user_id"`
	ClientID  string   `json:"client_id"`
	Scopes    []string `json:"scopes"`
	GrantedAt string   `json:"granted_at"`
	LastUsed  string   `json:"last_used,omitempty"`
}

// ListConsents lists all consents for a user.
func (c *Client) ListConsents(ctx context.Context, token, userID string) ([]Consent, error) {
	data, err := c.do(ctx, http.MethodGet, "/api/v1/oauth/consent/list?user_id="+userID, nil, token)
	if err != nil {
		return nil, err
	}
	var consents []Consent
	if err := json.Unmarshal(data, &consents); err != nil {
		return nil, fmt.Errorf("unmarshal consents: %w", err)
	}
	return consents, nil
}

// RevokeConsent revokes a specific consent by ID.
func (c *Client) RevokeConsent(ctx context.Context, token, consentID string) error {
	_, err := c.do(ctx, http.MethodDelete, "/api/v1/oauth/consent/"+consentID, nil, token)
	return err
}

// --- Policy: ABAC + Delegation ---

// ABACCondition represents a single attribute-based condition.
type ABACCondition struct {
	Field    string `json:"field"`
	Operator string `json:"operator"` // eq, ne, in, regex, startsWith, endsWith, gt, lt
	Value    string `json:"value"`
}

// ABACEvalRequest is the request body for ABAC evaluation.
type ABACEvalRequest struct {
	Attributes map[string]string `json:"attributes"`
	Conditions []ABACCondition   `json:"conditions"`
}

// ABACEvalResult is the response from ABAC evaluation.
type ABACEvalResult struct {
	Matched      bool     `json:"matched"`
	MatchedRules []string `json:"matched_rules,omitempty"`
}

// EvaluateABAC evaluates attributes against ABAC conditions.
func (c *Client) EvaluateABAC(ctx context.Context, token string, req ABACEvalRequest) (*ABACEvalResult, error) {
	data, err := c.do(ctx, http.MethodPost, "/api/v1/policies/abac/evaluate", req, token)
	if err != nil {
		return nil, err
	}
	var result ABACEvalResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("unmarshal ABAC result: %w", err)
	}
	return &result, nil
}

// ValidateDelegation validates a delegation chain for depth, consistency, and cycles.
func (c *Client) ValidateDelegation(ctx context.Context, token string, chain []string, maxDepth int) (bool, error) {
	body := map[string]interface{}{
		"chain":     chain,
		"max_depth": maxDepth,
	}
	data, err := c.do(ctx, http.MethodPost, "/api/v1/policy/delegation/validate", body, token)
	if err != nil {
		return false, err
	}
	var result struct {
		Valid         bool     `json:"valid"`
		Depth         int      `json:"depth"`
		CycleDetected bool     `json:"cycle_detected"`
		Errors        []string `json:"errors,omitempty"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return false, fmt.Errorf("unmarshal delegation validation: %w", err)
	}
	return result.Valid, nil
}

// --- SIEM Health ---

// SIEMHealth represents the health status of the SIEM forwarder.
type SIEMHealth struct {
	LastForwardTime string `json:"last_forward_time,omitempty"`
	PendingEvents   int    `json:"pending_events"`
	ErrorCount      int    `json:"error_count"`
	DestURL         string `json:"dest_url"`
}

// GetSIEMHealth retrieves the current SIEM forwarder health.
func (c *Client) GetSIEMHealth(ctx context.Context, token string) (*SIEMHealth, error) {
	data, err := c.do(ctx, http.MethodGet, "/api/v1/audit/siem/health", nil, token)
	if err != nil {
		return nil, err
	}
	var health SIEMHealth
	if err := json.Unmarshal(data, &health); err != nil {
		return nil, fmt.Errorf("unmarshal SIEM health: %w", err)
	}
	return &health, nil
}

// --- Alert Webhooks ---

// AlertWebhook represents an alert webhook configuration.
type AlertWebhook struct {
	ID        string   `json:"id,omitempty"`
	URL       string   `json:"url"`
	Secret    string   `json:"secret,omitempty"`
	Events    []string `json:"events,omitempty"`
	CreatedAt string   `json:"created_at,omitempty"`
}

// ListAlertWebhooks lists all configured alert webhooks.
func (c *Client) ListAlertWebhooks(ctx context.Context, token string) ([]AlertWebhook, error) {
	data, err := c.do(ctx, http.MethodGet, "/api/v1/audit/alert-webhooks", nil, token)
	if err != nil {
		return nil, err
	}
	var webhooks []AlertWebhook
	if err := json.Unmarshal(data, &webhooks); err != nil {
		return nil, fmt.Errorf("unmarshal alert webhooks: %w", err)
	}
	return webhooks, nil
}

// CreateAlertWebhook creates a new alert webhook.
func (c *Client) CreateAlertWebhook(ctx context.Context, token string, webhook AlertWebhook) error {
	_, err := c.do(ctx, http.MethodPost, "/api/v1/audit/alert-webhooks", webhook, token)
	return err
}

// DeleteAlertWebhook removes an alert webhook by ID.
func (c *Client) DeleteAlertWebhook(ctx context.Context, token, webhookID string) error {
	_, err := c.do(ctx, http.MethodDelete, "/api/v1/audit/alert-webhooks?id="+webhookID, nil, token)
	return err
}

// --- Compliance Schedules ---

// ComplianceSchedule represents a scheduled compliance report.
type ComplianceSchedule struct {
	ID         string   `json:"id,omitempty"`
	ReportType string   `json:"report_type"`
	Frequency  string   `json:"frequency"` // daily, weekly, monthly
	Recipients []string `json:"recipients"`
	NextRunAt  string   `json:"next_run_at,omitempty"`
}

// ListComplianceSchedules lists all compliance report schedules.
func (c *Client) ListComplianceSchedules(ctx context.Context, token string) ([]ComplianceSchedule, error) {
	data, err := c.do(ctx, http.MethodGet, "/api/v1/audit/compliance-schedules", nil, token)
	if err != nil {
		return nil, err
	}
	var schedules []ComplianceSchedule
	if err := json.Unmarshal(data, &schedules); err != nil {
		return nil, fmt.Errorf("unmarshal compliance schedules: %w", err)
	}
	return schedules, nil
}

// CreateComplianceSchedule creates a new compliance report schedule.
func (c *Client) CreateComplianceSchedule(ctx context.Context, token string, schedule ComplianceSchedule) error {
	_, err := c.do(ctx, http.MethodPost, "/api/v1/audit/compliance-schedules", schedule, token)
	return err
}

// DeleteComplianceSchedule removes a compliance schedule by ID.
func (c *Client) DeleteComplianceSchedule(ctx context.Context, token, scheduleID string) error {
	_, err := c.do(ctx, http.MethodDelete, "/api/v1/audit/compliance-schedules?id="+scheduleID, nil, token)
	return err
}

// --- User Import Validation ---

// ImportValidationResult is the response from validating a user import.
type ImportValidationResult struct {
	ValidCount   int                    `json:"valid_count"`
	InvalidCount int                    `json:"invalid_count"`
	Errors       []ImportValidationError `json:"errors,omitempty"`
}

// ImportValidationError represents a single row validation error.
type ImportValidationError struct {
	Row   int    `json:"row"`
	Field string `json:"field"`
	Error string `json:"error"`
}

// ValidateUserImport pre-validates user import data.
func (c *Client) ValidateUserImport(ctx context.Context, token string, users []map[string]string) (*ImportValidationResult, error) {
	body := map[string]interface{}{"users": users}
	data, err := c.do(ctx, http.MethodPost, "/api/v1/users/import/validate", body, token)
	if err != nil {
		return nil, err
	}
	var result ImportValidationResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("unmarshal import validation: %w", err)
	}
	return &result, nil
}
