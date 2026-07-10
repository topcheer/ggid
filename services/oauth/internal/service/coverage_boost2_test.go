package service

import (
	"context"
	stdcrypto "crypto/rand"
	"crypto/rsa"
	"strings"
	"testing"
	"time"

	"github.com/ggid/ggid/pkg/crypto"
	"github.com/ggid/ggid/pkg/tenant"
	"github.com/ggid/ggid/services/oauth/internal/domain"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// =====================================================
// Helper: create a signed RSA JWT for test scenarios
// =====================================================

// makeTestAssertion creates a JWT signed by a separate key (simulating a third-party issuer).
func makeTestAssertion(t *testing.T, claims jwt.MapClaims) string {
	t.Helper()
	priv, err := rsa.GenerateKey(stdcrypto.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := token.SignedString(priv)
	if err != nil {
		t.Fatalf("sign assertion: %v", err)
	}
	return signed
}

// makeUnsignedToken creates an unsigned JWT string using jwt.Parser (for backchannel logout tests).
func makeUnsignedToken(t *testing.T, claims jwt.MapClaims) string {
	t.Helper()
	priv, err := rsa.GenerateKey(stdcrypto.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := token.SignedString(priv)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return signed
}

// =====================================================
// Device Authorization Flow (RFC 8628)
// =====================================================

func TestCreateDeviceAuthorization_Success(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	resp, err := svc.CreateDeviceAuthorization(&DeviceAuthorizationRequest{
		TenantID: testTenantID,
		ClientID: "device-client",
		Scope:    []string{"openid", "profile"},
		Issuer:   "https://test.ggid.dev",
	})
	if err != nil {
		t.Fatalf("CreateDeviceAuthorization: %v", err)
	}

	if resp.DeviceCode == "" {
		t.Error("expected non-empty device_code")
	}
	if resp.UserCode == "" {
		t.Error("expected non-empty user_code")
	}
	if resp.VerificationURI == "" {
		t.Error("expected non-empty verification_uri")
	}
	if resp.ExpiresIn != 900 {
		t.Errorf("expected expires_in=900, got %d", resp.ExpiresIn)
	}
	if resp.Interval != 5 {
		t.Errorf("expected interval=5, got %d", resp.Interval)
	}

	// User code should be in XXXX-XXXX format.
	if len(resp.UserCode) != 9 || resp.UserCode[4] != '-' {
		t.Errorf("expected user_code in XXXX-XXXX format, got '%s'", resp.UserCode)
	}
}

func TestCreateDeviceAuthorization_EmptyIssuer(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	resp, err := svc.CreateDeviceAuthorization(&DeviceAuthorizationRequest{
		TenantID: testTenantID,
		ClientID: "device-client",
		Issuer:   "",
	})
	if err != nil {
		t.Fatalf("CreateDeviceAuthorization: %v", err)
	}

	if resp.VerificationURI != "/device" {
		t.Errorf("expected '/device' for empty issuer, got '%s'", resp.VerificationURI)
	}
}

func TestPollDeviceToken_Pending(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	resp, err := svc.CreateDeviceAuthorization(&DeviceAuthorizationRequest{
		TenantID: testTenantID,
		ClientID: "device-client",
		Issuer:   "https://test.ggid.dev",
	})
	if err != nil {
		t.Fatalf("CreateDeviceAuthorization: %v", err)
	}

	// Polling immediately should return authorization_pending.
	_, err = svc.PollDeviceToken(context.Background(), resp.DeviceCode, "device-client")
	if err == nil {
		t.Fatal("expected error for pending device code")
	}
	if !strings.Contains(err.Error(), "authorization_pending") {
		t.Errorf("expected 'authorization_pending' error, got '%s'", err.Error())
	}
}

func TestPollDeviceToken_InvalidDeviceCode(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	_, err := svc.PollDeviceToken(context.Background(), "nonexistent-device-code", "device-client")
	if err == nil {
		t.Fatal("expected error for invalid device code")
	}
	if !strings.Contains(err.Error(), "invalid_device_code") {
		t.Errorf("expected 'invalid_device_code' error, got '%s'", err.Error())
	}
}

func TestPollDeviceToken_Approved(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	resp, err := svc.CreateDeviceAuthorization(&DeviceAuthorizationRequest{
		TenantID: testTenantID,
		ClientID: "device-client",
		Scope:    []string{"openid", "profile"},
		Issuer:   "https://test.ggid.dev",
	})
	if err != nil {
		t.Fatalf("CreateDeviceAuthorization: %v", err)
	}

	userID := uuid.New()

	// Approve the device code.
	if err := svc.ApproveDeviceCode(resp.UserCode, userID); err != nil {
		t.Fatalf("ApproveDeviceCode: %v", err)
	}

	// Now polling should return a token.
	tokenResp, err := svc.PollDeviceToken(context.Background(), resp.DeviceCode, "device-client")
	if err != nil {
		t.Fatalf("PollDeviceToken after approval: %v", err)
	}

	if tokenResp.AccessToken == "" {
		t.Error("expected non-empty access_token")
	}
	if tokenResp.TokenType != "Bearer" {
		t.Errorf("expected Bearer, got %s", tokenResp.TokenType)
	}
	if tokenResp.Scope != "openid profile" {
		t.Errorf("expected scope 'openid profile', got '%s'", tokenResp.Scope)
	}

	// After success, the device code should be cleaned up.
	_, err = svc.PollDeviceToken(context.Background(), resp.DeviceCode, "device-client")
	if err == nil {
		t.Error("expected error when polling after token issued (code should be cleaned up)")
	}
}

func TestPollDeviceToken_Expired(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	resp, err := svc.CreateDeviceAuthorization(&DeviceAuthorizationRequest{
		TenantID: testTenantID,
		ClientID: "device-client",
		Issuer:   "https://test.ggid.dev",
	})
	if err != nil {
		t.Fatalf("CreateDeviceAuthorization: %v", err)
	}

	// Manually expire the device code.
	deviceCodeMu.Lock()
	info := deviceCodeStore[resp.DeviceCode]
	info.ExpiresAt = time.Now().Add(-1 * time.Minute)
	deviceCodeMu.Unlock()

	_, err = svc.PollDeviceToken(context.Background(), resp.DeviceCode, "device-client")
	if err == nil {
		t.Fatal("expected error for expired device code")
	}
	if !strings.Contains(err.Error(), "expired_token") {
		t.Errorf("expected 'expired_token' error, got '%s'", err.Error())
	}

	// The expired code should be cleaned up.
	_, err = svc.PollDeviceToken(context.Background(), resp.DeviceCode, "device-client")
	if err == nil {
		t.Error("expected error after expired code cleanup")
	}
}

func TestApproveDeviceCode_InvalidUserCode(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	err := svc.ApproveDeviceCode("INVALID-CODE", uuid.New())
	if err == nil {
		t.Fatal("expected error for invalid user_code")
	}
	if !strings.Contains(err.Error(), "invalid user_code") {
		t.Errorf("expected 'invalid user_code' error, got '%s'", err.Error())
	}
}

func TestApproveDeviceCode_ExpiredUserCode(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	resp, err := svc.CreateDeviceAuthorization(&DeviceAuthorizationRequest{
		TenantID: testTenantID,
		ClientID: "device-client",
		Issuer:   "https://test.ggid.dev",
	})
	if err != nil {
		t.Fatalf("CreateDeviceAuthorization: %v", err)
	}

	// Expire the code.
	deviceCodeMu.Lock()
	info := deviceCodeStore[resp.DeviceCode]
	info.ExpiresAt = time.Now().Add(-1 * time.Minute)
	deviceCodeMu.Unlock()

	err = svc.ApproveDeviceCode(resp.UserCode, uuid.New())
	if err == nil {
		t.Fatal("expected error for expired user_code")
	}
	if !strings.Contains(err.Error(), "expired") {
		t.Errorf("expected 'expired' error, got '%s'", err.Error())
	}
}

func TestGenerateUserCode_Format(t *testing.T) {
	code := generateUserCode()
	if len(code) != 9 {
		t.Errorf("expected 9 chars (XXXX-XXXX), got %d", len(code))
	}
	if code[4] != '-' {
		t.Errorf("expected dash at position 4, got '%c'", code[4])
	}
}

func TestGenerateDeviceCode_Length(t *testing.T) {
	code := generateDeviceCode(20)
	if len(code) != 20 {
		t.Errorf("expected length 20, got %d", len(code))
	}
}

func TestCryptoRandInt(t *testing.T) {
	// Normal case.
	v := cryptoRandInt(10)
	if v < 0 || v >= 10 {
		t.Errorf("expected 0-9, got %d", v)
	}

	// Edge case: max <= 0.
	if v := cryptoRandInt(0); v != 0 {
		t.Errorf("expected 0 for max=0, got %d", v)
	}
	if v := cryptoRandInt(-1); v != 0 {
		t.Errorf("expected 0 for max=-1, got %d", v)
	}
}

// =====================================================
// JWT Bearer Assertion Grant (RFC 7523)
// =====================================================

func TestJWTBearerGrant_Success(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	userID := uuid.New()
	now := time.Now()
	assertion := makeTestAssertion(t, jwt.MapClaims{
		"iss": "trusted-issuer",
		"sub": userID.String(),
		"exp": now.Add(1 * time.Hour).Unix(),
	})

	resp, err := svc.JWTBearerGrant(context.Background(), &JWTBearerRequest{
		TenantID:  testTenantID,
		Assertion: assertion,
		Scope:     []string{"read", "write"},
	})
	if err != nil {
		t.Fatalf("JWTBearerGrant: %v", err)
	}

	if resp.AccessToken == "" {
		t.Error("expected non-empty access_token")
	}
	if resp.TokenType != "Bearer" {
		t.Errorf("expected Bearer, got %s", resp.TokenType)
	}
	if resp.Scope != "read write" {
		t.Errorf("expected scope 'read write', got '%s'", resp.Scope)
	}
	if resp.ExpiresIn <= 0 {
		t.Error("expected positive expires_in")
	}

	// Verify the token can be parsed back.
	claims, err := svc.ParseAccessToken(resp.AccessToken)
	if err != nil {
		t.Fatalf("ParseAccessToken: %v", err)
	}
	if sub, _ := claims["sub"].(string); sub != userID.String() {
		t.Errorf("expected sub=%s, got %s", userID, sub)
	}
	if assertionIss, _ := claims["assertion_iss"].(string); assertionIss != "trusted-issuer" {
		t.Errorf("expected assertion_iss=trusted-issuer, got %s", assertionIss)
	}
}

func TestJWTBearerGrant_MissingAssertion(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	_, err := svc.JWTBearerGrant(context.Background(), &JWTBearerRequest{
		TenantID: testTenantID,
	})
	if err == nil {
		t.Fatal("expected error for missing assertion")
	}
	if !strings.Contains(err.Error(), "assertion is required") {
		t.Errorf("expected 'assertion is required', got '%s'", err.Error())
	}
}

func TestJWTBearerGrant_InvalidAssertion(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	_, err := svc.JWTBearerGrant(context.Background(), &JWTBearerRequest{
		TenantID:  testTenantID,
		Assertion: "not.a.valid.jwt",
	})
	if err == nil {
		t.Fatal("expected error for invalid assertion")
	}
	if !strings.Contains(err.Error(), "invalid assertion") {
		t.Errorf("expected 'invalid assertion', got '%s'", err.Error())
	}
}

func TestJWTBearerGrant_Expired(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	userID := uuid.New()
	assertion := makeTestAssertion(t, jwt.MapClaims{
		"sub": userID.String(),
		"exp": float64(time.Now().Add(-1 * time.Hour).Unix()),
	})

	_, err := svc.JWTBearerGrant(context.Background(), &JWTBearerRequest{
		TenantID:  testTenantID,
		Assertion: assertion,
	})
	if err == nil {
		t.Fatal("expected error for expired assertion")
	}
	if !strings.Contains(err.Error(), "expired") {
		t.Errorf("expected 'expired' error, got '%s'", err.Error())
	}
}

func TestJWTBearerGrant_MissingSub(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	now := time.Now()
	assertion := makeTestAssertion(t, jwt.MapClaims{
		"exp": now.Add(1 * time.Hour).Unix(),
		// no sub claim
	})

	_, err := svc.JWTBearerGrant(context.Background(), &JWTBearerRequest{
		TenantID:  testTenantID,
		Assertion: assertion,
	})
	if err == nil {
		t.Fatal("expected error for missing sub")
	}
	if !strings.Contains(err.Error(), "sub") {
		t.Errorf("expected error about 'sub', got '%s'", err.Error())
	}
}

func TestJWTBearerGrant_MissingExp(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	userID := uuid.New()
	assertion := makeTestAssertion(t, jwt.MapClaims{
		"sub": userID.String(),
		// no exp claim
	})

	_, err := svc.JWTBearerGrant(context.Background(), &JWTBearerRequest{
		TenantID:  testTenantID,
		Assertion: assertion,
	})
	if err == nil {
		t.Fatal("expected error for missing exp")
	}
}

func TestJWTBearerGrant_InvalidSubUUID(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	now := time.Now()
	assertion := makeTestAssertion(t, jwt.MapClaims{
		"sub": "not-a-uuid",
		"exp": now.Add(1 * time.Hour).Unix(),
	})

	_, err := svc.JWTBearerGrant(context.Background(), &JWTBearerRequest{
		TenantID:  testTenantID,
		Assertion: assertion,
	})
	if err == nil {
		t.Fatal("expected error for invalid UUID sub")
	}
}

// =====================================================
// ParseBackchannelLogoutToken
// =====================================================

func TestParseBackchannelLogoutToken_Success(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	tokenStr := makeUnsignedToken(t, jwt.MapClaims{
		"sub": "user-123",
		"events": map[string]any{
			"http://schemas.openid.net/event/backchannel-logout": map[string]any{},
		},
	})

	claims, err := svc.ParseBackchannelLogoutToken(tokenStr)
	if err != nil {
		t.Fatalf("ParseBackchannelLogoutToken: %v", err)
	}
	if sub, _ := claims["sub"].(string); sub != "user-123" {
		t.Errorf("expected sub=user-123, got %v", claims["sub"])
	}
}

func TestParseBackchannelLogoutToken_WithSID(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	tokenStr := makeUnsignedToken(t, jwt.MapClaims{
		"sid": "session-456",
		"events": map[string]any{
			"http://schemas.openid.net/event/backchannel-logout": map[string]any{},
		},
	})

	claims, err := svc.ParseBackchannelLogoutToken(tokenStr)
	if err != nil {
		t.Fatalf("ParseBackchannelLogoutToken: %v", err)
	}
	if sid, _ := claims["sid"].(string); sid != "session-456" {
		t.Errorf("expected sid=session-456, got %v", claims["sid"])
	}
}

func TestParseBackchannelLogoutToken_InvalidToken(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	_, err := svc.ParseBackchannelLogoutToken("not-a-valid-jwt")
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
	if !strings.Contains(err.Error(), "invalid logout token") {
		t.Errorf("expected 'invalid logout token', got '%s'", err.Error())
	}
}

func TestParseBackchannelLogoutToken_MissingSubAndSID(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	tokenStr := makeUnsignedToken(t, jwt.MapClaims{
		"events": map[string]any{
			"http://schemas.openid.net/event/backchannel-logout": map[string]any{},
		},
		// no sub or sid
	})

	_, err := svc.ParseBackchannelLogoutToken(tokenStr)
	if err == nil {
		t.Fatal("expected error for missing sub and sid")
	}
	if !strings.Contains(err.Error(), "sub") || !strings.Contains(err.Error(), "sid") {
		t.Errorf("expected error mentioning sub or sid, got '%s'", err.Error())
	}
}

func TestParseBackchannelLogoutToken_MissingEvents(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	tokenStr := makeUnsignedToken(t, jwt.MapClaims{
		"sub": "user-123",
		// no events claim
	})

	_, err := svc.ParseBackchannelLogoutToken(tokenStr)
	if err == nil {
		t.Fatal("expected error for missing events")
	}
	if !strings.Contains(err.Error(), "events") {
		t.Errorf("expected error about 'events', got '%s'", err.Error())
	}
}

func TestParseBackchannelLogoutToken_WrongEvent(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	tokenStr := makeUnsignedToken(t, jwt.MapClaims{
		"sub": "user-123",
		"events": map[string]any{
			"http://schemas.openid.net/event/some-other-event": map[string]any{},
		},
	})

	_, err := svc.ParseBackchannelLogoutToken(tokenStr)
	if err == nil {
		t.Fatal("expected error for wrong event type")
	}
	if !strings.Contains(err.Error(), "backchannel-logout") {
		t.Errorf("expected error about backchannel-logout, got '%s'", err.Error())
	}
}

func TestParseBackchannelLogoutToken_HasNonce(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	tokenStr := makeUnsignedToken(t, jwt.MapClaims{
		"sub":   "user-123",
		"nonce": "abc123",
		"events": map[string]any{
			"http://schemas.openid.net/event/backchannel-logout": map[string]any{},
		},
	})

	_, err := svc.ParseBackchannelLogoutToken(tokenStr)
	if err == nil {
		t.Fatal("expected error for logout token with nonce")
	}
	if !strings.Contains(err.Error(), "nonce") {
		t.Errorf("expected error about 'nonce', got '%s'", err.Error())
	}
}

// =====================================================
// BackchannelLogout
// =====================================================

func TestBackchannelLogout_StoresLogout(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	sub := "user-backchannel-logout-test"
	svc.BackchannelLogout(sub)

	// Verify the logout was stored.
	key := "ggid:backchannel_logout:" + sub
	_, ok := backchannelLogoutList.Load(key)
	if !ok {
		t.Error("expected backchannel logout entry to be stored")
	}
}

// =====================================================
// RevokeToken + IsTokenRevoked
// =====================================================

func TestRevokeAndIsTokenRevoked_ValidToken(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	// Create a valid token via issueAccessToken.
	userID := uuid.New()
	token, _, err := svc.issueAccessToken(userID, testTenantID, "test-client")
	if err != nil {
		t.Fatalf("issueAccessToken: %v", err)
	}

	// Token should not be revoked initially.
	if svc.IsTokenRevoked(token) {
		t.Error("token should not be revoked before RevokeToken call")
	}

	// Revoke it.
	if err := svc.RevokeToken(token); err != nil {
		t.Fatalf("RevokeToken: %v", err)
	}

	// Now it should be revoked.
	if !svc.IsTokenRevoked(token) {
		t.Error("token should be revoked after RevokeToken call")
	}
}

func TestRevokeToken_InvalidTokenReturnsNil(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	// RFC 7009: invalid token should return nil (200 OK).
	err := svc.RevokeToken("totally-invalid-jwt-string")
	if err != nil {
		t.Errorf("expected nil for invalid token (RFC 7009), got: %v", err)
	}
}

func TestRevokeToken_EmptyTokenReturnsNil(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	err := svc.RevokeToken("")
	if err != nil {
		t.Errorf("expected nil for empty token, got: %v", err)
	}
}

func TestHashTokenSHA256_Deterministic(t *testing.T) {
	h1 := hashTokenSHA256("test-token")
	h2 := hashTokenSHA256("test-token")
	if h1 != h2 {
		t.Error("hashTokenSHA256 should be deterministic")
	}
	h3 := hashTokenSHA256("other-token")
	if h1 == h3 {
		t.Error("hashTokenSHA256 should differ for different inputs")
	}
}

// =====================================================
// IntrospectToken with valid token
// =====================================================

func TestIntrospectToken_ValidToken(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	userID := uuid.New()
	token, _, err := svc.issueAccessToken(userID, testTenantID, "introspect-client")
	if err != nil {
		t.Fatalf("issueAccessToken: %v", err)
	}

	result := svc.IntrospectToken(token)
	if !result.Active {
		t.Error("expected active=true for valid token")
	}
	if result.Sub != userID.String() {
		t.Errorf("expected sub=%s, got %s", userID, result.Sub)
	}
	if result.Iss != "https://test.ggid.dev" {
		t.Errorf("expected iss=https://test.ggid.dev, got %s", result.Iss)
	}
}

func TestIntrospectToken_RevokedToken(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	userID := uuid.New()
	token, _, err := svc.issueAccessToken(userID, testTenantID, "revoked-client")
	if err != nil {
		t.Fatalf("issueAccessToken: %v", err)
	}

	// Revoke then introspect.
	svc.RevokeToken(token)

	result := svc.IntrospectToken(token)
	if result.Active {
		t.Error("expected active=false for revoked token")
	}
}

// =====================================================
// GetUserInfo with valid token
// =====================================================

func TestGetUserInfo_ValidToken(t *testing.T) {
	userID := uuid.New()

	// Build a token with extra claims.
	now := time.Now()
	expiresAt := now.Add(15 * time.Minute)
	claims := jwt.MapClaims{
		"iss":       "https://test.ggid.dev",
		"sub":       userID.String(),
		"aud":       "userinfo-client",
		"iat":       now.Unix(),
		"exp":       expiresAt.Unix(),
		"jti":       uuid.New().String(),
		"tenant_id": testTenantID.String(),
		"name":      "Jane Doe",
		"email":     "jane@example.com",
		"picture":   "https://example.com/avatar.png",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = "test-kid"
	kp := newMockKeyProvider()
	signed, err := token.SignedString(kp.PrivateKey())
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	// Recreate service with same key provider.
	svc2 := NewOAuthService(newMockClientRepo(), newMockCodeRepo(), &mockTokenRepo{}, kp, "https://test.ggid.dev")

	userInfo, err := svc2.GetUserInfo(signed)
	if err != nil {
		t.Fatalf("GetUserInfo: %v", err)
	}

	if userInfo.Sub != userID.String() {
		t.Errorf("expected sub=%s, got %s", userID, userInfo.Sub)
	}
	if userInfo.Name != "Jane Doe" {
		t.Errorf("expected name=Jane Doe, got %s", userInfo.Name)
	}
	if userInfo.Email != "jane@example.com" {
		t.Errorf("expected email=jane@example.com, got %s", userInfo.Email)
	}
	if userInfo.Picture != "https://example.com/avatar.png" {
		t.Errorf("expected picture URL, got %s", userInfo.Picture)
	}
	if userInfo.TenantID != testTenantID.String() {
		t.Errorf("expected tenant_id=%s, got %s", testTenantID, userInfo.TenantID)
	}
}

// =====================================================
// Token Exchange (RFC 8693)
// =====================================================

func TestExchangeToken_Success(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	userID := uuid.New()
	subjectToken, _, err := svc.issueAccessToken(userID, testTenantID, "source-client")
	if err != nil {
		t.Fatalf("issueAccessToken: %v", err)
	}

	resp, err := svc.ExchangeToken(context.Background(), &TokenExchangeRequestRFC8693{
		TenantID:         testTenantID,
		SubjectToken:     subjectToken,
		SubjectTokenType: "urn:ietf:params:oauth:token-type:access_token",
		Audience:         "target-service",
		Scope:            []string{"read"},
	})
	if err != nil {
		t.Fatalf("ExchangeToken: %v", err)
	}

	if resp.AccessToken == "" {
		t.Error("expected non-empty access_token")
	}
	if !strings.HasPrefix(resp.AccessToken, "exchanged_") {
		t.Errorf("expected 'exchanged_' prefix, got %s", resp.AccessToken)
	}
	if resp.TokenType != "N_A" {
		t.Errorf("expected N_A, got %s", resp.TokenType)
	}
	if resp.ExpiresIn != 3600 {
		t.Errorf("expected 3600, got %d", resp.ExpiresIn)
	}
}

func TestExchangeToken_MissingSubjectToken(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	_, err := svc.ExchangeToken(context.Background(), &TokenExchangeRequestRFC8693{
		TenantID:         testTenantID,
		SubjectTokenType: "urn:ietf:params:oauth:token-type:access_token",
	})
	if err == nil {
		t.Fatal("expected error for missing subject_token")
	}
	if !strings.Contains(err.Error(), "subject_token is required") {
		t.Errorf("expected 'subject_token is required', got '%s'", err.Error())
	}
}

func TestExchangeToken_MissingSubjectTokenType(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	_, err := svc.ExchangeToken(context.Background(), &TokenExchangeRequestRFC8693{
		TenantID:     testTenantID,
		SubjectToken: "some-token",
	})
	if err == nil {
		t.Fatal("expected error for missing subject_token_type")
	}
	if !strings.Contains(err.Error(), "subject_token_type is required") {
		t.Errorf("expected 'subject_token_type is required', got '%s'", err.Error())
	}
}

func TestExchangeToken_InvalidSubjectToken(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	_, err := svc.ExchangeToken(context.Background(), &TokenExchangeRequestRFC8693{
		TenantID:         testTenantID,
		SubjectToken:     "not-a-valid-jwt",
		SubjectTokenType: "urn:ietf:params:oauth:token-type:access_token",
	})
	if err == nil {
		t.Fatal("expected error for invalid subject_token")
	}
	if !strings.Contains(err.Error(), "invalid subject_token") {
		t.Errorf("expected 'invalid subject_token', got '%s'", err.Error())
	}
}

func TestExchangeToken_MissingSub(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	// Build a token with no sub claim using the service's own key.
	now := time.Now()
	claims := jwt.MapClaims{
		"iss": "https://test.ggid.dev",
		"aud": "test-client",
		"iat": now.Unix(),
		"exp": now.Add(1 * time.Hour).Unix(),
		// no sub
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = "test-kid"
	kp := newMockKeyProvider()
	signed, err := token.SignedString(kp.PrivateKey())
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	_, err = svc.ExchangeToken(context.Background(), &TokenExchangeRequestRFC8693{
		TenantID:         testTenantID,
		SubjectToken:     signed,
		SubjectTokenType: "urn:ietf:params:oauth:token-type:access_token",
	})
	if err == nil {
		t.Fatal("expected error for token missing sub")
	}
	if !strings.Contains(err.Error(), "sub") {
		t.Errorf("expected error about 'sub', got '%s'", err.Error())
	}
}

// =====================================================
// RefreshToken grant — more scenarios
// =====================================================

func TestRefreshToken_ClientNotFound(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	_, err := svc.RefreshToken(context.Background(), &RefreshTokenRequest{
		TenantID:     testTenantID,
		RefreshToken: "user123.refreshsecret",
		ClientID:     "nonexistent-client",
	})
	if err == nil {
		t.Fatal("expected error for non-existent client")
	}
}

func TestRefreshToken_WrongSecret(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	// Create a client.
	result, _ := svc.CreateClient(context.Background(), &CreateClientInput{
		TenantID:   testTenantID,
		Name:       "refresh-client",
		Type:       domain.ClientTypeConfidential,
		GrantTypes: []string{"refresh_token"},
	})

	_, err := svc.RefreshToken(context.Background(), &RefreshTokenRequest{
		TenantID:     testTenantID,
		RefreshToken: uuid.New().String() + ".refresh",
		ClientID:     result.Client.ClientID,
		ClientSecret: "wrong-secret",
	})
	if err == nil {
		t.Fatal("expected error for wrong secret")
	}
}

func TestRefreshToken_UnsupportedGrant(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	// Create a client that does NOT support refresh_token.
	result, _ := svc.CreateClient(context.Background(), &CreateClientInput{
		TenantID:   testTenantID,
		Name:       "no-refresh-client",
		Type:       domain.ClientTypeConfidential,
		GrantTypes: []string{"authorization_code"},
	})

	_, err := svc.RefreshToken(context.Background(), &RefreshTokenRequest{
		TenantID:     testTenantID,
		RefreshToken: uuid.New().String() + ".refresh",
		ClientID:     result.Client.ClientID,
		ClientSecret: result.ClientSecret,
	})
	if err == nil {
		t.Fatal("expected error for unsupported grant type")
	}
}

func TestRefreshToken_Success(t *testing.T) {
	svc, _, _, tokenRepo := newTestOAuthService()

	result, _ := svc.CreateClient(context.Background(), &CreateClientInput{
		TenantID:   testTenantID,
		Name:       "refresh-ok-client",
		Type:       domain.ClientTypeConfidential,
		GrantTypes: []string{"authorization_code", "refresh_token"},
	})

	userID := uuid.New()
	refreshToken := userID.String() + ".some-refresh-secret"

	// Store a valid refresh token record (required by the token rotation logic).
	tokenRepo.StoreRefreshToken(context.Background(), &domain.RefreshTokenRecord{
		ID:        uuid.New(),
		TenantID:  testTenantID,
		ClientID:  result.Client.ID,
		UserID:    userID,
		TokenHash: hashTokenSHA256(refreshToken),
		Scope:     []string{"openid", "profile"},
		ExpiresAt: time.Now().Add(24 * time.Hour),
	})

	resp, err := svc.RefreshToken(context.Background(), &RefreshTokenRequest{
		TenantID:     testTenantID,
		RefreshToken: refreshToken,
		ClientID:     result.Client.ClientID,
		ClientSecret: result.ClientSecret,
		Scope:        []string{"openid", "profile"},
	})
	if err != nil {
		t.Fatalf("RefreshToken: %v", err)
	}

	if resp.AccessToken == "" {
		t.Error("expected non-empty access_token")
	}
	if resp.TokenType != "Bearer" {
		t.Errorf("expected Bearer, got %s", resp.TokenType)
	}
	if resp.Scope != "openid profile" {
		t.Errorf("expected scope 'openid profile', got '%s'", resp.Scope)
	}
}

func TestRefreshToken_InvalidUserIDInToken(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	result, _ := svc.CreateClient(context.Background(), &CreateClientInput{
		TenantID:   testTenantID,
		Name:       "refresh-bad-uid",
		Type:       domain.ClientTypeConfidential,
		GrantTypes: []string{"refresh_token"},
	})

	// refresh token with no dot (invalid format).
	_, err := svc.RefreshToken(context.Background(), &RefreshTokenRequest{
		TenantID:     testTenantID,
		RefreshToken: "invalidformat-nodot",
		ClientID:     result.Client.ClientID,
		ClientSecret: result.ClientSecret,
	})
	if err == nil {
		t.Fatal("expected error for invalid refresh token format")
	}

	// refresh token with non-UUID user id.
	_, err = svc.RefreshToken(context.Background(), &RefreshTokenRequest{
		TenantID:     testTenantID,
		RefreshToken: "notauuid.somesecret",
		ClientID:     result.Client.ClientID,
		ClientSecret: result.ClientSecret,
	})
	if err == nil {
		t.Fatal("expected error for non-UUID user ID in refresh token")
	}
}

// =====================================================
// ClientCredentials — more scenarios
// =====================================================

func TestClientCredentials_ClientNotFound(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	_, err := svc.ClientCredentials(context.Background(), &ClientCredentialsRequest{
		TenantID: testTenantID,
		ClientID: "nonexistent-m2m",
	})
	if err == nil {
		t.Fatal("expected error for non-existent client")
	}
}

// =====================================================
// ExchangeAuthorizationCode — wrong client (code issued to different client)
// =====================================================

func TestExchangeAuthorizationCode_CodeIssuedToDifferentClient(t *testing.T) {
	svc, clientRepo, _, _ := newTestOAuthService()

	// Client A — will receive the code.
	secretHashA, _ := crypto.HashPassword("secret-a")
	clientA := &domain.OAuthClient{
		ID:               uuid.New(),
		TenantID:         testTenantID,
		ClientID:         "client_a",
		ClientSecretHash: secretHashA,
		Type:             domain.ClientTypeConfidential,
		RedirectURIs:     []string{"https://a.example.com/cb"},
		Enabled:          true,
	}
	clientRepo.clients["client_a"] = clientA

	// Client B — will try to use the code.
	secretHashB, _ := crypto.HashPassword("secret-b")
	clientB := &domain.OAuthClient{
		ID:               uuid.New(),
		TenantID:         testTenantID,
		ClientID:         "client_b",
		ClientSecretHash: secretHashB,
		Type:             domain.ClientTypeConfidential,
		RedirectURIs:     []string{"https://b.example.com/cb"},
		Enabled:          true,
	}
	clientRepo.clients["client_b"] = clientB

	// Create code for client A.
	plainCode, err := svc.CreateAuthorizationCode(context.Background(), &AuthorizeRequest{
		TenantID:    testTenantID,
		ClientID:    "client_a",
		RedirectURI: "https://a.example.com/cb",
		ResponseType: "code",
		State:       "test-state",
		UserID:      uuid.New(),
	})
	if err != nil {
		t.Fatalf("CreateAuthorizationCode: %v", err)
	}

	// Try to exchange with client B.
	_, err = svc.ExchangeAuthorizationCode(context.Background(), &TokenExchangeRequest{
		TenantID:     testTenantID,
		GrantType:    "authorization_code",
		Code:         plainCode,
		RedirectURI:  "https://a.example.com/cb",
		ClientID:     "client_b",
		ClientSecret: "secret-b",
	})
	if err == nil {
		t.Fatal("expected error for code issued to different client")
	}
}

// cryptoHashPassword was removed — use crypto.HashPassword directly.

// =====================================================
// PKCE exchange failure
// =====================================================

func TestExchangeAuthorizationCode_PKCEFailure(t *testing.T) {
	svc, clientRepo, _, _ := newTestOAuthService()

	secretHash, _ := crypto.HashPassword("secret")
	clientRepo.clients["pkce_client"] = &domain.OAuthClient{
		ID:               uuid.New(),
		TenantID:         testTenantID,
		ClientID:         "pkce_client",
		ClientSecretHash: secretHash,
		Type:             domain.ClientTypeConfidential,
		RedirectURIs:     []string{"https://pkce.example.com/cb"},
		Enabled:          true,
	}

	// Create code with PKCE challenge.
	plainCode, err := svc.CreateAuthorizationCode(context.Background(), &AuthorizeRequest{
		TenantID:            testTenantID,
		ClientID:            "pkce_client",
		RedirectURI:         "https://pkce.example.com/cb",
		ResponseType:        "code",
		State:               "test-state",
		CodeChallenge:       "some-challenge-value",
		CodeChallengeMethod: "S256",
		UserID:              uuid.New(),
	})
	if err != nil {
		t.Fatalf("CreateAuthorizationCode: %v", err)
	}

	// Exchange with wrong verifier — should fail.
	_, err = svc.ExchangeAuthorizationCode(context.Background(), &TokenExchangeRequest{
		TenantID:     testTenantID,
		GrantType:    "authorization_code",
		Code:         plainCode,
		RedirectURI:  "https://pkce.example.com/cb",
		ClientID:     "pkce_client",
		ClientSecret: "secret",
		CodeVerifier: "wrong-verifier",
	})
	if err == nil {
		t.Fatal("expected error for PKCE verification failure")
	}
}

// =====================================================
// GetClient / ListClients / DeleteClient — missing tenant
// =====================================================

func TestGetClient_MissingTenant(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	// No tenant context.
	_, err := svc.GetClient(context.Background(), "test-client")
	if err == nil {
		t.Fatal("expected error for missing tenant context")
	}
}

func TestListClients_MissingTenant(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	_, _, err := svc.ListClients(context.Background(), 10, 0)
	if err == nil {
		t.Fatal("expected error for missing tenant context")
	}
}

func TestDeleteClient_MissingTenant(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	err := svc.DeleteClient(context.Background(), "test-client")
	if err == nil {
		t.Fatal("expected error for missing tenant context")
	}
}

// =====================================================
// CreateClient — error path (hash failure impossible with real crypto,
// but we test the non-confidential default auth method path)
// =====================================================

func TestCreateClient_PublicClientDefaultAuth(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	result, err := svc.CreateClient(context.Background(), &CreateClientInput{
		TenantID: testTenantID,
		Name:     "Public Default",
		Type:     domain.ClientTypePublic,
	})
	if err != nil {
		t.Fatalf("CreateClient: %v", err)
	}

	// Public client should still get default auth method.
	if result.Client.TokenEndpointAuthMethod != "client_secret_basic" {
		t.Errorf("expected default auth method, got '%s'", result.Client.TokenEndpointAuthMethod)
	}
}

// =====================================================
// DynamicClientRegister — additional metadata fields
// =====================================================

func TestDynamicClientRegister_WithMetadata(t *testing.T) {
	svc, clientRepo, _, _ := newTestOAuthService()

	ctx := tenant.WithContext(context.Background(), &tenant.Context{
		TenantID:       testTenantID,
		IsolationLevel: tenant.IsolationShared,
	})

	resp, err := svc.DynamicClientRegister(ctx, &DynamicRegistrationRequest{
		RedirectURIs:    []string{"https://app.example.com/cb"},
		ClientName:      "Meta App",
		ClientURI:       "https://meta.example.com",
		LogoURI:         "https://meta.example.com/logo.png",
		PolicyURI:       "https://meta.example.com/policy",
		TosURI:          "https://meta.example.com/tos",
		JwksURI:         "https://meta.example.com/jwks",
		SoftwareID:      "sw-123",
		SoftwareVersion: "1.0.0",
	})
	if err != nil {
		t.Fatalf("DynamicClientRegister: %v", err)
	}

	// Verify metadata was stored.
	stored := clientRepo.clients[resp.ClientID]
	if stored.Metadata["client_uri"] != "https://meta.example.com" {
		t.Errorf("expected client_uri in metadata, got %v", stored.Metadata["client_uri"])
	}
	if stored.Metadata["software_id"] != "sw-123" {
		t.Errorf("expected software_id in metadata, got %v", stored.Metadata["software_id"])
	}
}

func TestDynamicClientRegister_DefaultClientName(t *testing.T) {
	svc, clientRepo, _, _ := newTestOAuthService()

	ctx := tenant.WithContext(context.Background(), &tenant.Context{
		TenantID:       testTenantID,
		IsolationLevel: tenant.IsolationShared,
	})

	resp, err := svc.DynamicClientRegister(ctx, &DynamicRegistrationRequest{
		RedirectURIs: []string{"https://app.example.com/cb"},
		// no ClientName — should default
	})
	if err != nil {
		t.Fatalf("DynamicClientRegister: %v", err)
	}

	if resp.ClientName != "Dynamic Client" {
		t.Errorf("expected default name 'Dynamic Client', got '%s'", resp.ClientName)
	}

	stored := clientRepo.clients[resp.ClientID]
	if stored.Name != "Dynamic Client" {
		t.Errorf("expected stored name 'Dynamic Client', got '%s'", stored.Name)
	}
}

// =====================================================
// CreateAuthorizationCode — response_type not allowed
// =====================================================

func TestCreateAuthorizationCode_ResponseTypeNotAllowed(t *testing.T) {
	svc, clientRepo, _, _ := newTestOAuthService()

	clientRepo.clients["rt_client"] = &domain.OAuthClient{
		ID:            uuid.New(),
		TenantID:      testTenantID,
		ClientID:      "rt_client",
		RedirectURIs:  []string{"https://rt.example.com/cb"},
		ResponseTypes: []string{"code"}, // only "code" allowed
		Enabled:       true,
	}

	_, err := svc.CreateAuthorizationCode(context.Background(), &AuthorizeRequest{
		TenantID:     testTenantID,
		ClientID:     "rt_client",
		RedirectURI:  "https://rt.example.com/cb",
		ResponseType: "token", // not allowed
	})
	if err == nil {
		t.Fatal("expected error for disallowed response_type")
	}
}

// =====================================================
// ExchangeAuthorizationCode — client not found
// =====================================================

func TestExchangeAuthorizationCode_ClientNotFound(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	_, err := svc.ExchangeAuthorizationCode(context.Background(), &TokenExchangeRequest{
		TenantID: testTenantID,
		Code:     "some-code",
		ClientID: "nonexistent",
	})
	if err == nil {
		t.Fatal("expected error for non-existent client")
	}
}

// =====================================================
// ExchangeAuthorizationCode — consumed code (already used)
// =====================================================

func TestExchangeAuthorizationCode_CodeAlreadyConsumed(t *testing.T) {
	svc, clientRepo, _, _ := newTestOAuthService()

	secretHash, _ := crypto.HashPassword("secret")
	clientRepo.clients["consumed_client"] = &domain.OAuthClient{
		ID:               uuid.New(),
		TenantID:         testTenantID,
		ClientID:         "consumed_client",
		ClientSecretHash: secretHash,
		Type:             domain.ClientTypePublic,
		RedirectURIs:     []string{"https://c.example.com/cb"},
		Enabled:          true,
	}

	// Create code for public client (with PKCE).
	plainCode, err := svc.CreateAuthorizationCode(context.Background(), &AuthorizeRequest{
		TenantID:            testTenantID,
		ClientID:            "consumed_client",
		RedirectURI:         "https://c.example.com/cb",
		ResponseType:        "code",
		State:               "test-state",
		CodeChallenge:       "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
		CodeChallengeMethod: "S256",
		UserID:              uuid.New(),
	})
	if err != nil {
		t.Fatalf("CreateAuthorizationCode: %v", err)
	}

	// First exchange succeeds (correct PKCE verifier from RFC 7636).
	_, err = svc.ExchangeAuthorizationCode(context.Background(), &TokenExchangeRequest{
		TenantID:     testTenantID,
		GrantType:    "authorization_code",
		Code:         plainCode,
		RedirectURI:  "https://c.example.com/cb",
		ClientID:     "consumed_client",
		CodeVerifier: "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk",
	})
	if err != nil {
		t.Fatalf("first exchange: %v", err)
	}

	// Second exchange fails (already consumed).
	_, err = svc.ExchangeAuthorizationCode(context.Background(), &TokenExchangeRequest{
		TenantID:     testTenantID,
		GrantType:    "authorization_code",
		Code:         plainCode,
		RedirectURI:  "https://c.example.com/cb",
		ClientID:     "consumed_client",
		CodeVerifier: "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk",
	})
	if err == nil {
		t.Fatal("expected error for already consumed code")
	}
}

// =====================================================
// RotateClientSecret — public client (no secret check)
// =====================================================

func TestRotateClientSecret_PublicClient(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	result, _ := svc.CreateClient(context.Background(), &CreateClientInput{
		TenantID: testTenantID,
		Name:     "public-rotate",
		Type:     domain.ClientTypePublic,
	})

	newSecret, err := svc.RotateClientSecret(context.Background(), testTenantID, result.Client.ClientID, "")
	if err != nil {
		t.Fatalf("RotateClientSecret for public client: %v", err)
	}
	if newSecret == "" {
		t.Error("expected non-empty new secret")
	}
}

// =====================================================
// issueAccessToken direct test
// =====================================================

func TestIssueAccessToken_Success(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	userID := uuid.New()
	token, expiresIn, err := svc.issueAccessToken(userID, testTenantID, "direct-test-client")
	if err != nil {
		t.Fatalf("issueAccessToken: %v", err)
	}
	if token == "" {
		t.Error("expected non-empty token")
	}
	if expiresIn <= 0 {
		t.Error("expected positive expires_in")
	}

	// Parse it back.
	claims, err := svc.ParseAccessToken(token)
	if err != nil {
		t.Fatalf("ParseAccessToken: %v", err)
	}
	if sub, _ := claims["sub"].(string); sub != userID.String() {
		t.Errorf("expected sub=%s, got %v", userID, claims["sub"])
	}
}

// =====================================================
// IntrospectToken with scope claim
// =====================================================

func TestIntrospectToken_WithScope(t *testing.T) {
	now := time.Now()
	claims := jwt.MapClaims{
		"iss":   "https://test.ggid.dev",
		"sub":   "scope-user",
		"aud":   "scope-client",
		"iat":   now.Unix(),
		"exp":   now.Add(1 * time.Hour).Unix(),
		"scope": "read write",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = "test-kid"
	kp := newMockKeyProvider()
	signed, err := token.SignedString(kp.PrivateKey())
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	svc2 := NewOAuthService(newMockClientRepo(), newMockCodeRepo(), &mockTokenRepo{}, kp, "https://test.ggid.dev")

	result := svc2.IntrospectToken(signed)
	if !result.Active {
		t.Error("expected active=true")
	}
	if result.Scope != "read write" {
		t.Errorf("expected scope 'read write', got '%s'", result.Scope)
	}
}

// =====================================================
// defaultIfEmpty2 helper
// =====================================================

func TestDefaultIfEmpty2(t *testing.T) {
	if defaultIfEmpty2("", "fallback") != "fallback" {
		t.Error("expected 'fallback' for empty input")
	}
	if defaultIfEmpty2("value", "fallback") != "value" {
		t.Error("expected 'value' for non-empty input")
	}
}

// =====================================================
// ClientType.IsValid
// =====================================================

func TestClientType_IsValid(t *testing.T) {
	if !domain.ClientTypeConfidential.IsValid() {
		t.Error("confidential should be valid")
	}
	if !domain.ClientTypePublic.IsValid() {
		t.Error("public should be valid")
	}
	if domain.ClientType("invalid").IsValid() {
		t.Error("invalid type should not be valid")
	}
}

// =====================================================
// OAuthClient.MetadataJSON
// =====================================================

func TestOAuthClient_MetadataJSON_Nil(t *testing.T) {
	client := &domain.OAuthClient{}
	raw := client.MetadataJSON()
	if string(raw) != "{}" {
		t.Errorf("expected '{}' for nil metadata, got '%s'", string(raw))
	}
}

func TestOAuthClient_MetadataJSON_WithData(t *testing.T) {
	client := &domain.OAuthClient{
		Metadata: map[string]any{"key": "value"},
	}
	raw := client.MetadataJSON()
	if !strings.Contains(string(raw), "key") {
		t.Errorf("expected JSON to contain 'key', got '%s'", string(raw))
	}
}

// =====================================================
// AuthorizationCode.IsExpired
// =====================================================

func TestAuthorizationCode_IsExpired(t *testing.T) {
	code := &domain.AuthorizationCode{
		ExpiresAt: time.Now().Add(-1 * time.Minute),
	}
	if !code.IsExpired() {
		t.Error("expected code to be expired")
	}

	code2 := &domain.AuthorizationCode{
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}
	if code2.IsExpired() {
		t.Error("expected code to not be expired")
	}
}
