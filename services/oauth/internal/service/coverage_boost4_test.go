package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/ggid/ggid/pkg/crypto"
	"github.com/ggid/ggid/services/oauth/internal/domain"
	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// ClientCredentials edge cases
// ---------------------------------------------------------------------------

func TestClientCredentials_PublicClient(t *testing.T) {
	svc, clientRepo, _, _ := newTestOAuthService()

	// Create a public client (no secret) that supports client_credentials.
	client := &domain.OAuthClient{
		ID:         uuid.New(),
		TenantID:   testTenantID,
		ClientID:   "gcid_pub_cc",
		Name:       "Public M2M",
		Type:       domain.ClientTypePublic,
		GrantTypes: []string{"client_credentials"},
		Enabled:    true,
	}
	_ = clientRepo.CreateClient(context.Background(), client)

	resp, err := svc.ClientCredentials(context.Background(), &ClientCredentialsRequest{
		TenantID: testTenantID,
		ClientID: "gcid_pub_cc",
	})
	if err != nil {
		t.Fatalf("public client_credentials: %v", err)
	}
	if resp.AccessToken == "" {
		t.Error("expected access token")
	}
}

func TestClientCredentials_WrongGrantType(t *testing.T) {
	svc, clientRepo, _, _ := newTestOAuthService()

	client := &domain.OAuthClient{
		ID:         uuid.New(),
		TenantID:   testTenantID,
		ClientID:   "gcid_no_cc",
		Name:       "Auth Code Only",
		Type:       domain.ClientTypeConfidential,
		GrantTypes: []string{"authorization_code"},
		Enabled:    true,
	}
	_ = clientRepo.CreateClient(context.Background(), client)

	_, err := svc.ClientCredentials(context.Background(), &ClientCredentialsRequest{
		TenantID:     testTenantID,
		ClientID:     "gcid_no_cc",
		ClientSecret: "anything",
	})
	if err == nil {
		t.Fatal("expected error for wrong grant type")
	}
}

// ---------------------------------------------------------------------------
// RefreshToken reuse detection + expired token
// ---------------------------------------------------------------------------

func TestRefreshToken_ExpiredToken(t *testing.T) {
	svc, clientRepo, _, tokenRepo := newTestOAuthService()

	// Create client with refresh_token grant.
	client := &domain.OAuthClient{
		ID:         uuid.New(),
		TenantID:   testTenantID,
		ClientID:   "gcid_refresh_exp",
		Name:       "Refresh Test",
		Type:       domain.ClientTypePublic,
		GrantTypes: []string{"authorization_code", "refresh_token"},
		Enabled:    true,
	}
	_ = clientRepo.CreateClient(context.Background(), client)

	// Store an expired refresh token.
	oldToken := "expired_refresh_token_123"
	tokenHash := hashTokenSHA256(oldToken)
	tokenRepo.StoreRefreshToken(context.Background(), &domain.RefreshTokenRecord{
		ID:        uuid.New(),
		TenantID:  testTenantID,
		ClientID:  client.ID,
		UserID:    uuid.New(),
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	})

	_, err := svc.RefreshToken(context.Background(), &RefreshTokenRequest{
		TenantID:     testTenantID,
		RefreshToken: oldToken,
		ClientID:     "gcid_refresh_exp",
	})
	if err == nil {
		t.Fatal("expected error for expired refresh token")
	}
}

func TestRefreshToken_ReuseDetection(t *testing.T) {
	svc, clientRepo, _, tokenRepo := newTestOAuthService()

	client := &domain.OAuthClient{
		ID:         uuid.New(),
		TenantID:   testTenantID,
		ClientID:   "gcid_reuse_detect",
		Name:       "Reuse Detect",
		Type:       domain.ClientTypePublic,
		GrantTypes: []string{"authorization_code", "refresh_token"},
		Enabled:    true,
	}
	_ = clientRepo.CreateClient(context.Background(), client)

	// Store a refresh token that has been used.
	usedToken := "used_refresh_token_456"
	usedHash := hashTokenSHA256(usedToken)
	tokenRepo.StoreRefreshToken(context.Background(), &domain.RefreshTokenRecord{
		ID:        uuid.New(),
		TenantID:  testTenantID,
		ClientID:  client.ID,
		UserID:    uuid.New(),
		TokenHash: usedHash,
		Used:      true,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	})

	_, err := svc.RefreshToken(context.Background(), &RefreshTokenRequest{
		TenantID:     testTenantID,
		RefreshToken: usedToken,
		ClientID:     "gcid_reuse_detect",
	})
	if err == nil {
		t.Fatal("expected reuse detection error")
	}
}

func TestRefreshToken_RevokedToken(t *testing.T) {
	svc, clientRepo, _, tokenRepo := newTestOAuthService()

	client := &domain.OAuthClient{
		ID:         uuid.New(),
		TenantID:   testTenantID,
		ClientID:   "gcid_revoked_rt",
		Name:       "Revoked RT",
		Type:       domain.ClientTypePublic,
		GrantTypes: []string{"authorization_code", "refresh_token"},
		Enabled:    true,
	}
	_ = clientRepo.CreateClient(context.Background(), client)

	revokedToken := "revoked_refresh_token_789"
	revokedHash := hashTokenSHA256(revokedToken)
	tokenRepo.StoreRefreshToken(context.Background(), &domain.RefreshTokenRecord{
		ID:        uuid.New(),
		TenantID:  testTenantID,
		ClientID:  client.ID,
		UserID:    uuid.New(),
		TokenHash: revokedHash,
		Revoked:   true,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	})

	_, err := svc.RefreshToken(context.Background(), &RefreshTokenRequest{
		TenantID:     testTenantID,
		RefreshToken: revokedToken,
		ClientID:     "gcid_revoked_rt",
	})
	if err == nil {
		t.Fatal("expected error for revoked token")
	}
}

func TestRefreshToken_WrongGrantType(t *testing.T) {
	svc, clientRepo, _, _ := newTestOAuthService()

	client := &domain.OAuthClient{
		ID:         uuid.New(),
		TenantID:   testTenantID,
		ClientID:   "gcid_no_refresh",
		Name:       "No Refresh",
		Type:       domain.ClientTypePublic,
		GrantTypes: []string{"authorization_code"},
		Enabled:    true,
	}
	_ = clientRepo.CreateClient(context.Background(), client)

	_, err := svc.RefreshToken(context.Background(), &RefreshTokenRequest{
		TenantID:     testTenantID,
		RefreshToken: "anything",
		ClientID:     "gcid_no_refresh",
	})
	if err == nil {
		t.Fatal("expected error for wrong grant type")
	}
}

// ---------------------------------------------------------------------------
// CreateAuthorizationCode — nonce/PKCE enforcement
// ---------------------------------------------------------------------------

func TestCreateAuthorizationCode_OIDCNonceRequired(t *testing.T) {
	svc, clientRepo, _, _ := newTestOAuthService()

	client := &domain.OAuthClient{
		ID:           uuid.New(),
		TenantID:     testTenantID,
		ClientID:     "gcid_oidc_nonce",
		Name:         "OIDC Nonce Test",
		Type:         domain.ClientTypeConfidential,
		GrantTypes:   []string{"authorization_code"},
		ResponseTypes: []string{"code", "id_token"},
		RedirectURIs: []string{"https://app.example.com/cb"},
		Scopes:       []string{"openid"},
		Enabled:      true,
	}
	_ = clientRepo.CreateClient(context.Background(), client)

	_, err := svc.CreateAuthorizationCode(context.Background(), &AuthorizeRequest{
		TenantID:     testTenantID,
		ClientID:     "gcid_oidc_nonce",
		RedirectURI:  "https://app.example.com/cb",
		ResponseType: "id_token",
		Scope:        []string{"openid"},
		State:        "state123",
		Nonce:        "", // missing!
	})
	if err == nil {
		t.Fatal("expected error for missing OIDC nonce")
	}
}

func TestCreateAuthorizationCode_PKCEEnforcedForPublicClient(t *testing.T) {
	svc, clientRepo, _, _ := newTestOAuthService()

	client := &domain.OAuthClient{
		ID:           uuid.New(),
		TenantID:     testTenantID,
		ClientID:     "gcid_pkce_pub",
		Name:         "PKCE Public",
		Type:         domain.ClientTypePublic,
		GrantTypes:   []string{"authorization_code"},
		ResponseTypes: []string{"code"},
		RedirectURIs: []string{"https://app.example.com/cb"},
		Scopes:       []string{"openid"},
		Enabled:      true,
	}
	_ = clientRepo.CreateClient(context.Background(), client)

	_, err := svc.CreateAuthorizationCode(context.Background(), &AuthorizeRequest{
		TenantID:     testTenantID,
		ClientID:     "gcid_pkce_pub",
		RedirectURI:  "https://app.example.com/cb",
		ResponseType: "code",
		Scope:        []string{"openid"},
		State:        "state123",
		// No CodeChallenge — PKCE should be enforced for public client
	})
	if err == nil {
		t.Fatal("expected PKCE enforcement error")
	}
}

// ---------------------------------------------------------------------------
// IntrospectToken — revoked token returns inactive
// ---------------------------------------------------------------------------

func TestIntrospectToken_RevokedToken_C4(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	// Issue a token first.
	client := &domain.OAuthClient{
		ID:         uuid.New(),
		TenantID:   testTenantID,
		ClientID:   "gcid_intro_revoked",
		Name:       "Intro Revoked",
		Type:       domain.ClientTypeConfidential,
		GrantTypes: []string{"client_credentials"},
		Enabled:    true,
	}

	accessToken, _, err := svc.issueAccessToken(uuid.New(), testTenantID, client.ClientID)
	if err != nil {
		t.Fatalf("issueAccessToken: %v", err)
	}

	// Revoke it.
	_ = svc.RevokeToken(accessToken)

	// Introspect — should be inactive.
	resp := svc.IntrospectToken(accessToken)
	if resp.Active {
		t.Fatal("expected revoked token to be inactive")
	}
}

func TestIntrospectToken_InvalidToken(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	resp := svc.IntrospectToken("invalid.jwt.token")
	if resp.Active {
		t.Fatal("expected inactive for invalid token")
	}
}

func TestIntrospectToken_WithScopeClaim(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	accessToken, _, err := svc.issueAccessToken(uuid.New(), testTenantID, "test_client")
	if err != nil {
		t.Fatalf("issueAccessToken: %v", err)
	}

	resp := svc.IntrospectToken(accessToken)
	if !resp.Active {
		t.Fatal("expected active token")
	}
	if resp.Iss == "" {
		t.Error("expected non-empty issuer")
	}
}

// ---------------------------------------------------------------------------
// OIDC Discovery endpoint
// ---------------------------------------------------------------------------

func TestGetDiscoveryConfig_AllFields(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	cfg := svc.GetDiscoveryConfig()
	if cfg.Issuer != "https://test.ggid.dev" {
		t.Errorf("issuer = %s", cfg.Issuer)
	}
	if cfg.AuthorizationEndpoint == "" {
		t.Error("missing authorization_endpoint")
	}
	if cfg.TokenEndpoint == "" {
		t.Error("missing token_endpoint")
	}
	if cfg.JwksURI == "" {
		t.Error("missing jwks_uri")
	}
	if len(cfg.GrantTypesSupported) < 3 {
		t.Error("expected at least 3 grant types")
	}
	if len(cfg.ScopesSupported) < 4 {
		t.Error("expected at least 4 scopes")
	}
	if len(cfg.ClaimsSupported) < 5 {
		t.Error("expected at least 5 claims")
	}
}

// ---------------------------------------------------------------------------
// Device code grant — denial path
// ---------------------------------------------------------------------------

func TestPollDeviceToken_DeniedCode(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	resp, err := svc.CreateDeviceAuthorization(&DeviceAuthorizationRequest{
		TenantID: testTenantID,
		ClientID: "gcid_device_test",
		Scope:    []string{"openid"},
		Issuer:   "https://test.ggid.dev",
	})
	if err != nil {
		t.Fatalf("CreateDeviceAuthorization: %v", err)
	}

	// Deny the device code by setting status to denied.
	deviceCodeMu.Lock()
	if info, ok := deviceCodeStore[resp.DeviceCode]; ok {
		info.Status = "denied"
	}
	deviceCodeMu.Unlock()

	_, err = svc.PollDeviceToken(context.Background(), resp.DeviceCode, "gcid_device_test")
	if err == nil {
		t.Fatal("expected access_denied error")
	}
}

// ---------------------------------------------------------------------------
// ParseAccessToken — invalid signing method
// ---------------------------------------------------------------------------

func TestParseAccessToken_InvalidSigningMethod(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	// This will fail parsing since it's not a valid JWT at all.
	_, err := svc.ParseAccessToken("not-a-jwt")
	if err == nil {
		t.Fatal("expected error for invalid JWT")
	}
}

// ---------------------------------------------------------------------------
// RevokeToken — empty token is a no-op
// ---------------------------------------------------------------------------


func TestRevokeToken_InvalidToken(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	err := svc.RevokeToken("invalid.jwt.token")
	if err != nil {
		t.Fatalf("expected nil for invalid token (RFC 7009), got %v", err)
	}
}

// ---------------------------------------------------------------------------
// GetUserInfo
// ---------------------------------------------------------------------------

func TestGetUserInfo_ValidToken_C4(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	userID := uuid.New()
	token, _, err := svc.issueAccessToken(userID, testTenantID, "client_x")
	if err != nil {
		t.Fatalf("issueAccessToken: %v", err)
	}

	info, err := svc.GetUserInfo(token)
	if err != nil {
		t.Fatalf("GetUserInfo: %v", err)
	}
	if info.Sub != userID.String() {
		t.Errorf("sub = %s, want %s", info.Sub, userID.String())
	}
}

func TestGetUserInfo_InvalidToken(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	_, err := svc.GetUserInfo("invalid.token.here")
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}

// ---------------------------------------------------------------------------
// RotateClientSecret — public client
// ---------------------------------------------------------------------------

func TestRotateClientSecret_PublicClient_C4(t *testing.T) {
	svc, clientRepo, _, _ := newTestOAuthService()

	client := &domain.OAuthClient{
		ID:         uuid.New(),
		TenantID:   testTenantID,
		ClientID:   "gcid_rotate_pub",
		Name:       "Rotate Pub",
		Type:       domain.ClientTypePublic,
		GrantTypes: []string{"authorization_code"},
		Enabled:    true,
	}
	_ = clientRepo.CreateClient(context.Background(), client)

	newSecret, err := svc.RotateClientSecret(context.Background(), testTenantID, "gcid_rotate_pub", "")
	if err != nil {
		t.Fatalf("RotateClientSecret: %v", err)
	}
	if newSecret == "" {
		t.Error("expected non-empty new secret")
	}
}

func TestRotateClientSecret_WrongOldSecret(t *testing.T) {
	svc, clientRepo, _, _ := newTestOAuthService()

	secretHash, _ := crypto.HashPassword("correct_secret")
	client := &domain.OAuthClient{
		ID:               uuid.New(),
		TenantID:         testTenantID,
		ClientID:         "gcid_rotate_wrong",
		Name:             "Rotate Wrong",
		Type:             domain.ClientTypeConfidential,
		GrantTypes:       []string{"authorization_code"},
		ClientSecretHash: secretHash,
		Enabled:          true,
	}
	_ = clientRepo.CreateClient(context.Background(), client)

	_, err := svc.RotateClientSecret(context.Background(), testTenantID, "gcid_rotate_wrong", "wrong_secret")
	if err == nil {
		t.Fatal("expected error for wrong old secret")
	}
}

// ---------------------------------------------------------------------------
// BackchannelLogout
// ---------------------------------------------------------------------------

func TestBackchannelLogout_MarksSubject(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	svc.BackchannelLogout("user-123")
	// Verify it was stored.
	_, ok := backchannelLogoutList.Load(fmt.Sprintf("ggid:backchannel_logout:%s", "user-123"))
	if !ok {
		t.Fatal("expected subject to be marked as logged out")
	}
}

// ---------------------------------------------------------------------------
// ParseBackchannelLogoutToken — more edge cases
// ---------------------------------------------------------------------------

func TestParseBackchannelLogoutToken_MissingEvents_C4(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	// JWT with sub but no events claim.
	token := createJWTWithClaims(t, map[string]any{
		"sub": "user-123",
	})
	_, err := svc.ParseBackchannelLogoutToken(token)
	if err == nil {
		t.Fatal("expected error for missing events")
	}
}

func TestParseBackchannelLogoutToken_WithNonce(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	token := createJWTWithClaims(t, map[string]any{
		"sub":    "user-123",
		"nonce":  "should-not-be-here",
		"events": map[string]any{"http://schemas.openid.net/event/backchannel-logout": map[string]any{}},
	})
	_, err := svc.ParseBackchannelLogoutToken(token)
	if err == nil {
		t.Fatal("expected error for nonce in logout token")
	}
}

func TestParseBackchannelLogoutToken_ValidWithSID(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	token := createJWTWithClaims(t, map[string]any{
		"sid":    "session-abc",
		"events": map[string]any{"http://schemas.openid.net/event/backchannel-logout": map[string]any{}},
	})
	claims, err := svc.ParseBackchannelLogoutToken(token)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if claims["sid"] != "session-abc" {
		t.Errorf("sid = %v", claims["sid"])
	}
}

// createJWTWithClaims creates an unsigned JWT for testing ParseBackchannelLogoutToken.
func createJWTWithClaims(t *testing.T, claims map[string]any) string {
	t.Helper()
	header := `{"alg":"none","typ":"JWT"}`
	body, _ := json.Marshal(claims)
	headerB64 := base64.RawURLEncoding.EncodeToString([]byte(header))
	bodyB64 := base64.RawURLEncoding.EncodeToString(body)
	return headerB64 + "." + bodyB64 + "."
}
