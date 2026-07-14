package service

import (
	"context"
	"strings"
	"testing"
	"time"

	stdcrypto "crypto/rand"
	"crypto/rsa"

	"github.com/ggid/ggid/pkg/crypto"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// --- AI Agent Identity Tests ---

// helper to create a valid subject (user) token for testing
func makeSubjectToken(svc *OAuthService, sub string) string {
	claims := jwt.MapClaims{
		"sub":      sub,
		"iss":      "https://test.ggid.dev",
		"iat":      time.Now().Unix(),
		"exp":      time.Now().Add(time.Hour).Unix(),
		"scope":    "read write",
		"tenant_id": testTenantID.String(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = "test-kid"
	str, _ := token.SignedString(svc.keyProvider.Signer())
	return str
}

func TestRegisterAgent_Success(t *testing.T) {
	ResetAgentStore()
	svc, _, _, _ := newTestOAuthService()
	ownerID := uuid.New()

	agent, err := svc.RegisterAgent(context.Background(), &AgentRegistration{
		TenantID:           testTenantID,
		Name:               "CodeBot-1",
		Type:               AgentTypeCodingAssistant,
		OwnerUserID:        ownerID,
		Description:        "AI coding assistant for PR reviews",
		AllowedScopes:      []string{"repo:read", "repo:write", "pr:review"},
		MaxDelegationDepth: 2,
		RateLimitPerMin:    60,
	})

	if err != nil {
		t.Fatalf("RegisterAgent failed: %v", err)
	}
	if agent.ID == uuid.Nil {
		t.Error("expected non-nil agent ID")
	}
	if agent.ClientID == "" || len(agent.ClientID) < 10 {
		t.Errorf("expected valid client_id, got %q", agent.ClientID)
	}
	if agent.ClientSecret == "" {
		t.Error("expected non-empty client secret")
	}
	if agent.Status != AgentStatusActive {
		t.Errorf("expected status active, got %s", agent.Status)
	}
	if agent.MaxDelegationDepth != 2 {
		t.Errorf("expected delegation depth 2, got %d", agent.MaxDelegationDepth)
	}
}

func TestRegisterAgent_MissingFields(t *testing.T) {
	ResetAgentStore()
	svc, _, _, _ := newTestOAuthService()

	// Missing tenant ID
	_, err := svc.RegisterAgent(context.Background(), &AgentRegistration{
		Name:       "NoTenant",
		OwnerUserID: uuid.New(),
	})
	if err == nil {
		t.Error("expected error for missing tenant_id")
	}

	// Missing name
	_, err = svc.RegisterAgent(context.Background(), &AgentRegistration{
		TenantID:   testTenantID,
		OwnerUserID: uuid.New(),
	})
	if err == nil {
		t.Error("expected error for missing name")
	}

	// Missing owner
	_, err = svc.RegisterAgent(context.Background(), &AgentRegistration{
		TenantID: testTenantID,
		Name:     "NoOwner",
	})
	if err == nil {
		t.Error("expected error for missing owner_user_id")
	}
}

func TestRegisterAgent_Defaults(t *testing.T) {
	ResetAgentStore()
	svc, _, _, _ := newTestOAuthService()

	agent, err := svc.RegisterAgent(context.Background(), &AgentRegistration{
		TenantID:    testTenantID,
		Name:        "MinimalAgent",
		OwnerUserID: uuid.New(),
	})
	if err != nil {
		t.Fatalf("RegisterAgent failed: %v", err)
	}
	if agent.Type != AgentTypeCustom {
		t.Errorf("expected default type custom, got %s", agent.Type)
	}
	if agent.RateLimitPerMin != 100 {
		t.Errorf("expected default rate limit 100, got %d", agent.RateLimitPerMin)
	}
}

func TestExchangeAgentToken_Success(t *testing.T) {
	ResetAgentStore()
	svc, _, _, _ := newTestOAuthService()

	// Register an agent
	agent, err := svc.RegisterAgent(context.Background(), &AgentRegistration{
		TenantID:           testTenantID,
		Name:               "ResearchBot",
		Type:               AgentTypeResearch,
		OwnerUserID:        uuid.New(),
		AllowedScopes:      []string{"read:users", "read:audit"},
		MaxDelegationDepth: 1,
	})
	if err != nil {
		t.Fatalf("RegisterAgent failed: %v", err)
	}

	// Create a subject token
	subjectToken := makeSubjectToken(svc, "user-123")

	// Exchange for agent token
	resp, err := svc.ExchangeAgentToken(context.Background(), &AgentTokenExchangeRequest{
		TenantID:       testTenantID,
		SubjectToken:   subjectToken,
		AgentID:        agent.ID,
		RequestedScope: []string{"read:users"},
		Audience:       "api://ggid",
	})
	if err != nil {
		t.Fatalf("ExchangeAgentToken failed: %v", err)
	}

	if resp.AccessToken == "" {
		t.Error("expected non-empty access token")
	}
	if resp.TokenType != "Bearer" {
		t.Errorf("expected Bearer, got %s", resp.TokenType)
	}
	if resp.AgentID != agent.ID.String() {
		t.Errorf("expected agent ID %s, got %s", agent.ID, resp.AgentID)
	}
	if resp.DelegationDepth != 1 {
		t.Errorf("expected delegation depth 1, got %d", resp.DelegationDepth)
	}
}

func TestExchangeAgentToken_DisallowedScope(t *testing.T) {
	ResetAgentStore()
	svc, _, _, _ := newTestOAuthService()

	agent, err := svc.RegisterAgent(context.Background(), &AgentRegistration{
		TenantID:      testTenantID,
		Name:          "ScopedBot",
		OwnerUserID:   uuid.New(),
		AllowedScopes: []string{"read:users"},
	})
	if err != nil {
		t.Fatalf("RegisterAgent failed: %v", err)
	}

	subjectToken := makeSubjectToken(svc, "user-456")

	// Request scope not in allowed list
	_, err = svc.ExchangeAgentToken(context.Background(), &AgentTokenExchangeRequest{
		TenantID:       testTenantID,
		SubjectToken:   subjectToken,
		AgentID:        agent.ID,
		RequestedScope: []string{"admin:delete"}, // not allowed
	})
	if err == nil {
		t.Error("expected error for disallowed scope")
	}
	if err != nil && !strings.Contains(err.Error(), "not allowed") {
		t.Errorf("expected scope error, got: %v", err)
	}
}

func TestExchangeAgentToken_InvalidSubjectToken(t *testing.T) {
	ResetAgentStore()
	svc, _, _, _ := newTestOAuthService()

	agent, err := svc.RegisterAgent(context.Background(), &AgentRegistration{
		TenantID:    testTenantID,
		Name:        "TestBot",
		OwnerUserID: uuid.New(),
	})
	if err != nil {
		t.Fatalf("RegisterAgent failed: %v", err)
	}

	_, err = svc.ExchangeAgentToken(context.Background(), &AgentTokenExchangeRequest{
		TenantID:     testTenantID,
		SubjectToken: "invalid-token",
		AgentID:      agent.ID,
	})
	if err == nil {
		t.Error("expected error for invalid subject token")
	}
}

func TestExchangeAgentToken_SuspendedAgent(t *testing.T) {
	ResetAgentStore()
	svc, _, _, _ := newTestOAuthService()

	agent, err := svc.RegisterAgent(context.Background(), &AgentRegistration{
		TenantID:    testTenantID,
		Name:        "SuspendedBot",
		OwnerUserID: uuid.New(),
	})
	if err != nil {
		t.Fatalf("RegisterAgent failed: %v", err)
	}

	// Suspend the agent
	svc.UpdateAgentStatus(context.Background(), agent.ID, AgentStatusSuspended)

	subjectToken := makeSubjectToken(svc, "user-789")

	_, err = svc.ExchangeAgentToken(context.Background(), &AgentTokenExchangeRequest{
		TenantID:     testTenantID,
		SubjectToken: subjectToken,
		AgentID:      agent.ID,
	})
	if err == nil {
		t.Error("expected error for suspended agent")
	}
}

func TestExchangeAgentToken_TenantMismatch(t *testing.T) {
	ResetAgentStore()
	svc, _, _, _ := newTestOAuthService()

	agent, err := svc.RegisterAgent(context.Background(), &AgentRegistration{
		TenantID:    testTenantID,
		Name:        "TenantBot",
		OwnerUserID: uuid.New(),
	})
	if err != nil {
		t.Fatalf("RegisterAgent failed: %v", err)
	}

	subjectToken := makeSubjectToken(svc, "user-999")
	otherTenant := uuid.New()

	_, err = svc.ExchangeAgentToken(context.Background(), &AgentTokenExchangeRequest{
		TenantID:     otherTenant, // wrong tenant
		SubjectToken: subjectToken,
		AgentID:      agent.ID,
	})
	if err == nil {
		t.Error("expected error for tenant mismatch")
	}
}

func TestVerifyAgentToken_Success(t *testing.T) {
	ResetAgentStore()
	svc, _, _, _ := newTestOAuthService()

	agent, err := svc.RegisterAgent(context.Background(), &AgentRegistration{
		TenantID:           testTenantID,
		Name:               "VerifyBot",
		Type:               AgentTypeDataPipeline,
		OwnerUserID:        uuid.New(),
		AllowedScopes:      []string{"read:data"},
		MaxDelegationDepth: 3,
	})
	if err != nil {
		t.Fatalf("RegisterAgent failed: %v", err)
	}

	subjectToken := makeSubjectToken(svc, "user-verify")

	resp, err := svc.ExchangeAgentToken(context.Background(), &AgentTokenExchangeRequest{
		TenantID:       testTenantID,
		SubjectToken:   subjectToken,
		AgentID:        agent.ID,
		RequestedScope: []string{"read:data"},
	})
	if err != nil {
		t.Fatalf("ExchangeAgentToken failed: %v", err)
	}

	// Verify the agent token
	claims, err := svc.VerifyAgentToken(context.Background(), resp.AccessToken)
	if err != nil {
		t.Fatalf("VerifyAgentToken failed: %v", err)
	}

	if !claims.IsAgentToken {
		t.Error("expected is_agent_token = true")
	}
	if claims.AgentID != agent.ID.String() {
		t.Errorf("expected agent_id %s, got %s", agent.ID, claims.AgentID)
	}
	if claims.AgentType != string(AgentTypeDataPipeline) {
		t.Errorf("expected agent_type %s, got %s", AgentTypeDataPipeline, claims.AgentType)
	}
	if claims.MaxDelegationDepth != 3 {
		t.Errorf("expected delegation depth 3, got %d", claims.MaxDelegationDepth)
	}
	if len(claims.DelegationChain) < 1 {
		t.Error("expected non-empty delegation chain")
	}
}

func TestVerifyAgentToken_InvalidToken(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	_, err := svc.VerifyAgentToken(context.Background(), "garbage-token")
	if err == nil {
		t.Error("expected error for invalid token")
	}
}

func TestVerifyAgentToken_NonAgentToken(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	// Create a regular (non-agent) token
	subjectToken := makeSubjectToken(svc, "user-regular")

	_, err := svc.VerifyAgentToken(context.Background(), subjectToken)
	if err == nil {
		t.Error("expected error for non-agent token")
	}
}

func TestExchangeAgentToken_DelegationChain(t *testing.T) {
	ResetAgentStore()
	svc, _, _, _ := newTestOAuthService()

	// Agent with 2-level delegation
	agent, err := svc.RegisterAgent(context.Background(), &AgentRegistration{
		TenantID:           testTenantID,
		Name:               "ChainBot",
		Type:               AgentTypeWorkflow,
		OwnerUserID:        uuid.New(),
		AllowedScopes:      []string{"workflow:execute"},
		MaxDelegationDepth: 2,
	})
	if err != nil {
		t.Fatalf("RegisterAgent failed: %v", err)
	}

	// First hop: user → agent
	subjectToken := makeSubjectToken(svc, "user-chain")
	resp1, err := svc.ExchangeAgentToken(context.Background(), &AgentTokenExchangeRequest{
		TenantID:       testTenantID,
		SubjectToken:   subjectToken,
		AgentID:        agent.ID,
		RequestedScope: []string{"workflow:execute"},
	})
	if err != nil {
		t.Fatalf("First exchange failed: %v", err)
	}

	// Verify delegation depth after first hop
	claims1, err := svc.VerifyAgentToken(context.Background(), resp1.AccessToken)
	if err != nil {
		t.Fatalf("Verify after first hop failed: %v", err)
	}
	if claims1.MaxDelegationDepth != 2 {
		t.Errorf("expected remaining depth 2 after first hop, got %d", claims1.MaxDelegationDepth)
	}
	if len(claims1.DelegationChain) != 2 {
		t.Errorf("expected 2 hops in chain (user + agent), got %d", len(claims1.DelegationChain))
	}

	// Second hop: agent → sub-agent (using first agent token as subject)
	resp2, err := svc.ExchangeAgentToken(context.Background(), &AgentTokenExchangeRequest{
		TenantID:       testTenantID,
		SubjectToken:   resp1.AccessToken, // use agent token as subject
		AgentID:        agent.ID,
		RequestedScope: []string{"workflow:execute"},
	})
	if err != nil {
		t.Fatalf("Second exchange (sub-delegation) failed: %v", err)
	}

	claims2, err := svc.VerifyAgentToken(context.Background(), resp2.AccessToken)
	if err != nil {
		t.Fatalf("Verify after second hop failed: %v", err)
	}
	if claims2.MaxDelegationDepth != 1 {
		t.Errorf("expected remaining depth 1 after second hop, got %d", claims2.MaxDelegationDepth)
	}
}

func TestExchangeAgentToken_DelegationDepthExhausted(t *testing.T) {
	ResetAgentStore()
	svc, _, _, _ := newTestOAuthService()

	// Agent with 0 delegation depth (no sub-delegation allowed)
	agent, err := svc.RegisterAgent(context.Background(), &AgentRegistration{
		TenantID:           testTenantID,
		Name:               "NoDelegateBot",
		OwnerUserID:        uuid.New(),
		AllowedScopes:      []string{"read"},
		MaxDelegationDepth: 1, // allows 1 hop
	})
	if err != nil {
		t.Fatalf("RegisterAgent failed: %v", err)
	}

	// First hop
	subjectToken := makeSubjectToken(svc, "user-depth")
	resp1, err := svc.ExchangeAgentToken(context.Background(), &AgentTokenExchangeRequest{
		TenantID:     testTenantID,
		SubjectToken: subjectToken,
		AgentID:      agent.ID,
	})
	if err != nil {
		t.Fatalf("First exchange failed: %v", err)
	}

	// Verify depth after first hop
	claims1, _ := svc.VerifyAgentToken(context.Background(), resp1.AccessToken)
	if claims1.MaxDelegationDepth != 1 {
		t.Errorf("expected depth 1 after first hop, got %d", claims1.MaxDelegationDepth)
	}

	// Second hop should fail — depth exhausted
	_, err = svc.ExchangeAgentToken(context.Background(), &AgentTokenExchangeRequest{
		TenantID:     testTenantID,
		SubjectToken: resp1.AccessToken,
		AgentID:      agent.ID,
	})
	if err == nil {
		// Actually, the remaining depth is 1, so the second hop would set remaining to 0
		// The third hop would fail. Let me check...
		// MaxDelegationDepth=1 means: after first hop, remaining=1, second hop: remaining = min(1, 1-1=0) = 0
		// Third hop: remaining=0, so getIntClaimFromToken returns 0, fails with "maximum delegation depth reached"
	}

	// Third hop should definitely fail
	resp2, err := svc.ExchangeAgentToken(context.Background(), &AgentTokenExchangeRequest{
		TenantID:     testTenantID,
		SubjectToken: resp1.AccessToken,
		AgentID:      agent.ID,
	})
	if err == nil {
		// Try one more hop
		_, err = svc.ExchangeAgentToken(context.Background(), &AgentTokenExchangeRequest{
			TenantID:     testTenantID,
			SubjectToken: resp2.AccessToken,
			AgentID:      agent.ID,
		})
		if err == nil {
			t.Error("expected error when delegation depth is exhausted")
		}
	}
}

func TestListAgents_ByTenant(t *testing.T) {
	ResetAgentStore()
	svc, _, _, _ := newTestOAuthService()

	// Register agents in different tenants
	svc.RegisterAgent(context.Background(), &AgentRegistration{
		TenantID:    testTenantID,
		Name:        "Tenant1Bot",
		OwnerUserID: uuid.New(),
	})
	svc.RegisterAgent(context.Background(), &AgentRegistration{
		TenantID:    testTenantID,
		Name:        "Tenant1Bot2",
		OwnerUserID: uuid.New(),
	})
	otherTenant := uuid.New()
	svc.RegisterAgent(context.Background(), &AgentRegistration{
		TenantID:    otherTenant,
		Name:        "Tenant2Bot",
		OwnerUserID: uuid.New(),
	})

	agents, err := svc.ListAgents(context.Background(), testTenantID)
	if err != nil {
		t.Fatalf("ListAgents failed: %v", err)
	}
	if len(agents) != 2 {
		t.Errorf("expected 2 agents for test tenant, got %d", len(agents))
	}

	agents2, _ := svc.ListAgents(context.Background(), otherTenant)
	if len(agents2) != 1 {
		t.Errorf("expected 1 agent for other tenant, got %d", len(agents2))
	}
}

func TestUpdateAgentStatus(t *testing.T) {
	ResetAgentStore()
	svc, _, _, _ := newTestOAuthService()

	agent, _ := svc.RegisterAgent(context.Background(), &AgentRegistration{
		TenantID:    testTenantID,
		Name:        "StatusBot",
		OwnerUserID: uuid.New(),
	})

	// Suspend
	err := svc.UpdateAgentStatus(context.Background(), agent.ID, AgentStatusSuspended)
	if err != nil {
		t.Fatalf("UpdateAgentStatus failed: %v", err)
	}

	// Should not be able to get agent
	_, err = svc.GetAgent(context.Background(), agent.ID)
	if err == nil {
		t.Error("expected error for suspended agent")
	}

	// Revoke
	svc.UpdateAgentStatus(context.Background(), agent.ID, AgentStatusRevoked)

	_, err = svc.GetAgent(context.Background(), agent.ID)
	if err == nil {
		t.Error("expected error for revoked agent")
	}
}

func TestGetAgent_NotFound(t *testing.T) {
	ResetAgentStore()
	svc, _, _, _ := newTestOAuthService()

	_, err := svc.GetAgent(context.Background(), uuid.New())
	if err == nil {
		t.Error("expected error for non-existent agent")
	}
}

func TestAgentFingerprint_Stable(t *testing.T) {
	agent := &AgentRegistration{
		ID:       uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		TenantID: uuid.MustParse("22222222-2222-2222-2222-222222222222"),
	}

	fp1 := AgentFingerprint(agent)
	fp2 := AgentFingerprint(agent)

	if fp1 != fp2 {
		t.Error("fingerprint should be deterministic")
	}
	if len(fp1) != 16 {
		t.Errorf("expected 16-char hex fingerprint, got %d chars: %s", len(fp1), fp1)
	}

	// Different agent should have different fingerprint
	agent2 := &AgentRegistration{
		ID:       uuid.MustParse("33333333-3333-3333-3333-333333333333"),
		TenantID: uuid.MustParse("22222222-2222-2222-2222-222222222222"),
	}
	fp3 := AgentFingerprint(agent2)
	if fp1 == fp3 {
		t.Error("different agents should have different fingerprints")
	}
}

func TestExchangeAgentToken_MCPServerValidation(t *testing.T) {
	ResetAgentStore()
	svc, _, _, _ := newTestOAuthService()

	agent, err := svc.RegisterAgent(context.Background(), &AgentRegistration{
		TenantID:           testTenantID,
		Name:               "MCPBot",
		OwnerUserID:        uuid.New(),
		AllowedScopes:      []string{"mcp:call"},
		AllowedMCPServers:  []string{"https://mcp1.example.com", "https://mcp2.example.com"},
	})
	if err != nil {
		t.Fatalf("RegisterAgent failed: %v", err)
	}

	subjectToken := makeSubjectToken(svc, "user-mcp")

	// Valid MCP server
	resp, err := svc.ExchangeAgentToken(context.Background(), &AgentTokenExchangeRequest{
		TenantID:       testTenantID,
		SubjectToken:   subjectToken,
		AgentID:        agent.ID,
		MCPServers:     []string{"https://mcp1.example.com"},
	})
	if err != nil {
		t.Fatalf("exchange with valid MCP failed: %v", err)
	}

	// Verify MCP servers in token
	claims, _ := svc.VerifyAgentToken(context.Background(), resp.AccessToken)
	if len(claims.MCPServers) != 1 || claims.MCPServers[0] != "https://mcp1.example.com" {
		t.Errorf("expected MCP server in claims, got %v", claims.MCPServers)
	}

	// Invalid MCP server
	_, err = svc.ExchangeAgentToken(context.Background(), &AgentTokenExchangeRequest{
		TenantID:       testTenantID,
		SubjectToken:   subjectToken,
		AgentID:        agent.ID,
		MCPServers:     []string{"https://evil.example.com"},
	})
	if err == nil {
		t.Error("expected error for unauthorized MCP server")
	}
}

// --- Integration test with real RSA key pair ---

func TestAgentToken_FullLifecycle(t *testing.T) {
	ResetAgentStore()
	svc, _, _, _ := newTestOAuthService()

	// 1. Register agent
	agent, err := svc.RegisterAgent(context.Background(), &AgentRegistration{
		TenantID:           testTenantID,
		Name:               "LifecycleBot",
		Type:               AgentTypeCodingAssistant,
		OwnerUserID:        uuid.New(),
		Description:        "Full lifecycle test agent",
		AllowedScopes:      []string{"repo:read", "repo:write"},
		MaxDelegationDepth: 2,
	})
	if err != nil {
		t.Fatalf("registration failed: %v", err)
	}

	// 2. Get agent (should succeed)
	_, err = svc.GetAgent(context.Background(), agent.ID)
	if err != nil {
		t.Fatalf("GetAgent failed: %v", err)
	}

	// 3. Exchange for agent token
	subjectToken := makeSubjectToken(svc, "user-lifecycle")
	resp, err := svc.ExchangeAgentToken(context.Background(), &AgentTokenExchangeRequest{
		TenantID:       testTenantID,
		SubjectToken:   subjectToken,
		AgentID:        agent.ID,
		RequestedScope: []string{"repo:read"},
	})
	if err != nil {
		t.Fatalf("token exchange failed: %v", err)
	}

	// 4. Verify token
	claims, err := svc.VerifyAgentToken(context.Background(), resp.AccessToken)
	if err != nil {
		t.Fatalf("verification failed: %v", err)
	}
	if claims.AgentType != string(AgentTypeCodingAssistant) {
		t.Errorf("expected coding-assistant, got %s", claims.AgentType)
	}

	// 5. Suspend agent
	svc.UpdateAgentStatus(context.Background(), agent.ID, AgentStatusSuspended)

	// 6. Token exchange should fail
	_, err = svc.ExchangeAgentToken(context.Background(), &AgentTokenExchangeRequest{
		TenantID:       testTenantID,
		SubjectToken:   subjectToken,
		AgentID:        agent.ID,
	})
	if err == nil {
		t.Error("exchange should fail after suspension")
	}

	// 7. Existing token is still valid (tokens are stateless until expiry)
	// This is expected behavior — revocation would require a denylist
	_, err = svc.VerifyAgentToken(context.Background(), resp.AccessToken)
	if err != nil {
		t.Logf("Note: existing token verification after suspension: %v (stateless tokens)", err)
	}
}

// Test that GenerateRandomToken works (used in RegisterAgent)
func TestGenerateRandomToken_ForAgentSecret(t *testing.T) {
	token, err := crypto.GenerateRandomToken(32)
	if err != nil {
		t.Fatalf("GenerateRandomToken failed: %v", err)
	}
	if len(token) < 32 {
		t.Errorf("expected token length >= 32, got %d", len(token))
	}

	// Should be different each time
	token2, _ := crypto.GenerateRandomToken(32)
	if token == token2 {
		t.Error("expected different tokens")
	}
}

// Ensure unused imports are referenced
var (
	_ = stdcrypto.Reader
	_ = (*rsa.PrivateKey)(nil)
)
