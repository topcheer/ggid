package handler

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestTenantIDFromContext_NoTenant(t *testing.T) {
	ctx := context.Background()
	id := tenantIDFromContext(ctx)
	if id != uuid.Nil {
		t.Errorf("expected uuid.Nil, got %v", id)
	}
}

func TestTenantIDFromContext_WithTenant(t *testing.T) {
	tenantID := uuid.New()
	ctx := WithTenantContext(context.Background(), tenantID)
	id := tenantIDFromContext(ctx)
	if id != tenantID {
		t.Errorf("expected %v, got %v", tenantID, id)
	}
}

func TestWithTenantContext_ReturnsNewContext(t *testing.T) {
	ctx1 := context.Background()
	tenantID := uuid.New()
	ctx2 := WithTenantContext(ctx1, tenantID)
	if ctx1 == ctx2 {
		t.Error("WithTenantContext should return a new context")
	}
}
