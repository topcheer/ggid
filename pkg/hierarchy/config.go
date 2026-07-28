// Package hierarchy provides three-level (App → Tenant → Instance) configuration
// resolution with automatic fallback. Each config key is looked up at the most
// specific scope first; if not found or disabled, it falls back to the next
// broader scope, and finally to a code-provided default.
package hierarchy

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ScopeType identifies the configuration scope level.
type ScopeType string

const (
	ScopeInstance ScopeType = "instance"
	ScopeTenant   ScopeType = "tenant"
	ScopeApp      ScopeType = "app"
)

// ProviderConfig is a row in the provider_configs table.
type ProviderConfig struct {
	ID           uuid.UUID       `json:"id"`
	ConfigKey    string          `json:"config_key"`
	ScopeType    ScopeType       `json:"scope_type"`
	TenantID     *uuid.UUID      `json:"tenant_id,omitempty"`
	ClientID     *string         `json:"client_id,omitempty"`
	ProviderType string          `json:"provider_type"`
	Config       json.RawMessage `json:"config"`
	Enabled      bool            `json:"enabled"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// ResolvedConfig is the result of a hierarchical lookup. It includes the
// source scope so callers can know which level the value came from.
type ResolvedConfig struct {
	ProviderConfig
	Source string `json:"source"` // "app", "tenant", "instance", "default"
}

// GetConfig performs a hierarchical lookup: app → tenant → instance → default.
// Returns the first enabled configuration found, or the provided default if none.
func GetConfig(ctx context.Context, pool *pgxpool.Pool, key string, tenantID *uuid.UUID, clientID *string, defaultConfig *ProviderConfig) (*ResolvedConfig, error) {
	// 1. Try app-level
	if tenantID != nil && clientID != nil {
		cfg, err := queryConfig(ctx, pool, key, ScopeApp, tenantID, clientID)
		if err == nil && cfg != nil && cfg.Enabled {
			return &ResolvedConfig{*cfg, "app"}, nil
		}
	}

	// 2. Try tenant-level
	if tenantID != nil {
		cfg, err := queryConfig(ctx, pool, key, ScopeTenant, tenantID, nil)
		if err == nil && cfg != nil && cfg.Enabled {
			return &ResolvedConfig{*cfg, "tenant"}, nil
		}
	}

	// 3. Try instance-level
	cfg, err := queryConfig(ctx, pool, key, ScopeInstance, nil, nil)
	if err == nil && cfg != nil && cfg.Enabled {
		return &ResolvedConfig{*cfg, "instance"}, nil
	}

	// 4. Code default
	if defaultConfig != nil {
		return &ResolvedConfig{*defaultConfig, "default"}, nil
	}
	return nil, fmt.Errorf("no config found for key %q at any scope", key)
}

// IsAvailable checks if a provider config exists and is enabled at any scope.
func IsAvailable(ctx context.Context, pool *pgxpool.Pool, key string, tenantID *uuid.UUID, clientID *string) bool {
	_, err := GetConfig(ctx, pool, key, tenantID, clientID, nil)
	return err == nil
}

// SetConfig inserts or updates a provider config at the specified scope.
func SetConfig(ctx context.Context, pool *pgxpool.Pool, cfg *ProviderConfig) error {
	var tenantVal interface{}
	if cfg.TenantID != nil {
		tenantVal = *cfg.TenantID
	}
	var clientVal interface{}
	if cfg.ClientID != nil {
		clientVal = *cfg.ClientID
	}

	_, err := pool.Exec(ctx, `
		INSERT INTO provider_configs (config_key, scope_type, tenant_id, client_id, provider_type, config, enabled)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (config_key, scope_type, COALESCE(tenant_id, '00000000-0000-0000-0000-000000000000'), COALESCE(client_id, ''))
		DO UPDATE SET provider_type = EXCLUDED.provider_type,
			config = EXCLUDED.config,
			enabled = EXCLUDED.enabled,
			updated_at = NOW()
	`, cfg.ConfigKey, cfg.ScopeType, tenantVal, clientVal, cfg.ProviderType, cfg.Config, cfg.Enabled)
	if err != nil {
		return fmt.Errorf("upsert provider config: %w", err)
	}
	return nil
}

// queryConfig queries a single scope level for a config key.
func queryConfig(ctx context.Context, pool *pgxpool.Pool, key string, scope ScopeType, tenantID *uuid.UUID, clientID *string) (*ProviderConfig, error) {
	var cfg ProviderConfig
	var tenantVal interface{}
	if tenantID != nil {
		tenantVal = *tenantID
	}
	var clientVal interface{}
	if clientID != nil {
		clientVal = *clientID
	}

	query := `
		SELECT id, config_key, scope_type, tenant_id, client_id, provider_type, config, enabled, created_at, updated_at
		FROM provider_configs
		WHERE config_key = $1 AND scope_type = $2`
	args := []interface{}{key, scope}

	if tenantID != nil {
		query += ` AND tenant_id = $3`
		args = append(args, tenantVal)
	} else if scope == ScopeInstance {
		query += ` AND tenant_id IS NULL`
	}

	if clientID != nil {
		placeholder := "$3"
		if tenantID != nil {
			placeholder = "$4"
		}
		query += fmt.Sprintf(` AND client_id = %s`, placeholder)
		args = append(args, clientVal)
	} else if scope != ScopeInstance {
		query += ` AND client_id IS NULL`
	}

	query += ` LIMIT 1`

	err := pool.QueryRow(ctx, query, args...).Scan(
		&cfg.ID, &cfg.ConfigKey, &cfg.ScopeType, &cfg.TenantID, &cfg.ClientID,
		&cfg.ProviderType, &cfg.Config, &cfg.Enabled, &cfg.CreatedAt, &cfg.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

// ConfigKey constants for well-known provider types.
const (
	KeySMSProvider   = "sms_provider"
	KeyEmailProvider = "email_provider"
)
