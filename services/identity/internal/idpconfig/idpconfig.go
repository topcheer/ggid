// Package idpconfig provides per-tenant IdP configuration management.
//
// Each tenant can configure its own Identity Provider settings (SAML, OIDC, LDAP),
// enabling multi-tenant IdP federation without global configuration.
package idpconfig

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// IdPType identifies the identity provider protocol.
type IdPType string

const (
	IdPTypeSAML IdPType = "saml"
	IdPTypeOIDC IdPType = "oidc"
	IdPTypeLDAP IdPType = "ldap"
)

// TenantIdPConfig represents per-tenant IdP configuration.
type TenantIdPConfig struct {
	ID         uuid.UUID  `json:"id"`
	TenantID   uuid.UUID  `json:"tenant_id"`
	IdPType    IdPType   `json:"idp_type"`
	Name       string    `json:"name"`
	ConfigJSON string    `json:"config_json"` // protocol-specific config
	Enabled    bool      `json:"enabled"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// Store is the persistence interface for IdP configs.
type Store interface {
	Create(ctx context.Context, config *TenantIdPConfig) error
	GetByID(ctx context.Context, id uuid.UUID) (*TenantIdPConfig, error)
	ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*TenantIdPConfig, error)
	Update(ctx context.Context, config *TenantIdPConfig) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// Service manages per-tenant IdP configurations.
type Service struct {
	store Store
}

// NewService creates a new IdP config service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Create adds a new IdP configuration for a tenant.
func (s *Service) Create(ctx context.Context, tenantID uuid.UUID, idpType IdPType, name, configJSON string) (*TenantIdPConfig, error) {
	if tenantID == uuid.Nil {
		return nil, fmt.Errorf("tenant_id is required")
	}
	if idpType != IdPTypeSAML && idpType != IdPTypeOIDC && idpType != IdPTypeLDAP {
		return nil, fmt.Errorf("invalid idp_type: %s", idpType)
	}
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}

	now := time.Now()
	cfg := &TenantIdPConfig{
		ID:         uuid.New(),
		TenantID:   tenantID,
		IdPType:    idpType,
		Name:       name,
		ConfigJSON: configJSON,
		Enabled:    true,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := s.store.Create(ctx, cfg); err != nil {
		return nil, fmt.Errorf("failed to create IdP config: %w", err)
	}
	return cfg, nil
}

// Get retrieves an IdP config by ID.
func (s *Service) Get(ctx context.Context, id uuid.UUID) (*TenantIdPConfig, error) {
	cfg, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("IdP config not found: %w", err)
	}
	return cfg, nil
}

// List returns all IdP configs for a tenant.
func (s *Service) List(ctx context.Context, tenantID uuid.UUID) ([]*TenantIdPConfig, error) {
	return s.store.ListByTenant(ctx, tenantID)
}

// Update modifies an existing IdP config.
func (s *Service) Update(ctx context.Context, id uuid.UUID, name, configJSON string, enabled bool) (*TenantIdPConfig, error) {
	cfg, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("IdP config not found: %w", err)
	}

	cfg.Name = name
	cfg.ConfigJSON = configJSON
	cfg.Enabled = enabled
	cfg.UpdatedAt = time.Now()

	if err := s.store.Update(ctx, cfg); err != nil {
		return nil, fmt.Errorf("failed to update: %w", err)
	}
	return cfg, nil
}

// Delete removes an IdP config.
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.store.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete: %w", err)
	}
	return nil
}

// --- In-memory store ---

type MemoryStore struct {
	mu      sync.RWMutex
	configs map[uuid.UUID]*TenantIdPConfig
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{configs: make(map[uuid.UUID]*TenantIdPConfig)}
}

func (m *MemoryStore) Create(_ context.Context, c *TenantIdPConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.configs[c.ID] = c
	return nil
}

func (m *MemoryStore) GetByID(_ context.Context, id uuid.UUID) (*TenantIdPConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	c, ok := m.configs[id]
	if !ok {
		return nil, fmt.Errorf("not found: %s", id)
	}
	return c, nil
}

func (m *MemoryStore) ListByTenant(_ context.Context, tenantID uuid.UUID) ([]*TenantIdPConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*TenantIdPConfig
	for _, c := range m.configs {
		if c.TenantID == tenantID {
			result = append(result, c)
		}
	}
	return result, nil
}

func (m *MemoryStore) Update(_ context.Context, c *TenantIdPConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.configs[c.ID] = c
	return nil
}

func (m *MemoryStore) Delete(_ context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.configs, id)
	return nil
}
