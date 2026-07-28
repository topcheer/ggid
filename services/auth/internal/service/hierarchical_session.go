package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ggid/ggid/pkg/hierarchy"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SessionConfig holds session timeout settings for hierarchical config.
type SessionConfig struct {
	IdleTimeoutSeconds     int `json:"idle_timeout_seconds"`     // 0 = no idle timeout
	AbsoluteTimeoutSeconds int `json:"absolute_timeout_seconds"` // 0 = no absolute timeout
	MaxConcurrentSessions  int `json:"max_concurrent_sessions"`  // 0 = unlimited
}

// TokenConfig holds token expiry settings for hierarchical config.
type TokenConfig struct {
	AccessTokenTTLSeconds  int `json:"access_token_ttl_seconds"`
	RefreshTokenTTLSeconds int `json:"refresh_token_ttl_seconds"`
}

// GetSessionConfig reads session config from hierarchical config.
func GetSessionConfig(ctx context.Context, pool *pgxpool.Pool, tenantID *uuid.UUID, clientID *string, def SessionConfig) SessionConfig {
	if pool == nil {
		return def
	}
	resolved, err := hierarchy.GetConfig(ctx, pool, "session_config", tenantID, clientID, nil)
	if err != nil || resolved == nil {
		return def
	}
	var cfg SessionConfig
	if err := json.Unmarshal(resolved.Config, &cfg); err != nil {
		return def
	}
	if cfg.IdleTimeoutSeconds == 0 {
		cfg.IdleTimeoutSeconds = def.IdleTimeoutSeconds
	}
	if cfg.AbsoluteTimeoutSeconds == 0 {
		cfg.AbsoluteTimeoutSeconds = def.AbsoluteTimeoutSeconds
	}
	if cfg.MaxConcurrentSessions == 0 {
		cfg.MaxConcurrentSessions = def.MaxConcurrentSessions
	}
	return cfg
}

// GetTokenConfig reads token config from hierarchical config.
func GetTokenConfig(ctx context.Context, pool *pgxpool.Pool, tenantID *uuid.UUID, clientID *string, def TokenConfig) TokenConfig {
	if pool == nil {
		return def
	}
	resolved, err := hierarchy.GetConfig(ctx, pool, "token_config", tenantID, clientID, nil)
	if err != nil || resolved == nil {
		return def
	}
	var cfg TokenConfig
	if err := json.Unmarshal(resolved.Config, &cfg); err != nil {
		return def
	}
	if cfg.AccessTokenTTLSeconds == 0 {
		cfg.AccessTokenTTLSeconds = def.AccessTokenTTLSeconds
	}
	if cfg.RefreshTokenTTLSeconds == 0 {
		cfg.RefreshTokenTTLSeconds = def.RefreshTokenTTLSeconds
	}
	return cfg
}

// SetGenericConfig saves any config type at the specified scope.
// Reused by session, token, and future hierarchical config items.
func SetGenericConfig(ctx context.Context, pool *pgxpool.Pool, key string, scope hierarchy.ScopeType, tenantID *uuid.UUID, clientID *string, configJSON json.RawMessage) error {
	if pool == nil {
		return fmt.Errorf("database not configured")
	}
	pc := &hierarchy.ProviderConfig{
		ConfigKey:    key,
		ScopeType:    scope,
		TenantID:     tenantID,
		ClientID:     clientID,
		ProviderType: key, // reuse key as provider_type for non-provider configs
		Config:       configJSON,
		Enabled:      true,
	}
	return hierarchy.SetConfig(ctx, pool, pc)
}
