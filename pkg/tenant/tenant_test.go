package tenant

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestContext_FromContext_NoTenant(t *testing.T) {
	ctx := context.Background()
	_, err := FromContext(ctx)
	if err == nil {
		t.Fatal("should error when no tenant context")
	}
}

func TestContext_WithContext_FromContext_RoundTrip(t *testing.T) {
	tenantID := uuid.New()
	tc := &Context{
		TenantID:       tenantID,
		IsolationLevel: IsolationShared,
		Settings:       map[string]any{"plan": "pro"},
	}

	ctx := WithContext(context.Background(), tc)

	extracted, err := FromContext(ctx)
	if err != nil {
		t.Fatalf("FromContext failed: %v", err)
	}
	if extracted.TenantID != tenantID {
		t.Fatalf("tenant ID mismatch: got %v, want %v", extracted.TenantID, tenantID)
	}
	if extracted.IsolationLevel != IsolationShared {
		t.Fatalf("isolation level mismatch: got %v, want %v", extracted.IsolationLevel, IsolationShared)
	}
	if extracted.Settings["plan"] != "pro" {
		t.Fatal("settings should be preserved")
	}
}

func TestIsolationLevels(t *testing.T) {
	levels := []IsolationLevel{IsolationShared, IsolationSchema, IsolationDatabase}
	expected := []string{"shared", "schema", "database"}
	for i, level := range levels {
		if string(level) != expected[i] {
			t.Fatalf("isolation level[%d]: got %s, want %s", i, level, expected[i])
		}
	}
}

func TestMustFromContext_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("should panic when no tenant context")
		}
	}()
	MustFromContext(context.Background())
}

func TestMustFromContext_Success(t *testing.T) {
	tc := &Context{TenantID: uuid.New(), IsolationLevel: IsolationShared}
	ctx := WithContext(context.Background(), tc)

	result := MustFromContext(ctx)
	if result.TenantID != tc.TenantID {
		t.Fatal("should extract tenant context without panic")
	}
}
