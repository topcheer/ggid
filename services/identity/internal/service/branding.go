package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ggid/ggid/services/identity/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

// BrandingStore is a DB-backed branding store with in-memory fallback for tests.
type BrandingStore struct {
	db       *pgxpool.Pool
	inMemory map[string]*domain.TenantBranding // fallback when db is nil
}

// NewBrandingStore creates a new BrandingStore.
func NewBrandingStore(db *pgxpool.Pool) *BrandingStore {
	return &BrandingStore{db: db, inMemory: make(map[string]*domain.TenantBranding)}
}

// GetBranding retrieves branding config for a tenant from DB.
func (bs *BrandingStore) GetBranding(ctx context.Context, tenantID string) (*domain.TenantBranding, error) {
	if bs.db == nil {
		if b, ok := bs.inMemory[tenantID]; ok {
			return b, nil
		}
		return domain.DefaultBranding(tenantID), nil
	}

	var b domain.TenantBranding
	var settingsJSON []byte
	err := bs.db.QueryRow(ctx,
		`SELECT tenant_id, COALESCE(settings->>'logo_url',''), COALESCE(settings->>'favicon_url',''),
		        COALESCE(settings->>'primary_color','#2563eb'), COALESCE(settings->>'accent_color','#1e40af'),
		        COALESCE(settings->>'secondary_color','#1e40af'), COALESCE(settings->>'font_family','Inter'),
		        COALESCE((settings->>'border_radius')::int, 8), COALESCE(settings->>'default_mode','light'),
		        COALESCE(settings->>'email_template','default'), COALESCE(settings->>'custom_domain','')
		 FROM tenant_branding WHERE tenant_id = $1`, tenantID).Scan(
		&b.TenantID, &b.LogoURL, &b.FaviconURL, &b.PrimaryColor, &b.AccentColor,
		&b.SecondaryColor, &b.FontFamily, &b.BorderRadius, &b.DefaultMode,
		&b.EmailTemplate, &b.CustomDomain)
	if err != nil {
		return domain.DefaultBranding(tenantID), nil
	}
	_ = settingsJSON
	return &b, nil
}

// UpdateBranding creates or updates branding config for a tenant in DB.
func (bs *BrandingStore) UpdateBranding(ctx context.Context, tenantID string, req *domain.TenantBranding) (*domain.TenantBranding, error) {
	req.TenantID = tenantID
	req.UpdatedAt = time.Now()

	if bs.db == nil {
		bs.inMemory[tenantID] = req
		return req, nil // in-memory fallback
	}

	settings, _ := json.Marshal(map[string]any{
		"logo_url":        req.LogoURL,
		"favicon_url":     req.FaviconURL,
		"primary_color":   req.PrimaryColor,
		"accent_color":    req.AccentColor,
		"secondary_color": req.SecondaryColor,
		"font_family":     req.FontFamily,
		"border_radius":   req.BorderRadius,
		"default_mode":    req.DefaultMode,
		"email_template":  req.EmailTemplate,
		"custom_domain":   req.CustomDomain,
	})

	_, err := bs.db.Exec(ctx,
		`INSERT INTO tenant_branding (tenant_id, settings, updated_at) VALUES ($1, $2, $3)
		 ON CONFLICT (tenant_id) DO UPDATE SET settings = $2, updated_at = $3`,
		tenantID, settings, req.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("update branding: %w", err)
	}
	return req, nil
}

// EnsureBrandingTable creates the tenant_branding table if it doesn't exist.
func (bs *BrandingStore) EnsureBrandingTable(ctx context.Context) error {
	if bs.db == nil {
		return nil
	}
	_, err := bs.db.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS tenant_branding (
			tenant_id   UUID NOT NULL PRIMARY KEY,
			settings    JSONB NOT NULL DEFAULT '{}'::jsonb,
			updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`)
	return err
}
