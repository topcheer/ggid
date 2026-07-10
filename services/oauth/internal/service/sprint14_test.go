package service

import (
	"context"
	"testing"
	"time"

	"github.com/ggid/ggid/services/oauth/internal/domain"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func TestSprint14_ExchangeToken_Success(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	subjectToken := signTestToken(svc, map[string]interface{}{
		"sub": "delegation-user", "exp": time.Now().Add(1 * time.Hour).Unix(), "iss": "https://test.ggid.dev",
	})
	resp, err := svc.ExchangeToken(context.Background(), &TokenExchangeRequestRFC8693{
		SubjectToken: subjectToken, SubjectTokenType: "urn:ietf:params:oauth:token-type:access_token",
		Scope: []string{"openid"},
	})
	if err != nil {
		t.Fatalf("ExchangeToken: %v", err)
	}
	if resp.AccessToken == "" {
		t.Error("expected non-empty access_token")
	}
}

func TestSprint14_DeviceFlow_FullSuccess(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	dr, _ := svc.CreateDeviceAuthorization(&DeviceAuthorizationRequest{TenantID: testTenantID, ClientID: "dev-full", Scope: []string{"openid"}})
	if err := svc.ApproveDeviceCode(dr.UserCode, uuid.New()); err != nil {
		t.Fatalf("ApproveDeviceCode: %v", err)
	}
	token, err := svc.PollDeviceToken(context.Background(), dr.DeviceCode, "dev-full")
	if err != nil {
		t.Fatalf("PollDeviceToken: %v", err)
	}
	if token.AccessToken == "" {
		t.Error("expected non-empty token")
	}
}

func TestSprint14_PAR_JAR_Integration(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	uri := "urn:ietf:params:oauth:request_uri:sprint14-integ"
	parStore.Store(uri, parEntry{
		Request: &PushedAuthorizationRequest{ClientID: "par-jar", ResponseType: "code", Scope: "openid"},
		ExpiresAt: time.Now().Add(5 * time.Minute),
	})
	claims, err := svc.ValidateAuthorizationRequest(context.Background(), "par-jar", "", uri)
	if err != nil {
		t.Fatalf("ValidateAuthorizationRequest: %v", err)
	}
	if claims["response_type"] != "code" {
		t.Errorf("expected code, got %v", claims["response_type"])
	}
}

func TestSprint14_JAR_Direct(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	tok := makeSimpleJARJWT(jwt.MapClaims{
		"iss": "jar-c", "aud": "https://test.ggid.dev",
		"exp": float64(time.Now().Add(5 * time.Minute).Unix()), "response_type": "code",
	})
	claims, err := svc.ValidateJARRequest(context.Background(), "jar-c", tok)
	if err != nil {
		t.Fatalf("ValidateJARRequest: %v", err)
	}
	if claims.ClientID != "jar-c" {
		t.Errorf("expected jar-c, got %s", claims.ClientID)
	}
}

func TestSprint14_CC_FullFlow(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	result, _ := svc.CreateClient(testCtx(), &CreateClientInput{
		TenantID: testTenantID, Name: "CC", Type: domain.ClientTypeConfidential,
		GrantTypes: []string{"client_credentials"}, Scopes: []string{"read"},
	})
	resp, err := svc.ClientCredentials(context.Background(), &ClientCredentialsRequest{
		TenantID: testTenantID, ClientID: result.Client.ClientID, ClientSecret: result.ClientSecret,
	})
	if err != nil {
		t.Fatalf("ClientCredentials: %v", err)
	}
	if resp.AccessToken == "" {
		t.Error("expected token")
	}
}

func TestSprint14_Discovery_Fields(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	c := svc.GetDiscoveryConfig()
	if c.Issuer == "" || c.AuthorizationEndpoint == "" || c.TokenEndpoint == "" || c.JwksURI == "" {
		t.Error("missing required discovery fields")
	}
	if c.CheckSessionIFrame == "" {
		t.Error("missing check_session_iframe")
	}
	if !c.BackchannelLogoutSupported {
		t.Error("expected backchannel_logout_supported")
	}
}

func TestSprint14_DynReg_NoTenant(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	_, err := svc.DynamicClientRegister(context.Background(), &DynamicRegistrationRequest{
		ClientName: "test", RedirectURIs: []string{"https://app.example.com/cb"},
	})
	if err == nil {
		t.Error("expected error for missing tenant")
	}
}

func TestSprint14_CryptoRandInt_Boundary(t *testing.T) {
	for i := 0; i < 20; i++ {
		if v := cryptoRandInt(1); v != 0 {
			t.Errorf("expected 0 for max=1, got %d", v)
		}
	}
}

func TestSprint14_issueDeviceAccessToken(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	tok, exp, err := svc.issueDeviceAccessToken(testTenantID, uuid.New())
	if err != nil {
		t.Fatalf("issueDeviceAccessToken: %v", err)
	}
	if tok == "" || exp <= 0 {
		t.Error("expected valid token and expiry")
	}
}
