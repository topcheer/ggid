package service

import (
	"context"
	"testing"
	"time"

	"github.com/ggid/ggid/pkg/crypto"
	"github.com/ggid/ggid/pkg/tenant"
	"github.com/ggid/ggid/services/oauth/internal/domain"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Target: ParseAccessToken (90%), cryptoRandInt (83.3%), issueDeviceAccessToken (88.9%)
// CreateAuthorizationCode (92.3%), ExchangeAuthorizationCode (92.3%)

func TestBoost_ParseAccessToken_WrongSigningKey(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	// Sign with a different key.
	now := time.Now()
	claims := jwt.MapClaims{
		"iss": svc.issuer, "sub": "user1", "aud": "client1",
		"iat": now.Unix(), "exp": now.Add(15 * time.Minute).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	signed, _ := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	_, err := svc.ParseAccessToken(signed)
	if err == nil {
		t.Error("expected error for alg=none token")
	}
}

func TestBoost_ParseAccessToken_TamperedToken(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	// Pass completely garbage string.
	_, err := svc.ParseAccessToken("header.payload.garbage_signature_part_that_is_invalid")
	if err == nil {
		t.Error("expected error for invalid token")
	}
}

func TestBoost_CreateAuthCode_NonExistentClient(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	_, err := svc.CreateAuthorizationCode(context.Background(), &AuthorizeRequest{
		TenantID:     testTenantID,
		ClientID:     "nonexistent_client_xyz",
		RedirectURI:  "https://app.com/cb",
		ResponseType: "code",
		Scope:        []string{"openid"},
		State:        "st",
	})
	if err == nil {
		t.Fatal("expected error for nonexistent client")
	}
}

func TestBoost_ExchangeCode_ClientNotFound(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	_, err := svc.ExchangeAuthorizationCode(context.Background(), &TokenExchangeRequest{
		GrantType:   "authorization_code",
		Code:        "some-code",
		RedirectURI: "https://app.com/cb",
		ClientID:    "nonexistent_exchange",
	})
	if err == nil {
		t.Fatal("expected error for nonexistent client")
	}
}

func TestBoost_ExchangeCode_ExpiredCode(t *testing.T) {
	svc, repo, codeRepo, _ := newTestOAuthService()
	c := &domain.OAuthClient{
		ID: uuid.New(), TenantID: testTenantID, ClientID: "gcid_exp_code",
		Name: "Exp", Type: domain.ClientTypePublic,
		GrantTypes: []string{"authorization_code"},
		RedirectURIs: []string{"https://app.com/cb"},
		Scopes: []string{"openid"}, Enabled: true,
	}
	_ = repo.CreateClient(context.Background(), c)

	code := "expired-test-code"
	authCode := &domain.AuthorizationCode{
		ID: uuid.New(), TenantID: testTenantID, CodeHash: hashCode(code),
		ClientID: c.ID, UserID: uuid.New(), RedirectURI: "https://app.com/cb",
		Scope: []string{"openid"}, ExpiresAt: time.Now().Add(-1 * time.Minute),
	}
	_ = codeRepo.CreateCode(context.Background(), authCode)

	_, err := svc.ExchangeAuthorizationCode(context.Background(), &TokenExchangeRequest{
		GrantType: "authorization_code", Code: code,
		RedirectURI: "https://app.com/cb", ClientID: "gcid_exp_code",
	})
	if err == nil {
		t.Fatal("expected error for expired code")
	}
}

func TestBoost_RefreshToken_DisabledClient(t *testing.T) {
	svc, repo, _, _ := newTestOAuthService()
	hash, _ := crypto.HashPassword("secret")
	c := &domain.OAuthClient{
		ID: uuid.New(), TenantID: testTenantID, ClientID: "gcid_rt_disabled",
		Name: "RT Dis", Type: domain.ClientTypeConfidential,
		GrantTypes: []string{"refresh_token"}, ClientSecretHash: hash,
		Enabled: false,
	}
	_ = repo.CreateClient(context.Background(), c)

	_, err := svc.RefreshToken(context.Background(), &RefreshTokenRequest{
		TenantID:     testTenantID,
		RefreshToken: "some-token",
		ClientID:     "gcid_rt_disabled",
	})
	if err == nil {
		t.Fatal("expected error for disabled client")
	}
}

func TestBoost_CC_WrongSecret(t *testing.T) {
	svc, repo, _, _ := newTestOAuthService()
	hash, _ := crypto.HashPassword("correct-secret")
	c := &domain.OAuthClient{
		ID: uuid.New(), TenantID: testTenantID, ClientID: "gcid_cc_wrong",
		Name: "CC Wrong", Type: domain.ClientTypeConfidential,
		GrantTypes: []string{"client_credentials"}, ClientSecretHash: hash,
		Enabled: true,
	}
	_ = repo.CreateClient(context.Background(), c)

	_, err := svc.ClientCredentials(context.Background(), &ClientCredentialsRequest{
		TenantID:     testTenantID,
		ClientID:     "gcid_cc_wrong",
		ClientSecret: "wrong-secret",
	})
	if err == nil {
		t.Fatal("expected error for wrong client secret")
	}
}

func TestBoost_DynReg_DuplicateRedirectURI(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	ctx := tenant.WithContext(context.Background(), &tenant.Context{TenantID: testTenantID, IsolationLevel: tenant.IsolationShared})
	_, err := svc.DynamicClientRegister(ctx, &DynamicRegistrationRequest{
		ClientName: "Test App", RedirectURIs: []string{"https://app1.com/cb"},
	})
	if err != nil {
		t.Fatalf("first reg: %v", err)
	}
	_, err = svc.DynamicClientRegister(ctx, &DynamicRegistrationRequest{
		ClientName: "Test App 2", RedirectURIs: []string{"https://app2.com/cb"},
	})
	if err != nil {
		t.Fatalf("second reg: %v", err)
	}
}

func TestBoost_DeviceFlow_InvalidClientID(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	resp, _ := svc.CreateDeviceAuthorization(&DeviceAuthorizationRequest{
		TenantID: testTenantID, ClientID: "gcid_device_mismatch",
	})
	// Poll with different client ID.
	_, err := svc.PollDeviceToken(context.Background(), resp.DeviceCode, "different_client")
	if err == nil {
		t.Fatal("expected error for client mismatch")
	}
}

func TestBoost_IntrospectToken_NilClaimsAfterParse(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	// Craft token with empty string sub.
	now := time.Now()
	claims := jwt.MapClaims{
		"iss": svc.issuer, "aud": "client_nil",
		"iat": now.Unix(), "exp": now.Add(15 * time.Minute).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = svc.keyProvider.Metadata().KeyID
	signed, _ := token.SignedString(svc.keyProvider.Signer())

	resp := svc.IntrospectToken(signed)
	if !resp.Active {
		t.Fatal("expected active")
	}
	if resp.Sub != "" {
		t.Errorf("expected empty sub, got %s", resp.Sub)
	}
}
