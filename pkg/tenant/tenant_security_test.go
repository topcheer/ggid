package tenant

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

// TestFromContext_NilContextReturnsError tests what happens when
// a nil context is passed to FromContext.
func TestFromContext_NilContextReturnsError(t *testing.T) {
	// BUG: FromContext at line 34-39 returns an error if tc is nil
	// but doesn't handle nil context itself
	tc, err := FromContext(nil)
	if err == nil {
		t.Error("BUG: FromContext(nil) should return an error")
	}
	if tc != nil {
		t.Error("BUG: FromContext(nil) should return nil tenant context")
	}
}

// TestFromContext_EmptyContextReturnsError tests what happens when
// a context without tenant info is passed to FromContext.
func TestFromContext_EmptyContextReturnsError(t *testing.T) {
	ctx := context.Background()
	tc, err := FromContext(ctx)
	if err == nil {
		t.Error("BUG: FromContext(empty context) should return an error")
	}
	if tc != nil {
		t.Error("BUG: FromContext(empty context) should return nil tenant context")
	}
}

// TestFromContext_NilTenantInContextReturnsError tests what happens when
// a context has a nil tenant context value.
func TestFromContext_NilTenantInContextReturnsError(t *testing.T) {
	// Create a context with a nil tenant value
	ctx := context.WithValue(context.Background(), contextKey{}, nil)

	tc, err := FromContext(ctx)
	if err == nil {
		t.Error("BUG: FromContext with nil tenant value should return an error")
	}
	if tc != nil {
		t.Error("BUG: FromContext with nil tenant value should return nil")
	}
}

// TestWithContext_ProperlyPropagated tests that WithContext
// properly propagates the tenant context through the context chain.
func TestWithContext_ProperlyPropagated(t *testing.T) {
	tenantID := uuid.New()
	originalTC := &Context{
		TenantID: tenantID,
	}

	// Create context with tenant info
	ctx := WithContext(context.Background(), originalTC)

	// Extract it back
	extractedTC, err := FromContext(ctx)
	if err != nil {
		t.Fatalf("Failed to extract tenant context: %v", err)
	}

	if extractedTC.TenantID != tenantID {
		t.Error("BUG: Tenant ID was not properly propagated")
	}

	// Create a derived context (common middleware pattern)
	derivedCtx := context.WithValue(ctx, "other_key", "other_value")

	// Extract from derived context
	derivedTC, err := FromContext(derivedCtx)
	if err != nil {
		t.Fatalf("Failed to extract from derived context: %v", err)
	}

	if derivedTC.TenantID != tenantID {
		t.Error("BUG: Tenant context was lost in derived context")
	}
}

// TestMustFromContext_PanicsWithoutTenant tests that MustFromContext
// panics when no tenant context is present.
func TestMustFromContext_PanicsWithoutTenant(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("BUG: MustFromContext should panic when no tenant context")
		}
	}()

	ctx := context.Background()
	_ = MustFromContext(ctx)
}

// TestWithNilContext tests what happens with a nil parent context.
func TestWithNilContext(t *testing.T) {
	tenantID := uuid.New()
	tc := &Context{
		TenantID: tenantID,
	}

	// WithContext calls context.WithValue which panics with nil parent
	// This is expected Go behavior, not a bug
	defer func() {
		if r := recover(); r != nil {
			t.Logf("Panicked as expected (Go stdlib behavior): %v", r)
		}
	}()

	_ = WithContext(nil, tc)

	// If we get here, it's unexpected
	t.Error("WithContext with nil parent should panic (Go stdlib behavior)")
}
