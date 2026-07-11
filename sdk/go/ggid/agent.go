// Package ggid provides AI Agent Identity helpers for the GGID SDK.
//
// This file adds support for the AI Agent Identity API, enabling
// Go applications to register agents, exchange user tokens for
// agent-scoped tokens, and verify agent tokens.
//
// Quick start:
//
//	client := ggid.NewClient("https://iam.example.com",
//		ggid.WithTenantID("00000000-0000-0000-0000-000000000001"))
//	agent, _ := client.RegisterAgent(ctx, &ggid.AgentRegistration{
//		Name: "CodeBot", Type: "coding-assistant",
//		AllowedScopes: []string{"repo:read"},
//	}, accessToken)
//	resp, _ := client.ExchangeAgentToken(ctx, agent.ID, accessToken, []string{"repo:read"})

package ggid

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// AgentType classifies the kind of AI agent.
type AgentType string

const (
	AgentTypeCodingAssistant  AgentType = "coding-assistant"
	AgentTypeDataPipeline     AgentType = "data-pipeline"
	AgentTypeCustomerService  AgentType = "customer-service"
	AgentTypeWorkflow         AgentType = "workflow-orchestrator"
	AgentTypeResearch         AgentType = "research-agent"
	AgentTypeCustom           AgentType = "custom"
)

// AgentRegistration holds parameters for registering a new AI agent.
type AgentRegistration struct {
	Name               string     `json:"name"`
	Type               AgentType  `json:"type"`
	OwnerUserID        string     `json:"owner_user_id"`
	Description        string     `json:"description,omitempty"`
	AllowedScopes      []string   `json:"allowed_scopes"`
	AllowedMCPServers  []string   `json:"allowed_mcp_servers,omitempty"`
	MaxDelegationDepth int        `json:"max_delegation_depth"`
	RateLimitPerMin    int        `json:"rate_limit_per_min"`
}

// Agent is the response from agent registration.
type Agent struct {
	ID                 string    `json:"id"`
	TenantID           string    `json:"tenant_id"`
	Name               string    `json:"name"`
	Type               AgentType `json:"type"`
	OwnerUserID        string    `json:"owner_user_id"`
	ClientID           string    `json:"client_id"`
	Status             string    `json:"status"`
	AllowedScopes      []string  `json:"allowed_scopes"`
	MaxDelegationDepth int       `json:"max_delegation_depth"`
	CreatedAt          time.Time `json:"created_at"`
}

// AgentTokenResponse is the response from agent token exchange.
type AgentTokenResponse struct {
	AccessToken     string `json:"access_token"`
	TokenType       string `json:"token_type"`
	ExpiresIn       int    `json:"expires_in"`
	Scope           string `json:"scope"`
	AgentID         string `json:"agent_id"`
	DelegationDepth int    `json:"delegation_depth_remaining"`
}

// AgentTokenClaims represents the claims in an agent JWT.
type AgentTokenClaims struct {
	Sub                string `json:"sub"`
	Iss                string `json:"iss"`
	Exp                int64  `json:"exp"`
	Iat                int64  `json:"iat"`
	AgentID            string `json:"agent_id"`
	AgentType          string `json:"agent_type"`
	IsAgentToken       bool   `json:"is_agent_token"`
	MaxDelegationDepth int    `json:"max_delegation_depth"`
}

// RegisterAgent registers a new AI agent identity.
func (c *Client) RegisterAgent(ctx context.Context, reg *AgentRegistration, accessToken string) (*Agent, error) {
	body, err := json.Marshal(reg)
	if err != nil {
		return nil, fmt.Errorf("marshal registration: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST",
		c.gatewayURL+"/api/v1/agents/register", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	if c.tenantID != "" {
		req.Header.Set("X-Tenant-ID", c.tenantID)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("register agent: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		var errResp map[string]string
		json.NewDecoder(resp.Body).Decode(&errResp)
		return nil, fmt.Errorf("register agent failed (%d): %s", resp.StatusCode, errResp["error"])
	}

	var agent Agent
	if err := json.NewDecoder(resp.Body).Decode(&agent); err != nil {
		return nil, fmt.Errorf("decode agent response: %w", err)
	}
	return &agent, nil
}

// ListAgents lists all agents for the configured tenant.
func (c *Client) ListAgents(ctx context.Context, accessToken string) ([]Agent, error) {
	req, err := http.NewRequestWithContext(ctx, "GET",
		c.gatewayURL+"/api/v1/agents", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	if c.tenantID != "" {
		req.Header.Set("X-Tenant-ID", c.tenantID)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list agents: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list agents failed (%d)", resp.StatusCode)
	}

	var result struct {
		Agents []Agent `json:"agents"`
		Total  int     `json:"total"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode agents response: %w", err)
	}
	return result.Agents, nil
}

// ExchangeAgentToken exchanges a user access token for an agent-scoped token.
// The resulting token contains agent identity claims and a delegation chain.
func (c *Client) ExchangeAgentToken(ctx context.Context, agentID, subjectToken string, scopes []string) (*AgentTokenResponse, error) {
	body, _ := json.Marshal(map[string]any{
		"subject_token": subjectToken,
		"agent_id":      agentID,
		"scope":         scopes,
	})

	req, err := http.NewRequestWithContext(ctx, "POST",
		c.gatewayURL+"/api/v1/agents/token", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.tenantID != "" {
		req.Header.Set("X-Tenant-ID", c.tenantID)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("exchange agent token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]string
		json.NewDecoder(resp.Body).Decode(&errResp)
		return nil, fmt.Errorf("agent token exchange failed (%d): %s", resp.StatusCode, errResp["error"])
	}

	var tokenResp AgentTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("decode token response: %w", err)
	}
	return &tokenResp, nil
}

// VerifyAgentToken verifies an agent token and returns its claims.
func (c *Client) VerifyAgentToken(ctx context.Context, token string) (*AgentTokenClaims, error) {
	body, _ := json.Marshal(map[string]string{"token": token})

	req, err := http.NewRequestWithContext(ctx, "POST",
		c.gatewayURL+"/api/v1/agents/verify", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("verify agent token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]string
		json.NewDecoder(resp.Body).Decode(&errResp)
		return nil, fmt.Errorf("agent token invalid (%d): %s", resp.StatusCode, errResp["error"])
	}

	var result struct {
		Active             bool   `json:"active"`
		AgentID            string `json:"agent_id"`
		AgentType          string `json:"agent_type"`
		IsAgentToken       bool   `json:"is_agent_token"`
		MaxDelegationDepth int    `json:"max_delegation_depth"`
		Sub                string `json:"sub"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode verify response: %w", err)
	}

	return &AgentTokenClaims{
		Sub:                result.Sub,
		AgentID:            result.AgentID,
		AgentType:          result.AgentType,
		IsAgentToken:       result.IsAgentToken,
		MaxDelegationDepth: result.MaxDelegationDepth,
	}, nil
}
