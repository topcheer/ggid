package domain

import (
	"crypto/sha256"
	"encoding/base64"
	"testing"
	"time"
)

func TestClientType_IsValid(t *testing.T) {
	if !ClientTypeConfidential.IsValid() {
		t.Error("confidential should be valid")
	}
	if !ClientTypePublic.IsValid() {
		t.Error("public should be valid")
	}
	if ClientType("invalid").IsValid() {
		t.Error("invalid type should not be valid")
	}
}

func TestOAuthClient_IsConfidential(t *testing.T) {
	c := &OAuthClient{Type: ClientTypeConfidential}
	if !c.IsConfidential() {
		t.Error("confidential client should return true")
	}
	if c.IsPublic() {
		t.Error("confidential client should not be public")
	}
}

func TestOAuthClient_IsPublic(t *testing.T) {
	c := &OAuthClient{Type: ClientTypePublic}
	if !c.IsPublic() {
		t.Error("public client should return true")
	}
	if c.IsConfidential() {
		t.Error("public client should not be confidential")
	}
}

func TestOAuthClient_RequiresPKCE(t *testing.T) {
	// Public clients always require PKCE
	pub := &OAuthClient{Type: ClientTypePublic}
	if !pub.RequiresPKCE() {
		t.Error("public client should require PKCE")
	}

	// Confidential with RequirePKCE flag
	conf := &OAuthClient{Type: ClientTypeConfidential, RequirePKCE: true}
	if !conf.RequiresPKCE() {
		t.Error("confidential with RequirePKCE should require PKCE")
	}

	// Confidential without flag
	confNoPKCE := &OAuthClient{Type: ClientTypeConfidential}
	if confNoPKCE.RequiresPKCE() {
		t.Error("confidential without RequirePKCE should not require PKCE")
	}
}

func TestOAuthClient_FAPI2_0(t *testing.T) {
	// No metadata
	c := &OAuthClient{}
	if c.FAPI2_0() {
		t.Error("nil metadata should return false")
	}

	// With FAPI enabled
	c.Metadata = map[string]any{"fapi_2_0": true}
	if !c.FAPI2_0() {
		t.Error("metadata[fapi_2_0]=true should return true")
	}

	// With FAPI disabled
	c.Metadata = map[string]any{"fapi_2_0": false}
	if c.FAPI2_0() {
		t.Error("metadata[fapi_2_0]=false should return false")
	}

	// With non-bool value
	c.Metadata = map[string]any{"fapi_2_0": "yes"}
	if c.FAPI2_0() {
		t.Error("non-bool metadata should return false")
	}
}

func TestOAuthClient_SetFAPI2_0(t *testing.T) {
	c := &OAuthClient{}
	c.SetFAPI2_0(true)
	if !c.FAPI2_0() {
		t.Error("after SetFAPI2_0(true) should return true")
	}

	c.SetFAPI2_0(false)
	if c.FAPI2_0() {
		t.Error("after SetFAPI2_0(false) should return false")
	}

	// Existing metadata should not be lost
	c.Metadata = map[string]any{"other": "value"}
	c.SetFAPI2_0(true)
	if c.Metadata["other"] != "value" {
		t.Error("existing metadata should be preserved")
	}
}

func TestOAuthClient_SupportsGrantType(t *testing.T) {
	c := &OAuthClient{GrantTypes: []string{"client_credentials", "authorization_code"}}
	if !c.SupportsGrantType("client_credentials") {
		t.Error("should support client_credentials")
	}
	if !c.SupportsGrantType("authorization_code") {
		t.Error("should support authorization_code")
	}
	if c.SupportsGrantType("password") {
		t.Error("should not support password")
	}
}

func TestOAuthClient_ValidateRedirectURI(t *testing.T) {
	c := &OAuthClient{RedirectURIs: []string{
		"https://app.example.com/callback",
		"https://localhost:3000/callback",
	}}
	if !c.ValidateRedirectURI("https://app.example.com/callback") {
		t.Error("should validate registered URI")
	}
	if c.ValidateRedirectURI("https://evil.com/callback") {
		t.Error("should reject unregistered URI")
	}
}

func TestOAuthClient_MetadataJSON(t *testing.T) {
	// Nil metadata → empty JSON object
	c := &OAuthClient{}
	if string(c.MetadataJSON()) != "{}" {
		t.Errorf("nil metadata should return {}, got %s", c.MetadataJSON())
	}

	// With metadata
	c.Metadata = map[string]any{"key": "value"}
	j := c.MetadataJSON()
	if string(j) == "{}" {
		t.Error("non-nil metadata should not be empty object")
	}
}

func TestAuthorizationCode_IsExpired(t *testing.T) {
	// Expired
	expired := &AuthorizationCode{ExpiresAt: time.Now().Add(-1 * time.Minute)}
	if !expired.IsExpired() {
		t.Error("past time should be expired")
	}

	// Not expired
	future := &AuthorizationCode{ExpiresAt: time.Now().Add(10 * time.Minute)}
	if future.IsExpired() {
		t.Error("future time should not be expired")
	}
}

func TestAuthorizationCode_ValidatePKCE(t *testing.T) {
	// No challenge — PKCE not required
	code := &AuthorizationCode{}
	if !code.ValidatePKCE("") {
		t.Error("no challenge should accept any verifier")
	}

	// Plain method
	code = &AuthorizationCode{CodeChallenge: "myverifier", CodeChallengeMethod: "plain"}
	if !code.ValidatePKCE("myverifier") {
		t.Error("plain method: matching verifier should pass")
	}
	if code.ValidatePKCE("wrong") {
		t.Error("plain method: non-matching verifier should fail")
	}

	// Empty verifier with challenge
	code = &AuthorizationCode{CodeChallenge: "abc123", CodeChallengeMethod: "plain"}
	if code.ValidatePKCE("") {
		t.Error("empty verifier with challenge should fail")
	}

	// S256 method
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	h := sha256.Sum256([]byte(verifier))
	expectedChallenge := base64.RawURLEncoding.EncodeToString(h[:])
	code = &AuthorizationCode{CodeChallenge: expectedChallenge, CodeChallengeMethod: "S256"}
	if !code.ValidatePKCE(verifier) {
		t.Error("S256: correct verifier should pass")
	}
	if code.ValidatePKCE("wrong-verifier") {
		t.Error("S256: wrong verifier should fail")
	}

	// Unknown method
	code = &AuthorizationCode{CodeChallenge: "abc", CodeChallengeMethod: "unknown"}
	if code.ValidatePKCE("abc") {
		t.Error("unknown method should fail")
	}
}

func TestRefreshTokenRecord(t *testing.T) {
	// Basic struct validation
	rt := &RefreshTokenRecord{
		Scope:     []string{"openid", "profile"},
		Revoked:   false,
		Used:      false,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	if rt.Revoked {
		t.Error("new token should not be revoked")
	}
	if rt.Used {
		t.Error("new token should not be used")
	}
	if len(rt.Scope) != 2 {
		t.Errorf("expected 2 scopes, got %d", len(rt.Scope))
	}
}
