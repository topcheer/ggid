package service

import (
	"context"
	"strings"
	"testing"

	"github.com/ggid/ggid/pkg/tenant"
	"github.com/google/uuid"
)

// TestDynamicClientRegister_ClientCredentials_Flow verifies that a DCR client
// registered with grant_types=[client_credentials] can actually obtain a token
// via the client_credentials grant. This is a regression test for the
// productization gap: "DCR accepts grant_types but doesn't persist".
func TestDynamicClientRegister_ClientCredentials_Flow(t *testing.T) {
	svc, clientRepo, _, _ := newTestOAuthService()

	tenantID := uuid.New()
	ctx := tenant.WithContext(context.Background(), &tenant.Context{
		TenantID:       tenantID,
		IsolationLevel: tenant.IsolationShared,
	})

	// 1. Register a dynamic client for machine-to-machine use.
	regResp, err := svc.DynamicClientRegister(ctx, &DynamicRegistrationRequest{
		ClientName:   "M2M Service",
		GrantTypes:   []string{"client_credentials"},
		ResponseTypes: []string{"token"},
		Scope:        "read write",
	})
	if err != nil {
		t.Fatalf("DynamicClientRegister: %v", err)
	}
	if regResp.ClientID == "" || regResp.ClientSecret == "" {
		t.Fatal("expected client_id and client_secret")
	}

	// 2. Verify the client was persisted with client_credentials grant type.
	stored, ok := clientRepo.clients[regResp.ClientID]
	if !ok {
		t.Fatal("client not persisted in repo")
	}
	if !stored.SupportsGrantType("client_credentials") {
		t.Fatalf("expected persisted client to support client_credentials, got %v", stored.GrantTypes)
	}

	// 3. Exchange via client_credentials grant.
	tokenResp, err := svc.ClientCredentials(ctx, &ClientCredentialsRequest{
		TenantID:     tenantID,
		ClientID:     regResp.ClientID,
		ClientSecret: regResp.ClientSecret,
		Scope:        []string{"read"},
	})
	if err != nil {
		t.Fatalf("ClientCredentials: %v", err)
	}
	if tokenResp.AccessToken == "" {
		t.Error("expected access token")
	}
	if !strings.Contains(tokenResp.Scope, "read") {
		t.Errorf("expected scope to include read, got %s", tokenResp.Scope)
	}
}

// TestDynamicClientRegister_MultipleGrantTypes_Persisted verifies that arbitrary
// grant types passed to DCR are persisted and reflected in GetClient.
func TestDynamicClientRegister_MultipleGrantTypes_Persisted(t *testing.T) {
	svc, clientRepo, _, _ := newTestOAuthService()

	tenantID := uuid.New()
	ctx := tenant.WithContext(context.Background(), &tenant.Context{
		TenantID:       tenantID,
		IsolationLevel: tenant.IsolationShared,
	})

	grants := []string{"authorization_code", "refresh_token", "client_credentials"}
	regResp, err := svc.DynamicClientRegister(ctx, &DynamicRegistrationRequest{
		ClientName:    "Multi-Grant App",
		RedirectURIs:  []string{"https://app.example.com/callback"},
		GrantTypes:    grants,
		ResponseTypes: []string{"code", "token"},
		Scope:         "openid profile email",
	})
	if err != nil {
		t.Fatalf("DynamicClientRegister: %v", err)
	}

	stored, ok := clientRepo.clients[regResp.ClientID]
	if !ok {
		t.Fatal("client not persisted")
	}
	if len(stored.GrantTypes) != len(grants) {
		t.Fatalf("expected %d grant types, got %d: %v", len(grants), len(stored.GrantTypes), stored.GrantTypes)
	}
	for _, gt := range grants {
		if !stored.SupportsGrantType(gt) {
			t.Errorf("expected client to support %s", gt)
		}
	}
}