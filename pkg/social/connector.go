// Package social implements pluggable social login connectors for GGID.
// Each provider (Google, GitHub, Microsoft, Apple, generic OIDC) implements
// the Connector interface and can be registered per-tenant.
package social

import (
	"context"
	"errors"
)

// UserInfo represents the normalized user profile returned by a social provider.
type UserInfo struct {
	Provider      string // "google", "github", etc.
	ExternalID    string // unique ID from the provider
	Email         string
	Name          string
	AvatarURL     string
	EmailVerified bool            // whether the IdP has verified the email
	RawClaims     map[string]any // full claims from the IdP
}

// Connector defines the interface every social login provider must implement.
// The flow is: GetAuthURL → redirect user → HandleCallback → UserInfo.
type Connector interface {
	// ID returns the connector identifier (e.g. "google", "github").
	ID() string
	// DisplayName returns a human-readable name.
	DisplayName() string
	// GetAuthURL builds the authorization redirect URL.
	GetAuthURL(ctx context.Context, state string, redirectURI string) (string, error)
	// HandleCallback exchanges the authorization code for user info.
	HandleCallback(ctx context.Context, code string, state string, redirectURI string) (*UserInfo, error)
}

// Registry holds all registered social connectors by ID.
type Registry struct {
	connectors map[string]Connector
}

// NewRegistry creates an empty connector registry.
func NewRegistry() *Registry {
	return &Registry{connectors: make(map[string]Connector)}
}

// Register adds a connector to the registry.
func (r *Registry) Register(c Connector) {
	r.connectors[c.ID()] = c
}

// Get returns the connector with the given ID.
func (r *Registry) Get(id string) (Connector, error) {
	c, ok := r.connectors[id]
	if !ok {
		return nil, errors.New("social connector not registered: " + id)
	}
	return c, nil
}

// List returns all registered connector IDs.
func (r *Registry) List() []string {
	ids := make([]string, 0, len(r.connectors))
	for id := range r.connectors {
		ids = append(ids, id)
	}
	return ids
}
