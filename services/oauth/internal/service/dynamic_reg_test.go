package service

import (
	"context"
	"testing"

	"github.com/ggid/ggid/pkg/tenant"
	"github.com/ggid/ggid/services/oauth/internal/domain"
	"github.com/google/uuid"
)

func TestDynamicClientRegister_Success(t *testing.T) {
	svc, clientRepo, _, _ := newTestOAuthService()

	ctx := tenant.WithContext(context.Background(), &tenant.Context{
		TenantID:       uuid.New(),
		IsolationLevel: tenant.IsolationShared,
	})

	req := &DynamicRegistrationRequest{
		ClientName:    "My Dynamic App",
		RedirectURIs: []string{"https://app.example.com/callback"},
		GrantTypes:   []string{"authorization_code", "refresh_token"},
		ResponseTypes: []string{"code"},
		Scope:        "openid profile email",
	}

	resp, err := svc.DynamicClientRegister(ctx, req)
	if err != nil {
		t.Fatalf("DynamicClientRegister: %v", err)
	}
	if resp.ClientID == "" {
		t.Error("expected non-empty client_id")
	}
	if resp.ClientSecret == "" {
		t.Error("expected non-empty client_secret for confidential client")
	}
	if resp.ClientName != "My Dynamic App" {
		t.Errorf("expected client_name 'My Dynamic App', got %s", resp.ClientName)
	}

	// Verify client was persisted.
	stored, ok := clientRepo.clients[resp.ClientID]
	if !ok {
		t.Fatal("client not persisted in repo")
	}
	if stored.Type != domain.ClientTypeConfidential {
		t.Error("expected confidential type")
	}
}

func TestDynamicClientRegister_NoRedirectURIs(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	ctx := tenant.WithContext(context.Background(), &tenant.Context{
		TenantID:       uuid.New(),
		IsolationLevel: tenant.IsolationShared,
	})

	_, err := svc.DynamicClientRegister(ctx, &DynamicRegistrationRequest{})
	if err == nil {
		t.Error("expected error for missing redirect_uris")
	}
}

func TestDynamicClientRegister_Defaults(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	ctx := tenant.WithContext(context.Background(), &tenant.Context{
		TenantID:       uuid.New(),
		IsolationLevel: tenant.IsolationShared,
	})

	resp, err := svc.DynamicClientRegister(ctx, &DynamicRegistrationRequest{
		RedirectURIs: []string{"https://app.example.com/callback"},
	})
	if err != nil {
		t.Fatalf("DynamicClientRegister: %v", err)
	}

	if len(resp.GrantTypes) != 2 || resp.GrantTypes[0] != "authorization_code" {
		t.Errorf("expected default grant_types, got %v", resp.GrantTypes)
	}
	if resp.TokenEndpointAuthMethod != "client_secret_basic" {
		t.Errorf("expected default auth method, got %s", resp.TokenEndpointAuthMethod)
	}
	if resp.Scope != "openid profile email" {
		t.Errorf("expected default scope, got %s", resp.Scope)
	}
}

func TestDynamicClientRegister_NoTenant(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	_, err := svc.DynamicClientRegister(context.Background(), &DynamicRegistrationRequest{
		RedirectURIs: []string{"https://app.example.com/callback"},
	})
	if err == nil {
		t.Error("expected error for missing tenant context")
	}
}

func TestDynamicClientRegister_Persists(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	tenantID := uuid.New()
	ctx := tenant.WithContext(context.Background(), &tenant.Context{
		TenantID:       tenantID,
		IsolationLevel: tenant.IsolationShared,
	})

	resp, err := svc.DynamicClientRegister(ctx, &DynamicRegistrationRequest{
		RedirectURIs: []string{"https://app.example.com/callback"},
		ClientName:   "Test App",
	})
	if err != nil {
		t.Fatalf("DynamicClientRegister: %v", err)
	}

	// Verify persisted via GetClient.
	client, err := svc.GetClient(ctx, resp.ClientID)
	if err != nil {
		t.Fatalf("GetClient: %v", err)
	}
	if client.Name != "Test App" {
		t.Errorf("expected 'Test App', got %s", client.Name)
	}
}

func TestClaimRulesEngine_DynamicReg_NilReceiver(t *testing.T) {
	var engine *ClaimRulesEngine
	claims := map[string]any{"sub": "user123"}
	engine.ApplyRules(claims, nil)
}

func TestClaimRulesEngine_DynamicReg_DefaultValue(t *testing.T) {
	engine := NewClaimRulesEngine([]ClaimRule{
		{ClaimName: "department", SourceAttr: "dept", Default: "Engineering"},
	})
	claims := map[string]any{"sub": "user123"}
	engine.ApplyRules(claims, map[string]any{})
	if dept, ok := claims["department"]; !ok || dept != "Engineering" {
		t.Errorf("expected department='Engineering', got %v", dept)
	}
}

func TestClaimRulesEngine_DynamicReg_FromSource(t *testing.T) {
	engine := NewClaimRulesEngine([]ClaimRule{
		{ClaimName: "department", SourceAttr: "dept", Default: "Engineering"},
	})
	claims := map[string]any{"sub": "user123"}
	engine.ApplyRules(claims, map[string]any{"dept": "Sales"})
	if dept, ok := claims["department"]; !ok || dept != "Sales" {
		t.Errorf("expected department='Sales', got %v", dept)
	}
}

func TestIntrospectToken_DynamicReg_Invalid(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	result := svc.IntrospectToken("invalid-token")
	if result.Active {
		t.Error("expected active=false for invalid token")
	}
}
