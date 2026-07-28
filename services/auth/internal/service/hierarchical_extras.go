package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ggid/ggid/pkg/hierarchy"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CORSConfig holds per-scope CORS settings.
type CORSConfig struct {
	AllowedOrigins   []string `json:"allowed_origins"`
	AllowedMethods   []string `json:"allowed_methods,omitempty"`
	AllowedHeaders   []string `json:"allowed_headers,omitempty"`
	AllowCredentials bool     `json:"allow_credentials,omitempty"`
	MaxAgeSeconds    int      `json:"max_age_seconds,omitempty"`
}

// MFAEnforcementConfig holds per-scope MFA enforcement rules.
type MFAEnforcementConfig struct {
	Required       bool     `json:"required"`
	AllowedMethods []string `json:"allowed_methods,omitempty"` // totp, sms, email, passkey
	RememberDays   int      `json:"remember_days,omitempty"`
}

// BrandingConfig holds UI branding for per-scope customization.
type BrandingConfig struct {
	LogoURL       string `json:"logo_url,omitempty"`
	PrimaryColor  string `json:"primary_color,omitempty"`
	SecondaryColor string `json:"secondary_color,omitempty"`
	FontFamily    string `json:"font_family,omitempty"`
	DefaultMode   string `json:"default_mode,omitempty"` // light/dark
	CustomDomain  string `json:"custom_domain,omitempty"`
}

// AuditRetentionConfig holds retention policy per scope.
type AuditRetentionConfig struct {
	RetentionDays int    `json:"retention_days"`
	Action        string `json:"action"` // "delete" or "anonymize"
}

// GetCORSConfig reads CORS config from hierarchical config.
func GetCORSConfig(ctx context.Context, pool *pgxpool.Pool, tenantID *uuid.UUID, clientID *string, def CORSConfig) CORSConfig {
	return getHierarchical(ctx, pool, "cors_config", tenantID, clientID, def)
}

// GetMFAEnforcement reads MFA enforcement from hierarchical config.
func GetMFAEnforcement(ctx context.Context, pool *pgxpool.Pool, tenantID *uuid.UUID, clientID *string, def MFAEnforcementConfig) MFAEnforcementConfig {
	return getHierarchical(ctx, pool, "mfa_enforcement", tenantID, clientID, def)
}

// GetBranding reads branding from hierarchical config with instance-level fallback.
func GetBranding(ctx context.Context, pool *pgxpool.Pool, tenantID *uuid.UUID, clientID *string, def BrandingConfig) BrandingConfig {
	return getHierarchical(ctx, pool, "branding", tenantID, clientID, def)
}

// GetAuditRetention reads audit retention from hierarchical config.
func GetAuditRetention(ctx context.Context, pool *pgxpool.Pool, tenantID *uuid.UUID, def AuditRetentionConfig) AuditRetentionConfig {
	return getHierarchical(ctx, pool, "audit_retention", tenantID, nil, def)
}

// SetCORSConfig saves CORS config at the specified scope.
func SetCORSConfig(ctx context.Context, pool *pgxpool.Pool, scope hierarchy.ScopeType, tenantID *uuid.UUID, clientID *string, cfg CORSConfig) error {
	return marshalAndSet(ctx, pool, "cors_config", scope, tenantID, clientID, cfg)
}

// SetMFAEnforcement saves MFA enforcement config.
func SetMFAEnforcement(ctx context.Context, pool *pgxpool.Pool, scope hierarchy.ScopeType, tenantID *uuid.UUID, clientID *string, cfg MFAEnforcementConfig) error {
	return marshalAndSet(ctx, pool, "mfa_enforcement", scope, tenantID, clientID, cfg)
}

// SetBranding saves branding config.
func SetBranding(ctx context.Context, pool *pgxpool.Pool, scope hierarchy.ScopeType, tenantID *uuid.UUID, clientID *string, cfg BrandingConfig) error {
	return marshalAndSet(ctx, pool, "branding", scope, tenantID, clientID, cfg)
}

// SetAuditRetention saves audit retention config.
func SetAuditRetention(ctx context.Context, pool *pgxpool.Pool, scope hierarchy.ScopeType, tenantID *uuid.UUID, cfg AuditRetentionConfig) error {
	return marshalAndSet(ctx, pool, "audit_retention", scope, tenantID, nil, cfg)
}

// getHierarchical is a generic helper that resolves config via hierarchy.
func getHierarchical[T any](ctx context.Context, pool *pgxpool.Pool, key string, tenantID *uuid.UUID, clientID *string, def T) T {
	if pool == nil {
		return def
	}
	resolved, err := hierarchy.GetConfig(ctx, pool, key, tenantID, clientID, nil)
	if err != nil || resolved == nil {
		return def
	}
	var cfg T
	if err := json.Unmarshal(resolved.Config, &cfg); err != nil {
		return def
	}
	return cfg
}

// marshalAndSet serializes config and stores via SetGenericConfig.
func marshalAndSet(ctx context.Context, pool *pgxpool.Pool, key string, scope hierarchy.ScopeType, tenantID *uuid.UUID, clientID *string, cfg any) error {
	configJSON, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	return SetGenericConfig(ctx, pool, key, scope, tenantID, clientID, configJSON)
}
