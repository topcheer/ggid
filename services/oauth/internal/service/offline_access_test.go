package service

import (
	"context"
	"testing"
	"time"

	"github.com/ggid/ggid/services/oauth/internal/domain"
	"github.com/google/uuid"
)

// TestExchangeCode_OfflineAccessIssuesRefreshToken verifies that an
// authorization_code exchange with offline_access scope returns a refresh
// token rooted in a new family (regression: previously never issued, making
// the refresh_token grant unreachable for web clients).
func TestExchangeCode_OfflineAccessIssuesRefreshToken(t *testing.T) {
	svc, clientRepo, codeRepo, tokenRepo := newTestOAuthService()

	client := &domain.OAuthClient{
		ID:         uuid.New(),
		TenantID:   testTenantID,
		ClientID:   "gcid_offline_1",
		Name:       "Offline App",
		Type:       domain.ClientTypePublic,
		GrantTypes: []string{"authorization_code", "refresh_token"},
		Enabled:    true,
	}
	_ = clientRepo.CreateClient(context.Background(), client)

	plainCode := "test-code-offline-1"
	_ = codeRepo.CreateCode(context.Background(), &domain.AuthorizationCode{
		ID:          uuid.New(),
		TenantID:    testTenantID,
		ClientID:    client.ID,
		UserID:      uuid.New(),
		CodeHash:    hashCode(plainCode),
		RedirectURI: "https://app.example.com/cb",
		Scope:       []string{"openid", "offline_access"},
		ExpiresAt:   time.Now().Add(5 * time.Minute),
	})

	resp, err := svc.ExchangeAuthorizationCode(context.Background(), &TokenExchangeRequest{
		TenantID:    testTenantID,
		Code:        plainCode,
		RedirectURI: "https://app.example.com/cb",
		ClientID:    "gcid_offline_1",
	})
	if err != nil {
		t.Fatalf("ExchangeAuthorizationCode: %v", err)
	}
	if resp.RefreshToken == "" {
		t.Fatal("offline_access exchange must return a refresh token")
	}

	// Stored record roots its own family.
	if len(tokenRepo.refreshTokens) != 1 {
		t.Fatalf("expected 1 stored refresh token, got %d", len(tokenRepo.refreshTokens))
	}
	rec := tokenRepo.refreshTokens[0]
	if rec.FamilyID != rec.ID.String() {
		t.Errorf("FamilyID = %q, want root %q", rec.FamilyID, rec.ID.String())
	}
}

// TestExchangeCode_NoOfflineAccess_NoRefreshToken preserves existing
// behavior: plain OIDC exchanges do NOT get a refresh token.
func TestExchangeCode_NoOfflineAccess_NoRefreshToken(t *testing.T) {
	svc, clientRepo, codeRepo, tokenRepo := newTestOAuthService()

	client := &domain.OAuthClient{
		ID:         uuid.New(),
		TenantID:   testTenantID,
		ClientID:   "gcid_offline_2",
		Name:       "Plain OIDC",
		Type:       domain.ClientTypePublic,
		GrantTypes: []string{"authorization_code", "refresh_token"},
		Enabled:    true,
	}
	_ = clientRepo.CreateClient(context.Background(), client)

	plainCode := "test-code-plain-1"
	_ = codeRepo.CreateCode(context.Background(), &domain.AuthorizationCode{
		ID:          uuid.New(),
		TenantID:    testTenantID,
		ClientID:    client.ID,
		UserID:      uuid.New(),
		CodeHash:    hashCode(plainCode),
		RedirectURI: "https://app.example.com/cb",
		Scope:       []string{"openid", "profile"},
		ExpiresAt:   time.Now().Add(5 * time.Minute),
	})

	resp, err := svc.ExchangeAuthorizationCode(context.Background(), &TokenExchangeRequest{
		TenantID:    testTenantID,
		Code:        plainCode,
		RedirectURI: "https://app.example.com/cb",
		ClientID:    "gcid_offline_2",
	})
	if err != nil {
		t.Fatalf("ExchangeAuthorizationCode: %v", err)
	}
	if resp.RefreshToken != "" {
		t.Error("plain OIDC exchange must NOT return a refresh token")
	}
	if len(tokenRepo.refreshTokens) != 0 {
		t.Error("no refresh token should be stored")
	}
}

// TestExchangeCode_OfflineAccess_RequiresRefreshGrant ensures clients
// without the refresh_token grant type don't get a refresh token even with
// offline_access scope.
func TestExchangeCode_OfflineAccess_RequiresRefreshGrant(t *testing.T) {
	svc, clientRepo, codeRepo, _ := newTestOAuthService()

	client := &domain.OAuthClient{
		ID:         uuid.New(),
		TenantID:   testTenantID,
		ClientID:   "gcid_offline_3",
		Name:       "No Refresh Grant",
		Type:       domain.ClientTypePublic,
		GrantTypes: []string{"authorization_code"},
		Enabled:    true,
	}
	_ = clientRepo.CreateClient(context.Background(), client)

	plainCode := "test-code-nogrant-1"
	_ = codeRepo.CreateCode(context.Background(), &domain.AuthorizationCode{
		ID:          uuid.New(),
		TenantID:    testTenantID,
		ClientID:    client.ID,
		UserID:      uuid.New(),
		CodeHash:    hashCode(plainCode),
		RedirectURI: "https://app.example.com/cb",
		Scope:       []string{"openid", "offline_access"},
		ExpiresAt:   time.Now().Add(5 * time.Minute),
	})

	resp, err := svc.ExchangeAuthorizationCode(context.Background(), &TokenExchangeRequest{
		TenantID:    testTenantID,
		Code:        plainCode,
		RedirectURI: "https://app.example.com/cb",
		ClientID:    "gcid_offline_3",
	})
	if err != nil {
		t.Fatalf("ExchangeAuthorizationCode: %v", err)
	}
	if resp.RefreshToken != "" {
		t.Error("client without refresh_token grant must NOT receive a refresh token")
	}
}
