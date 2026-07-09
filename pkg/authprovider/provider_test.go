package authprovider

import (
	"context"
	"testing"
)

// stubProvider is a test double for the Provider interface.
type stubProvider struct {
	providerType ProviderType
	name         string
	succeed      bool
	result       *AuthResult
}

func (s *stubProvider) Type() ProviderType { return s.providerType }
func (s *stubProvider) Name() string       { return s.name }
func (s *stubProvider) Authenticate(ctx context.Context, creds Credentials) (*AuthResult, error) {
	if s.succeed {
		return s.result, nil
	}
	return nil, &stubError{msg: "authentication failed"}
}

type stubError struct{ msg string }

func (e *stubError) Error() string { return e.msg }

func TestProviderType_Values(t *testing.T) {
	expected := map[ProviderType]string{
		ProviderLocal:  "local",
		ProviderLDAP:   "ldap",
		ProviderOIDC:   "oidc",
		ProviderSAML:   "saml",
		ProviderOAuth2: "oauth2",
	}
	for pt, val := range expected {
		if string(pt) != val {
			t.Fatalf("ProviderType mismatch: got %s, want %s", pt, val)
		}
	}
}

func TestChain_Authenticate_FirstProviderSucceeds(t *testing.T) {
	successResult := &AuthResult{
		ExternalID: "user-123",
		Provider:   ProviderLocal,
	}

	chain := NewChain(
		&stubProvider{providerType: ProviderLocal, name: "local", succeed: true, result: successResult},
		&stubProvider{providerType: ProviderLDAP, name: "ldap", succeed: false},
	)

	result, err := chain.Authenticate(context.Background(), Credentials{Username: "test", Password: "pass"})
	if err != nil {
		t.Fatalf("should not error when first provider succeeds: %v", err)
	}
	if result.ExternalID != "user-123" {
		t.Fatalf("unexpected ExternalID: got %s", result.ExternalID)
	}
	if result.Provider != ProviderLocal {
		t.Fatalf("unexpected Provider: got %s", result.Provider)
	}
}

func TestChain_Authenticate_FallbackToSecondProvider(t *testing.T) {
	ldapResult := &AuthResult{
		ExternalID: "cn=john,dc=corp,dc=local",
		Provider:   ProviderLDAP,
		NewUser:    true,
	}

	chain := NewChain(
		&stubProvider{providerType: ProviderLocal, name: "local", succeed: false},
		&stubProvider{providerType: ProviderLDAP, name: "ldap", succeed: true, result: ldapResult},
	)

	result, err := chain.Authenticate(context.Background(), Credentials{Username: "john", Password: "pass"})
	if err != nil {
		t.Fatalf("should succeed via fallback: %v", err)
	}
	if result.Provider != ProviderLDAP {
		t.Fatalf("expected LDAP provider, got %s", result.Provider)
	}
	if !result.NewUser {
		t.Fatal("NewUser should be true")
	}
}

func TestChain_Authenticate_AllFail(t *testing.T) {
	chain := NewChain(
		&stubProvider{providerType: ProviderLocal, name: "local", succeed: false},
		&stubProvider{providerType: ProviderLDAP, name: "ldap", succeed: false},
	)

	_, err := chain.Authenticate(context.Background(), Credentials{Username: "x", Password: "x"})
	if err == nil {
		t.Fatal("should error when all providers fail")
	}
}

func TestChain_Authenticate_EmptyChain(t *testing.T) {
	chain := NewChain()
	_, err := chain.Authenticate(context.Background(), Credentials{})
	if err == nil {
		t.Fatal("should error with empty chain")
	}
}

func TestCredentials_Fields(t *testing.T) {
	c := Credentials{
		Username: "admin",
		Password: "secret",
		Token:    "bearer-xyz",
	}
	if c.Username != "admin" || c.Password != "secret" || c.Token != "bearer-xyz" {
		t.Fatal("credentials fields not set correctly")
	}
}

func TestAuthResult_Fields(t *testing.T) {
	r := AuthResult{
		ExternalID: "ext-1",
		Provider:   ProviderOIDC,
		Attributes: map[string]any{"email": "test@example.com"},
		MustLink:   true,
		NewUser:    true,
	}
	if r.ExternalID != "ext-1" || r.Provider != ProviderOIDC {
		t.Fatal("auth result fields not set")
	}
	if r.Attributes["email"] != "test@example.com" {
		t.Fatal("attributes not set")
	}
	if !r.MustLink || !r.NewUser {
		t.Fatal("flags not set")
	}
}
