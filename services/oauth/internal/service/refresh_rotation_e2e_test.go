package service

import (
	"context"
	"testing"
	"time"

	"github.com/ggid/ggid/services/oauth/internal/domain"
	"github.com/google/uuid"
)

// TestRefreshToken_RotationAndReuseDetection is a comprehensive E2E test
// for the refresh token rotation flow (RFC 6749 §6 + §10.4):
//
// 1. Store a valid refresh token (simulating auth code flow output)
// 2. Refresh → verify new access + refresh tokens returned
// 3. Try reusing OLD refresh token → must fail (rotation enforcement)
// 4. Verify old token is marked Used in the store
func TestRefreshToken_RotationAndReuseDetection(t *testing.T) {
	svc, clientRepo, _, tokenRepo := newTestOAuthService()

	// Create client with refresh_token grant
	client := &domain.OAuthClient{
		ID:         uuid.New(),
		TenantID:   testTenantID,
		ClientID:   "gcid-rotation-test",
		Name:       "Rotation Test",
		Type:       domain.ClientTypePublic,
		GrantTypes: []string{"refresh_token"},
		Enabled:    true,
	}
	_ = clientRepo.CreateClient(context.Background(), client)

	// Store initial refresh token
	initialToken := "initial-refresh-token-value"
	tokenHash := hashTokenSHA256Helper(initialToken)
	userID := uuid.New()
	tokenRepo.refreshTokens = append(tokenRepo.refreshTokens, &domain.RefreshTokenRecord{
		ID:        uuid.New(),
		TenantID:  testTenantID,
		ClientID:  client.ID,
		UserID:    userID,
		TokenHash: tokenHash,
		Scope:     []string{"openid", "profile"},
		ExpiresAt: time.Now().Add(24 * time.Hour),
	})

	// Step 1: Refresh — should succeed and return new tokens
	resp1, err := svc.RefreshToken(context.Background(), &RefreshTokenRequest{
		TenantID:     testTenantID,
		RefreshToken: initialToken,
		ClientID:     "gcid-rotation-test",
		Scope:        []string{"openid"},
	})
	if err != nil {
		t.Fatalf("Step 1: refresh should succeed: %v", err)
	}
	if resp1.AccessToken == "" {
		t.Error("Step 1: access token should be present")
	}
	if resp1.RefreshToken == "" {
		t.Error("Step 1: new refresh token should be present")
	}
	if resp1.RefreshToken == initialToken {
		t.Error("Step 1: new refresh token must differ from old (rotation)")
	}

	// Step 2: Try reusing the OLD refresh token — must fail
	_, err = svc.RefreshToken(context.Background(), &RefreshTokenRequest{
		TenantID:     testTenantID,
		RefreshToken: initialToken,
		ClientID:     "gcid-rotation-test",
		Scope:        []string{"openid"},
	})
	if err == nil {
		t.Error("Step 2: old refresh token reuse must fail")
	}

	// Step 3: Verify old token is marked Used in the store
	oldRecord, _ := tokenRepo.GetRefreshToken(nil, testTenantID, tokenHash)
	if oldRecord != nil && !oldRecord.Used && !oldRecord.Revoked {
		t.Error("Step 3: old refresh token should be marked Used or Revoked")
	}
}

// TestRefreshToken_ExpiredTokenRejected verifies that expired refresh
// tokens are rejected.
func TestRefreshToken_ExpiredTokenRejected(t *testing.T) {
	svc, clientRepo, _, tokenRepo := newTestOAuthService()

	client := &domain.OAuthClient{
		ID:         uuid.New(),
		TenantID:   testTenantID,
		ClientID:   "gcid-expired-test",
		Name:       "Expired Test",
		Type:       domain.ClientTypePublic,
		GrantTypes: []string{"refresh_token"},
		Enabled:    true,
	}
	_ = clientRepo.CreateClient(context.Background(), client)

	expiredToken := "expired-refresh-token"
	tokenRepo.refreshTokens = append(tokenRepo.refreshTokens, &domain.RefreshTokenRecord{
		ID:        uuid.New(),
		TenantID:  testTenantID,
		ClientID:  client.ID,
		UserID:    uuid.New(),
		TokenHash: hashTokenSHA256Helper(expiredToken),
		Scope:     []string{"openid"},
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired 1 hour ago
	})

	_, err := svc.RefreshToken(context.Background(), &RefreshTokenRequest{
		TenantID:     testTenantID,
		RefreshToken: expiredToken,
		ClientID:     "gcid-expired-test",
	})
	if err == nil {
		t.Error("expired refresh token must be rejected")
	}
}

// TestRefreshToken_WrongClientRejected verifies cross-client token usage fails.
func TestRefreshToken_WrongClientRejected(t *testing.T) {
	svc, clientRepo, _, tokenRepo := newTestOAuthService()

	clientA := &domain.OAuthClient{
		ID:         uuid.New(),
		TenantID:   testTenantID,
		ClientID:   "gcid-client-a",
		Name:       "Client A",
		Type:       domain.ClientTypePublic,
		GrantTypes: []string{"refresh_token"},
		Enabled:    true,
	}
	clientB := &domain.OAuthClient{
		ID:         uuid.New(),
		TenantID:   testTenantID,
		ClientID:   "gcid-client-b",
		Name:       "Client B",
		Type:       domain.ClientTypePublic,
		GrantTypes: []string{"refresh_token"},
		Enabled:    true,
	}
	_ = clientRepo.CreateClient(context.Background(), clientA)
	_ = clientRepo.CreateClient(context.Background(), clientB)

	// Token issued to client A
	tokenA := "token-issued-to-client-a"
	tokenRepo.refreshTokens = append(tokenRepo.refreshTokens, &domain.RefreshTokenRecord{
		ID:        uuid.New(),
		TenantID:  testTenantID,
		ClientID:  clientA.ID,
		UserID:    uuid.New(),
		TokenHash: hashTokenSHA256Helper(tokenA),
		Scope:     []string{"openid"},
		ExpiresAt: time.Now().Add(24 * time.Hour),
	})

	// Client B tries to use client A's token
	_, err := svc.RefreshToken(context.Background(), &RefreshTokenRequest{
		TenantID:     testTenantID,
		RefreshToken: tokenA,
		ClientID:     "gcid-client-b", // Wrong client
	})
	if err == nil {
		t.Error("cross-client refresh token usage must fail")
	}
}
