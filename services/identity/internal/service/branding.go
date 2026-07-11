package service

import (
	"context"
	"sync"
	"time"

	"github.com/ggid/ggid/services/identity/internal/domain"
)

// BrandingStore is an in-memory branding store (replace with DB-backed repo in production).
// This provides the CRUD interface for per-tenant branding configuration.
type BrandingStore struct {
	mu       sync.RWMutex
	branding map[string]*domain.TenantBranding // key: tenant_id
}

// NewBrandingStore creates a new BrandingStore.
func NewBrandingStore() *BrandingStore {
	return &BrandingStore{branding: make(map[string]*domain.TenantBranding)}
}

// GetBranding retrieves branding config for a tenant.
func (bs *BrandingStore) GetBranding(ctx context.Context, tenantID string) (*domain.TenantBranding, error) {
	bs.mu.RLock()
	defer bs.mu.RUnlock()
	if b, ok := bs.branding[tenantID]; ok {
		return b, nil
	}
	// Return default branding
	return &domain.TenantBranding{
		TenantID:       tenantID,
		LogoURL:        "",
		PrimaryColor:   "#2563eb",
		SecondaryColor: "#1e40af",
		CustomDomain:   "",
		EmailTemplate:  "default",
	}, nil
}

// UpdateBranding creates or updates branding config for a tenant.
func (bs *BrandingStore) UpdateBranding(ctx context.Context, tenantID string, req *domain.TenantBranding) (*domain.TenantBranding, error) {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	req.TenantID = tenantID
	req.UpdatedAt = time.Now()
	bs.branding[tenantID] = req
	return req, nil
}
