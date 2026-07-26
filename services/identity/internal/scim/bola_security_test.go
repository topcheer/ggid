package scim

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	ggidtenant "github.com/ggid/ggid/pkg/tenant"
	"github.com/google/uuid"
)

// TestInjectTenant_RespectsTokenContext proves P0 BOLA fix:
// SCIM token's tenant context must NOT be overwritten by X-Tenant-ID header.
func TestInjectTenant_RespectsTokenContext(t *testing.T) {
	tokenTenant := uuid.New()
	headerTenant := uuid.New() // attacker injects different tenant

	// Simulate scimTokenAuth: sets context from token
	tokenCtx := ggidtenant.WithContext(context.Background(), &ggidtenant.Context{
		TenantID:       tokenTenant,
		IsolationLevel: ggidtenant.IsolationShared,
	})

	req := httptest.NewRequest(http.MethodGet, "/scim/v2/Users", nil)
	req.Header.Set("X-Tenant-ID", headerTenant.String())
	req = req.WithContext(tokenCtx)

	ok, ctx := injectTenant(req)
	if !ok {
		t.Fatal("expected injectTenant to return ok=true")
	}

	tc, err := ggidtenant.FromContext(ctx)
	if err != nil {
		t.Fatalf("expected tenant context: %v", err)
	}

	// CRITICAL: context must reflect TOKEN tenant, not header tenant
	if tc.TenantID != tokenTenant {
		t.Errorf("BOLA NOT FIXED: tenant=%s (from header), expected=%s (from token)",
			tc.TenantID, tokenTenant)
	}
	if tc.TenantID == headerTenant {
		t.Fatal("BOLA VULNERABILITY: attacker's X-Tenant-ID header overwrote token tenant!")
	}
}

// TestInjectTenant_FallbackToHeader ensures non-token paths still work.
func TestInjectTenant_FallbackToHeader(t *testing.T) {
	headerTenant := uuid.New()

	req := httptest.NewRequest(http.MethodGet, "/scim/v2/Users", nil)
	req.Header.Set("X-Tenant-ID", headerTenant.String())

	ok, ctx := injectTenant(req)
	if !ok {
		t.Fatal("expected injectTenant to return ok=true")
	}

	tc, err := ggidtenant.FromContext(ctx)
	if err != nil {
		t.Fatalf("expected tenant context: %v", err)
	}

	if tc.TenantID != headerTenant {
		t.Errorf("expected header tenant %s, got %s", headerTenant, tc.TenantID)
	}
}

// TestInjectTenant_MissingBothTokenAndHeader ensures fail-closed behavior.
func TestInjectTenant_MissingBothTokenAndHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/scim/v2/Users", nil)

	ok, _ := injectTenant(req)
	if ok {
		t.Fatal("expected injectTenant to return ok=false when no tenant source exists")
	}
}
