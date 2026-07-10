package service

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/ggid/ggid/services/oauth/internal/domain"
)

// --- PAR (RFC 9126) Tests ---

func TestPushAuthorizationRequest_Success(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	result, err := svc.CreateClient(context.Background(), &CreateClientInput{
		TenantID:      testTenantID,
		Name:          "PAR Test Client",
		Type:          domain.ClientTypeConfidential,
		GrantTypes:    []string{"authorization_code"},
		ResponseTypes: []string{"code"},
		RedirectURIs:  []string{"https://app.example.com/callback"},
	})
	if err != nil {
		t.Fatalf("CreateClient: %v", err)
	}

	parResp, err := svc.PushAuthorizationRequest(context.Background(), &PushedAuthorizationRequest{
		TenantID:     testTenantID,
		ClientID:     result.Client.ClientID,
		ClientSecret: result.ClientSecret,
		RedirectURI:  "https://app.example.com/callback",
		ResponseType: "code",
		Scope:        "openid profile",
		State:        "xyz123",
	})
	if err != nil {
		t.Fatalf("PushAuthorizationRequest: %v", err)
	}
	if !strings.HasPrefix(parResp.RequestURI, "urn:ietf:params:oauth:request_uri:") {
		t.Errorf("RequestURI = %s, want prefix", parResp.RequestURI)
	}
	if parResp.ExpiresIn != 60 {
		t.Errorf("ExpiresIn = %d, want 60", parResp.ExpiresIn)
	}
}

func TestPushAuthorizationRequest_InvalidClient(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	_, err := svc.PushAuthorizationRequest(context.Background(), &PushedAuthorizationRequest{
		TenantID:     testTenantID,
		ClientID:     "nonexistent-client",
		RedirectURI:  "https://app.example.com/callback",
		ResponseType: "code",
	})
	if err == nil {
		t.Error("expected error for invalid client")
	}
}

func TestPushAuthorizationRequest_BadRedirectURI(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	result, _ := svc.CreateClient(context.Background(), &CreateClientInput{
		TenantID:      testTenantID,
		Name:          "PAR Bad URI",
		Type:          domain.ClientTypePublic,
		GrantTypes:    []string{"authorization_code"},
		ResponseTypes: []string{"code"},
		RedirectURIs:  []string{"https://app.example.com/callback"},
	})

	_, err := svc.PushAuthorizationRequest(context.Background(), &PushedAuthorizationRequest{
		TenantID:     testTenantID,
		ClientID:     result.Client.ClientID,
		RedirectURI:  "https://evil.example.com/callback",
		ResponseType: "code",
	})
	if err == nil {
		t.Error("expected error for unregistered redirect_uri")
	}
}

func TestPushAuthorizationRequest_UnsupportedResponseType(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	result, _ := svc.CreateClient(context.Background(), &CreateClientInput{
		TenantID:      testTenantID,
		Name:          "PAR Bad RT",
		Type:          domain.ClientTypePublic,
		GrantTypes:    []string{"authorization_code"},
		ResponseTypes: []string{"code"},
		RedirectURIs:  []string{"https://app.example.com/callback"},
	})

	_, err := svc.PushAuthorizationRequest(context.Background(), &PushedAuthorizationRequest{
		TenantID:     testTenantID,
		ClientID:     result.Client.ClientID,
		RedirectURI:  "https://app.example.com/callback",
		ResponseType: "token",
	})
	if err == nil {
		t.Error("expected error for unsupported response_type")
	}
}

func TestPushAuthorizationRequest_BadSecret(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	result, _ := svc.CreateClient(context.Background(), &CreateClientInput{
		TenantID:      testTenantID,
		Name:          "PAR Bad Secret",
		Type:          domain.ClientTypeConfidential,
		GrantTypes:    []string{"authorization_code"},
		ResponseTypes: []string{"code"},
		RedirectURIs:  []string{"https://app.example.com/callback"},
	})

	_, err := svc.PushAuthorizationRequest(context.Background(), &PushedAuthorizationRequest{
		TenantID:     testTenantID,
		ClientID:     result.Client.ClientID,
		ClientSecret: "wrong-secret",
		RedirectURI:  "https://app.example.com/callback",
		ResponseType: "code",
	})
	if err == nil {
		t.Error("expected error for bad client secret")
	}
}

func TestGetPushedAuthorizationRequest_Success(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	result, _ := svc.CreateClient(context.Background(), &CreateClientInput{
		TenantID:      testTenantID,
		Name:          "PAR Get",
		Type:          domain.ClientTypePublic,
		GrantTypes:    []string{"authorization_code"},
		ResponseTypes: []string{"code"},
		RedirectURIs:  []string{"https://app.example.com/callback"},
	})

	parResp, _ := svc.PushAuthorizationRequest(context.Background(), &PushedAuthorizationRequest{
		TenantID:     testTenantID,
		ClientID:     result.Client.ClientID,
		RedirectURI:  "https://app.example.com/callback",
		ResponseType: "code",
		Scope:        "openid",
		State:        "abc",
	})

	stored, err := svc.GetPushedAuthorizationRequest(parResp.RequestURI)
	if err != nil {
		t.Fatalf("GetPushedAuthorizationRequest: %v", err)
	}
	if stored.Scope != "openid" {
		t.Errorf("Scope = %s, want openid", stored.Scope)
	}
	if stored.State != "abc" {
		t.Errorf("State = %s, want abc", stored.State)
	}
}

func TestGetPushedAuthorizationRequest_NotFound(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	_, err := svc.GetPushedAuthorizationRequest("urn:ietf:params:oauth:request_uri:nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent request_uri")
	}
}

func TestGetPushedAuthorizationRequest_InvalidFormat(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	_, err := svc.GetPushedAuthorizationRequest("invalid-format")
	if err == nil {
		t.Error("expected error for invalid request_uri format")
	}
}

func TestGetPushedAuthorizationRequest_SingleUse(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	result, _ := svc.CreateClient(context.Background(), &CreateClientInput{
		TenantID:      testTenantID,
		Name:          "PAR Single Use",
		Type:          domain.ClientTypePublic,
		GrantTypes:    []string{"authorization_code"},
		ResponseTypes: []string{"code"},
		RedirectURIs:  []string{"https://app.example.com/callback"},
	})

	parResp, _ := svc.PushAuthorizationRequest(context.Background(), &PushedAuthorizationRequest{
		TenantID:     testTenantID,
		ClientID:     result.Client.ClientID,
		RedirectURI:  "https://app.example.com/callback",
		ResponseType: "code",
	})

	_, err := svc.GetPushedAuthorizationRequest(parResp.RequestURI)
	if err != nil {
		t.Fatalf("first use should succeed: %v", err)
	}

	_, err = svc.GetPushedAuthorizationRequest(parResp.RequestURI)
	if err == nil {
		t.Error("expected error on second use (single-use)")
	}
}

// --- RP-Initiated Logout Tests ---

func TestRPInitiatedLogout_WithRedirect(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	result, err := svc.RPInitiatedLogout(&RPInitiatedLogoutRequest{
		PostLogoutRedirectURI: "https://app.example.com/post-logout",
		State:                 "logout-state-123",
	})
	if err != nil {
		t.Fatalf("RPInitiatedLogout: %v", err)
	}
	if result.RedirectURL == "" {
		t.Error("expected non-empty redirect URL")
	}
	if !strings.Contains(result.RedirectURL, "state=logout-state-123") {
		t.Errorf("redirect URL should contain state: %s", result.RedirectURL)
	}
}

func TestRPInitiatedLogout_NoRedirect(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	result, err := svc.RPInitiatedLogout(&RPInitiatedLogoutRequest{})
	if err != nil {
		t.Fatalf("RPInitiatedLogout: %v", err)
	}
	if result.RedirectURL != "" {
		t.Errorf("expected empty redirect URL, got %s", result.RedirectURL)
	}
}

func TestRPInitiatedLogout_InvalidRedirectURI(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	_, err := svc.RPInitiatedLogout(&RPInitiatedLogoutRequest{
		PostLogoutRedirectURI: "not-a-url",
	})
	if err == nil {
		t.Error("expected error for invalid post_logout_redirect_uri")
	}
}

func TestRPInitiatedLogout_WithIDTokenHint(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	// Use a malformed token - should not error, just empty subject.
	result, err := svc.RPInitiatedLogout(&RPInitiatedLogoutRequest{
		IDTokenHint: "invalid.token.hint",
	})
	if err != nil {
		t.Fatalf("RPInitiatedLogout with invalid id_token_hint should not fail: %v", err)
	}
	if result.Subject != "" {
		t.Errorf("expected empty subject for invalid token, got '%s'", result.Subject)
	}
	if result.Revoked {
		t.Error("expected not revoked for invalid token")
	}
}

func TestBackchannelLogoutEndpoint_EmptyToken(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	err := svc.BackchannelLogoutEndpoint("")
	if err == nil {
		t.Error("expected error for empty logout_token")
	}
}

func TestBackchannelLogoutEndpoint_ValidToken(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	logoutToken := makeTestJWT(
		`{"alg":"none","typ":"JWT"}`,
		`{"sub":"user-123","events":{"http://schemas.openid.net/event/backchannel-logout":{}}}`,
	)

	err := svc.BackchannelLogoutEndpoint(logoutToken)
	if err != nil {
		t.Fatalf("BackchannelLogoutEndpoint: %v", err)
	}
}

func TestBackchannelLogoutEndpoint_MissingEvents(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	logoutToken := makeTestJWT(
		`{"alg":"none","typ":"JWT"}`,
		`{"sub":"user-456"}`,
	)

	err := svc.BackchannelLogoutEndpoint(logoutToken)
	if err == nil {
		t.Error("expected error for missing events claim")
	}
}

func TestBackchannelLogoutEndpoint_WithNonce(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	logoutToken := makeTestJWT(
		`{"alg":"none","typ":"JWT"}`,
		`{"sub":"user-789","nonce":"abc","events":{"http://schemas.openid.net/event/backchannel-logout":{}}}`,
	)

	err := svc.BackchannelLogoutEndpoint(logoutToken)
	if err == nil {
		t.Error("expected error for nonce in logout token")
	}
}

// --- CIBA Tests ---

func TestBackchannelAuthentication_Success(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	result, _ := svc.CreateClient(context.Background(), &CreateClientInput{
		TenantID:      testTenantID,
		Name:          "CIBA Success",
		Type:          domain.ClientTypeConfidential,
		GrantTypes:    []string{"urn:openid:params:grant-type:ciba"},
		ResponseTypes: []string{"code"},
		RedirectURIs:  []string{"https://app.example.com/callback"},
	})

	resp, err := svc.BackchannelAuthentication(context.Background(), &BackchannelAuthRequest{
		TenantID:       testTenantID,
		ClientID:       result.Client.ClientID,
		ClientSecret:   result.ClientSecret,
		LoginHint:      "user@example.com",
		BindingMessage: "Approve login from laptop",
	})
	if err != nil {
		t.Fatalf("BackchannelAuthentication: %v", err)
	}
	if resp.AuthReqID == "" {
		t.Error("expected non-empty auth_req_id")
	}
	if resp.ExpiresIn != 300 {
		t.Errorf("ExpiresIn = %d, want 300", resp.ExpiresIn)
	}
	if resp.Interval != 5 {
		t.Errorf("Interval = %d, want 5", resp.Interval)
	}
}

func TestBackchannelAuthentication_NoHint(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	result, _ := svc.CreateClient(context.Background(), &CreateClientInput{
		TenantID:      testTenantID,
		Name:          "CIBA No Hint",
		Type:          domain.ClientTypePublic,
		GrantTypes:    []string{"urn:openid:params:grant-type:ciba"},
		ResponseTypes: []string{"code"},
		RedirectURIs:  []string{"https://app.example.com/callback"},
	})

	_, err := svc.BackchannelAuthentication(context.Background(), &BackchannelAuthRequest{
		TenantID: testTenantID,
		ClientID: result.Client.ClientID,
	})
	if err == nil {
		t.Error("expected error when no hint provided")
	}
}

func TestBackchannelAuthentication_NoCIBAGrant(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	result, _ := svc.CreateClient(context.Background(), &CreateClientInput{
		TenantID:      testTenantID,
		Name:          "Non-CIBA",
		Type:          domain.ClientTypePublic,
		GrantTypes:    []string{"authorization_code"},
		ResponseTypes: []string{"code"},
		RedirectURIs:  []string{"https://app.example.com/callback"},
	})

	_, err := svc.BackchannelAuthentication(context.Background(), &BackchannelAuthRequest{
		TenantID:  testTenantID,
		ClientID:  result.Client.ClientID,
		LoginHint: "user@example.com",
	})
	if err == nil {
		t.Error("expected error for client without CIBA grant")
	}
}

func TestBackchannelAuthentication_InvalidClient(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	_, err := svc.BackchannelAuthentication(context.Background(), &BackchannelAuthRequest{
		TenantID:  testTenantID,
		ClientID:  "nonexistent",
		LoginHint: "user@example.com",
	})
	if err == nil {
		t.Error("expected error for invalid client")
	}
}

func TestBackchannelAuthentication_BadSecret(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	result, _ := svc.CreateClient(context.Background(), &CreateClientInput{
		TenantID:      testTenantID,
		Name:          "CIBA Bad Secret",
		Type:          domain.ClientTypeConfidential,
		GrantTypes:    []string{"urn:openid:params:grant-type:ciba"},
		ResponseTypes: []string{"code"},
		RedirectURIs:  []string{"https://app.example.com/callback"},
	})

	_, err := svc.BackchannelAuthentication(context.Background(), &BackchannelAuthRequest{
		TenantID:     testTenantID,
		ClientID:     result.Client.ClientID,
		ClientSecret: "wrong-secret",
		LoginHint:    "user@example.com",
	})
	if err == nil {
		t.Error("expected error for bad client secret")
	}
}

func TestBackchannelAuthentication_CustomExpiry(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	result, _ := svc.CreateClient(context.Background(), &CreateClientInput{
		TenantID:      testTenantID,
		Name:          "CIBA Expiry",
		Type:          domain.ClientTypePublic,
		GrantTypes:    []string{"urn:openid:params:grant-type:ciba"},
		ResponseTypes: []string{"code"},
		RedirectURIs:  []string{"https://app.example.com/callback"},
	})

	resp, err := svc.BackchannelAuthentication(context.Background(), &BackchannelAuthRequest{
		TenantID:        testTenantID,
		ClientID:        result.Client.ClientID,
		LoginHint:       "user@example.com",
		RequestedExpiry: 120,
	})
	if err != nil {
		t.Fatalf("BackchannelAuthentication: %v", err)
	}
	if resp.ExpiresIn != 120 {
		t.Errorf("ExpiresIn = %d, want 120", resp.ExpiresIn)
	}
}

func TestPollCIBAToken_Pending(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	result, _ := svc.CreateClient(context.Background(), &CreateClientInput{
		TenantID:      testTenantID,
		Name:          "CIBA Poll Pending",
		Type:          domain.ClientTypePublic,
		GrantTypes:    []string{"urn:openid:params:grant-type:ciba"},
		ResponseTypes: []string{"code"},
		RedirectURIs:  []string{"https://app.example.com/callback"},
	})

	authResp, _ := svc.BackchannelAuthentication(context.Background(), &BackchannelAuthRequest{
		TenantID:  testTenantID,
		ClientID:  result.Client.ClientID,
		LoginHint: "user@example.com",
	})

	_, err := svc.PollCIBAToken(context.Background(), testTenantID, authResp.AuthReqID, result.Client.ClientID, "")
	if err == nil {
		t.Fatal("expected authorization_pending error")
	}
	if !strings.Contains(err.Error(), "authorization_pending") {
		t.Errorf("expected authorization_pending, got: %v", err)
	}
}

func TestPollCIBAToken_UnknownAuthReqID(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	_, err := svc.PollCIBAToken(context.Background(), testTenantID, "unknown-req-id", "client", "")
	if err == nil {
		t.Error("expected error for unknown auth_req_id")
	}
	if !strings.Contains(err.Error(), "invalid_grant") {
		t.Errorf("expected invalid_grant, got: %v", err)
	}
}

func TestPollCIBAToken_Approved(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	result, _ := svc.CreateClient(context.Background(), &CreateClientInput{
		TenantID:      testTenantID,
		Name:          "CIBA Approved",
		Type:          domain.ClientTypePublic,
		GrantTypes:    []string{"urn:openid:params:grant-type:ciba"},
		ResponseTypes: []string{"code"},
		RedirectURIs:  []string{"https://app.example.com/callback"},
	})

	authResp, _ := svc.BackchannelAuthentication(context.Background(), &BackchannelAuthRequest{
		TenantID:  testTenantID,
		ClientID:  result.Client.ClientID,
		LoginHint: "user@example.com",
		Scope:     "openid profile",
	})

	if err := svc.ApproveCIBAAuth(authResp.AuthReqID); err != nil {
		t.Fatalf("ApproveCIBAAuth: %v", err)
	}

	token, err := svc.PollCIBAToken(context.Background(), testTenantID, authResp.AuthReqID, result.Client.ClientID, "")
	if err != nil {
		t.Fatalf("PollCIBAToken: %v", err)
	}
	if token.AccessToken == "" {
		t.Error("expected non-empty access token")
	}
	if token.TokenType != "Bearer" {
		t.Errorf("TokenType = %s, want Bearer", token.TokenType)
	}
}

func TestPollCIBAToken_Denied(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	result, _ := svc.CreateClient(context.Background(), &CreateClientInput{
		TenantID:      testTenantID,
		Name:          "CIBA Denied",
		Type:          domain.ClientTypePublic,
		GrantTypes:    []string{"urn:openid:params:grant-type:ciba"},
		ResponseTypes: []string{"code"},
		RedirectURIs:  []string{"https://app.example.com/callback"},
	})

	authResp, _ := svc.BackchannelAuthentication(context.Background(), &BackchannelAuthRequest{
		TenantID:  testTenantID,
		ClientID:  result.Client.ClientID,
		LoginHint: "user@example.com",
	})

	if err := svc.DenyCIBAAuth(authResp.AuthReqID); err != nil {
		t.Fatalf("DenyCIBAAuth: %v", err)
	}

	_, err := svc.PollCIBAToken(context.Background(), testTenantID, authResp.AuthReqID, result.Client.ClientID, "")
	if err == nil {
		t.Fatal("expected access_denied error")
	}
	if !strings.Contains(err.Error(), "access_denied") {
		t.Errorf("expected access_denied, got: %v", err)
	}
}

func TestPollCIBAToken_SlowDown(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	result, _ := svc.CreateClient(context.Background(), &CreateClientInput{
		TenantID:      testTenantID,
		Name:          "CIBA Slow",
		Type:          domain.ClientTypePublic,
		GrantTypes:    []string{"urn:openid:params:grant-type:ciba"},
		ResponseTypes: []string{"code"},
		RedirectURIs:  []string{"https://app.example.com/callback"},
	})

	authResp, _ := svc.BackchannelAuthentication(context.Background(), &BackchannelAuthRequest{
		TenantID:  testTenantID,
		ClientID:  result.Client.ClientID,
		LoginHint: "user@example.com",
	})

	// First poll: pending.
	svc.PollCIBAToken(context.Background(), testTenantID, authResp.AuthReqID, result.Client.ClientID, "")

	// Immediate second poll: slow_down.
	_, err := svc.PollCIBAToken(context.Background(), testTenantID, authResp.AuthReqID, result.Client.ClientID, "")
	if err == nil {
		t.Fatal("expected slow_down error")
	}
	if !strings.Contains(err.Error(), "slow_down") {
		t.Errorf("expected slow_down, got: %v", err)
	}
}

func TestApproveCIBAAuth_NotFound(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	err := svc.ApproveCIBAAuth("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent auth_req_id")
	}
}

func TestDenyCIBAAuth_NotFound(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	err := svc.DenyCIBAAuth("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent auth_req_id")
	}
}

func TestCIBAError_Error(t *testing.T) {
	e := &CIBAError{Err: "authorization_pending", Desc: "waiting for user"}
	expected := "authorization_pending: waiting for user"
	if e.Error() != expected {
		t.Errorf("Error() = %s, want %s", e.Error(), expected)
	}
}

// --- Helpers ---

// makeTestJWT creates an unsigned JWT for testing (alg: none).
func makeTestJWT(header, payload string) string {
	h := base64.RawURLEncoding.EncodeToString([]byte(header))
	p := base64.RawURLEncoding.EncodeToString([]byte(payload))
	return h + "." + p + "."
}
