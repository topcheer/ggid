package service

import (
	cryptorand "crypto/rand"
	"crypto/rsa"
	"testing"

	"github.com/ggid/ggid/services/oauth/internal/domain"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// TestTokenExchange_FullFlow exercises RFC 8693 token exchange:
// issue a valid subject_token -> exchange -> verify new token with reduced scope.
func TestTokenExchange_FullFlow(t *testing.T) {
	kp := newMockKeyProvider()
	svc := NewOAuthService(newMockClientRepo(), newMockCodeRepo(), &mockTokenRepo{}, kp, "https://test.ggid.dev")

	userUUID := uuid.New()

	// Issue a valid access token to use as subject_token.
	subjectToken, _, err := svc.issueAccessToken(userUUID, testTenantID, "ggid", "openid profile email")
	if err != nil {
		t.Fatalf("issueAccessToken: %v", err)
	}

	// Exchange the token with reduced scope.
	resp, err := svc.ExchangeToken(t.Context(), &TokenExchangeRequestRFC8693{
		TenantID:         testTenantID,
		SubjectToken:     subjectToken,
		SubjectTokenType: "urn:ietf:params:oauth:token-type:access_token",
		Scope:            []string{"openid"},
		Audience:         "downstream-api",
	})
	if err != nil {
		t.Fatalf("ExchangeToken: %v", err)
	}

	if resp.AccessToken == "" {
		t.Fatal("exchanged access token is empty")
	}
	if resp.TokenType != "N_A" {
		t.Fatalf("expected token_type=N_A, got %s", resp.TokenType)
	}
	if resp.ExpiresIn != 3600 {
		t.Fatalf("expected expires_in=3600, got %d", resp.ExpiresIn)
	}
	if resp.Scope != "openid" {
		t.Fatalf("expected scope='openid', got '%s'", resp.Scope)
	}

	// Verify the exchanged token is different from the subject token.
	if resp.AccessToken == subjectToken {
		t.Fatal("exchanged token should be different from subject token")
	}
}

// TestTokenExchange_MissingSubjectToken verifies validation error.
func TestTokenExchange_MissingSubjectToken(t *testing.T) {
	svc := NewOAuthService(newMockClientRepo(), newMockCodeRepo(), &mockTokenRepo{}, newMockKeyProvider(), "https://test.ggid.dev")

	_, err := svc.ExchangeToken(t.Context(), &TokenExchangeRequestRFC8693{
		SubjectToken:     "",
		SubjectTokenType: "urn:ietf:params:oauth:token-type:access_token",
	})
	if err == nil {
		t.Fatal("expected error for missing subject_token")
	}
	if err.Error() != "subject_token is required" {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestTokenExchange_MissingTokenType verifies validation error.
func TestTokenExchange_MissingTokenType(t *testing.T) {
	svc := NewOAuthService(newMockClientRepo(), newMockCodeRepo(), &mockTokenRepo{}, newMockKeyProvider(), "https://test.ggid.dev")

	_, err := svc.ExchangeToken(t.Context(), &TokenExchangeRequestRFC8693{
		SubjectToken:     "some-token",
		SubjectTokenType: "",
	})
	if err == nil {
		t.Fatal("expected error for missing subject_token_type")
	}
	if err.Error() != "subject_token_type is required" {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestTokenExchange_InvalidSubjectToken verifies that a malformed JWT is rejected.
func TestTokenExchange_InvalidSubjectToken(t *testing.T) {
	svc := NewOAuthService(newMockClientRepo(), newMockCodeRepo(), &mockTokenRepo{}, newMockKeyProvider(), "https://test.ggid.dev")

	_, err := svc.ExchangeToken(t.Context(), &TokenExchangeRequestRFC8693{
		SubjectToken:     "invalid.jwt.token",
		SubjectTokenType: "urn:ietf:params:oauth:token-type:access_token",
	})
	if err == nil {
		t.Fatal("expected error for invalid subject_token")
	}
}

// TestTokenExchange_WrongSignature verifies that a token signed with a different key is rejected.
func TestTokenExchange_WrongSignature(t *testing.T) {
	// Service uses the singleton mock key provider.
	svcA := NewOAuthService(newMockClientRepo(), newMockCodeRepo(), &mockTokenRepo{}, newMockKeyProvider(), "https://test.ggid.dev")

	// Generate a completely separate RSA key for the "attacker" token.
	attackerKey, err := rsa.GenerateKey(cryptorand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate attacker key: %v", err)
	}

	claims := jwt.MapClaims{
		"sub": uuid.New().String(),
		"iss": "https://test.ggid.dev",
		"aud": "ggid",
		"iat": 1700000000,
		"exp": 9999999999,
		"jti": uuid.New().String(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := token.SignedString(attackerKey)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	// Exchange should fail because the signature doesn't match the service's key.
	_, err = svcA.ExchangeToken(t.Context(), &TokenExchangeRequestRFC8693{
		SubjectToken:     signed,
		SubjectTokenType: "urn:ietf:params:oauth:token-type:access_token",
		Scope:            []string{"openid"},
	})
	if err == nil {
		t.Fatal("expected error for token with wrong signature")
	}
}

// TestTokenExchange_SubjectTokenMissingSub verifies error when subject_token lacks sub claim.
func TestTokenExchange_SubjectTokenMissingSub(t *testing.T) {
	kp := newMockKeyProvider()

	// Create a valid-signed JWT but without sub claim.
	claims := jwt.MapClaims{
		"iss":   "https://test.ggid.dev",
		"aud":   "ggid",
		"iat":   1700000000,
		"exp":   9999999999,
		"jti":   uuid.New().String(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kp.KeyID()
	signed, err := token.SignedString(kp.PrivateKey())
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	svc := NewOAuthService(newMockClientRepo(), newMockCodeRepo(), &mockTokenRepo{}, kp, "https://test.ggid.dev")
	_, err = svc.ExchangeToken(t.Context(), &TokenExchangeRequestRFC8693{
		SubjectToken:     signed,
		SubjectTokenType: "urn:ietf:params:oauth:token-type:access_token",
		Scope:            []string{"openid"},
	})
	if err == nil {
		t.Fatal("expected error for subject_token missing sub")
	}

	// Reference domain to avoid unused import in some build configs.
	_ = domain.ClientTypeConfidential
}
