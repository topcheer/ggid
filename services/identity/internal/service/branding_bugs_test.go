package service

import (
	"context"
	"testing"

	"github.com/ggid/ggid/services/identity/internal/domain"
	"github.com/google/uuid"
)

// BUG 1: EnsureBrandingTable doesn't handle errors properly
func TestEnsureBrandingTable_ErrorHandling_Bug(t *testing.T) {
	// Create a branding store with nil DB
	bs := NewBrandingStore(nil)

	// Ensure table with nil DB - should return nil (no-op)
	err := bs.EnsureBrandingTable(context.Background())
	if err != nil {
		t.Fatalf("EnsureBrandingTable with nil DB should not error: %v", err)
	}

	// BUG: The method silently does nothing when db is nil
	// It doesn't log a warning or return an error indicating the DB is missing
	// This could lead to branding not being set up in production
}

// BUG 2: GetBranding returns default branding on DB error
func TestGetBranding_DBErrorHandling_Bug(t *testing.T) {
	// Create a branding store with nil DB (simulating DB error)
	bs := NewBrandingStore(nil)
	tenantID := uuid.New().String()

	// Get branding with nil DB
	branding, err := bs.GetBranding(context.Background(), tenantID)

	if err != nil {
		t.Fatalf("GetBranding failed: %v", err)
	}

	// BUG: On DB error (nil db), returns default branding
	// This hides DB errors and could lead to incorrect branding being displayed
	if branding == nil {
		t.Fatal("Expected branding to be returned")
	}

	// The returned branding is default, not from DB
	if branding.TenantID != tenantID {
		t.Errorf("Expected tenant ID %s, got %s", tenantID, branding.TenantID)
	}

	t.Log("BUG WARNING: GetBranding returns default branding on DB error instead of error")
}

// BUG 3: UpdateBranding with nil DB updates in-memory only
func TestUpdateBranding_InMemoryFallback_Bug(t *testing.T) {
	bs := NewBrandingStore(nil)
	tenantID := uuid.New().String()

	req := &domain.TenantBranding{
		LogoURL:      "https://example.com/logo.png",
		PrimaryColor: "#ff0000",
	}

	// Update branding with nil DB
	updated, err := bs.UpdateBranding(context.Background(), tenantID, req)
	if err != nil {
		t.Fatalf("UpdateBranding failed: %v", err)
	}

	// BUG: Update succeeds with nil DB (in-memory)
	// In production with DB connectivity issues, this would silently fail
	// The caller thinks branding is updated, but it's only in memory

	if updated.LogoURL != "https://example.com/logo.png" {
		t.Errorf("Expected logo URL to be updated, got %s", updated.LogoURL)
	}

	// Try to get the branding back
	retrieved, err := bs.GetBranding(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("GetBranding failed: %v", err)
	}

	// In-memory update is reflected (good for tests)
	if retrieved.LogoURL != "https://example.com/logo.png" {
		t.Error("In-memory update should be retrievable")
	}

	t.Log("BUG WARNING: UpdateBranding with nil DB uses in-memory fallback, hiding DB errors")
}

// BUG 4: Multiple tenant branding operations with nil DB share state
func TestBranding_InMemoryState_Bug(t *testing.T) {
	bs := NewBrandingStore(nil)

	tenant1 := uuid.New().String()
	tenant2 := uuid.New().String()

	// Update branding for tenant1
	req1 := &domain.TenantBranding{
		LogoURL: "https://tenant1.com/logo.png",
	}
	_, _ = bs.UpdateBranding(context.Background(), tenant1, req1)

	// Update branding for tenant2
	req2 := &domain.TenantBranding{
		LogoURL: "https://tenant2.com/logo.png",
	}
	_, _ = bs.UpdateBranding(context.Background(), tenant2, req2)

	// Get branding for both tenants
	branding1, _ := bs.GetBranding(context.Background(), tenant1)
	branding2, _ := bs.GetBranding(context.Background(), tenant2)

	// BUG: In-memory state is not isolated per tenant in a real sense
	// It's just a map, which is fine, but the fallback mechanism hides issues

	if branding1.LogoURL != "https://tenant1.com/logo.png" {
		t.Errorf("Tenant1 logo mismatch: %s", branding1.LogoURL)
	}

	if branding2.LogoURL != "https://tenant2.com/logo.png" {
		t.Errorf("Tenant2 logo mismatch: %s", branding2.LogoURL)
	}

	// This works, but the issue is that in production with DB issues,
	// all tenants would be affected and the in-memory map would be shared
}

// BUG 5: GetBranding doesn't validate tenant ID format
func TestGetBranding_InvalidTenantID_Bug(t *testing.T) {
	bs := NewBrandingStore(nil)

	// Try with empty tenant ID
	branding, err := bs.GetBranding(context.Background(), "")
	if err != nil {
		t.Fatalf("GetBranding failed: %v", err)
	}

	if branding == nil {
		t.Fatal("Expected branding to be returned")
	}

	// BUG: Empty or invalid tenant IDs are accepted
	// Should validate the tenant ID format
	t.Log("BUG WARNING: Empty tenant ID accepted without validation")
}

// BUG 6: UpdateBranding doesn't validate input
func TestUpdateBranding_NoValidation_Bug(t *testing.T) {
	bs := NewBrandingStore(nil)
	tenantID := uuid.New().String()

	// Update with invalid color
	req := &domain.TenantBranding{
		PrimaryColor: "not-a-color", // Invalid color format
		LogoURL:      "not-a-url",   // Invalid URL
	}

	updated, err := bs.UpdateBranding(context.Background(), tenantID, req)
	if err != nil {
		t.Fatalf("UpdateBranding failed: %v", err)
	}

	// BUG: Invalid values are accepted without validation
	if updated.PrimaryColor != "not-a-color" {
		t.Error("Invalid color was accepted")
	}

	if updated.LogoURL != "not-a-url" {
		t.Error("Invalid URL was accepted")
	}

	t.Log("BUG WARNING: No input validation in UpdateBranding")
}

// BUG 7: GetBranding with non-existent tenant returns default
func TestGetBranding_NonExistentTenant_Bug(t *testing.T) {
	bs := NewBrandingStore(nil)
	nonExistentTenant := uuid.New().String()

	branding, err := bs.GetBranding(context.Background(), nonExistentTenant)
	if err != nil {
		t.Fatalf("GetBranding failed: %v", err)
	}

	// BUG: Returns default branding instead of "not found" error
	// This makes it impossible to distinguish between:
	// 1. Tenant exists with default branding
	// 2. Tenant doesn't exist

	if branding == nil {
		t.Fatal("Expected branding to be returned")
	}

	t.Log("BUG WARNING: Non-existent tenant returns default branding instead of error")
}

// BUG 8: EnsureBrandingTable doesn't check if table already has correct schema
func TestEnsureBrandingTable_SchemaCheck_Bug(t *testing.T) {
	bs := NewBrandingStore(nil)

	// Call EnsureBrandingTable multiple times
	err1 := bs.EnsureBrandingTable(context.Background())
	err2 := bs.EnsureBrandingTable(context.Background())

	if err1 != nil || err2 != nil {
		t.Fatalf("EnsureBrandingTable failed: %v, %v", err1, err2)
	}

	// BUG: With nil DB, we can't test this properly
	// But in real DB, the CREATE TABLE IF NOT EXISTS doesn't check schema
	// If the table exists but has wrong schema, it won't be updated

	t.Log("BUG WARNING: EnsureBrandingTable uses IF NOT EXISTS, doesn't verify schema")
}

// BUG 9: Branding operations are not atomic
func TestBranding_Concurrency_Bug(t *testing.T) {
	bs := NewBrandingStore(nil)
	tenantID := uuid.New().String()

	// Simulate concurrent updates (not really concurrent in this test)
	req1 := &domain.TenantBranding{LogoURL: "url1"}
	req2 := &domain.TenantBranding{LogoURL: "url2"}
	req3 := &domain.TenantBranding{LogoURL: "url3"}

	_, _ = bs.UpdateBranding(context.Background(), tenantID, req1)
	_, _ = bs.UpdateBranding(context.Background(), tenantID, req2)
	_, _ = bs.UpdateBranding(context.Background(), tenantID, req3)

	// Get final branding
	branding, _ := bs.GetBranding(context.Background(), tenantID)

	// BUG: Without transactions, concurrent updates could overwrite each other
	// The last write wins (which is expected), but there's no version checking

	if branding.LogoURL != "url3" {
		t.Errorf("Expected url3, got %s", branding.LogoURL)
	}

	t.Log("BUG WARNING: No optimistic concurrency control - last write wins")
}

// BUG 10: GetBranding doesn't cache results
func TestBranding_NoCaching_Bug(t *testing.T) {
	bs := NewBrandingStore(nil)
	tenantID := uuid.New().String()

	// Get branding multiple times
	branding1, _ := bs.GetBranding(context.Background(), tenantID)
	branding2, _ := bs.GetBranding(context.Background(), tenantID)
	branding3, _ := bs.GetBranding(context.Background(), tenantID)

	// BUG: Each call would hit the DB (if it existed)
	// No caching layer to reduce DB load

	if branding1 != branding2 || branding2 != branding3 {
		t.Log("Different instances returned (no caching)")
	}

	t.Log("BUG WARNING: No caching for frequently accessed branding data")
}
