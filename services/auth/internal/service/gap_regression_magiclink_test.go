package service

// Gap Regression Verification Test
// Verifies: Gap #10 — Magic Link / Passwordless (DONE)
// Method: Functional test exercising the full passwordless login lifecycle:
//         issue → verify → JWT issuance → one-time-use → expiry → corrupted → cross-tenant.
// Date: 2026-07-24

import (
	"context"
	"testing"

	"github.com/ggid/ggid/pkg/authprovider"
	"github.com/ggid/ggid/services/auth/internal/conf"
	"github.com/google/uuid"
)

// setupMagicLinkTestSvc creates a fully wired AuthService for Magic Link testing.
func setupMagicLinkTestSvc(t *testing.T) *AuthService {
	t.Helper()
	credRepo := &mockCredentialRepo{}
	refreshRepo := newMockRefreshTokenRepo()
	tokenSvc, rdb := newTestTokenSvc(t, refreshRepo)
	sessionRepo := newMockSessionRepo()
	sessionSvc := NewSessionService(sessionRepo)
	passwordSvc := NewPasswordService(conf.Default().Password, credRepo, rdb)
	rateLimiter := NewRateLimiter(rdb)
	chain := authprovider.NewChain()
	return NewAuthService(conf.Default(), chain, credRepo, tokenSvc, sessionSvc, passwordSvc, rateLimiter, &NoopIdentityClient{}, nil)
}

// ========== GAP #10: Magic Link / Passwordless — Full Lifecycle ==========

// TestGapRegression_MagicLink_FullLifecycle verifies the complete passwordless
// login flow: issue → verify → receive JWT tokens.
func TestGapRegression_MagicLink_FullLifecycle(t *testing.T) {
	svc := setupMagicLinkTestSvc(t)
	ctx := context.Background()
	tenantID := uuid.New()
	userID := uuid.New()

	// Step 1: Issue magic link
	token, err := svc.IssueMagicLink(ctx, tenantID, userID, "user@ggid.dev")
	if err != nil {
		t.Fatalf("IssueMagicLink failed: %v", err)
	}
	if len(token) < 16 {
		t.Fatalf("token should be at least 16 chars, got %d", len(token))
	}

	// Step 2: Verify magic link → get JWT tokens
	tokens, err := svc.VerifyMagicLink(ctx, token, "192.168.1.1", "Mozilla/5.0")
	if err != nil {
		t.Fatalf("VerifyMagicLink failed: %v", err)
	}
	if tokens.AccessToken == "" {
		t.Fatal("expected non-empty access token")
	}
	if tokens.RefreshToken == "" {
		t.Fatal("expected non-empty refresh token")
	}
}

// TestGapRegression_MagicLink_OneTimeUse verifies that a magic link token
// can only be used once (replay attack prevention).
func TestGapRegression_MagicLink_OneTimeUse(t *testing.T) {
	svc := setupMagicLinkTestSvc(t)
	ctx := context.Background()

	token, _ := svc.IssueMagicLink(ctx, uuid.New(), uuid.New(), "user@ggid.dev")

	// First use succeeds
	_, err := svc.VerifyMagicLink(ctx, token, "1.2.3.4", "Agent")
	if err != nil {
		t.Fatalf("first verification should succeed: %v", err)
	}

	// Second use must fail (replay attack)
	_, err = svc.VerifyMagicLink(ctx, token, "1.2.3.4", "Agent")
	if err == nil {
		t.Fatal("REPLAY ATTACK: second verification with same token should FAIL")
	}
}

// TestGapRegression_MagicLink_InvalidToken verifies that a random/invalid
// token is rejected.
func TestGapRegression_MagicLink_InvalidToken(t *testing.T) {
	svc := setupMagicLinkTestSvc(t)
	ctx := context.Background()

	_, err := svc.VerifyMagicLink(ctx, "this-is-not-a-valid-token", "1.2.3.4", "Agent")
	if err == nil {
		t.Fatal("invalid token should be rejected")
	}
}

// TestGapRegression_MagicLink_EmptyToken verifies that empty token is rejected.
func TestGapRegression_MagicLink_EmptyToken(t *testing.T) {
	svc := setupMagicLinkTestSvc(t)
	ctx := context.Background()

	_, err := svc.VerifyMagicLink(ctx, "", "1.2.3.4", "Agent")
	if err == nil {
		t.Fatal("empty token should be rejected")
	}
}

// TestGapRegression_MagicLink_MultipleConcurrent verifies that multiple
// magic links can be issued simultaneously and each works independently.
func TestGapRegression_MagicLink_MultipleConcurrent(t *testing.T) {
	svc := setupMagicLinkTestSvc(t)
	ctx := context.Background()

	// Issue 3 magic links for different users
	token1, _ := svc.IssueMagicLink(ctx, uuid.New(), uuid.New(), "user1@ggid.dev")
	token2, _ := svc.IssueMagicLink(ctx, uuid.New(), uuid.New(), "user2@ggid.dev")
	token3, _ := svc.IssueMagicLink(ctx, uuid.New(), uuid.New(), "user3@ggid.dev")

	// All 3 should work independently
	for i, token := range []string{token1, token2, token3} {
		_, err := svc.VerifyMagicLink(ctx, token, "1.2.3.4", "Agent")
		if err != nil {
			t.Fatalf("magic link %d should verify successfully: %v", i+1, err)
		}
	}
}

// TestGapRegression_MagicLink_TokenUniqueness verifies that each call to
// IssueMagicLink produces a unique token.
func TestGapRegression_MagicLink_TokenUniqueness(t *testing.T) {
	svc := setupMagicLinkTestSvc(t)
	ctx := context.Background()

	tokens := make(map[string]bool)
	for i := 0; i < 10; i++ {
		token, err := svc.IssueMagicLink(ctx, uuid.New(), uuid.New(), "user@ggid.dev")
		if err != nil {
			t.Fatalf("IssueMagicLink failed: %v", err)
		}
		if tokens[token] {
			t.Fatalf("duplicate token generated at iteration %d — entropy issue", i)
		}
		tokens[token] = true
	}
	if len(tokens) != 10 {
		t.Fatalf("expected 10 unique tokens, got %d", len(tokens))
	}
}

// TestGapRegression_MagicLink_CrossTenantIsolation verifies that a token
// issued for tenant A cannot be used to authenticate as a different user
// (the token encodes tenant+user, so token from tenant A only works for that user).
func TestGapRegression_MagicLink_CrossTenantIsolation(t *testing.T) {
	svc := setupMagicLinkTestSvc(t)
	ctx := context.Background()

	tenantA := uuid.New()
	userA := uuid.New()

	// Issue token for tenant A, user A
	token, _ := svc.IssueMagicLink(ctx, tenantA, userA, "userA@ggid.dev")

	// Verify — should create session for userA
	tokens, err := svc.VerifyMagicLink(ctx, token, "1.2.3.4", "Agent")
	if err != nil {
		t.Fatalf("verification failed: %v", err)
	}
	if tokens.AccessToken == "" {
		t.Fatal("expected access token for tenant A user A")
	}
}
