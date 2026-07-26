package service

import (
	"context"
	"testing"
	"time"

	"github.com/ggid/ggid/services/oauth/internal/domain"
	"github.com/google/uuid"
)

// TestRefreshToken_IssuesIDToken verifies that a refresh token grant
// with openid scope returns a non-empty id_token (OIDC Core §12).
func TestRefreshToken_IssuesIDToken(t *testing.T) {
	svc, clientRepo, _, tokenRepo := newTestOAuthService()

	// Create client with refresh_token grant
	client := &domain.OAuthClient{
		ID:         uuid.New(),
		TenantID:   testTenantID,
		ClientID:   "ggid-test-refresh",
		Name:       "Test",
		Type:       domain.ClientTypePublic,
		GrantTypes: []string{"refresh_token"},
		Enabled:    true,
	}
	_ = clientRepo.CreateClient(context.Background(), client)

	// Store a refresh token with openid scope
	userID := uuid.New()
	tokenHash := hashTokenSHA256Helper("existing-refresh-token")
	tokenRepo.refreshTokens = append(tokenRepo.refreshTokens, &domain.RefreshTokenRecord{
		ID:        uuid.New(),
		TenantID:  testTenantID,
		ClientID:  client.ID,
		UserID:    userID,
		TokenHash: tokenHash,
		Scope:     []string{"openid", "profile"},
		ExpiresAt: time.Now().Add(24 * time.Hour),
	})

	resp, err := svc.RefreshToken(context.Background(), &RefreshTokenRequest{
		TenantID:     testTenantID,
		RefreshToken: "existing-refresh-token",
		ClientID:     "ggid-test-refresh",
		Scope:        []string{"openid"},
	})
	if err != nil {
		t.Fatalf("refresh failed: %v", err)
	}

	// id_token MUST be present when openid scope is in the refresh
	if resp.IDToken == "" {
		t.Error("FAIL: id_token should be issued on refresh with openid scope (OIDC §12)")
	}
}
