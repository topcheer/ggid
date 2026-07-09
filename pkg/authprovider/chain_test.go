package authprovider

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

// --- ChainEnhanced tests ---

func TestChainEnhanced_TypeFiltering(t *testing.T) {
	ldapResult := &AuthResult{ExternalID: "cn=john", Provider: ProviderLDAP}

	chain := NewChainEnhanced(
		&stubProvider{providerType: ProviderLocal, name: "local", succeed: false},
		&stubProvider{providerType: ProviderLDAP, name: "ldap", succeed: true, result: ldapResult},
	).OnlyTypes(ProviderLDAP) // skip local

	result, err := chain.Authenticate(context.Background(), Credentials{Username: "x", Password: "x"})
	if err != nil {
		t.Fatalf("expected success: %v", err)
	}
	if result.Provider != ProviderLDAP {
		t.Errorf("expected LDAP, got %s", result.Provider)
	}
}

func TestChainEnhanced_TypeFiltering_ExcludesAll(t *testing.T) {
	chain := NewChainEnhanced(
		&stubProvider{providerType: ProviderLocal, name: "local", succeed: true, result: &AuthResult{Provider: ProviderLocal}},
		&stubProvider{providerType: ProviderLDAP, name: "ldap", succeed: true, result: &AuthResult{Provider: ProviderLDAP}},
	).OnlyTypes(ProviderOIDC) // no OIDP configured

	_, err := chain.Authenticate(context.Background(), Credentials{Username: "x", Password: "x"})
	if err == nil {
		t.Fatal("expected error when all providers filtered out")
	}
}

func TestChainEnhanced_FallbackToSecond(t *testing.T) {
	ldapResult := &AuthResult{
		ExternalID: "cn=jane,dc=corp,dc=local",
		Provider:   ProviderLDAP,
	}

	chain := NewChainEnhanced(
		&stubProvider{providerType: ProviderLocal, name: "local", succeed: false},
		&stubProvider{providerType: ProviderLDAP, name: "ldap", succeed: true, result: ldapResult},
	)

	result, err := chain.Authenticate(context.Background(), Credentials{Username: "jane", Password: "pw"})
	if err != nil {
		t.Fatalf("fallback failed: %v", err)
	}
	if result.Provider != ProviderLDAP {
		t.Errorf("expected LDAP, got %s", result.Provider)
	}
}

func TestChainEnhanced_AllFail(t *testing.T) {
	chain := NewChainEnhanced(
		&stubProvider{providerType: ProviderLocal, name: "local", succeed: false},
		&stubProvider{providerType: ProviderLDAP, name: "ldap", succeed: false},
	)

	_, err := chain.Authenticate(context.Background(), Credentials{Username: "x", Password: "x"})
	if err == nil {
		t.Fatal("expected error when all fail")
	}
}

func TestChainEnhanced_Empty(t *testing.T) {
	chain := NewChainEnhanced()
	_, err := chain.Authenticate(context.Background(), Credentials{})
	if err == nil {
		t.Fatal("expected error with empty chain")
	}
}

func TestChainEnhanced_FirstProviderSucceeds(t *testing.T) {
	result := &AuthResult{ExternalID: "user-1", Provider: ProviderLocal}
	chain := NewChainEnhanced(
		&stubProvider{providerType: ProviderLocal, name: "local", succeed: true, result: result},
		&stubProvider{providerType: ProviderLDAP, name: "ldap", succeed: false},
	)

	r, err := chain.Authenticate(context.Background(), Credentials{Username: "a", Password: "b"})
	if err != nil {
		t.Fatalf("expected success: %v", err)
	}
	if r.ExternalID != "user-1" {
		t.Errorf("unexpected ExternalID: %s", r.ExternalID)
	}
}

func TestChainEnhanced_ProviderTypes(t *testing.T) {
	chain := NewChainEnhanced(
		&stubProvider{providerType: ProviderLocal, name: "local"},
		&stubProvider{providerType: ProviderLDAP, name: "ldap"},
		&stubProvider{providerType: ProviderOIDC, name: "oidc"},
	)

	types := chain.ProviderTypes()
	if len(types) != 3 {
		t.Fatalf("expected 3 types, got %d", len(types))
	}
	if types[0] != ProviderLocal || types[1] != ProviderLDAP || types[2] != ProviderOIDC {
		t.Errorf("unexpected types: %v", types)
	}
}

func TestChainEnhanced_ProviderNames(t *testing.T) {
	chain := NewChainEnhanced(
		&stubProvider{providerType: ProviderLocal, name: "local"},
		&stubProvider{providerType: ProviderLDAP, name: "ldap"},
	)

	names := chain.ProviderNames()
	if names != "local, ldap" {
		t.Errorf("expected 'local, ldap', got '%s'", names)
	}
}

func TestChainEnhanced_OnlyTypesChained(t *testing.T) {
	// Verify OnlyTypes returns the chain for fluent chaining.
	chain := NewChainEnhanced(
		&stubProvider{providerType: ProviderLocal, name: "local", succeed: true, result: &AuthResult{Provider: ProviderLocal}},
	)
	ret := chain.OnlyTypes(ProviderLocal)
	if ret != chain {
		t.Error("OnlyTypes should return the chain for fluent API")
	}
}

// --- WithTenantContext / resolveTenantID tests ---

func TestWithTenantContext_ResolveSuccess(t *testing.T) {
	tid := uuid.New()
	ctx := WithTenantContext(context.Background(), tid)

	resolved, err := resolveTenantID(ctx)
	if err != nil {
		t.Fatalf("resolveTenantID failed: %v", err)
	}
	if resolved != tid {
		t.Errorf("expected %s, got %s", tid, resolved)
	}
}

func TestResolveTenantID_NoContext(t *testing.T) {
	_, err := resolveTenantID(context.Background())
	if err == nil {
		t.Fatal("expected error without tenant context")
	}
}
