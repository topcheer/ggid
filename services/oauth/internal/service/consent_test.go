package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestPGConsentStore_NilPool(t *testing.T) {
	store := NewPGConsentStore(nil)
	ctx := context.Background()
	tenantID := uuid.New()
	userID := uuid.New()

	// Get should return nil (not found) with nil pool.
	rec, err := store.Get(ctx, tenantID, userID, "client-1")
	if err != nil {
		t.Fatalf("nil pool Get should not error: %v", err)
	}
	if rec != nil {
		t.Error("nil pool should return nil record")
	}

	// Save should be no-op with nil pool.
	cr := &ConsentRecord{
		TenantID: tenantID, UserID: userID, ClientID: "client-1",
		Scopes: []string{"read:profile"},
	}
	if err := store.Save(ctx, cr); err != nil {
		t.Errorf("nil pool Save should not error: %v", err)
	}

	// Delete should be no-op with nil pool.
	if err := store.Delete(ctx, tenantID, userID, "client-1"); err != nil {
		t.Errorf("nil pool Delete should not error: %v", err)
	}
}

func TestPGConsentStore_EnsureSchemaNilPool(t *testing.T) {
	store := NewPGConsentStore(nil)
	if err := store.(*pgConsentStore).EnsureSchema(context.Background()); err != nil {
		t.Errorf("nil pool EnsureSchema should not error: %v", err)
	}
}

func TestConsentKey(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	key := consentKey(tenantID, userID, "client-1")
	if key == "" {
		t.Error("consent key should not be empty")
	}
	// Keys should be unique per client.
	key2 := consentKey(tenantID, userID, "client-2")
	if key == key2 {
		t.Error("keys should differ for different clients")
	}
}

func TestConsentRecord_Fields(t *testing.T) {
	now := time.Now().UTC()
	cr := &ConsentRecord{
		ID:        uuid.New(),
		TenantID:  uuid.New(),
		UserID:    uuid.New(),
		ClientID:  "test-client",
		Scopes:    []string{"openid", "profile", "email"},
		GrantedAt: now,
		ExpiresAt: now.Add(24 * time.Hour),
	}
	if cr.ClientID != "test-client" {
		t.Error("client ID mismatch")
	}
	if len(cr.Scopes) != 3 {
		t.Error("scope count mismatch")
	}
	if !cr.ExpiresAt.After(cr.GrantedAt) {
		t.Error("expiry should be after grant")
	}
}
