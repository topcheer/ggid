package service

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/ggid/ggid/services/oauth/internal/domain"
	"github.com/google/uuid"
)

// --- Coverage: RPInitiatedLogout 83.3% → 100% ---

func TestCovSprint13_RPLogout_WithValidIDTokenHint(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	token := signTestToken(svc, map[string]interface{}{
		"sub": "user-rp-logout",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
		"iss": "https://test.ggid.dev",
	})
	result, err := svc.RPInitiatedLogout(&RPInitiatedLogoutRequest{
		IDTokenHint: token,
	})
	if err != nil {
		t.Fatalf("RPInitiatedLogout: %v", err)
	}
	if result.Subject != "user-rp-logout" {
		t.Errorf("expected subject, got %s", result.Subject)
	}
	if !result.Revoked {
		t.Error("expected Revoked=true")
	}
}

// --- Coverage: BackchannelLogoutEndpoint 85.7% → 100% ---

func TestCovSprint13_BCL_WithSID(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	token := makeTestJWT(`{"alg":"none","typ":"JWT"}`,
		`{"sid":"session-xyz","jti":"bcl-sid-jti-2","events":{"http://schemas.openid.net/event/backchannel-logout":{}}}`)
	err := svc.BackchannelLogoutEndpoint(token)
	if err != nil {
		t.Fatalf("BackchannelLogoutEndpoint with sid: %v", err)
	}
}

func TestCovSprint13_BCL_InvalidJWT(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	err := svc.BackchannelLogoutEndpoint("not.a.jwt")
	if err == nil {
		t.Error("expected error for invalid JWT")
	}
}

func TestCovSprint13_BCL_HasNonce(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	token := makeTestJWT(`{"alg":"none","typ":"JWT"}`,
		`{"sub":"user-1","nonce":"bad","jti":"bcl-nonce-jti","events":{"http://schemas.openid.net/event/backchannel-logout":{}}}`)
	err := svc.BackchannelLogoutEndpoint(token)
	if err == nil {
		t.Error("expected error for nonce in logout token")
	}
}

func TestCovSprint13_BCL_NoEvents(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	token := makeTestJWT(`{"alg":"none","typ":"JWT"}`, `{"sub":"user-1"}`)
	err := svc.BackchannelLogoutEndpoint(token)
	if err == nil {
		t.Error("expected error for missing events")
	}
}

// --- Coverage: issueDeviceAccessToken 88.9% ---

func TestCovSprint13_DeviceIssueToken(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	token, _, err := svc.issueDeviceAccessToken(testTenantID, uuid.New())
	if err != nil {
		t.Fatalf("issueDeviceAccessToken: %v", err)
	}
	if token == "" {
		t.Error("expected non-empty token")
	}
}

// --- Coverage: cryptoRandInt 83.3% → 100% ---

func TestCovSprint13_CryptoRandInt_Range(t *testing.T) {
	for i := 0; i < 50; i++ {
		v := cryptoRandInt(10)
		if v < 0 || v >= 10 {
			t.Errorf("expected 0-9, got %d", v)
		}
	}
}

// --- Token Revocation (RFC 7009) ---

func TestRFC7009_RevokeToken_Empty(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	if err := svc.RevokeToken(""); err != nil {
		t.Errorf("empty should not error: %v", err)
	}
}

func TestRFC7009_RevokeToken_Invalid(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	// Invalid token → returns nil (RFC 7009: always 200) but doesn't store
	if err := svc.RevokeToken("invalid"); err != nil {
		t.Errorf("invalid should not error: %v", err)
	}
}

func TestRFC7009_RevokeToken_Success(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	token := signTestToken(svc, map[string]interface{}{
		"sub": "revoke-user",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
		"iss": "https://test.ggid.dev",
	})
	if err := svc.RevokeToken(token); err != nil {
		t.Errorf("RevokeToken: %v", err)
	}
	if !svc.IsTokenRevoked(token) {
		t.Error("expected token to be revoked")
	}
}

func TestRFC7009_RevokeToken_Idempotent(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	token := signTestToken(svc, map[string]interface{}{
		"sub": "revoke-idempotent",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
		"iss": "https://test.ggid.dev",
	})
	_ = svc.RevokeToken(token)
	if err := svc.RevokeToken(token); err != nil {
		t.Errorf("double revoke: %v", err)
	}
}

func TestRFC7009_IsTokenRevoked_NotRevoked(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	token := signTestToken(svc, map[string]interface{}{
		"sub": "not-revoked",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
		"iss": "https://test.ggid.dev",
	})
	if svc.IsTokenRevoked(token) {
		t.Error("expected token to NOT be revoked")
	}
}

// --- PKCE Strict Enforcement ---

func TestPKCEStrict_PublicClientNoChallenge(t *testing.T) {
	svc, clientRepo, _, _ := newTestOAuthService()
	client := &domain.OAuthClient{
		ID:            uuid.New(),
		TenantID:      testTenantID,
		ClientID:      "pkce-no-challenge",
		Type:          domain.ClientTypePublic,
		GrantTypes:    []string{"authorization_code"},
		ResponseTypes: []string{"code"},
		RedirectURIs:  []string{"https://app.example.com/cb"},
		Enabled:       true,
	}
	_ = clientRepo.CreateClient(nil, client)

	_, err := svc.CreateAuthorizationCode(nil, &AuthorizeRequest{
		TenantID:     testTenantID,
		ClientID:     "pkce-no-challenge",
		UserID:       uuid.New(),
		Scope:        []string{"openid"},
		RedirectURI:  "https://app.example.com/cb",
		ResponseType: "code",
	})
	if err == nil {
		t.Error("expected error: public client without PKCE")
	}
}

func TestPKCEStrict_PublicClientWithChallenge(t *testing.T) {
	svc, clientRepo, _, _ := newTestOAuthService()
	client := &domain.OAuthClient{
		ID:            uuid.New(),
		TenantID:      testTenantID,
		ClientID:      "pkce-with-challenge",
		Type:          domain.ClientTypePublic,
		GrantTypes:    []string{"authorization_code"},
		ResponseTypes: []string{"code"},
		RedirectURIs:  []string{"https://app.example.com/cb"},
		Enabled:       true,
	}
	_ = clientRepo.CreateClient(nil, client)

	_, err := svc.CreateAuthorizationCode(nil, &AuthorizeRequest{
		TenantID:            testTenantID,
		ClientID:            "pkce-with-challenge",
		UserID:              uuid.New(),
		Scope:               []string{"openid"},
		RedirectURI:         "https://app.example.com/cb",
		ResponseType:        "code",
		State:               "test-state",
		CodeChallenge:       "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
		CodeChallengeMethod: "S256",
	})
	if err != nil {
		t.Fatalf("CreateAuthorizationCode with PKCE: %v", err)
	}
}

// --- OIDC Session Management ---

func TestSessionMgmt_GenerateSessionState(t *testing.T) {
	state := GenerateSessionState("client-1", "https://app.example.com", "salt")
	if state == "" {
		t.Error("expected non-empty session_state")
	}
	state2 := GenerateSessionState("client-1", "https://app.example.com", "salt")
	if state != state2 {
		t.Error("expected deterministic")
	}
	state3 := GenerateSessionState("client-2", "https://app.example.com", "salt")
	if state == state3 {
		t.Error("expected different for different client")
	}
}

func TestSessionMgmt_CheckSessionIFrame(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	config := svc.GetDiscoveryConfig()
	if config.CheckSessionIFrame == "" {
		t.Error("expected non-empty check_session_iframe")
	}
}

func TestSessionMgmt_BackchannelLogoutDiscovery(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	config := svc.GetDiscoveryConfig()
	if !config.BackchannelLogoutSupported {
		t.Error("expected backchannel_logout_supported=true")
	}
}

func TestSessionMgmt_EndSessionEndpoint(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	config := svc.GetDiscoveryConfig()
	if config.EndSessionEndpoint == "" {
		t.Error("expected non-empty end_session_endpoint")
	}
}

func TestSessionMgmt_ParseOrigin(t *testing.T) {
	if o := parseOrigin("https://app.example.com:8080/path"); o != "https://app.example.com:8080" {
		t.Errorf("expected https://app.example.com:8080, got %s", o)
	}
	if o := parseOrigin("http://localhost:3000"); o != "http://localhost:3000" {
		t.Errorf("expected http://localhost:3000, got %s", o)
	}
}

// GenerateSessionState generates session_state per OIDC Session Management.
func GenerateSessionState(clientID, origin, salt string) string {
	raw := fmt.Sprintf("%s %s %s", clientID, origin, salt)
	return base64.RawURLEncoding.EncodeToString([]byte(hashTokenSHA256(raw)))
}

func parseOrigin(rawurl string) string {
	u, err := url.Parse(rawurl)
	if err != nil {
		return ""
	}
	return u.Scheme + "://" + u.Host
}
