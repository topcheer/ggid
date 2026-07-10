package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/ggid/ggid/pkg/crypto"
	"github.com/ggid/ggid/pkg/tenant"
	"github.com/ggid/ggid/services/oauth/internal/domain"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// ===========================================================================
// Task 1: Token Introspection (RFC 7662) — 6 tests
// ===========================================================================

func TestSec_IntrospectToken_ActiveWithAllFields(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	userID := uuid.New()
	token, _, err := svc.issueAccessToken(userID, testTenantID, "gcid_intro_full")
	if err != nil {
		t.Fatalf("issueAccessToken: %v", err)
	}

	resp := svc.IntrospectToken(token)
	if !resp.Active {
		t.Fatal("expected active token")
	}
	if resp.TokenType != "Bearer" {
		t.Errorf("token_type = %s, want Bearer", resp.TokenType)
	}
	if resp.Sub != userID.String() {
		t.Errorf("sub = %s", resp.Sub)
	}
	if resp.Iss == "" {
		t.Error("expected non-empty issuer")
	}
	if resp.Exp == 0 {
		t.Error("expected non-zero exp")
	}
	if resp.Iat == 0 {
		t.Error("expected non-zero iat")
	}
}

func TestSec_IntrospectToken_ExpiredToken(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	now := time.Now()
	claims := jwt.MapClaims{
		"iss": "https://test.ggid.dev",
		"sub": "user-expired",
		"aud": "client1",
		"iat": now.Add(-2 * time.Hour).Unix(),
		"exp": now.Add(-1 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = "test-kid"
	signed, _ := token.SignedString(svc.keyProvider.PrivateKey())

	resp := svc.IntrospectToken(signed)
	if resp.Active {
		t.Fatal("expected inactive for expired token")
	}
}

func TestSec_IntrospectToken_MalformedJWT(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	resp := svc.IntrospectToken("not.a.valid.jwt")
	if resp.Active {
		t.Fatal("expected inactive")
	}
}

func TestSec_IntrospectToken_EmptyToken(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	resp := svc.IntrospectToken("")
	if resp.Active {
		t.Fatal("expected inactive")
	}
}

func TestSec_IntrospectToken_WithScope(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	now := time.Now()
	claims := jwt.MapClaims{
		"iss":   svc.issuer,
		"sub":   uuid.New().String(),
		"aud":   "gcid_scope",
		"iat":   now.Unix(),
		"exp":   now.Add(15 * time.Minute).Unix(),
		"scope": "openid profile",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = svc.keyProvider.KeyID()
	signed, _ := token.SignedString(svc.keyProvider.PrivateKey())

	resp := svc.IntrospectToken(signed)
	if resp.Scope != "openid profile" {
		t.Errorf("scope = %s", resp.Scope)
	}
}

func TestSec_IntrospectToken_WithUsername(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	now := time.Now()
	claims := jwt.MapClaims{
		"iss":                svc.issuer,
		"sub":                uuid.New().String(),
		"aud":                "gcid_uname",
		"iat":                now.Unix(),
		"exp":                now.Add(15 * time.Minute).Unix(),
		"preferred_username": "testuser@example.com",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = svc.keyProvider.KeyID()
	signed, _ := token.SignedString(svc.keyProvider.PrivateKey())

	resp := svc.IntrospectToken(signed)
	if resp.Username != "testuser@example.com" {
		t.Errorf("username = %s", resp.Username)
	}
}

// ===========================================================================
// Task 2: OIDC UserInfo — 4 tests
// ===========================================================================

func TestSec_UserInfo_ValidToken(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	userID := uuid.New()
	token, _, err := svc.issueAccessToken(userID, testTenantID, "gcid_ui")
	if err != nil {
		t.Fatalf("issueAccessToken: %v", err)
	}
	info, err := svc.GetUserInfo(token)
	if err != nil {
		t.Fatalf("GetUserInfo: %v", err)
	}
	if info.Sub != userID.String() {
		t.Errorf("sub = %s", info.Sub)
	}
}

func TestSec_UserInfo_InvalidToken(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	_, err := svc.GetUserInfo("invalid.token")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSec_UserInfo_EmptyToken(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	_, err := svc.GetUserInfo("")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSec_UserInfo_WithEmailVerified(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	now := time.Now()
	claims := jwt.MapClaims{
		"iss":            svc.issuer,
		"sub":            "user-ev",
		"aud":            "gcid_ev",
		"iat":            now.Unix(),
		"exp":            now.Add(15 * time.Minute).Unix(),
		"email":          "user@test.com",
		"email_verified": true,
		"name":           "Test User",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = svc.keyProvider.KeyID()
	signed, _ := token.SignedString(svc.keyProvider.PrivateKey())

	info, err := svc.GetUserInfo(signed)
	if err != nil {
		t.Fatalf("GetUserInfo: %v", err)
	}
	if !info.EmailVerified {
		t.Error("expected email_verified=true")
	}
	if info.Email != "user@test.com" {
		t.Errorf("email = %s", info.Email)
	}
}

// ===========================================================================
// Task 3: Device Flow (RFC 8628) — 8 tests
// ===========================================================================

func TestSec_Device_Create(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	resp, err := svc.CreateDeviceAuthorization(&DeviceAuthorizationRequest{
		TenantID: testTenantID, ClientID: "dc1", Scope: []string{"openid"},
		Issuer: "https://test.ggid.dev",
	})
	if err != nil {
		t.Fatalf("CreateDeviceAuthorization: %v", err)
	}
	if resp.DeviceCode == "" || len(resp.UserCode) != 9 {
		t.Errorf("bad codes: dc=%s uc=%s", resp.DeviceCode, resp.UserCode)
	}
	if resp.VerificationURI != "https://test.ggid.dev/device" {
		t.Errorf("uri=%s", resp.VerificationURI)
	}
}

func TestSec_Device_PollPending(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	resp, _ := svc.CreateDeviceAuthorization(&DeviceAuthorizationRequest{
		TenantID: testTenantID, ClientID: "dc2",
	})
	_, err := svc.PollDeviceToken(context.Background(), resp.DeviceCode, "dc2")
	if err == nil || err.Error() != "authorization_pending" {
		t.Fatalf("expected authorization_pending, got %v", err)
	}
}

func TestSec_Device_SlowDown(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	resp, _ := svc.CreateDeviceAuthorization(&DeviceAuthorizationRequest{
		TenantID: testTenantID, ClientID: "dc3",
	})
	// First poll
	_, _ = svc.PollDeviceToken(context.Background(), resp.DeviceCode, "dc3")
	// Second poll immediately
	_, err := svc.PollDeviceToken(context.Background(), resp.DeviceCode, "dc3")
	if err == nil || err.Error() != "slow_down" {
		t.Fatalf("expected slow_down, got %v", err)
	}
}

func TestSec_Device_Approved(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	resp, _ := svc.CreateDeviceAuthorization(&DeviceAuthorizationRequest{
		TenantID: testTenantID, ClientID: "dc4",
	})
	userID := uuid.New()
	_ = svc.ApproveDeviceCode(resp.UserCode, userID)
	tok, err := svc.PollDeviceToken(context.Background(), resp.DeviceCode, "dc4")
	if err != nil {
		t.Fatalf("PollDeviceToken: %v", err)
	}
	if tok.AccessToken == "" {
		t.Error("expected access token")
	}
}

func TestSec_Device_Denied(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	resp, _ := svc.CreateDeviceAuthorization(&DeviceAuthorizationRequest{
		TenantID: testTenantID, ClientID: "dc5",
	})
	deviceCodeMu.Lock()
	deviceCodeStore[resp.DeviceCode].Status = "denied"
	deviceCodeMu.Unlock()
	_, err := svc.PollDeviceToken(context.Background(), resp.DeviceCode, "dc5")
	if err == nil || err.Error() != "access_denied" {
		t.Fatalf("expected access_denied, got %v", err)
	}
}

func TestSec_Device_Expired(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	resp, _ := svc.CreateDeviceAuthorization(&DeviceAuthorizationRequest{
		TenantID: testTenantID, ClientID: "dc6",
	})
	deviceCodeMu.Lock()
	deviceCodeStore[resp.DeviceCode].ExpiresAt = time.Now().Add(-1 * time.Second)
	deviceCodeMu.Unlock()
	_, err := svc.PollDeviceToken(context.Background(), resp.DeviceCode, "dc6")
	if err == nil || err.Error() != "expired_token" {
		t.Fatalf("expected expired_token, got %v", err)
	}
}

func TestSec_Device_InvalidCode(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	_, err := svc.PollDeviceToken(context.Background(), "nonexistent", "dc7")
	if err == nil || err.Error() != "invalid_device_code" {
		t.Fatalf("expected invalid_device_code, got %v", err)
	}
}

func TestSec_Device_ApproveBadUserCode(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	err := svc.ApproveDeviceCode("BADCODE", uuid.New())
	if err == nil {
		t.Fatal("expected error")
	}
}

// ===========================================================================
// Task 4: JWT amr/acr/auth_time — 5 tests
// ===========================================================================

func TestSec_JWT_AMR(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	token, _ := svc.issueIDToken(uuid.New(), testTenantID, "c1", "n1", &IDTokenOptions{
		AMR: []string{"pwd", "otp"},
	})
	parsed, _ := jwt.ParseWithClaims(token, jwt.MapClaims{}, func(t *jwt.Token) (any, error) {
		return svc.keyProvider.PublicKey(), nil
	})
	claims := parsed.Claims.(jwt.MapClaims)
	amr, ok := claims["amr"].([]any)
	if !ok || len(amr) != 2 {
		t.Fatalf("amr = %v", claims["amr"])
	}
}

func TestSec_JWT_ACR(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	token, _ := svc.issueIDToken(uuid.New(), testTenantID, "c2", "n2", &IDTokenOptions{
		ACR: "urn:mace:incommon:iap:silver",
	})
	parsed, _ := jwt.ParseWithClaims(token, jwt.MapClaims{}, func(t *jwt.Token) (any, error) {
		return svc.keyProvider.PublicKey(), nil
	})
	claims := parsed.Claims.(jwt.MapClaims)
	if claims["acr"] != "urn:mace:incommon:iap:silver" {
		t.Errorf("acr = %v", claims["acr"])
	}
}

func TestSec_JWT_AuthTime(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	at := time.Now().Unix()
	token, _ := svc.issueIDToken(uuid.New(), testTenantID, "c3", "n3", &IDTokenOptions{
		AuthTime: at,
	})
	parsed, _ := jwt.ParseWithClaims(token, jwt.MapClaims{}, func(t *jwt.Token) (any, error) {
		return svc.keyProvider.PublicKey(), nil
	})
	claims := parsed.Claims.(jwt.MapClaims)
	val, ok := claims["auth_time"].(float64)
	if !ok {
		t.Fatalf("auth_time type = %T", claims["auth_time"])
	}
	if int64(val) != at {
		t.Errorf("auth_time = %d, want %d", int64(val), at)
	}
}

func TestSec_JWT_NoOptsNoExtraClaims(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	token, _ := svc.issueIDToken(uuid.New(), testTenantID, "c4", "n4", nil)
	parsed, _ := jwt.ParseWithClaims(token, jwt.MapClaims{}, func(t *jwt.Token) (any, error) {
		return svc.keyProvider.PublicKey(), nil
	})
	claims := parsed.Claims.(jwt.MapClaims)
	for _, k := range []string{"amr", "acr", "auth_time"} {
		if _, exists := claims[k]; exists {
			t.Errorf("%s should not be present", k)
		}
	}
}

func TestSec_JWT_AllClaimsCombined(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	token, _ := svc.issueIDToken(uuid.New(), testTenantID, "c5", "n5", &IDTokenOptions{
		AMR: []string{"pwd", "mfa"}, ACR: "urn:ggid:2fa", AuthTime: time.Now().Unix(),
	})
	parsed, _ := jwt.ParseWithClaims(token, jwt.MapClaims{}, func(t *jwt.Token) (any, error) {
		return svc.keyProvider.PublicKey(), nil
	})
	claims := parsed.Claims.(jwt.MapClaims)
	if _, ok := claims["amr"]; !ok {
		t.Error("missing amr")
	}
	if _, ok := claims["acr"]; !ok {
		t.Error("missing acr")
	}
	if _, ok := claims["auth_time"]; !ok {
		t.Error("missing auth_time")
	}
}

// ===========================================================================
// Task 5: Coverage boost → 97%
// ===========================================================================

func TestSec_CreateClient_Public(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	r, err := svc.CreateClient(context.Background(), &CreateClientInput{
		TenantID: testTenantID, Name: "Pub", Type: domain.ClientTypePublic,
		GrantTypes: []string{"authorization_code"},
	})
	if err != nil {
		t.Fatalf("CreateClient: %v", err)
	}
	if r.ClientSecret != "" {
		t.Error("public client should not have secret")
	}
}

func TestSec_CreateClient_WithMetadata(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	r, err := svc.CreateClient(context.Background(), &CreateClientInput{
		TenantID: testTenantID, Name: "Meta", Type: domain.ClientTypeConfidential,
		GrantTypes: []string{"client_credentials"},
		Metadata:   map[string]any{"logo": "https://app.com/logo.png"},
	})
	if err != nil {
		t.Fatalf("CreateClient: %v", err)
	}
	if r.Client.ClientID == "" {
		t.Error("expected client_id")
	}
}

func TestSec_DeleteClient(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	r, _ := svc.CreateClient(context.Background(), &CreateClientInput{
		TenantID: testTenantID, Name: "Del", Type: domain.ClientTypePublic,
		GrantTypes: []string{"authorization_code"},
	})
	ctx := tenantCtx(testTenantID)
	err := svc.DeleteClient(ctx, r.Client.ClientID)
	if err != nil {
		t.Fatalf("DeleteClient: %v", err)
	}
}

func TestSec_Authorize_InvalidRedirectURI(t *testing.T) {
	svc, repo, _, _ := newTestOAuthService()
	c := &domain.OAuthClient{
		ID: uuid.New(), TenantID: testTenantID, ClientID: "gcid_bad_uri",
		Name: "Bad URI", Type: domain.ClientTypeConfidential,
		GrantTypes: []string{"authorization_code"}, RedirectURIs: []string{"https://correct.com/cb"},
		Scopes: []string{"openid"}, Enabled: true,
	}
	_ = repo.CreateClient(context.Background(), c)
	_, err := svc.CreateAuthorizationCode(context.Background(), &AuthorizeRequest{
		TenantID: testTenantID, ClientID: "gcid_bad_uri",
		RedirectURI: "https://evil.com/cb", ResponseType: "code",
		Scope: []string{"openid"}, State: "st",
	})
	if err == nil {
		t.Fatal("expected error for invalid redirect URI")
	}
}

func TestSec_Authorize_DisabledClient(t *testing.T) {
	svc, repo, _, _ := newTestOAuthService()
	c := &domain.OAuthClient{
		ID: uuid.New(), TenantID: testTenantID, ClientID: "gcid_dis",
		Name: "Disabled", Type: domain.ClientTypeConfidential,
		GrantTypes: []string{"authorization_code"}, RedirectURIs: []string{"https://app.com/cb"},
		Scopes: []string{"openid"}, Enabled: false,
	}
	_ = repo.CreateClient(context.Background(), c)
	_, err := svc.CreateAuthorizationCode(context.Background(), &AuthorizeRequest{
		TenantID: testTenantID, ClientID: "gcid_dis",
		RedirectURI: "https://app.com/cb", ResponseType: "code",
		Scope: []string{"openid"}, State: "st",
	})
	if err == nil {
		t.Fatal("expected error for disabled client")
	}
}


func TestSec_CC_DisabledClient(t *testing.T) {
	svc, repo, _, _ := newTestOAuthService()
	hash, _ := crypto.HashPassword("secret123")
	c := &domain.OAuthClient{
		ID: uuid.New(), TenantID: testTenantID, ClientID: "gcid_cc_dis",
		Name: "CC Dis", Type: domain.ClientTypeConfidential,
		GrantTypes: []string{"client_credentials"}, ClientSecretHash: hash, Enabled: false,
	}
	_ = repo.CreateClient(context.Background(), c)
	_, err := svc.ClientCredentials(context.Background(), &ClientCredentialsRequest{
		TenantID: testTenantID, ClientID: "gcid_cc_dis", ClientSecret: "secret123",
	})
	if err == nil {
		t.Fatal("expected error for disabled client")
	}
}

func TestSec_DynReg_AllFields(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	ctx := tenantCtx(testTenantID)
	resp, err := svc.DynamicClientRegister(ctx, &DynamicRegistrationRequest{
		ClientName: "Full Reg", RedirectURIs: []string{"https://app.com/cb"},
		GrantTypes: []string{"authorization_code", "refresh_token"},
		ResponseTypes: []string{"code"}, Scope: "openid profile",
		ClientURI: "https://app.com", LogoURI: "https://app.com/logo.png",
		PolicyURI: "https://app.com/policy",
	})
	if err != nil {
		t.Fatalf("DynamicClientRegister: %v", err)
	}
	if resp.ClientID == "" {
		t.Error("expected client_id")
	}
}

func TestSec_JWTBearer_InvalidAssertion(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	_, err := svc.JWTBearerGrant(context.Background(), &JWTBearerRequest{
		TenantID: testTenantID, Assertion: "invalid.jwt", Issuer: "https://test.ggid.dev",
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSec_RevokeToken_Idempotent(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	token, _, _ := svc.issueAccessToken(uuid.New(), testTenantID, "c_rev")
	_ = svc.RevokeToken(token)
	err := svc.RevokeToken(token)
	if err != nil {
		t.Fatalf("idempotent revoke: %v", err)
	}
	if !svc.IsTokenRevoked(token) {
		t.Error("expected revoked")
	}
}

func TestSec_BackchannelLogout_ValidToken(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	header := `{"alg":"none","typ":"JWT"}`
	body, _ := json.Marshal(map[string]any{
		"sub": "user-logout",
		"events": map[string]any{
			"http://schemas.openid.net/event/backchannel-logout": map[string]any{},
		},
	})
	token := base64.RawURLEncoding.EncodeToString([]byte(header)) + "." +
		base64.RawURLEncoding.EncodeToString(body) + "."
	claims, err := svc.ParseBackchannelLogoutToken(token)
	if err != nil {
		t.Fatalf("ParseBackchannelLogoutToken: %v", err)
	}
	if claims["sub"] != "user-logout" {
		t.Errorf("sub = %v", claims["sub"])
	}
}

func TestSec_BackchannelLogout_WrongEvents(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	header := `{"alg":"none","typ":"JWT"}`
	body, _ := json.Marshal(map[string]any{
		"sub": "user-x",
		"events": map[string]any{"other-event": map[string]any{}},
	})
	token := base64.RawURLEncoding.EncodeToString([]byte(header)) + "." +
		base64.RawURLEncoding.EncodeToString(body) + "."
	_, err := svc.ParseBackchannelLogoutToken(token)
	if err == nil {
		t.Fatal("expected error for wrong events URI")
	}
}

func TestSec_hashTokenSHA256(t *testing.T) {
	h1 := hashTokenSHA256("test")
	h2 := hashTokenSHA256("test")
	if h1 != h2 {
		t.Error("hash should be deterministic")
	}
	if hashTokenSHA256("other") == h1 {
		t.Error("different inputs should differ")
	}
}

func TestSec_generateClientID(t *testing.T) {
	id1 := generateClientID()
	id2 := generateClientID()
	if id1 == id2 {
		t.Error("expected unique IDs")
	}
}

func TestSec_generateClientSecret(t *testing.T) {
	s1 := generateClientSecret()
	s2 := generateClientSecret()
	if s1 == s2 {
		t.Error("expected unique secrets")
	}
}

func TestSec_contains(t *testing.T) {
	if !contains([]string{"a", "b"}, "a") {
		t.Error("expected true")
	}
	if contains([]string{"a", "b"}, "c") {
		t.Error("expected false")
	}
	if contains(nil, "x") {
		t.Error("nil should return false")
	}
}

func TestSec_joinScopes(t *testing.T) {
	if joinScopes([]string{"a", "b"}) != "a b" {
		t.Error("joinScopes failed")
	}
	if joinScopes(nil) != "" {
		t.Error("nil should return empty")
	}
}

func TestSec_defaultIfEmpty(t *testing.T) {
	if defaultIfEmpty("", "def") != "def" {
		t.Error("expected default")
	}
	if defaultIfEmpty("val", "def") != "val" {
		t.Error("expected val")
	}
}

func TestSec_Discovery_AllEndpoints(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	cfg := svc.GetDiscoveryConfig()
	for _, ep := range []string{cfg.UserInfoEndpoint, cfg.RevocationEndpoint, cfg.IntrospectionEndpoint} {
		if ep == "" {
			t.Error("missing endpoint")
		}
	}
}

func TestSec_GetClient_NotFound(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	ctx := tenantCtx(testTenantID)
	_, err := svc.GetClient(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}
}

// tenantCtx creates a context with tenant for OAuth service tests.
func tenantCtx(tid uuid.UUID) context.Context {
	return tenant.WithContext(context.Background(), &tenant.Context{
		TenantID:       tid,
		IsolationLevel: tenant.IsolationShared,
	})
}

var _ = fmt.Sprintf
