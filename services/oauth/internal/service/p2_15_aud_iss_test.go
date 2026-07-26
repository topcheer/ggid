package service

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// TestParseAccessToken_RejectsWrongIssuer verifies that ParseAccessToken
// enforces iss claim matching (RFC 7519 §4.1.1).
func TestParseAccessToken_RejectsWrongIssuer(t *testing.T) {
	userID := uuid.New()
	now := time.Now()
	claims := jwt.MapClaims{
		"iss":       "https://wrong-issuer.example.com",
		"sub":       userID.String(),
		"aud":       "test-client",
		"iat":       now.Unix(),
		"exp":       now.Add(15 * time.Minute).Unix(),
		"jti":       uuid.New().String(),
		"tenant_id": testTenantID.String(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = "test-kid"
	kp := newMockKeyProvider()
	signed, err := token.SignedString(kp.Signer())
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	svc := NewOAuthService(newMockClientRepo(), newMockCodeRepo(), &mockTokenRepo{}, kp, "https://test.ggid.dev")
	_, err = svc.ParseAccessToken(signed)
	if err == nil {
		t.Fatal("ParseAccessToken should reject token with wrong issuer")
	}
}

// TestParseAccessToken_AcceptsCorrectIssuer verifies that tokens with
// matching iss claim are accepted.
func TestParseAccessToken_AcceptsCorrectIssuer(t *testing.T) {
	userID := uuid.New()
	now := time.Now()
	claims := jwt.MapClaims{
		"iss":       "https://test.ggid.dev",
		"sub":       userID.String(),
		"aud":       "test-client",
		"iat":       now.Unix(),
		"exp":       now.Add(15 * time.Minute).Unix(),
		"jti":       uuid.New().String(),
		"tenant_id": testTenantID.String(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = "test-kid"
	kp := newMockKeyProvider()
	signed, err := token.SignedString(kp.Signer())
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	svc := NewOAuthService(newMockClientRepo(), newMockCodeRepo(), &mockTokenRepo{}, kp, "https://test.ggid.dev")
	parsedClaims, err := svc.ParseAccessToken(signed)
	if err != nil {
		t.Fatalf("ParseAccessToken should accept valid token: %v", err)
	}
	if parsedClaims["sub"] != userID.String() {
		t.Fatalf("expected sub=%s, got %v", userID.String(), parsedClaims["sub"])
	}
}

// TestParseAccessTokenWithAudience_RejectsWrongAudience verifies that
// ParseAccessTokenWithAudience enforces aud claim when expectedAudience is set.
func TestParseAccessTokenWithAudience_RejectsWrongAudience(t *testing.T) {
	userID := uuid.New()
	now := time.Now()
	claims := jwt.MapClaims{
		"iss":       "https://test.ggid.dev",
		"sub":       userID.String(),
		"aud":       "client-A",
		"iat":       now.Unix(),
		"exp":       now.Add(15 * time.Minute).Unix(),
		"jti":       uuid.New().String(),
		"tenant_id": testTenantID.String(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = "test-kid"
	kp := newMockKeyProvider()
	signed, err := token.SignedString(kp.Signer())
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	svc := NewOAuthService(newMockClientRepo(), newMockCodeRepo(), &mockTokenRepo{}, kp, "https://test.ggid.dev")
	_, err = svc.ParseAccessTokenWithAudience(signed, "client-B")
	if err == nil {
		t.Fatal("ParseAccessTokenWithAudience should reject token with wrong audience")
	}
}

// TestParseAccessTokenWithAudience_AcceptsMatchingAudience verifies that
// tokens with matching aud claim are accepted.
func TestParseAccessTokenWithAudience_AcceptsMatchingAudience(t *testing.T) {
	userID := uuid.New()
	now := time.Now()
	claims := jwt.MapClaims{
		"iss":       "https://test.ggid.dev",
		"sub":       userID.String(),
		"aud":       "resource-server-1",
		"iat":       now.Unix(),
		"exp":       now.Add(15 * time.Minute).Unix(),
		"jti":       uuid.New().String(),
		"tenant_id": testTenantID.String(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = "test-kid"
	kp := newMockKeyProvider()
	signed, err := token.SignedString(kp.Signer())
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	svc := NewOAuthService(newMockClientRepo(), newMockCodeRepo(), &mockTokenRepo{}, kp, "https://test.ggid.dev")
	_, err = svc.ParseAccessTokenWithAudience(signed, "resource-server-1")
	if err != nil {
		t.Fatalf("ParseAccessTokenWithAudience should accept matching audience: %v", err)
	}
}

// TestParseAccessTokenWithAudience_EmptyAudienceSkipsCheck verifies that
// empty expectedAudience skips aud verification (backward compatible).
func TestParseAccessTokenWithAudience_EmptyAudienceSkipsCheck(t *testing.T) {
	userID := uuid.New()
	now := time.Now()
	claims := jwt.MapClaims{
		"iss":       "https://test.ggid.dev",
		"sub":       userID.String(),
		"aud":       "any-client",
		"iat":       now.Unix(),
		"exp":       now.Add(15 * time.Minute).Unix(),
		"jti":       uuid.New().String(),
		"tenant_id": testTenantID.String(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = "test-kid"
	kp := newMockKeyProvider()
	signed, err := token.SignedString(kp.Signer())
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	svc := NewOAuthService(newMockClientRepo(), newMockCodeRepo(), &mockTokenRepo{}, kp, "https://test.ggid.dev")
	_, err = svc.ParseAccessTokenWithAudience(signed, "")
	if err != nil {
		t.Fatalf("ParseAccessTokenWithAudience with empty audience should skip aud check: %v", err)
	}
}