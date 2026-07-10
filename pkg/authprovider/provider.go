// Package authprovider defines the multi-backend authentication provider abstraction.
// GGID supports local users, LDAP/Active Directory, and external IdP (OIDC/SAML/OAuth2).
package authprovider

import (
	"context"

	ggiderrors "github.com/ggid/ggid/pkg/errors"
	"github.com/google/uuid"
)

// ProviderType identifies the authentication backend.
type ProviderType string

const (
	ProviderLocal   ProviderType = "local"
	ProviderLDAP    ProviderType = "ldap"
	ProviderOIDC    ProviderType = "oidc"
	ProviderSAML    ProviderType = "saml"
	ProviderOAuth2  ProviderType = "oauth2"
)

// Provider is the interface every auth backend must implement.
type Provider interface {
	// Type returns the provider type.
	Type() ProviderType
	// Name returns the human-readable provider name.
	Name() string
	// Authenticate verifies credentials and returns the result.
	Authenticate(ctx context.Context, creds Credentials) (*AuthResult, error)
}

// Credentials holds the data submitted by a user during authentication.
type Credentials struct {
	Username string
	Password string
	// For external providers (future use)
	Token     string
	Assertion string
}

// AuthResult is returned by a successful authentication.
type AuthResult struct {
	ExternalID  string         // ID in the external system (LDAP DN, OIDC sub, etc.)
	Provider    ProviderType
	Attributes  map[string]any // synced attributes (email, display name, groups...)
	MustLink    bool           // needs linking to a local account
	NewUser     bool           // first-time login, requires JIT provisioning
	LinkedUser  *uuid.UUID     // non-nil if already linked to a local user
	Roles       []string       // mapped roles from group membership
}

// Chain holds an ordered list of providers and tries them in sequence.
type Chain struct {
	providers []Provider
}

// NewChain creates a new provider chain.
func NewChain(providers ...Provider) *Chain {
	return &Chain{providers: providers}
}

// Authenticate tries each provider in order until one succeeds.
func (c *Chain) Authenticate(ctx context.Context, creds Credentials) (*AuthResult, error) {
	var lastErr error
	for _, p := range c.providers {
		result, err := p.Authenticate(ctx, creds)
		if err == nil {
			return result, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		return nil, ggiderrors.Unauthenticated("no auth providers configured")
	}
	return nil, lastErr
}
