package service

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/ggid/ggid/services/oauth/internal/domain"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// makeClientAssertionJWT builds a JWT for RFC 7523 testing.
func makeClientAssertionJWT(iss, sub, aud string, exp int64) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payload := `{"iss":"` + iss + `","sub":"` + sub + `","aud":"` + aud + `","exp":` + formatInt(exp) + `}`
	p := base64.RawURLEncoding.EncodeToString([]byte(payload))
	return header + "." + p + "."
}

// --- Coverage: RPInitiatedLogout (83.3%) ---

func TestCovSprint12_RPLogout_WithIDTokenHint(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	result, err := svc.RPInitiatedLogout(&RPInitiatedLogoutRequest{
		ClientID:              "client-123",
		PostLogoutRedirectURI: "https://app.example.com/done",
		State:                 "xyz",
	})
	if err != nil {
		t.Fatalf("RPInitiatedLogout: %v", err)
	}
	if result.RedirectURL == "" {
		t.Error("expected redirect URL")
	}
	if !strings.Contains(result.RedirectURL, "state=xyz") {
		t.Error("expected state in redirect URL")
	}
}

func TestCovSprint12_RPLogout_InvalidRedirectURI(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	_, err := svc.RPInitiatedLogout(&RPInitiatedLogoutRequest{
		PostLogoutRedirectURI: "not-a-url",
	})
	if err == nil {
		t.Error("expected error for invalid redirect URI")
	}
}

func TestCovSprint12_RPLogout_NoRedirect(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	result, err := svc.RPInitiatedLogout(&RPInitiatedLogoutRequest{})
	if err != nil {
		t.Fatalf("RPInitiatedLogout: %v", err)
	}
	if result.RedirectURL != "" {
		t.Error("expected empty redirect URL")
	}
}

// --- Coverage: BackchannelLogoutEndpoint (85.7%) ---

func TestCovSprint12_BCLEmpty(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	err := svc.BackchannelLogoutEndpoint("")
	if err == nil {
		t.Error("expected error for empty token")
	}
}

func TestCovSprint12_BCLValidWithSub(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	token := makeTestJWT(
		`{"alg":"none","typ":"JWT"}`,
		`{"sub":"user-bcl","jti":"bcl-jti-1","events":{"http://schemas.openid.net/event/backchannel-logout":{}}}`,
	)
	err := svc.BackchannelLogoutEndpoint(token)
	if err != nil {
		t.Fatalf("BackchannelLogoutEndpoint: %v", err)
	}
}

func TestCovSprint12_BCLNoSubOrSid(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	token := makeTestJWT(
		`{"alg":"none","typ":"JWT"}`,
		`{"events":{"http://schemas.openid.net/event/backchannel-logout":{}}}`,
	)
	err := svc.BackchannelLogoutEndpoint(token)
	if err == nil {
		t.Error("expected error for missing sub and sid")
	}
}

// --- Coverage: PollCIBAToken (81.8%) ---

func TestCovSprint12_PollCIBA_Approved(t *testing.T) {
	svc, clientRepo, _, _ := newTestOAuthService()

	client := &domain.OAuthClient{
		ID:            uuid.New(),
		TenantID:      testTenantID,
		ClientID:      "ciba-poll-client",
		Type:          domain.ClientTypePublic,
		GrantTypes:    []string{"urn:openid:params:grant-type:ciba"},
		Enabled:       true,
	}
	_ = clientRepo.CreateClient(context.Background(), client)

	resp, _ := svc.BackchannelAuthentication(context.Background(), &BackchannelAuthRequest{
		TenantID:  testTenantID,
		ClientID:  "ciba-poll-client",
		LoginHint: "user@example.com",
	})

	// Approve
	svc.ApproveCIBAAuth(resp.AuthReqID)

	// Poll — should get token
	token, err := svc.PollCIBAToken(context.Background(), testTenantID, resp.AuthReqID, "ciba-poll-client", "")
	if err != nil {
		t.Fatalf("PollCIBAToken approved: %v", err)
	}
	if token.AccessToken == "" {
		t.Error("expected non-empty access token")
	}
}

func TestCovSprint12_PollCIBA_Denied(t *testing.T) {
	svc, clientRepo, _, _ := newTestOAuthService()

	client := &domain.OAuthClient{
		ID:            uuid.New(),
		TenantID:      testTenantID,
		ClientID:      "ciba-deny-client",
		Type:          domain.ClientTypePublic,
		GrantTypes:    []string{"urn:openid:params:grant-type:ciba"},
		Enabled:       true,
	}
	_ = clientRepo.CreateClient(context.Background(), client)

	resp, _ := svc.BackchannelAuthentication(context.Background(), &BackchannelAuthRequest{
		TenantID:  testTenantID,
		ClientID:  "ciba-deny-client",
		LoginHint: "user@example.com",
	})

	svc.DenyCIBAAuth(resp.AuthReqID)

	_, err := svc.PollCIBAToken(context.Background(), testTenantID, resp.AuthReqID, "ciba-deny-client", "")
	if err == nil {
		t.Error("expected access_denied")
	}
}

func TestCovSprint12_PollCIBA_SlowDown(t *testing.T) {
	svc, clientRepo, _, _ := newTestOAuthService()

	client := &domain.OAuthClient{
		ID:            uuid.New(),
		TenantID:      testTenantID,
		ClientID:      "ciba-slow-client",
		Type:          domain.ClientTypePublic,
		GrantTypes:    []string{"urn:openid:params:grant-type:ciba"},
		Enabled:       true,
	}
	_ = clientRepo.CreateClient(context.Background(), client)

	resp, _ := svc.BackchannelAuthentication(context.Background(), &BackchannelAuthRequest{
		TenantID:  testTenantID,
		ClientID:  "ciba-slow-client",
		LoginHint: "user@example.com",
	})

	// First poll — sets LastPoll
	svc.PollCIBAToken(context.Background(), testTenantID, resp.AuthReqID, "ciba-slow-client", "")
	// Immediate second poll — should be slow_down
	_, err := svc.PollCIBAToken(context.Background(), testTenantID, resp.AuthReqID, "ciba-slow-client", "")
	if err == nil {
		t.Error("expected slow_down")
	}
}

func TestCovSprint12_PollCIBA_Expired(t *testing.T) {
	svc, clientRepo, _, _ := newTestOAuthService()

	client := &domain.OAuthClient{
		ID:            uuid.New(),
		TenantID:      testTenantID,
		ClientID:      "ciba-exp-client",
		Type:          domain.ClientTypePublic,
		GrantTypes:    []string{"urn:openid:params:grant-type:ciba"},
		Enabled:       true,
	}
	_ = clientRepo.CreateClient(context.Background(), client)

	resp, _ := svc.BackchannelAuthentication(context.Background(), &BackchannelAuthRequest{
		TenantID:  testTenantID,
		ClientID:  "ciba-exp-client",
		LoginHint: "user@example.com",
	})

	// Simulate expiry
	cibaStore.Store(resp.AuthReqID, cibaEntry{
		Status:    CIBAStatusPending,
		ExpiresAt: time.Now().Add(-1 * time.Minute),
	})

	_, err := svc.PollCIBAToken(context.Background(), testTenantID, resp.AuthReqID, "ciba-exp-client", "")
	if err == nil {
		t.Error("expected expired_token")
	}
}

// --- Coverage: UpdateClientMetadata (84.2%) ---

func TestCovSprint12_UpdateClient_AllFields(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	result, _ := svc.CreateClient(testCtx(), &CreateClientInput{
		TenantID:      testTenantID,
		Name:          "Original",
		Type:          domain.ClientTypeConfidential,
		GrantTypes:    []string{"authorization_code"},
		ResponseTypes: []string{"code"},
		RedirectURIs:  []string{"https://app.example.com/cb"},
	})

	newName := "New Name"
	newAuthMethod := "client_secret_post"
	updated, err := svc.UpdateClientMetadata(testCtx(), result.Client.ClientID, &ClientMetadataUpdate{
		Name:                    &newName,
		RedirectURIs:            []string{"https://new.example.com/cb"},
		GrantTypes:              []string{"authorization_code", "refresh_token"},
		ResponseTypes:           []string{"code", "token"},
		Scopes:                  []string{"openid", "profile"},
		TokenEndpointAuthMethod: &newAuthMethod,
	})
	if err != nil {
		t.Fatalf("UpdateClientMetadata: %v", err)
	}
	if updated.Name != "New Name" {
		t.Errorf("expected New Name, got %s", updated.Name)
	}
	if len(updated.RedirectURIs) != 1 || updated.RedirectURIs[0] != "https://new.example.com/cb" {
		t.Errorf("unexpected redirect URIs: %v", updated.RedirectURIs)
	}
}

// --- Coverage: cryptoRandInt (83.3%) ---

func TestCovSprint12_CryptoRandInt_Negative(t *testing.T) {
	v := cryptoRandInt(-5)
	if v != 0 {
		t.Errorf("expected 0 for negative max, got %d", v)
	}
}

func TestCovSprint12_CryptoRandInt_Zero(t *testing.T) {
	v := cryptoRandInt(0)
	if v != 0 {
		t.Errorf("expected 0 for zero max, got %d", v)
	}
}

// --- RFC 7523: JWT Client Auth ---

func TestRFC7523_ValidAssertion(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	token := makeClientAssertionJWT("client-1", "client-1", "https://test.ggid.dev", time.Now().Add(5*time.Minute).Unix())
	claims, err := svc.ValidateClientAssertion(token, "client-1")
	if err != nil {
		t.Fatalf("ValidateClientAssertion: %v", err)
	}
	if claims.ClientID != "client-1" {
		t.Errorf("expected client-1, got %s", claims.ClientID)
	}
}

func TestRFC7523_EmptyAssertion(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	_, err := svc.ValidateClientAssertion("", "client-1")
	if err == nil {
		t.Error("expected error for empty assertion")
	}
}

func TestRFC7523_EmptyClientID(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	_, err := svc.ValidateClientAssertion("token", "")
	if err == nil {
		t.Error("expected error for empty client_id")
	}
}

func TestRFC7523_InvalidJWT(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	_, err := svc.ValidateClientAssertion("not.a.jwt", "client-1")
	if err == nil {
		t.Error("expected error for invalid JWT")
	}
}

func TestRFC7523_IssMismatch(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	token := makeClientAssertionJWT("wrong", "client-1", "https://test.ggid.dev", time.Now().Add(5*time.Minute).Unix())
	_, err := svc.ValidateClientAssertion(token, "client-1")
	if err == nil {
		t.Error("expected error for iss mismatch")
	}
}

func TestRFC7523_SubMismatch(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	token := makeClientAssertionJWT("client-1", "wrong", "https://test.ggid.dev", time.Now().Add(5*time.Minute).Unix())
	_, err := svc.ValidateClientAssertion(token, "client-1")
	if err == nil {
		t.Error("expected error for sub mismatch")
	}
}

func TestRFC7523_WrongAud(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	token := makeClientAssertionJWT("client-1", "client-1", "https://wrong.example.com", time.Now().Add(5*time.Minute).Unix())
	_, err := svc.ValidateClientAssertion(token, "client-1")
	if err == nil {
		t.Error("expected error for wrong aud")
	}
}

func TestRFC7523_Expired(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	token := makeClientAssertionJWT("client-1", "client-1", "https://test.ggid.dev", 1)
	_, err := svc.ValidateClientAssertion(token, "client-1")
	if err == nil {
		t.Error("expected error for expired assertion")
	}
}

func TestRFC7523_MissingExp(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"iss":"client-1","sub":"client-1","aud":"https://test.ggid.dev"}`))
	token := header + "." + payload + "."
	_, err := svc.ValidateClientAssertion(token, "client-1")
	if err == nil {
		t.Error("expected error for missing exp")
	}
}

func TestRFC7523_JWTClientAuth_WrongType(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	_, err := svc.ValidateJWTClientAuth("wrong-type", "token", "client-1")
	if err == nil {
		t.Error("expected error for wrong assertion_type")
	}
}

func TestRFC7523_JWTClientAuth_Valid(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	token := makeClientAssertionJWT("client-1", "client-1", "https://test.ggid.dev", time.Now().Add(5*time.Minute).Unix())
	claims, err := svc.ValidateJWTClientAuth(ClientAssertionTypeRFC7523, token, "client-1")
	if err != nil {
		t.Fatalf("ValidateJWTClientAuth: %v", err)
	}
	if claims.ClientID != "client-1" {
		t.Errorf("expected client-1, got %s", claims.ClientID)
	}
}

// --- PKCE ---

func TestPKCE_VerifyCodeChallenge_S256(t *testing.T) {
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	challenge := hashTokenSHA256(verifier)
	if !VerifyCodeChallenge(challenge, verifier, "S256") {
		t.Error("expected S256 challenge to match")
	}
}

func TestPKCE_VerifyCodeChallenge_Plain(t *testing.T) {
	verifier := "plain-verifier-12345678901234567890"
	if !VerifyCodeChallenge(verifier, verifier, "plain") {
		t.Error("expected plain challenge to match")
	}
}

func TestPKCE_VerifyCodeChallenge_Mismatch(t *testing.T) {
	if VerifyCodeChallenge("wrong", "verifier", "S256") {
		t.Error("expected mismatch for wrong challenge")
	}
}

func TestPKCE_VerifyCodeChallenge_EmptyChallenge(t *testing.T) {
	if VerifyCodeChallenge("", "verifier", "S256") {
		t.Error("expected false for empty challenge")
	}
}

func TestPKCE_VerifyCodeChallenge_EmptyVerifier(t *testing.T) {
	if VerifyCodeChallenge("challenge", "", "S256") {
		t.Error("expected false for empty verifier")
	}
}

func TestPKCE_VerifyCodeChallenge_UnknownMethod(t *testing.T) {
	if VerifyCodeChallenge("challenge", "verifier", "unknown") {
		t.Error("expected false for unknown method")
	}
}

func TestPKCE_VerifyCodeChallenge_DefaultS256(t *testing.T) {
	verifier := "test-verifier-123456789012345678901234"
	challenge := hashTokenSHA256(verifier)
	if !VerifyCodeChallenge(challenge, verifier, "") {
		t.Error("expected default to be S256")
	}
}

func TestPKCE_IsPublicClient_Public(t *testing.T) {
	if !IsPublicClient("public") {
		t.Error("expected true for public")
	}
}

func TestPKCE_IsPublicClient_Confidential(t *testing.T) {
	if IsPublicClient("confidential") {
		t.Error("expected false for confidential")
	}
}

func TestPKCE_StringInSlice(t *testing.T) {
	if !StringInSlice("openid", []string{"openid", "profile"}) {
		t.Error("expected true for openid in slice")
	}
	if StringInSlice("missing", []string{"openid", "profile"}) {
		t.Error("expected false for missing")
	}
}

func TestPKCE_Authorize_RequiresChallenge(t *testing.T) {
	svc, clientRepo, _, _ := newTestOAuthService()

	// Create client that requires PKCE
	client := &domain.OAuthClient{
		ID:            uuid.New(),
		TenantID:      testTenantID,
		ClientID:      "pkce-required-client",
		Type:          domain.ClientTypePublic,
		GrantTypes:    []string{"authorization_code"},
		ResponseTypes: []string{"code"},
		RedirectURIs:  []string{"https://app.example.com/cb"},
		Enabled:       true,
	}
	_ = clientRepo.CreateClient(context.Background(), client)

	// Try to authorize without code_challenge
	_, err := svc.CreateAuthorizationCode(context.Background(), &AuthorizeRequest{
		TenantID:     testTenantID,
		ClientID:     "pkce-required-client",
		UserID:       uuid.New(),
		Scope:        []string{"openid"},
		RedirectURI:  "https://app.example.com/cb",
		ResponseType: "code",
	})
	if err == nil {
		t.Error("expected error for missing PKCE challenge on PKCE-required client")
	}
}

// --- OIDC UserInfo ---

func TestUserInfo_InvalidToken(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	_, err := svc.GetUserInfo("invalid-token")
	if err == nil {
		t.Error("expected error for invalid token")
	}
}

func TestUserInfo_EmptyToken(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	_, err := svc.GetUserInfo("")
	if err == nil {
		t.Error("expected error for empty token")
	}
}

func TestUserInfo_ValidToken(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	// Sign a token with the service's key
	token := signTestToken(svc, jwt.MapClaims{
		"sub":   "user-123",
		"email": "user@example.com",
		"name":  "Test User",
		"exp":   time.Now().Add(1 * time.Hour).Unix(),
		"iss":   "https://test.ggid.dev",
	})
	info, err := svc.GetUserInfo(token)
	if err != nil {
		t.Fatalf("GetUserInfo: %v", err)
	}
	if info.Sub != "user-123" {
		t.Errorf("expected sub=user-123, got %s", info.Sub)
	}
}

// --- Dynamic Client Registration ---

func TestDynReg_Success(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	resp, err := svc.DynamicClientRegister(testCtx(), &DynamicRegistrationRequest{
		ClientName:   "DynReg App",
		RedirectURIs: []string{"https://app.example.com/cb"},
		GrantTypes:   []string{"authorization_code"},
		ResponseTypes: []string{"code"},
	})
	if err != nil {
		t.Fatalf("DynamicClientRegister: %v", err)
	}
	if resp.ClientID == "" {
		t.Error("expected non-empty client_id")
	}
	if resp.ClientSecret == "" {
		t.Error("expected non-empty client_secret for confidential client")
	}
}

func TestDynReg_MissingRedirects(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	_, err := svc.DynamicClientRegister(testCtx(), &DynamicRegistrationRequest{
		ClientName: "No Redirects",
	})
	if err == nil {
		t.Error("expected error for missing redirect URIs")
	}
}

func TestDynReg_PublicClient(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	resp, err := svc.DynamicClientRegister(testCtx(), &DynamicRegistrationRequest{
		ClientName:              "Public App",
		RedirectURIs:            []string{"https://app.example.com/cb"},
		GrantTypes:              []string{"authorization_code"},
		ResponseTypes:           []string{"code"},
		TokenEndpointAuthMethod: "none",
	})
	if err != nil {
		t.Fatalf("DynamicClientRegister public: %v", err)
	}
	if resp.ClientID == "" {
		t.Error("expected non-empty client_id")
	}
}

// --- formatInt helper ---

func TestFormatInt(t *testing.T) {
	if formatInt(0) != "0" {
		t.Error("expected 0")
	}
	if formatInt(123) != "123" {
		t.Error("expected 123")
	}
	if formatInt(-1) != "-1" {
		t.Error("expected -1")
	}
}
