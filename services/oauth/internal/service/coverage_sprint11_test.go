package service

import (
	"context"
	"testing"
	"time"

	"github.com/ggid/ggid/pkg/tenant"
	"github.com/ggid/ggid/services/oauth/internal/domain"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func testTenantCtx() context.Context {
	return tenant.WithContext(context.Background(), &tenant.Context{
		TenantID:       testTenantID,
		IsolationLevel: tenant.IsolationShared,
	})
}

// Coverage tests targeting error paths and edge cases.

func TestCovSprint11_Introspect_EmptyToken(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	resp := svc.IntrospectToken("")
	if resp.Active {
		t.Error("expected inactive for empty token")
	}
}

func TestCovSprint11_Introspect_InvalidToken(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	resp := svc.IntrospectToken("invalid.jwt.token")
	if resp.Active {
		t.Error("expected inactive for invalid token")
	}
}

func TestCovSprint11_Introspect_RevokedToken(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	// Revoke a dummy token - should not panic
	_ = svc.RevokeToken("dummy-token")
	resp := svc.IntrospectToken("dummy-token")
	// The token is not valid JWT, so introspect returns inactive
	if resp.Active {
		t.Error("expected inactive for dummy token")
	}
}

func TestCovSprint11_DiscoveryConfig(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	config := svc.GetDiscoveryConfig()
	if config.Issuer == "" {
		t.Error("expected non-empty issuer")
	}
}

func TestCovSprint11_JWKS(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	jwks := svc.GetJWKS()
	if len(jwks.Keys) == 0 {
		t.Error("expected at least one key")
	}
}

func TestCovSprint11_GetUserInfo_InvalidToken(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	_, err := svc.GetUserInfo("invalid-token")
	if err == nil {
		t.Error("expected error for invalid token")
	}
}

func TestCovSprint11_GetUserInfo_EmptyToken(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	_, err := svc.GetUserInfo("")
	if err == nil {
		t.Error("expected error for empty token")
	}
}

func TestCovSprint11_ClaimRulesEngine(t *testing.T) {
	engine := NewClaimRulesEngine(nil)
	engine.AddRule(ClaimRule{ClaimName: "role", SourceAttr: "ldap_role", Default: "user"})

	claims := jwt.MapClaims{}
	engine.ApplyRules(claims, map[string]any{"ldap_role": "admin"})
	if claims["role"] != "admin" {
		t.Errorf("expected role=admin, got %v", claims["role"])
	}

	claims2 := jwt.MapClaims{}
	engine.ApplyRules(claims2, map[string]any{})
	if claims2["role"] != "user" {
		t.Errorf("expected default role=user, got %v", claims2["role"])
	}
}

func TestCovSprint11_DeviceAuth_PollPending(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	resp, _ := svc.CreateDeviceAuthorization(&DeviceAuthorizationRequest{
		TenantID: testTenantID,
		ClientID: "test-device-client",
		Scope:    []string{"openid"},
		Issuer:   "https://test.ggid.dev",
	})

	_, err := svc.PollDeviceToken(context.Background(), resp.DeviceCode, "test-device-client")
	if err == nil {
		t.Error("expected authorization_pending error")
	}
}

func TestCovSprint11_DeviceAuth_PollUnknown(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	_, err := svc.PollDeviceToken(context.Background(), "unknown-device-code", "unknown-client")
	if err == nil {
		t.Error("expected error for unknown device code")
	}
}

func TestCovSprint11_DeviceAuth_ApproveAndPoll(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	resp, _ := svc.CreateDeviceAuthorization(&DeviceAuthorizationRequest{
		TenantID: testTenantID,
		ClientID: "device-approve-test",
		Scope:    []string{"openid"},
	})

	err := svc.ApproveDeviceCode(resp.UserCode, uuid.New())
	if err != nil {
		t.Fatalf("ApproveDeviceCode: %v", err)
	}

	token, err := svc.PollDeviceToken(context.Background(), resp.DeviceCode, "device-approve-test")
	if err != nil {
		t.Fatalf("PollDeviceToken after approval: %v", err)
	}
	if token.AccessToken == "" {
		t.Error("expected non-empty access token")
	}
}

func TestCovSprint11_DeviceAuth_ApproveUnknown(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	err := svc.ApproveDeviceCode("UNKNOWN-CODE", uuid.New())
	if err == nil {
		t.Error("expected error for unknown user_code")
	}
}

func TestCovSprint11_DeviceAuth_PollExpired(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	resp, _ := svc.CreateDeviceAuthorization(&DeviceAuthorizationRequest{
		TenantID: testTenantID,
		ClientID: "device-expired-test",
	})

	// Simulate expiry
	deviceCodeMu.Lock()
	if info, ok := deviceCodeStore[resp.DeviceCode]; ok {
		info.ExpiresAt = time.Now().Add(-1 * time.Minute)
	}
	deviceCodeMu.Unlock()

	_, err := svc.PollDeviceToken(context.Background(), resp.DeviceCode, "device-expired-test")
	if err == nil {
		t.Error("expected expired_token error")
	}
}

func TestCovSprint11_DeviceAuth_Denied(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	resp, _ := svc.CreateDeviceAuthorization(&DeviceAuthorizationRequest{
		TenantID: testTenantID,
		ClientID: "device-denied-test",
	})

	// Manually mark as denied
	deviceCodeMu.Lock()
	if info, ok := deviceCodeStore[resp.DeviceCode]; ok {
		info.Status = "denied"
	}
	deviceCodeMu.Unlock()

	_, err := svc.PollDeviceToken(context.Background(), resp.DeviceCode, "device-denied-test")
	if err == nil {
		t.Error("expected access_denied error")
	}
}

func TestCovSprint11_DeviceAuth_SlowDown(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	resp, _ := svc.CreateDeviceAuthorization(&DeviceAuthorizationRequest{
		TenantID: testTenantID,
		ClientID: "device-slow-test",
	})

	// First poll
	svc.PollDeviceToken(context.Background(), resp.DeviceCode, "device-slow-test")
	// Immediate second poll should trigger slow_down
	_, err := svc.PollDeviceToken(context.Background(), resp.DeviceCode, "device-slow-test")
	if err == nil {
		t.Error("expected slow_down error")
	}
}

func TestCovSprint11_ClientCredentials_InvalidClient(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	_, err := svc.ClientCredentials(context.Background(), &ClientCredentialsRequest{
		TenantID: testTenantID,
		ClientID: "nonexistent",
	})
	if err == nil {
		t.Error("expected error for invalid client")
	}
}

func TestCovSprint11_RefreshToken_InvalidClient(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	_, err := svc.RefreshToken(context.Background(), &RefreshTokenRequest{
		TenantID:     testTenantID,
		RefreshToken: "invalid",
		ClientID:     "nonexistent",
	})
	if err == nil {
		t.Error("expected error for invalid client")
	}
}

func TestCovSprint11_JWTBearer_EmptyAssertion(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	_, err := svc.JWTBearerGrant(context.Background(), &JWTBearerRequest{
		TenantID:  testTenantID,
		Assertion: "",
	})
	if err == nil {
		t.Error("expected error for empty assertion")
	}
}

func TestCovSprint11_ExchangeToken_EmptySubjectToken(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	_, err := svc.ExchangeToken(context.Background(), &TokenExchangeRequestRFC8693{
		TenantID:         testTenantID,
		SubjectToken:     "",
		SubjectTokenType: "urn:ietf:params:oauth:token-type:access_token",
	})
	if err == nil {
		t.Error("expected error for empty subject_token")
	}
}

func TestCovSprint11_ExchangeToken_InvalidSubjectToken(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	_, err := svc.ExchangeToken(context.Background(), &TokenExchangeRequestRFC8693{
		TenantID:         testTenantID,
		SubjectToken:     "invalid-token",
		SubjectTokenType: "urn:ietf:params:oauth:token-type:access_token",
	})
	if err == nil {
		t.Error("expected error for invalid subject_token")
	}
}

func TestCovSprint11_GetClient_NotFound(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	_, err := svc.GetClient(testTenantCtx(), "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent client")
	}
}

func TestCovSprint11_ListClients(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	_, _ = svc.CreateClient(testTenantCtx(), &CreateClientInput{
		TenantID:      testTenantID,
		Name:          "List 1",
		Type:          domain.ClientTypePublic,
		GrantTypes:    []string{"authorization_code"},
		ResponseTypes: []string{"code"},
		RedirectURIs:  []string{"https://app.example.com/callback"},
	})
	clients, total, err := svc.ListClients(testTenantCtx(), 10, 0)
	if err != nil {
		t.Fatalf("ListClients: %v", err)
	}
	if total == 0 || len(clients) == 0 {
		t.Error("expected at least 1 client")
	}
}

func TestCovSprint11_DeleteClient(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	result, _ := svc.CreateClient(testTenantCtx(), &CreateClientInput{
		TenantID:      testTenantID,
		Name:          "Delete Me",
		Type:          domain.ClientTypePublic,
		GrantTypes:    []string{"authorization_code"},
		ResponseTypes: []string{"code"},
		RedirectURIs:  []string{"https://app.example.com/callback"},
	})
	err := svc.DeleteClient(testTenantCtx(), result.Client.ClientID)
	if err != nil {
		t.Errorf("DeleteClient: %v", err)
	}
}

func TestCovSprint11_RotateClientSecret_InvalidClient(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	_, err := svc.RotateClientSecret(context.Background(), testTenantID, "nonexistent", "old")
	if err == nil {
		t.Error("expected error for invalid client")
	}
}

func TestCovSprint11_IssueSAMLToken(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	token, expires, err := svc.IssueSAMLToken(testTenantID, "nameid-123", "user@example.com", "Test User")
	if err != nil {
		t.Fatalf("IssueSAMLToken: %v", err)
	}
	if token == "" || expires <= 0 {
		t.Error("expected valid token and expiry")
	}
}

func TestCovSprint11_BackchannelLogout(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	svc.BackchannelLogout("user-cov-test")
	key := "ggid:backchannel_logout:user-cov-test"
	_, ok := backchannelLogoutList.Load(key)
	if !ok {
		t.Error("expected backchannel logout entry stored")
	}
}

func TestCovSprint11_DynamicReg_MissingRedirectURIs(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	_, err := svc.DynamicClientRegister(context.Background(), &DynamicRegistrationRequest{
		ClientName: "No Redirects",
	})
	if err == nil {
		t.Error("expected error for missing redirect_uris")
	}
}

func TestCovSprint11_CryptoRandInt(t *testing.T) {
	v := cryptoRandInt(100)
	if v < 0 || v >= 100 {
		t.Errorf("expected 0 <= v < 100, got %d", v)
	}
	if cryptoRandInt(0) != 0 {
		t.Error("expected 0 for max=0")
	}
	if cryptoRandInt(-1) != 0 {
		t.Error("expected 0 for negative max")
	}
}

func TestCovSprint11_DefaultIfEmpty(t *testing.T) {
	if defaultIfEmpty("", "def") != "def" {
		t.Error("expected default for empty")
	}
	if defaultIfEmpty("val", "def") != "val" {
		t.Error("expected value for non-empty")
	}
}

func TestCovSprint11_Contains(t *testing.T) {
	if !contains([]string{"a", "b"}, "b") {
		t.Error("expected true")
	}
	if contains([]string{"a", "b"}, "c") {
		t.Error("expected false")
	}
}

func TestCovSprint11_HashTokenSHA256(t *testing.T) {
	h1 := hashTokenSHA256("token")
	h2 := hashTokenSHA256("token")
	if h1 != h2 {
		t.Error("expected same hash for same input")
	}
	if hashTokenSHA256("other") == h1 {
		t.Error("expected different hash for different input")
	}
}

func TestCovSprint11_PAR_ExpiredRequest(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	requestURI := "urn:ietf:params:oauth:request_uri:" + uuid.New().String()
	parStore.Store(requestURI, parEntry{
		Request:   &PushedAuthorizationRequest{Scope: "test"},
		ExpiresAt: time.Now().Add(-1 * time.Minute),
	})
	_, err := svc.GetPushedAuthorizationRequest(requestURI)
	if err == nil {
		t.Error("expected error for expired request_uri")
	}
}

func TestCovSprint11_ApproveCIBAAuth_Expired(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	authReqID := "expired-approve-" + uuid.New().String()
	cibaStore.Store(authReqID, cibaEntry{
		Status:    CIBAStatusPending,
		ExpiresAt: time.Now().Add(-1 * time.Minute),
	})
	err := svc.ApproveCIBAAuth(authReqID)
	if err == nil {
		t.Error("expected error for expired auth_req_id")
	}
}

func TestCovSprint11_ParseBackchannelLogoutToken_NoSubOrSid(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	token := makeTestJWT(`{"alg":"none","typ":"JWT"}`, `{"events":{"http://schemas.openid.net/event/backchannel-logout":{}}}`)
	_, err := svc.ParseBackchannelLogoutToken(token)
	if err == nil {
		t.Error("expected error for missing sub and sid")
	}
}
