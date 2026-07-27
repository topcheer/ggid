package domain

import (
	"crypto/sha256"
	"encoding/base64"
	"testing"
)

// --- PKCE ValidatePKCE tests (supplementing existing models_test.go) ---

func TestValidatePKCE_S256_Match(t *testing.T) {
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	h := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(h[:])

	code := &AuthorizationCode{
		CodeChallenge:       challenge,
		CodeChallengeMethod: "S256",
	}
	if !code.ValidatePKCE(verifier) {
		t.Error("expected PKCE validation to pass with correct verifier")
	}
}

func TestValidatePKCE_S256_NoMatch(t *testing.T) {
	code := &AuthorizationCode{
		CodeChallenge:       "invalidchallenge",
		CodeChallengeMethod: "S256",
	}
	if code.ValidatePKCE("validverifier12345678901234567890123456789012345") {
		t.Error("expected PKCE validation to fail with mismatched verifier")
	}
}

func TestValidatePKCE_NoChallenge(t *testing.T) {
	code := &AuthorizationCode{
		CodeChallenge:       "",
		CodeChallengeMethod: "",
	}
	// No PKCE required — always passes
	if !code.ValidatePKCE("anything") {
		t.Error("expected PKCE to pass when no challenge is set")
	}
}

func TestValidatePKCE_Plain_Rejected(t *testing.T) {
	code := &AuthorizationCode{
		CodeChallenge:       "somechallenge",
		CodeChallengeMethod: "plain",
	}
	// OAuth 2.1: plain method not supported
	if code.ValidatePKCE("somechallenge") {
		t.Error("expected PKCE to reject plain method")
	}
}

func TestValidatePKCE_VerifierTooShort(t *testing.T) {
	code := &AuthorizationCode{
		CodeChallenge:       "challenge",
		CodeChallengeMethod: "S256",
	}
	if code.ValidatePKCE("short") {
		t.Error("expected PKCE to reject verifier < 43 chars")
	}
}

func TestValidatePKCE_VerifierTooLong(t *testing.T) {
	code := &AuthorizationCode{
		CodeChallenge:       "challenge",
		CodeChallengeMethod: "S256",
	}
	longVerifier := ""
	for i := 0; i < 129; i++ {
		longVerifier += "a"
	}
	if code.ValidatePKCE(longVerifier) {
		t.Error("expected PKCE to reject verifier > 128 chars")
	}
}

func TestValidatePKCE_VerifierInvalidChars(t *testing.T) {
	code := &AuthorizationCode{
		CodeChallenge:       "challenge",
		CodeChallengeMethod: "S256",
	}
	// Contains invalid character '!'
	invalidVerifier := "verifier_with_invalid_char!_padding_to_43_chars__"
	if code.ValidatePKCE(invalidVerifier) {
		t.Error("expected PKCE to reject verifier with invalid chars")
	}
}

func TestValidatePKCE_EmptyMethodDefaultsS256(t *testing.T) {
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	h := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(h[:])

	code := &AuthorizationCode{
		CodeChallenge:       challenge,
		CodeChallengeMethod: "", // empty defaults to S256
	}
	if !code.ValidatePKCE(verifier) {
		t.Error("expected PKCE to pass with empty method defaulting to S256")
	}
}

func TestValidatePKCE_ValidUnreservedChars(t *testing.T) {
	// All unreserved chars from RFC 7636: A-Z a-z 0-9 - . _ ~
	verifier := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-._~"
	if len(verifier) < 43 {
		t.Fatalf("test verifier too short: %d", len(verifier))
	}
	h := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(h[:])

	code := &AuthorizationCode{
		CodeChallenge:       challenge,
		CodeChallengeMethod: "S256",
	}
	if !code.ValidatePKCE(verifier) {
		t.Error("expected PKCE to pass with all valid unreserved chars")
	}
}

// --- FAPI 2.0 tests (supplementing existing tests) ---

func TestOAuthClient_FAPI2_0_NilMetadata(t *testing.T) {
	c := &OAuthClient{}
	if c.FAPI2_0() {
		t.Error("expected FAPI2_0=false for nil metadata")
	}
}

func TestOAuthClient_FAPI2_0_SetAndGet(t *testing.T) {
	c := &OAuthClient{}
	c.SetFAPI2_0(true)
	if !c.FAPI2_0() {
		t.Error("expected FAPI2_0=true after SetFAPI2_0(true)")
	}
	c.SetFAPI2_0(false)
	if c.FAPI2_0() {
		t.Error("expected FAPI2_0=false after SetFAPI2_0(false)")
	}
}

func TestOAuthClient_ValidateRedirectURI_PartialMatch(t *testing.T) {
	c := &OAuthClient{
		RedirectURIs: []string{"https://app.example.com/callback"},
	}
	// Ensure exact match required — no path prefix bypass
	if c.ValidateRedirectURI("https://app.example.com/callback/evil") {
		t.Error("expected exact URI match — prefix should not pass")
	}
	if c.ValidateRedirectURI("https://app.example.com/callback?param=1") {
		t.Error("expected exact URI match — query string should not pass")
	}
}
