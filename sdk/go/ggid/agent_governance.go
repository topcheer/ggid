// Package ggid provides agent governance SDK methods for the GGID IAM platform.
//
// This file adds support for agent privilege drift detection, shadow agent
// scanning, agent access reviews, NHI lifecycle management, and credential
// rotation scheduling — all driven by 2026 IAM trends research.

package ggid

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// ---------------------------------------------------------------------------
// Agent Drift Detection
// ---------------------------------------------------------------------------

// DriftReport describes a privilege drift detected for an agent.
type DriftReport struct {
	AgentID         string   `json:"agent_id"`
	AgentName       string   `json:"agent_name"`
	DetectedScopes  []string `json:"detected_scopes"`
	DeclaredScopes  []string `json:"declared_scopes"`
	DriftType       string   `json:"drift_type"`       // scope_expansion, new_tool_access, unauthorized_op
	Severity        string   `json:"severity"`          // low, medium, high, critical
	DetectedAt      string   `json:"detected_at"`
	Description     string   `json:"description"`
}

// DetectAgentDrift scans an agent's actual access vs declared permissions.
func (c *Client) DetectAgentDrift(ctx context.Context, agentID, accessToken string) (*DriftReport, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s/api/v1/agents/%s/drift", c.gatewayURL, agentID), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	if c.tenantID != "" {
		req.Header.Set("X-Tenant-ID", c.tenantID)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("detect drift: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("detect drift: status %d", resp.StatusCode)
	}

	var report DriftReport
	if err := json.NewDecoder(resp.Body).Decode(&report); err != nil {
		return nil, fmt.Errorf("decode drift report: %w", err)
	}
	return &report, nil
}

// ---------------------------------------------------------------------------
// Shadow Agent Scanner
// ---------------------------------------------------------------------------

// ShadowAgent describes an agent with active tokens but no registry entry.
type ShadowAgent struct {
	TokenCount   int    `json:"token_count"`
	AgentID      string `json:"agent_id"`
	FirstSeen    string `json:"first_seen"`
	LastActive   string `json:"last_active"`
	RiskLevel    string `json:"risk_level"`
}

// ShadowScanResult holds the results of a shadow agent scan.
type ShadowScanResult struct {
	ScanTime       time.Time      `json:"scan_time"`
	UnknownAgents  []ShadowAgent  `json:"unknown_agents"`
	TotalTokens    int            `json:"total_tokens"`
	TotalShadows   int            `json:"total_shadows"`
}

// ScanShadowAgents scans for agents with active tokens but no registry entry.
func (c *Client) ScanShadowAgents(ctx context.Context, accessToken string) (*ShadowScanResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s/api/v1/agents/shadows", c.gatewayURL), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	if c.tenantID != "" {
		req.Header.Set("X-Tenant-ID", c.tenantID)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("scan shadows: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("scan shadows: status %d", resp.StatusCode)
	}

	var result ShadowScanResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode shadow scan: %w", err)
	}
	return &result, nil
}

// ---------------------------------------------------------------------------
// Agent Access Review
// ---------------------------------------------------------------------------

// AgentReview represents an access review for an AI agent.
type AgentReview struct {
	ID              string   `json:"id,omitempty"`
	AgentID         string   `json:"agent_id"`
	Reviewer        string   `json:"reviewer"`
	ScopesReviewed  []string `json:"scopes_reviewed"`
	Decision        string   `json:"decision"`     // approve, reject, revoke
	Comment         string   `json:"comment"`
	Timestamp       string   `json:"timestamp,omitempty"`
}

// CreateAgentReview submits a new access review for an agent.
func (c *Client) CreateAgentReview(ctx context.Context, review *AgentReview, accessToken string) (*AgentReview, error) {
	body, err := json.Marshal(review)
	if err != nil {
		return nil, fmt.Errorf("marshal review: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/api/v1/agents/reviews", c.gatewayURL), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	if c.tenantID != "" {
		req.Header.Set("X-Tenant-ID", c.tenantID)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("create review: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("create review: status %d", resp.StatusCode)
	}

	var result AgentReview
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode review: %w", err)
	}
	return &result, nil
}

// ListAgentReviews retrieves all agent access reviews.
func (c *Client) ListAgentReviews(ctx context.Context, accessToken string) ([]AgentReview, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s/api/v1/agents/reviews", c.gatewayURL), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	if c.tenantID != "" {
		req.Header.Set("X-Tenant-ID", c.tenantID)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list reviews: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list reviews: status %d", resp.StatusCode)
	}

	var reviews []AgentReview
	if err := json.NewDecoder(resp.Body).Decode(&reviews); err != nil {
		return nil, fmt.Errorf("decode reviews: %w", err)
	}
	return reviews, nil
}

// GetAgentReview retrieves a specific agent access review by ID.
func (c *Client) GetAgentReview(ctx context.Context, reviewID, accessToken string) (*AgentReview, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s/api/v1/agents/reviews/%s", c.gatewayURL, reviewID), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	if c.tenantID != "" {
		req.Header.Set("X-Tenant-ID", c.tenantID)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get review: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get review: status %d", resp.StatusCode)
	}

	var review AgentReview
	if err := json.NewDecoder(resp.Body).Decode(&review); err != nil {
		return nil, fmt.Errorf("decode review: %w", err)
	}
	return &review, nil
}

// UpdateAgentReview updates an existing agent access review (e.g., approve/reject).
func (c *Client) UpdateAgentReview(ctx context.Context, reviewID string, review *AgentReview, accessToken string) (*AgentReview, error) {
	body, err := json.Marshal(review)
	if err != nil {
		return nil, fmt.Errorf("marshal review: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut,
		fmt.Sprintf("%s/api/v1/agents/reviews/%s", c.gatewayURL, reviewID), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	if c.tenantID != "" {
		req.Header.Set("X-Tenant-ID", c.tenantID)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("update review: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("update review: status %d", resp.StatusCode)
	}

	var result AgentReview
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode review: %w", err)
	}
	return &result, nil
}

// ---------------------------------------------------------------------------
// NHI Lifecycle Management
// ---------------------------------------------------------------------------

// NHIType classifies non-human identity types.
type NHIType string

const (
	NHITypeServiceAccount NHIType = "service-account"
	NHITypeAPIKey         NHIType = "api-key"
	NHITypeAIAgent        NHIType = "ai-agent"
	NHITypeIoT            NHIType = "iot-device"
	NHITypeOAuthClient    NHIType = "oauth-client"
	NHITypeServiceMesh    NHIType = "service-mesh"
)

// NHIEntry represents a non-human identity in the inventory.
type NHIEntry struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Type        NHIType `json:"type"`
	Status      string  `json:"status"`     // active, orphaned, decommissioned
	Created     string  `json:"created"`
	LastUsed    string  `json:"last_used"`
	Owner       string  `json:"owner"`
	RiskScore   int     `json:"risk_score"`
}

// NHIInventory holds the full non-human identity inventory.
type NHIInventory struct {
	Total      int       `json:"total"`
	Active     int       `json:"active"`
	Orphaned   int       `json:"orphaned"`
	Entries    []NHIEntry `json:"entries"`
}

// ListNHI retrieves the full non-human identity inventory.
func (c *Client) ListNHI(ctx context.Context, accessToken string) (*NHIInventory, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s/api/v1/identity/nhi", c.gatewayURL), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	if c.tenantID != "" {
		req.Header.Set("X-Tenant-ID", c.tenantID)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list nhi: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list nhi: status %d", resp.StatusCode)
	}

	var inventory NHIInventory
	if err := json.NewDecoder(resp.Body).Decode(&inventory); err != nil {
		return nil, fmt.Errorf("decode nhi inventory: %w", err)
	}
	return &inventory, nil
}

// DetectOrphans finds NHI entries with last_used > 90 days.
func (c *Client) DetectOrphans(ctx context.Context, accessToken string) ([]NHIEntry, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s/api/v1/identity/nhi/orphans", c.gatewayURL), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	if c.tenantID != "" {
		req.Header.Set("X-Tenant-ID", c.tenantID)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("detect orphans: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("detect orphans: status %d", resp.StatusCode)
	}

	var orphans []NHIEntry
	if err := json.NewDecoder(resp.Body).Decode(&orphans); err != nil {
		return nil, fmt.Errorf("decode orphans: %w", err)
	}
	return orphans, nil
}

// DecommissionNHI decommissions a non-human identity (revoke tokens + disable + audit).
func (c *Client) DecommissionNHI(ctx context.Context, nhiID, reason, accessToken string) error {
	body, _ := json.Marshal(map[string]string{"reason": reason})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/api/v1/identity/nhi/%s/decommission", c.gatewayURL, nhiID), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	if c.tenantID != "" {
		req.Header.Set("X-Tenant-ID", c.tenantID)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("decommission nhi: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("decommission nhi: status %d", resp.StatusCode)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Credential Rotation Scheduler
// ---------------------------------------------------------------------------

// RotationPolicy defines how a credential should be rotated.
type RotationPolicy struct {
	IntervalDays      int  `json:"interval_days"`
	AutoRotate        bool `json:"auto_rotate"`
	NotifyBeforeDays  int  `json:"notify_before_days"`
}

// RotationSchedule holds the schedule for a credential rotation.
type RotationSchedule struct {
	CredentialID    string         `json:"credential_id"`
	CredentialType  string         `json:"credential_type"`
	Policy          RotationPolicy `json:"policy"`
	NextRotation    string         `json:"next_rotation"`
	LastRotation    string         `json:"last_rotation,omitempty"`
	Status          string         `json:"status"`  // scheduled, due, overdue, rotated
}

// ScheduleRotation sets a rotation policy for a credential.
func (c *Client) ScheduleRotation(ctx context.Context, credentialID string, policy *RotationPolicy, accessToken string) (*RotationSchedule, error) {
	body, err := json.Marshal(policy)
	if err != nil {
		return nil, fmt.Errorf("marshal policy: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/api/v1/auth/credentials/%s/rotation", c.gatewayURL, credentialID), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	if c.tenantID != "" {
		req.Header.Set("X-Tenant-ID", c.tenantID)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("schedule rotation: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("schedule rotation: status %d", resp.StatusCode)
	}

	var schedule RotationSchedule
	if err := json.NewDecoder(resp.Body).Decode(&schedule); err != nil {
		return nil, fmt.Errorf("decode schedule: %w", err)
	}
	return &schedule, nil
}

// CheckDueRotations returns credentials that are due for rotation.
func (c *Client) CheckDueRotations(ctx context.Context, accessToken string) ([]RotationSchedule, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s/api/v1/auth/credentials/rotation/due", c.gatewayURL), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	if c.tenantID != "" {
		req.Header.Set("X-Tenant-ID", c.tenantID)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("check due rotations: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("check due rotations: status %d", resp.StatusCode)
	}

	var schedules []RotationSchedule
	if err := json.NewDecoder(resp.Body).Decode(&schedules); err != nil {
		return nil, fmt.Errorf("decode schedules: %w", err)
	}
	return schedules, nil
}

// ExecuteRotation triggers an immediate rotation for a credential.
func (c *Client) ExecuteRotation(ctx context.Context, credentialID, accessToken string) (*RotationSchedule, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/api/v1/auth/credentials/%s/rotation/execute", c.gatewayURL, credentialID), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	if c.tenantID != "" {
		req.Header.Set("X-Tenant-ID", c.tenantID)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute rotation: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("execute rotation: status %d", resp.StatusCode)
	}

	var schedule RotationSchedule
	if err := json.NewDecoder(resp.Body).Decode(&schedule); err != nil {
		return nil, fmt.Errorf("decode schedule: %w", err)
	}
	return &schedule, nil
}