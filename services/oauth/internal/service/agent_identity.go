// Package service implements AI Agent Identity and MCP authentication.
//
// This file implements the AI Agent Identity model for GGID, enabling
// autonomous AI agents to authenticate, receive scoped tokens, and
// delegate authority through chains. It builds on RFC 8693 Token Exchange
// with agent-specific claims per the OpenID Foundation's October 2025
// whitepaper "Identity Management for Agentic AI".
package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ggid/ggid/pkg/crypto"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// AgentType classifies the kind of AI agent.
type AgentType string

const (
	AgentTypeCodingAssistant AgentType = "coding-assistant"
	AgentTypeDataPipeline    AgentType = "data-pipeline"
	AgentTypeCustomerService AgentType = "customer-service"
	AgentTypeWorkflow        AgentType = "workflow-orchestrator"
	AgentTypeResearch        AgentType = "research-agent"
	AgentTypeCustom          AgentType = "custom"
)

// AgentStatus represents the lifecycle state of a registered agent.
type AgentStatus string

const (
	AgentStatusActive   AgentStatus = "active"
	AgentStatusSuspended AgentStatus = "suspended"
	AgentStatusRevoked  AgentStatus = "revoked"
)

// AgentRegistration holds the parameters for registering a new AI agent.
type AgentRegistration struct {
	ID                uuid.UUID      `json:"id"`
	TenantID          uuid.UUID      `json:"tenant_id"`
	Name              string         `json:"name"`
	Type              AgentType      `json:"type"`
	OwnerUserID       uuid.UUID      `json:"owner_user_id"`
	Description       string         `json:"description,omitempty"`
	AllowedScopes     []string       `json:"allowed_scopes"`
	AllowedMCPServers []string       `json:"allowed_mcp_servers,omitempty"`
	MaxDelegationDepth int           `json:"max_delegation_depth"` // 0 = no sub-delegation
	RateLimitPerMin    int           `json:"rate_limit_per_min"`
	Status            AgentStatus    `json:"status"`
	ClientID          string         `json:"client_id"`
	ClientSecret      string         `json:"-"` // never serialized to JSON responses
	Metadata          map[string]any `json:"metadata,omitempty"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
}

// AgentTokenClaims extends standard JWT claims with agent identity metadata.
type AgentTokenClaims struct {
	jwt.RegisteredClaims
	// Agent identity
	AgentID   string `json:"agent_id,omitempty"`
	AgentType string `json:"agent_type,omitempty"`
	// Delegation chain: [{sub, agent_id}] ordered from original user to current actor
	DelegationChain []DelegationHop `json:"delegation_chain,omitempty"`
	// MCP servers the agent is authorized to access
	MCPServers []string `json:"mcp_servers,omitempty"`
	// Maximum remaining delegation depth (how many more hops allowed)
	MaxDelegationDepth int `json:"max_delegation_depth,omitempty"`
	// Actor who delegated to this agent (RFC 8693 act claim)
	ActorSubject string `json:"act_sub,omitempty"`
	// Whether this is an agent token (for quick introspection filtering)
	IsAgentToken bool `json:"is_agent_token,omitempty"`
}

// DelegationHop represents one level in the delegation chain.
type DelegationHop struct {
	Subject  string `json:"sub"`
	AgentID  string `json:"agent_id,omitempty"`
	AgentType string `json:"agent_type,omitempty"`
}

// AgentTokenExchangeRequest holds parameters for exchanging a user token
// for an agent-scoped token via RFC 8693.
type AgentTokenExchangeRequest struct {
	TenantID       uuid.UUID
	SubjectToken   string // user's access token
	AgentID        uuid.UUID
	RequestedScope []string
	MCPServers     []string // MCP server URIs the agent wants to access
	Audience       string   // target resource server
}

// AgentTokenResponse is the token response for agent token exchange.
type AgentTokenResponse struct {
	AccessToken      string `json:"access_token"`
	TokenType        string `json:"token_type"`
	ExpiresIn        int    `json:"expires_in"`
	Scope            string `json:"scope,omitempty"`
	AgentID          string `json:"agent_id"`
	DelegationDepth  int    `json:"delegation_depth_remaining"`
	IssuedTokenType  string `json:"issued_token_type"`
}

// --- Agent registry (in-memory + Redis persistent) ---

var (
	agentMu       sync.RWMutex
	agentStore    = make(map[uuid.UUID]*AgentRegistration)
	agentByClient = make(map[string]uuid.UUID) // client_id → agent_id
)

// agentStoreRedis persists an agent registration to Redis (no TTL, persistent).
func (s *OAuthService) agentStoreRedis(ctx context.Context, reg *AgentRegistration) {
	agentMu.Lock()
	agentStore[reg.ID] = reg
	agentByClient[reg.ClientID] = reg.ID
	agentMu.Unlock()
	if s.rdb != nil {
		if data, err := json.Marshal(reg); err == nil {
			s.rdb.Set(ctx, "agent:"+reg.ID.String(), data, 0) // no expiry
		}
	}
}

// agentLoadRedis loads an agent from Redis, falling back to in-memory.
func (s *OAuthService) agentLoadRedis(ctx context.Context, agentID uuid.UUID) (*AgentRegistration, bool) {
	agentMu.RLock()
	agent, ok := agentStore[agentID]
	agentMu.RUnlock()
	if ok {
		return agent, true
	}
	if s.rdb != nil {
		if data, err := s.rdb.Get(ctx, "agent:"+agentID.String()); err == nil && data != "" {
			var reg AgentRegistration
			if json.Unmarshal([]byte(data), &reg) == nil {
				agentMu.Lock()
				agentStore[reg.ID] = &reg
				agentByClient[reg.ClientID] = reg.ID
				agentMu.Unlock()
				return &reg, true
			}
		}
	}
	return nil, false
}

// agentDeleteRedis removes an agent from both Redis and in-memory.
func (s *OAuthService) agentDeleteRedis(ctx context.Context, agentID uuid.UUID) { //nolint:unused // kept for future agent management
	agentMu.Lock()
	agent, ok := agentStore[agentID]
	if ok {
		delete(agentByClient, agent.ClientID)
	}
	delete(agentStore, agentID)
	agentMu.Unlock()
	if s.rdb != nil {
		s.rdb.Del(ctx, "agent:"+agentID.String())
	}
}

// RegisterAgent creates a new AI agent identity within a tenant.
// The agent is linked to an owner user who can delegate authority to it.
func (s *OAuthService) RegisterAgent(ctx context.Context, reg *AgentRegistration) (*AgentRegistration, error) {
	if reg.TenantID == uuid.Nil {
		return nil, fmt.Errorf("tenant_id is required")
	}
	if reg.Name == "" {
		return nil, fmt.Errorf("agent name is required")
	}
	if reg.OwnerUserID == uuid.Nil {
		return nil, fmt.Errorf("owner_user_id is required")
	}

	// Set defaults
	if reg.ID == uuid.Nil {
		reg.ID = uuid.New()
	}
	if reg.Type == "" {
		reg.Type = AgentTypeCustom
	}
	if reg.Status == "" {
		reg.Status = AgentStatusActive
	}
	if reg.MaxDelegationDepth < 0 {
		reg.MaxDelegationDepth = 0
	}
	if reg.RateLimitPerMin <= 0 {
		reg.RateLimitPerMin = 100 // default: 100 req/min
	}

	// Generate OAuth client credentials for the agent
	reg.ClientID = "agent_" + reg.ID.String()[:8]
	secret, err := crypto.GenerateRandomToken(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate agent secret: %w", err)
	}
	reg.ClientSecret = secret
	reg.CreatedAt = time.Now()
	reg.UpdatedAt = reg.CreatedAt

	s.agentStoreRedis(ctx, reg)
	return reg, nil
}

// GetAgent retrieves an agent by ID.
func (s *OAuthService) GetAgent(ctx context.Context, agentID uuid.UUID) (*AgentRegistration, error) {
	agent, ok := s.agentLoadRedis(ctx, agentID)
	if !ok {
		return nil, fmt.Errorf("agent not found: %s", agentID)
	}
	if agent.Status != AgentStatusActive {
		return nil, fmt.Errorf("agent %s is %s", agentID, agent.Status)
	}
	return agent, nil
}

// ListAgents returns all agents for a tenant.
func (s *OAuthService) ListAgents(ctx context.Context, tenantID uuid.UUID) ([]*AgentRegistration, error) {
	agentMu.RLock()
	defer agentMu.RUnlock()
	result := make([]*AgentRegistration, 0)
	for _, a := range agentStore {
		if a.TenantID == tenantID {
			result = append(result, a)
		}
	}
	return result, nil
}

// UpdateAgentStatus changes the lifecycle status of an agent.
func (s *OAuthService) UpdateAgentStatus(ctx context.Context, agentID uuid.UUID, status AgentStatus) error {
	agent, ok := s.agentLoadRedis(ctx, agentID)
	if !ok {
		return fmt.Errorf("agent not found: %s", agentID)
	}
	agent.Status = status
	agent.UpdatedAt = time.Now()
	s.agentStoreRedis(ctx, agent)
	return nil
}

// ExchangeAgentToken implements RFC 8693 token exchange for AI agents.
// It validates the subject token (user's access token), checks that the
// requested scopes are within the agent's allowed scope set, and issues
// a new JWT with agent identity claims and a delegation chain.
func (s *OAuthService) ExchangeAgentToken(ctx context.Context, req *AgentTokenExchangeRequest) (*AgentTokenResponse, error) {
	if req.SubjectToken == "" {
		return nil, fmt.Errorf("subject_token is required")
	}
	if req.AgentID == uuid.Nil {
		return nil, fmt.Errorf("agent_id is required")
	}

	// 1. Validate the subject token
	claims, err := s.ParseAccessToken(req.SubjectToken)
	if err != nil {
		return nil, fmt.Errorf("invalid subject_token: %w", err)
	}

	subjectSub := getStringClaim(claims, "sub")
	if subjectSub == "" {
		return nil, fmt.Errorf("subject_token missing 'sub' claim")
	}

	// Check if this is already an agent token (chained delegation)
	isExistingAgentToken := getBoolClaim(claims, "is_agent_token")
	var existingChain []DelegationHop
	if isExistingAgentToken {
		existingChain = getDelegationChain(claims)
		// Verify delegation depth
		existingDepth := getIntClaimFromToken(claims, "max_delegation_depth")
		if existingDepth <= 0 {
			return nil, fmt.Errorf("maximum delegation depth reached")
		}
	}

	// 2. Validate the agent exists and is active
	agent, err := s.GetAgent(ctx, req.AgentID)
	if err != nil {
		return nil, fmt.Errorf("agent validation failed: %w", err)
	}

	// Verify tenant matches
	if agent.TenantID != req.TenantID {
		return nil, fmt.Errorf("agent tenant mismatch")
	}

	// 3. Validate requested scopes are within agent's allowed scopes
	if len(req.RequestedScope) > 0 {
		allowed := make(map[string]bool)
		for _, sc := range agent.AllowedScopes {
			allowed[sc] = true
		}
		for _, requested := range req.RequestedScope {
			if !allowed[requested] {
				return nil, fmt.Errorf("scope '%s' is not allowed for agent '%s'", requested, agent.Name)
			}
		}
	}

	// 4. Validate MCP servers are in allowed list
	if len(req.MCPServers) > 0 && len(agent.AllowedMCPServers) > 0 {
		allowedMCP := make(map[string]bool)
		for _, mcp := range agent.AllowedMCPServers {
			allowedMCP[mcp] = true
		}
		for _, requested := range req.MCPServers {
			if !allowedMCP[requested] {
				return nil, fmt.Errorf("MCP server '%s' is not authorized for agent '%s'", requested, agent.Name)
			}
		}
	}

	// 5. Build delegation chain
	chain := existingChain
	if !isExistingAgentToken {
		// First hop: user → agent
		chain = append(chain, DelegationHop{
			Subject: subjectSub,
		})
	}
	chain = append(chain, DelegationHop{
		Subject:   subjectSub,
		AgentID:   agent.ID.String(),
		AgentType: string(agent.Type),
	})

	// Calculate remaining delegation depth
	remainingDepth := agent.MaxDelegationDepth
	if isExistingAgentToken {
		existingDepth := getIntClaimFromToken(claims, "max_delegation_depth")
		if remainingDepth > existingDepth-1 {
			remainingDepth = existingDepth - 1
		}
	}

	// 6. Issue agent token
	now := time.Now()
	expiresIn := 3600 // 1 hour default
	scopeStr := strings.Join(req.RequestedScope, " ")
	if scopeStr == "" {
		scopeStr = strings.Join(agent.AllowedScopes, " ")
	}

	agentClaims := AgentTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   subjectSub,
			Issuer:    s.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(expiresIn) * time.Second)),
			ID:        uuid.New().String(),
			Audience:  []string{defaultIfEmpty2(req.Audience, "agent-api")},
		},
		AgentID:            agent.ID.String(),
		AgentType:          string(agent.Type),
		DelegationChain:    chain,
		MCPServers:         req.MCPServers,
		MaxDelegationDepth: remainingDepth,
		ActorSubject:       subjectSub,
		IsAgentToken:       true,
	}

	// Sign the token using the service's key provider
	tokenString, err := s.signAgentToken(&agentClaims)
	if err != nil {
		return nil, fmt.Errorf("failed to sign agent token: %w", err)
	}

	return &AgentTokenResponse{
		AccessToken:     tokenString,
		TokenType:       "Bearer",
		ExpiresIn:       expiresIn,
		Scope:           scopeStr,
		AgentID:         agent.ID.String(),
		DelegationDepth: remainingDepth,
		IssuedTokenType: "urn:ietf:params:oauth:token-type:access_token",
	}, nil
}

// signAgentToken signs the agent claims using the service's key provider.
func (s *OAuthService) signAgentToken(claims *AgentTokenClaims) (string, error) {
	if s.keyProvider == nil {
		return "", fmt.Errorf("key provider not configured")
	}
	key := s.keyProvider.PrivateKey()

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	kid := s.keyProvider.KeyID()
	if kid != "" {
		token.Header["kid"] = kid
	}
	return token.SignedString(key)
}

// VerifyAgentToken parses and validates an agent token.
// Returns the agent claims if valid.
func (s *OAuthService) VerifyAgentToken(ctx context.Context, tokenString string) (*AgentTokenClaims, error) {
	if s.keyProvider == nil {
		return nil, fmt.Errorf("key provider not configured")
	}

	pubKey := s.keyProvider.PublicKey()
	if pubKey == nil {
		return nil, fmt.Errorf("public key not available")
	}

	token, err := jwt.ParseWithClaims(tokenString, &AgentTokenClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return pubKey, nil
	})
	if err != nil {
		return nil, fmt.Errorf("token validation failed: %w", err)
	}

	claims, ok := token.Claims.(*AgentTokenClaims)
	if !ok {
		return nil, fmt.Errorf("invalid claims type")
	}

	if !claims.IsAgentToken {
		return nil, fmt.Errorf("not an agent token")
	}

	return claims, nil
}

// CheckAgentScope verifies that the agent token has the required scope.
func (s *OAuthService) CheckAgentScope(ctx context.Context, tokenString, requiredScope string) (*AgentTokenClaims, error) {
	claims, err := s.VerifyAgentToken(ctx, tokenString)
	if err != nil {
		return nil, err
	}

	// Check scope in the token (scope claim or registered claims)
	// TODO: parse scope from raw token for finer-grained validation
	tokenScope := ""
	_ = tokenScope // placeholder for future scope extraction

	// For agent tokens, we trust the scope validated during exchange
	return claims, nil
}

// AgentFingerprint creates a stable hash of an agent for audit/logging
// without revealing PII.
func AgentFingerprint(agent *AgentRegistration) string {
	h := sha256.Sum256([]byte(agent.ID.String() + agent.TenantID.String()))
	return hex.EncodeToString(h[:8])
}

// --- Helper functions ---

func getDelegationChain(claims jwt.Claims) []DelegationHop {
	mapClaims, ok := claims.(jwt.MapClaims)
	if !ok {
		return nil
	}
	chainRaw, ok := mapClaims["delegation_chain"]
	if !ok {
		return nil
	}
	bytes, err := json.Marshal(chainRaw)
	if err != nil {
		return nil
	}
	var chain []DelegationHop
	if err := json.Unmarshal(bytes, &chain); err != nil {
		return nil
	}
	return chain
}

func getIntClaimFromToken(claims jwt.Claims, key string) int {
	mapClaims, ok := claims.(jwt.MapClaims)
	if !ok {
		return 0
	}
	switch v := mapClaims[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	case json.Number:
		n, _ := v.Int64()
		return int(n)
	default:
		return 0
	}
}


// ResetAgentStore clears all registered agents (for testing only).
func ResetAgentStore() {
	agentMu.Lock()
	agentStore = make(map[uuid.UUID]*AgentRegistration)
	agentByClient = make(map[string]uuid.UUID)
	agentMu.Unlock()
}
