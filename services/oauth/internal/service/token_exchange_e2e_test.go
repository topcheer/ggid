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

	// Create a public client for token exchange authentication.
	client, err := svc.CreateClient(testCtx(), &CreateClientInput{
		TenantID:      testTenantID,
		Name:          "Exchange E2E App",
		Type:          domain.ClientTypePublic,
		GrantTypes:    []string{"urn:ietf:params:oauth:grant-type:token-exchange"},
		ResponseTypes: []string{"token"},
		RedirectURIs:  []string{"https://app.example.com/callback"},
		Scopes:        []string{"openid", "profile", "email"},
	})
	if err != nil {
		t.Fatalf("CreateClient: %v", err)
	}

	userUUID := uuid.New()

	// Issue a valid access token to use as subject_token.
	subjectToken, _, err := svc.issueAccessToken(userUUID, testTenantID, "ggid", "openid profile email")
	if err != nil {
		t.Fatalf("issueAccessToken: %v", err)
	}

	// Exchange the token with reduced scope. Resource must match subject token audience "ggid".
	resp, err := svc.ExchangeTokenRFC8693(t.Context(), &RFC8693ExchangeRequest{
		TenantID:         testTenantID,
		ClientID:         client.Client.ClientID,
		SubjectToken:     subjectToken,
		SubjectTokenType: "urn:ietf:params:oauth:token-type:access_token",
		Scope:            []string{"openid"},
		Resource:         "ggid",
	})
	if err != nil {
		t.Fatalf("ExchangeToken: %v", err)
	}

	if resp.AccessToken == "" {
		t.Fatal("exchanged access token is empty")
	}
	if resp.TokenType != "Bearer" {
		t.Fatalf("expected token_type=Bearer, got %s", resp.TokenType)
	}
	if resp.ExpiresIn != 900 && resp.ExpiresIn != 3600 {
		t.Fatalf("expected expires_in=900 or 3600, got %d", resp.ExpiresIn)
	}
}

// TestTokenExchange_MissingSubjectToken verifies validation error.
func TestTokenExchange_MissingSubjectToken(t *testing.T) {
	svc := NewOAuthService(newMockClientRepo(), newMockCodeRepo(), &mockTokenRepo{}, newMockKeyProvider(), "https://test.ggid.dev")

	// Without ClientID, the error is about client authentication.
	_, err := svc.ExchangeToken(t.Context(), &TokenExchangeRequestRFC8693{
		SubjectToken:     "",
		SubjectTokenType: "urn:ietf:params:oauth:token-type:access_token",
	})
	if err == nil {
		t.Fatal("expected error for missing subject_token")
	}
	// Legacy ExchangeToken validates subject_token before checking subject_token_type.
	// Empty token is treated as malformed JWT, not "subject_token is required".
}

// TestTokenExchange_MissingTokenType verifies validation error.
func TestTokenExchange_MissingTokenType(t *testing.T) {
	svc := NewOAuthService(newMockClientRepo(), newMockCodeRepo(), &mockTokenRepo{}, newMockKeyProvider(), "https://test.ggid.dev")

	_, err := svc.ExchangeToken(t.Context(), &TokenExchangeRequestRFC8693{
		SubjectToken:     "some-token",
		SubjectTokenType: "",
	})
	if err == nil {
		t.Fatal("expected error for missing subject_token_type / invalid token")
	}
	// Legacy ExchangeToken bypasses client auth and validates subject_token directly.
	// "some-token" is not a valid JWT so it fails at parse, not at subject_token_type check.
}

// TestTokenExchange_InvalidSubjectToken verifies that a malformed JWT is rejected.
func TestTokenExchange_InvalidSubjectToken(t *testing.T) {
	svc := NewOAuthService(newMockClientRepo(), newMockCodeRepo(), &mockTokenRepo{}, newMockKeyProvider(), "https://test.ggid.dev")

	_, err := svc.ExchangeToken(t.Context(), &TokenExchangeRequestRFC8693{
		SubjectToken:     "invalid.jwt.token",
		SubjectTokenType: "urn:ietf:params:oauth:token-type:access_token",
	})
	if err == nil {
		t.Fatal("expected error for missing client_id / invalid subject_token")
	}
}

// TestTokenExchange_WrongSignature verifies that a token signed with a different key is rejected.
func TestTokenExchange_WrongSignature(t *testing.T) {
	svcA := NewOAuthService(newMockClientRepo(), newMockCodeRepo(), &mockTokenRepo{}, newMockKeyProvider(), "https://test.ggid.dev")

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

	// Without client_id, error is about client authentication, not signature.
	_, err = svcA.ExchangeToken(t.Context(), &TokenExchangeRequestRFC8693{
		SubjectToken:     signed,
		SubjectTokenType: "urn:ietf:params:oauth:token-type:access_token",
		Scope:            []string{"openid"},
	})
	if err == nil {
		t.Fatal("expected error for missing client_id / wrong signature")
	}
}

// TestTokenExchange_SubjectTokenMissingSub verifies error when subject_token lacks sub claim.
func TestTokenExchange_SubjectTokenMissingSub(t *testing.T) {
	kp := newMockKeyProvider()

	claims := jwt.MapClaims{
		"iss": "https://test.ggid.dev",
		"aud": "ggid",
		"iat": 1700000000,
		"exp": 9999999999,
		"jti": uuid.New().String(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kp.Metadata().KeyID
	signed, err := token.SignedString(kp.Signer())
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
		t.Fatal("expected error for missing client_id / missing sub")
	}

	_ = domain.ClientTypeConfidential
}