package service

import (
	"context"
	"testing"

	"github.com/ggid/ggid/pkg/tenant"
	"github.com/ggid/ggid/services/oauth/internal/domain"
)

func testCtx() context.Context {
	return tenant.WithContext(context.Background(), &tenant.Context{
		TenantID:       testTenantID,
		IsolationLevel: tenant.IsolationShared,
	})
}

// --- RFC 7592: UpdateClientMetadata ---

func TestRFC7592_UpdateClientMetadata_Success(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	result, _ := svc.CreateClient(testCtx(), &CreateClientInput{
		TenantID:      testTenantID,
		Name:          "Original",
		Type:          domain.ClientTypeConfidential,
		GrantTypes:    []string{"authorization_code"},
		ResponseTypes: []string{"code"},
		RedirectURIs:  []string{"https://app.example.com/callback"},
	})

	newName := "Updated Name"
	updated, err := svc.UpdateClientMetadata(testCtx(), result.Client.ClientID, &ClientMetadataUpdate{
		Name:         &newName,
		RedirectURIs: []string{"https://new.example.com/cb"},
	})
	if err != nil {
		t.Fatalf("UpdateClientMetadata: %v", err)
	}
	if updated.Name != "Updated Name" {
		t.Errorf("expected Updated Name, got %s", updated.Name)
	}
}

func TestRFC7592_UpdateClientMetadata_NotFound(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	_, err := svc.UpdateClientMetadata(testCtx(), "nonexistent", &ClientMetadataUpdate{})
	if err == nil {
		t.Error("expected error for nonexistent client")
	}
}

func TestRFC7592_UpdateClientMetadata_NoTenant(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	_, err := svc.UpdateClientMetadata(context.Background(), "any", &ClientMetadataUpdate{})
	if err == nil {
		t.Error("expected error for missing tenant context")
	}
}

func TestRFC7592_UpdateClientMetadata_PartialUpdate(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	result, _ := svc.CreateClient(testCtx(), &CreateClientInput{
		TenantID:      testTenantID,
		Name:          "Keep Me",
		Type:          domain.ClientTypeConfidential,
		GrantTypes:    []string{"authorization_code"},
		ResponseTypes: []string{"code"},
		RedirectURIs:  []string{"https://app.example.com/callback"},
	})

	// Only update scopes, leave name unchanged
	updated, err := svc.UpdateClientMetadata(testCtx(), result.Client.ClientID, &ClientMetadataUpdate{
		Scopes: []string{"openid", "profile", "email"},
	})
	if err != nil {
		t.Fatalf("UpdateClientMetadata partial: %v", err)
	}
	if updated.Name != "Keep Me" {
		t.Errorf("expected name unchanged, got %s", updated.Name)
	}
}

func TestRFC7592_DeleteClient_Success(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	result, _ := svc.CreateClient(testCtx(), &CreateClientInput{
		TenantID:      testTenantID,
		Name:          "Delete Me",
		Type:          domain.ClientTypePublic,
		GrantTypes:    []string{"authorization_code"},
		ResponseTypes: []string{"code"},
		RedirectURIs:  []string{"https://app.example.com/cb"},
	})
	err := svc.DeleteClient(testCtx(), result.Client.ClientID)
	if err != nil {
		t.Errorf("DeleteClient: %v", err)
	}
}

func TestRFC7592_RotateClientSecret_Success(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	result, _ := svc.CreateClient(testCtx(), &CreateClientInput{
		TenantID:      testTenantID,
		Name:          "Rotate Me",
		Type:          domain.ClientTypeConfidential,
		GrantTypes:    []string{"authorization_code"},
		ResponseTypes: []string{"code"},
		RedirectURIs:  []string{"https://app.example.com/cb"},
	})

	newSecret, err := svc.RotateClientSecret(testCtx(), testTenantID, result.Client.ClientID, result.ClientSecret)
	if err != nil {
		t.Fatalf("RotateClientSecret: %v", err)
	}
	if newSecret == "" || newSecret == result.ClientSecret {
		t.Error("expected new different secret")
	}
}

// --- Backchannel Logout Replay Prevention (jti) ---

func TestBackchannelReplay_NoReplay(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	token := makeTestJWT(
		`{"alg":"none","typ":"JWT"}`,
		`{"sub":"user-1","jti":"unique-jti-1","events":{"http://schemas.openid.net/event/backchannel-logout":{}}}`,
	)
	_, err := svc.ParseBackchannelLogoutToken(token)
	if err != nil {
		t.Fatalf("first parse should succeed: %v", err)
	}
}

func TestBackchannelReplay_DuplicateJti(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	token := makeTestJWT(
		`{"alg":"none","typ":"JWT"}`,
		`{"sub":"user-2","jti":"replay-jti-test","events":{"http://schemas.openid.net/event/backchannel-logout":{}}}`,
	)
	// First parse — should succeed
	_, err := svc.ParseBackchannelLogoutToken(token)
	if err != nil {
		t.Fatalf("first parse should succeed: %v", err)
	}
	// Second parse with same jti — should fail (replay detected)
	_, err = svc.ParseBackchannelLogoutToken(token)
	if err == nil {
		t.Error("expected replay detection error for duplicate jti")
	}
}

func TestBackchannelReplay_NoJti(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	token := makeTestJWT(
		`{"alg":"none","typ":"JWT"}`,
		`{"sub":"user-3","events":{"http://schemas.openid.net/event/backchannel-logout":{}}}`,
	)
	// Without jti, replay prevention doesn't apply — should succeed
	_, err := svc.ParseBackchannelLogoutToken(token)
	if err != nil {
		t.Fatalf("parse without jti should succeed: %v", err)
	}
}

func TestBackchannel_SIDExtraction(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	token := makeTestJWT(
		`{"alg":"none","typ":"JWT"}`,
		`{"sid":"session-abc-123","jti":"sid-jti-1","events":{"http://schemas.openid.net/event/backchannel-logout":{}}}`,
	)
	claims, err := svc.ParseBackchannelLogoutToken(token)
	if err != nil {
		t.Fatalf("parse with sid: %v", err)
	}
	if claims["sid"] != "session-abc-123" {
		t.Errorf("expected sid=session-abc-123, got %v", claims["sid"])
	}
}

// --- Discovery Well-Known Endpoint ---

func TestDiscovery_Issuer(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	config := svc.GetDiscoveryConfig()
	if config.Issuer != "https://test.ggid.dev" {
		t.Errorf("expected https://test.ggid.dev, got %s", config.Issuer)
	}
}

func TestDiscovery_AllEndpoints(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	config := svc.GetDiscoveryConfig()

	checks := map[string]string{
		"authorization_endpoint": config.AuthorizationEndpoint,
		"token_endpoint":         config.TokenEndpoint,
		"userinfo_endpoint":      config.UserInfoEndpoint,
		"jwks_uri":               config.JwksURI,
		"revocation_endpoint":    config.RevocationEndpoint,
		"introspection_endpoint": config.IntrospectionEndpoint,
	}
	for name, val := range checks {
		if val == "" {
			t.Errorf("expected non-empty %s", name)
		}
		if val[:4] != "http" {
			t.Errorf("expected URL for %s, got %s", name, val)
		}
	}
}

func TestDiscovery_SupportedScopes(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	config := svc.GetDiscoveryConfig()
	if len(config.ScopesSupported) < 3 {
		t.Error("expected at least 3 supported scopes")
	}
}

func TestDiscovery_SupportedGrants(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	config := svc.GetDiscoveryConfig()
	grants := config.GrantTypesSupported
	if len(grants) < 3 {
		t.Error("expected at least 3 grant types")
	}
	hasAuthCode := false
	for _, g := range grants {
		if g == "authorization_code" {
			hasAuthCode = true
		}
	}
	if !hasAuthCode {
		t.Error("expected authorization_code in grants")
	}
}

func TestDiscovery_SupportedResponseTypes(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	config := svc.GetDiscoveryConfig()
	if len(config.ResponseTypesSupported) == 0 {
		t.Error("expected at least 1 response type")
	}
}

func TestDiscovery_AuthMethods(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	config := svc.GetDiscoveryConfig()

	// Check mTLS methods are present (RFC 8705)
	hasTLS := false
	for _, m := range config.TokenEndpointAuthMethodsSupported {
		if m == "tls_client_auth" {
			hasTLS = true
		}
	}
	if !hasTLS {
		t.Error("expected tls_client_auth in token_endpoint_auth_methods_supported")
	}
}

func TestDiscovery_CodeChallengeMethods(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	config := svc.GetDiscoveryConfig()
	if len(config.CodeChallengeMethodsSupported) == 0 {
		t.Error("expected at least 1 code challenge method")
	}
	hasS256 := false
	for _, m := range config.CodeChallengeMethodsSupported {
		if m == "S256" {
			hasS256 = true
		}
	}
	if !hasS256 {
		t.Error("expected S256 in code_challenge_methods_supported")
	}
}

func TestDiscovery_ClaimsSupported(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	config := svc.GetDiscoveryConfig()
	if len(config.ClaimsSupported) < 3 {
		t.Error("expected at least 3 supported claims")
	}
}
