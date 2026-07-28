package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ggid/ggid/pkg/hierarchy"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PasswordPolicyConfig is the JSON shape stored in provider_configs.
type PasswordPolicyConfig struct {
	MinLength      int      `json:"min_length"`
	MaxLength      int      `json:"max_length,omitempty"`
	RequireUpper   bool     `json:"require_upper"`
	RequireLower   bool     `json:"require_lower"`
	RequireDigit   bool     `json:"require_digit"`
	RequireSpecial bool     `json:"require_special"`
	HistoryCount   int      `json:"history_count,omitempty"`
	Blacklist      []string `json:"blacklist,omitempty"`
}

// GetPasswordPolicyHierarchical reads password policy from the hierarchical
// config system (app → tenant → instance → default). Falls back to the
// provided defaultConfig if no DB config exists.
func GetPasswordPolicyHierarchical(ctx context.Context, pool *pgxpool.Pool, tenantID *uuid.UUID, clientID *string, defaultConfig PasswordPolicyConfig) PasswordPolicyConfig {
	if pool == nil {
		return defaultConfig
	}

	resolved, err := hierarchy.GetConfig(ctx, pool, "password_policy", tenantID, clientID, nil)
	if err != nil || resolved == nil {
		return defaultConfig
	}

	var cfg PasswordPolicyConfig
	if err := json.Unmarshal(resolved.Config, &cfg); err != nil {
		return defaultConfig
	}

	// Fill zero values with defaults
	if cfg.MinLength == 0 {
		cfg.MinLength = defaultConfig.MinLength
	}
	if cfg.MaxLength == 0 {
		cfg.MaxLength = defaultConfig.MaxLength
	}

	return cfg
}

// SetPasswordPolicyHierarchical saves password policy at the specified scope.
func SetPasswordPolicyHierarchical(ctx context.Context, pool *pgxpool.Pool, scope hierarchy.ScopeType, tenantID *uuid.UUID, clientID *string, cfg PasswordPolicyConfig) error {
	if pool == nil {
		return fmt.Errorf("database not configured")
	}

	configJSON, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal password policy: %w", err)
	}

	pc := &hierarchy.ProviderConfig{
		ConfigKey:    "password_policy",
		ScopeType:    scope,
		TenantID:     tenantID,
		ClientID:     clientID,
		ProviderType: "password_policy",
		Config:       configJSON,
		Enabled:      true,
	}

	return hierarchy.SetConfig(ctx, pool, pc)
}
