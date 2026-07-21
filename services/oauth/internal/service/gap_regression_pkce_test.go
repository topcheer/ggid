package service

// PKCE (RFC 7636) Functional Verification Tests
// Verifies: Gap #3 — PKCE code challenge/verifier flow (was DONE via grep, now functionally verified)
// Flow: GenerateCodeVerifier → CreateCodeChallenge(S256) → CreateAuthCode with challenge →
//       ExchangeCode with verifier → success. Also: mismatch → rejection.
// Date: 2026-07-25

import (
	"context"
	"crypto/sha256"
	"strings"
	"encoding/base64"
	"encoding/hex"
	"testing"

	"github.com/ggid/ggid/pkg/crypto"
	"github.com/ggid/ggid/services/oauth/internal/domain"
	"github.com/google/uuid"
)

// generateCodeVerifier creates a cryptographically random code_verifier per RFC 7636.
func generateCodeVerifier(t *testing.T) string {
	t.Helper()
	tok, err := crypto.GenerateRandomToken(32)
	if err != nil {
		t.Fatalf("GenerateRandomToken: %v", err)
	}
	return tok
}

// computeS256Challenge computes the S256 code_challenge from a code_verifier.
// Uses base64url encoding to match domain.AuthorizationCode.ValidatePKCE.
func computeS256Challenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

// ========== RFC 7636 Functional Tests ==========

// TestPKCE_FullFlow_S256_Success verifies the complete PKCE flow:
// verifier → S256 challenge → auth code with challenge → exchange with verifier → tokens issued.
func TestPKCE_FullFlow_S256_Success(t *testing.T) {
	svc, clientRepo, _, _ := newTestOAuthService()

	// Setup client
	secretHash, _ := crypto.HashPassword("secret")
	clientRepo.clients["pkce-flow-client"] = &domain.OAuthClient{
		ID:               uuid.New(),
		TenantID:         testTenantID,
		ClientID:         "pkce-flow-client",
		ClientSecretHash: secretHash,
		Type:             domain.ClientTypeConfidential,
		RedirectURIs:     []string{"https://app.example.com/cb"},
		Enabled:          true,
	}

	// 1. Client generates code_verifier
	verifier := generateCodeVerifier(t)

	// 2. Client computes S256 code_challenge
	challenge := computeS256Challenge(verifier)
	if challenge == "" {
		t.Fatal("code_challenge should not be empty")
	}

	// 3. Auth request includes code_challenge
	plaintextCode, err := svc.CreateAuthorizationCode(context.Background(), &AuthorizeRequest{
		TenantID:            testTenantID,
		ClientID:            "pkce-flow-client",
		RedirectURI:         "https://app.example.com/cb",
		ResponseType:        "code",
		Scope:               []string{"openid"},
		State:               "csrf-state-123",
		CodeChallenge:       challenge,
		CodeChallengeMethod: "S256",
		UserID:              uuid.New(),
	})
	if err != nil {
		t.Fatalf("CreateAuthorizationCode with PKCE: %v", err)
	}

	// 4. Token exchange with correct code_verifier
	resp, err := svc.ExchangeAuthorizationCode(context.Background(), &TokenExchangeRequest{
		TenantID:     testTenantID,
		GrantType:    "authorization_code",
		Code:         plaintextCode,
		RedirectURI:  "https://app.example.com/cb",
		ClientID:     "pkce-flow-client",
		ClientSecret: "secret",
		CodeVerifier: verifier,
		State:        "csrf-state-123",
	})
	if err != nil {
		t.Fatalf("ExchangeAuthorizationCode with correct verifier should succeed: %v", err)
	}

	if resp.AccessToken == "" {
		t.Error("access_token should be issued")
	}
	if resp.TokenType != "Bearer" {
		t.Errorf("token_type should be Bearer, got %s", resp.TokenType)
	}
}

// TestPKCE_VerifierMismatch_Rejected verifies that a wrong code_verifier is rejected.
func TestPKCE_VerifierMismatch_Rejected(t *testing.T) {
	svc, clientRepo, _, _ := newTestOAuthService()

	secretHash, _ := crypto.HashPassword("secret")
	clientRepo.clients["pkce-mismatch"] = &domain.OAuthClient{
		ID:               uuid.New(),
		TenantID:         testTenantID,
		ClientID:         "pkce-mismatch",
		ClientSecretHash: secretHash,
		Type:             domain.ClientTypeConfidential,
		RedirectURIs:     []string{"https://app.example.com/cb"},
		Enabled:          true,
	}

	verifier := generateCodeVerifier(t)
	challenge := computeS256Challenge(verifier)

	plaintextCode, err := svc.CreateAuthorizationCode(context.Background(), &AuthorizeRequest{
		TenantID:            testTenantID,
		ClientID:            "pkce-mismatch",
		RedirectURI:         "https://app.example.com/cb",
		ResponseType:        "code",
		Scope:               []string{"openid"},
		State:               "csrf-state",
		CodeChallenge:       challenge,
		CodeChallengeMethod: "S256",
		UserID:              uuid.New(),
	})
	if err != nil {
		t.Fatalf("CreateAuthorizationCode: %v", err)
	}

	// Exchange with WRONG verifier
	_, err = svc.ExchangeAuthorizationCode(context.Background(), &TokenExchangeRequest{
		TenantID:     testTenantID,
		GrantType:    "authorization_code",
		Code:         plaintextCode,
		RedirectURI:  "https://app.example.com/cb",
		ClientID:     "pkce-mismatch",
		ClientSecret: "secret",
		CodeVerifier: "completely-wrong-verifier",
		State:        "csrf-state",
	})
	if err == nil {
		t.Fatal("PKCE verifier mismatch should be rejected")
	}
}

// TestPKCE_PlainMethod verifies the "plain" code challenge method.
func TestPKCE_PlainMethod(t *testing.T) {
	svc, clientRepo, _, _ := newTestOAuthService()

	secretHash, _ := crypto.HashPassword("secret")
	clientRepo.clients["pkce-plain"] = &domain.OAuthClient{
		ID:               uuid.New(),
		TenantID:         testTenantID,
		ClientID:         "pkce-plain",
		ClientSecretHash: secretHash,
		Type:             domain.ClientTypeConfidential,
		RedirectURIs:     []string{"https://app.example.com/cb"},
		Enabled:          true,
	}

	// For plain method, challenge == verifier.
	// RFC 7636 §4.1: code_verifier length MUST be 43-128 chars.
	verifier := strings.Repeat("a", 43)
	challenge := verifier // plain: challenge equals verifier

	plaintextCode, err := svc.CreateAuthorizationCode(context.Background(), &AuthorizeRequest{
		TenantID:            testTenantID,
		ClientID:            "pkce-plain",
		RedirectURI:         "https://app.example.com/cb",
		ResponseType:        "code",
		Scope:               []string{"openid"},
		State:               "state-plain",
		CodeChallenge:       challenge,
		CodeChallengeMethod: "plain",
		UserID:              uuid.New(),
	})
	if err != nil {
		t.Fatalf("CreateAuthorizationCode: %v", err)
	}

	// Exchange with correct verifier (same as challenge for plain)
	resp, err := svc.ExchangeAuthorizationCode(context.Background(), &TokenExchangeRequest{
		TenantID:     testTenantID,
		GrantType:    "authorization_code",
		Code:         plaintextCode,
		RedirectURI:  "https://app.example.com/cb",
		ClientID:     "pkce-plain",
		ClientSecret: "secret",
		CodeVerifier: verifier,
		State:        "state-plain",
	})
	if err != nil {
		t.Fatalf("plain PKCE exchange should succeed: %v", err)
	}
	if resp.AccessToken == "" {
		t.Error("access_token should be issued with plain PKCE")
	}
}

// TestPKCE_PublicClientEnforced verifies that public clients are required to use PKCE.
func TestPKCE_PublicClientEnforced(t *testing.T) {
	svc, clientRepo, _, _ := newTestOAuthService()

	clientRepo.clients["pkce-public"] = &domain.OAuthClient{
		ID:           uuid.New(),
		TenantID:     testTenantID,
		ClientID:     "pkce-public",
		Type:         domain.ClientTypePublic,
		RedirectURIs: []string{"https://app.example.com/cb"},
		Enabled:      true,
	}

	// Public client without code_challenge should be rejected
	_, err := svc.CreateAuthorizationCode(context.Background(), &AuthorizeRequest{
		TenantID:     testTenantID,
		ClientID:     "pkce-public",
		RedirectURI:  "https://app.example.com/cb",
		ResponseType: "code",
		Scope:        []string{"openid"},
		State:        "state",
		// No CodeChallenge — should fail for public client
		UserID: uuid.New(),
	})
	if err == nil {
		t.Fatal("public client without PKCE should be rejected")
	}
}

// TestPKCE_VerifyCodeChallenge_UnitTests tests the VerifyCodeChallenge function directly.
func TestPKCE_VerifyCodeChallenge_UnitTests(t *testing.T) {
	// VerifyCodeChallenge uses hashTokenSHA256 (hex encoding)
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	h := sha256.Sum256([]byte(verifier))
	challenge := hex.EncodeToString(h[:])

	// S256 correct
	if !VerifyCodeChallenge(challenge, verifier, "S256") {
		t.Error("S256 verification should succeed with matching verifier")
	}

	// S256 wrong verifier
	if VerifyCodeChallenge(challenge, "wrong-verifier", "S256") {
		t.Error("S256 verification should fail with wrong verifier")
	}

	// Plain correct
	if !VerifyCodeChallenge(verifier, verifier, "plain") {
		t.Error("plain verification should succeed when challenge==verifier")
	}

	// Plain wrong
	if VerifyCodeChallenge("different", verifier, "plain") {
		t.Error("plain verification should fail with mismatched challenge")
	}

	// Empty challenge
	if VerifyCodeChallenge("", verifier, "S256") {
		t.Error("empty challenge should fail")
	}

	// Empty verifier
	if VerifyCodeChallenge(challenge, "", "S256") {
		t.Error("empty verifier should fail")
	}

	// Unknown method
	if VerifyCodeChallenge(challenge, verifier, "unknown") {
		t.Error("unknown method should fail")
	}

	// Default method (empty string should default to S256)
	if !VerifyCodeChallenge(challenge, verifier, "") {
		t.Error("empty method should default to S256 and succeed")
	}
}
