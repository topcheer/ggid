package handler

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestSvc2_WithTenantContextRoundTrip(t *testing.T) {
	tenantID := uuid.New()
	ctx := WithTenantContext(context.Background(), tenantID)
	extracted := tenantIDFromContext(ctx)
	if extracted != tenantID {
		t.Errorf("expected %v, got %v", tenantID, extracted)
	}
}

func TestSvc2_WithTenantContext_Nil(t *testing.T) {
	ctx := context.Background()
	id := tenantIDFromContext(ctx)
	if id != uuid.Nil {
		t.Errorf("expected uuid.Nil, got %v", id)
	}
}

func TestSvc2_WithTenantContext_NewContext(t *testing.T) {
	ctx1 := context.Background()
	tenantID := uuid.New()
	ctx2 := WithTenantContext(ctx1, tenantID)
	if ctx1 == ctx2 {
		t.Error("should return new context")
	}
	// Original should still be nil
	if tenantIDFromContext(ctx1) != uuid.Nil {
		t.Error("original context should be unaffected")
	}
	// New should have tenant
	if tenantIDFromContext(ctx2) != tenantID {
		t.Error("new context should have tenant")
	}
}
