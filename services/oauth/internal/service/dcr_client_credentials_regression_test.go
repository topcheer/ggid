package service

import (
	"context"
	"testing"

	"github.com/ggid/ggid/pkg/tenant"
	"github.com/google/uuid"
)

// TestDynamicClientRegister_AllowsStandardGrants verifies that DCR allows
// standard OAuth 2.0 grant types (authorization_code, refresh_token,
// client_credentials) while filtering dangerous ones like password.
func TestDynamicClientRegister_ClientCredentials_Flow(t *testing.T) {
	svc, clientRepo, _, _ := newTestOAuthService()

	tenantID := uuid.New()
	ctx := tenant.WithContext(context.Background(), &tenant.Context{
		TenantID:       tenantID,
		IsolationLevel: tenant.IsolationShared,
	})

	// DCR with client_credentials — should be allowed as standard OAuth 2.0 grant.
	regResp, err := svc.DynamicClientRegister(ctx, &DynamicRegistrationRequest{
		ClientName:    "M2M Service",
		GrantTypes:    []string{"client_credentials"},
		ResponseTypes: []string{"token"},
		Scope:         "read write",
	})
	if err != nil {
		t.Fatalf("DynamicClientRegister: %v", err)
	}
	if regResp.ClientID == "" || regResp.ClientSecret == "" {
		t.Fatal("expected client_id and client_secret")
	}

	// client_credentials is a standard OAuth 2.0 grant type — should be allowed.
	stored, ok := clientRepo.clients[regResp.ClientID]
	if !ok {
		t.Fatal("client not persisted in repo")
	}
	if !stored.SupportsGrantType("client_credentials") {
		t.Fatalf("DCR should allow client_credentials, got %v", stored.GrantTypes)
	}
}

// TestDynamicClientRegister_FiltersDangerousGrants verifies that dangerous grant types
// (password) are filtered, but standard grants (authorization_code, refresh_token,
// client_credentials) are persisted.
func TestDynamicClientRegister_MultipleGrantTypes_Persisted(t *testing.T) {
	svc, clientRepo, _, _ := newTestOAuthService()

	tenantID := uuid.New()
	ctx := tenant.WithContext(context.Background(), &tenant.Context{
		TenantID:       tenantID,
		IsolationLevel: tenant.IsolationShared,
	})

	regResp, err := svc.DynamicClientRegister(ctx, &DynamicRegistrationRequest{
		ClientName:   "Test Client",
		GrantTypes:   []string{"authorization_code", "refresh_token", "client_credentials", "password"},
		RedirectURIs: []string{"https://example.com/callback"},
		Scope:        "read",
	})
	if err != nil {
		t.Fatalf("DynamicClientRegister: %v", err)
	}

	stored, ok := clientRepo.clients[regResp.ClientID]
	if !ok {
		t.Fatal("client not persisted in repo")
	}
	// Standard grants should be persisted.
	for _, g := range []string{"authorization_code", "refresh_token", "client_credentials"} {
		if !stored.SupportsGrantType(g) {
			t.Fatalf("expected grant %s to be persisted, got %v", g, stored.GrantTypes)
		}
	}
	// password grant must be filtered out.
	if stored.SupportsGrantType("password") {
		t.Fatalf("password grant should be filtered out, got %v", stored.GrantTypes)
	}
}
