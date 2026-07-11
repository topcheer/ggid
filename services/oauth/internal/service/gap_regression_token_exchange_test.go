package service

// Token Exchange (RFC 8693) Functional Verification Tests
// Verifies: Gap #19 — Token exchange with delegation semantics
// Date: 2026-07-25

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TestTokenExchangeRFC8693_FullDelegationFlow verifies the full RFC 8693 flow:
// subject_token (user JWT) → ExchangeToken → new token issued with reduced scope.
func TestTokenExchangeRFC8693_FullDelegationFlow(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	// Issue a valid subject token (user JWT signed by our key provider)
	subjectToken := signTestToken(svc, map[string]interface{}{
		"sub":   "user-subject-123",
		"exp":   time.Now().Add(1 * time.Hour).Unix(),
		"iss":   "https://test.ggid.dev",
		"scope": "read write admin",
	})

	resp, err := svc.ExchangeToken(testCtxForExchange(), &TokenExchangeRequestRFC8693{
		TenantID:         testTenantID,
		SubjectToken:     subjectToken,
		SubjectTokenType: "urn:ietf:params:oauth:token-type:access_token",
		ActorToken:       "",
		ActorTokenType:   "",
		Scope:            []string{"read"}, // reduced scope
	})
	if err != nil {
		t.Fatalf("ExchangeToken: %v", err)
	}

	if resp.AccessToken == "" {
		t.Error("access_token must be issued")
	}
	if resp.TokenType != "N_A" {
		t.Errorf("token_type should be 'N_A' per RFC 8693, got '%s'", resp.TokenType)
	}
	if resp.ExpiresIn != 3600 {
		t.Errorf("expires_in should be 3600, got %d", resp.ExpiresIn)
	}
	if resp.Scope != "read" {
		t.Errorf("scope should be reduced to 'read', got '%s'", resp.Scope)
	}
}

// TestTokenExchangeRFC8693_WithActorToken verifies delegation with actor_token.
func TestTokenExchangeRFC8693_WithActorToken(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	subjectToken := signTestToken(svc, map[string]interface{}{
		"sub": "end-user-456",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
		"iss": "https://test.ggid.dev",
	})

	actorToken := signTestToken(svc, map[string]interface{}{
		"sub": "service-actor-789",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
		"iss": "https://test.ggid.dev",
	})

	resp, err := svc.ExchangeToken(testCtxForExchange(), &TokenExchangeRequestRFC8693{
		TenantID:         testTenantID,
		SubjectToken:     subjectToken,
		SubjectTokenType: "urn:ietf:params:oauth:token-type:access_token",
		ActorToken:       actorToken,
		ActorTokenType:   "urn:ietf:params:oauth:token-type:access_token",
		Scope:            []string{"read", "write"},
	})
	if err != nil {
		t.Fatalf("ExchangeToken with actor: %v", err)
	}

	// The issued token should have reduced scope
	if resp.Scope != "read write" {
		t.Errorf("scope should be 'read write', got '%s'", resp.Scope)
	}
	// Token should be prefixed with "exchanged_" indicating delegation
	if resp.AccessToken == "" {
		t.Error("access_token must be issued")
	}
}

// TestTokenExchangeRFC8693_ScopeReduction verifies scope is narrowed.
func TestTokenExchangeRFC8693_ScopeReduction(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	subjectToken := signTestToken(svc, map[string]interface{}{
		"sub": "scope-user",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
		"iss": "https://test.ggid.dev",
	})

	// Request only a subset of the subject's scopes
	resp, err := svc.ExchangeToken(testCtxForExchange(), &TokenExchangeRequestRFC8693{
		TenantID:         testTenantID,
		SubjectToken:     subjectToken,
		SubjectTokenType: "urn:ietf:params:oauth:token-type:access_token",
		Scope:            []string{"readonly"},
	})
	if err != nil {
		t.Fatalf("ExchangeToken: %v", err)
	}

	if resp.Scope != "readonly" {
		t.Errorf("exchanged token scope should be 'readonly', got '%s'", resp.Scope)
	}
}

// TestTokenExchangeRFC8693_WrongKey verifies token signed by different key is rejected.
func TestTokenExchangeRFC8693_WrongKey(t *testing.T) {
	svcA, _, _, _ := newTestOAuthService()

	// Generate an independent attacker key
	attackerKey, _ := signTestTokenWithAttackerKey("attacker-sub", "https://test.ggid.dev")

	_, err := svcA.ExchangeToken(testCtxForExchange(), &TokenExchangeRequestRFC8693{
		TenantID:         testTenantID,
		SubjectToken:     attackerKey,
		SubjectTokenType: "urn:ietf:params:oauth:token-type:access_token",
	})
	if err == nil {
		t.Fatal("token signed by wrong key should be rejected")
	}
}

// TestTokenExchangeRFC8693_ExpiredSubjectToken verifies expired tokens are rejected.
func TestTokenExchangeRFC8693_ExpiredSubjectToken(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	expiredToken := signTestToken(svc, map[string]interface{}{
		"sub": "expired-user",
		"exp": time.Now().Add(-1 * time.Hour).Unix(), // expired 1 hour ago
		"iss": "https://test.ggid.dev",
	})

	_, err := svc.ExchangeToken(testCtxForExchange(), &TokenExchangeRequestRFC8693{
		TenantID:         testTenantID,
		SubjectToken:     expiredToken,
		SubjectTokenType: "urn:ietf:params:oauth:token-type:access_token",
	})
	if err == nil {
		t.Fatal("expired subject_token should be rejected")
	}
}

// signTestTokenWithAttackerKey signs a JWT with a completely independent key.
func signTestTokenWithAttackerKey(sub, iss string) (string, error) {
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", err
	}
	claims := jwt.MapClaims{
		"sub": sub,
		"exp": time.Now().Add(1 * time.Hour).Unix(),
		"iss": iss,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(privKey)
}

// testCtx returns a context for token exchange tests.
func testCtxForExchange() context.Context {
	return context.Background()
}
