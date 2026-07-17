package httpserver

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestAuditMemoryMapRepo_NilPool(t *testing.T) {
	repo := newAuditMemoryMapRepo(nil)

	// storeJSON should not error with nil pool.
	if err := repo.storeJSON(context.Background(), "dashboard_widgets", "w1", map[string]any{"title": "test"}); err != nil {
		t.Errorf("nil pool storeJSON should not error: %v", err)
	}

	// listJSON should return empty.
	items, err := repo.listJSON(context.Background(), "dashboard_widgets")
	if err != nil {
		t.Fatalf("nil pool listJSON should not error: %v", err)
	}
	if len(items) != 0 {
		t.Error("nil pool should return empty list")
	}

	// deleteJSON should not error.
	if err := repo.deleteJSON(context.Background(), "dashboard_widgets", "w1"); err != nil {
		t.Errorf("nil pool deleteJSON should not error: %v", err)
	}
}

func TestAuditMemoryMapRepo_GetBranding_NilPool(t *testing.T) {
	repo := newAuditMemoryMapRepo(nil)
	branding, err := repo.GetBranding(context.Background(), uuid.New())
	_ = err
	if branding == nil {
		t.Fatal("should return default branding")
	}
	if branding["primary_color"] != "#6366f1" {
		t.Errorf("default color should be #6366f1, got %v", branding["primary_color"])
	}
}

func TestAuditMemoryMapRepo_UpsertBranding_NilPool(t *testing.T) {
	repo := newAuditMemoryMapRepo(nil)
	err := repo.UpsertBranding(context.Background(), uuid.New(), map[string]any{
		"primary_color": "#ff0000",
		"logo_url":      "https://example.com/logo.png",
	})
	if err != nil {
		t.Errorf("nil pool UpsertBranding should not error: %v", err)
	}
}

func TestAuditMemoryMapRepo_EnsureSchema_NilPool(t *testing.T) {
	repo := newAuditMemoryMapRepo(nil)
	if err := repo.EnsureSchema(context.Background()); err != nil {
		t.Errorf("nil pool EnsureSchema should not error: %v", err)
	}
}
