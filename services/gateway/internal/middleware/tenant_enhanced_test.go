package middleware

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ggid/ggid/pkg/tenant"
	"github.com/google/uuid"
)

func TestResolveTenantFromSubdomain(t *testing.T) {
	tests := []struct {
		host       string
		suffix     string
		want       string
	}{
		{"acme.iam.com", ".iam.com", "acme"},
		{"12345678-1234-1234-1234-123456789012.iam.com", ".iam.com", "12345678-1234-1234-1234-123456789012"},
		{"www.iam.com", ".iam.com", ""}, // www is excluded
		{"iam.com", ".iam.com", ""},     // no subdomain
		{"acme.iam.com:8080", ".iam.com", "acme"}, // with port
		{"acme.other.com", ".iam.com", ""}, // wrong domain
		{"", ".iam.com", ""},
		{"acme.iam.com", "", ""}, // no suffix configured
	}
	for _, tt := range tests {
		got := ResolveTenantFromSubdomain(tt.host, tt.suffix)
		if got != tt.want {
			t.Errorf("ResolveTenantFromSubdomain(%q, %q) = %q, want %q", tt.host, tt.suffix, got, tt.want)
		}
	}
}

func TestResolveTenantFromJWTClaim(t *testing.T) {
	tests := []struct {
		name    string
		claims  map[string]any
		wantErr bool
	}{
		{
			name:    "valid tenant_id",
			claims:  map[string]any{"tenant_id": "550e8400-e29b-41d4-a716-446655440000"},
			wantErr: false,
		},
		{
			name:    "missing tenant_id",
			claims:  map[string]any{"sub": "user123"},
			wantErr: true,
		},
		{
			name:    "invalid tenant_id",
			claims:  map[string]any{"tenant_id": 12345}, // wrong type
			wantErr: true,
		},
		{
			name:    "malformed uuid",
			claims:  map[string]any{"tenant_id": "not-a-uuid"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ResolveTenantFromJWTClaim(tt.claims)
			if (err != nil) != tt.wantErr {
				t.Errorf("expected err=%v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestEnhancedTenantResolver_HeaderPriority(t *testing.T) {
	tenantID := uuid.New()
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		tc, _ := tenant.FromContext(r.Context())
		if tc == nil || tc.TenantID != tenantID {
			t.Error("expected tenant from header")
		}
		source := TenantSourceFromRequest(r)
		if source != "header" {
			t.Errorf("expected source=header, got %s", source)
		}
	})

	mw := EnhancedTenantResolver(EnhancedTenantConfig{})(next)
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Tenant-ID", tenantID.String())
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req)
	if !called {
		t.Error("handler should be called")
	}
}

func TestEnhancedTenantResolver_NoTenant(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	mw := EnhancedTenantResolver(EnhancedTenantConfig{})(next)
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req)
	if !called {
		t.Error("handler should be called even without tenant")
	}
}

func TestEnhancedTenantResolver_Subdomain(t *testing.T) {
	tenantID := uuid.New()
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		tc, _ := tenant.FromContext(r.Context())
		if tc == nil || tc.TenantID != tenantID {
			t.Error("expected tenant from subdomain")
		}
	})

	mw := EnhancedTenantResolver(EnhancedTenantConfig{
		DomainSuffix: ".iam.com",
		AliasResolver: func(_ context.Context, alias string) (uuid.UUID, error) {
			if alias == "acme" {
				return tenantID, nil
			}
			return uuid.Nil, fmt.Errorf("unknown")
		},
	})(next)

	req := httptest.NewRequest("GET", "/", nil)
	req.Host = "acme.iam.com"
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req)
	if !called {
		t.Error("handler should be called")
	}
}

func TestEnhancedTenantResolver_SubdomainWWW(t *testing.T) {
	called := false
	var tenantResolved bool
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		tc, _ := tenant.FromContext(r.Context())
		if tc != nil {
			tenantResolved = true
		}
	})

	mw := EnhancedTenantResolver(EnhancedTenantConfig{
		DomainSuffix: ".iam.com",
	})(next)

	req := httptest.NewRequest("GET", "/", nil)
	req.Host = "www.iam.com"
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req)
	if !called {
		t.Error("handler should be called")
	}
	if tenantResolved {
		t.Error("www should not resolve to a tenant")
	}
}

func TestEnhancedTenantResolver_SubdomainWithPort(t *testing.T) {
	tenantID := uuid.New()
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		tc, _ := tenant.FromContext(r.Context())
		if tc == nil || tc.TenantID != tenantID {
			t.Error("expected tenant from subdomain with port")
		}
	})

	mw := EnhancedTenantResolver(EnhancedTenantConfig{
		DomainSuffix: ".iam.com",
		AliasResolver: func(_ context.Context, alias string) (uuid.UUID, error) {
			return tenantID, nil
		},
	})(next)

	req := httptest.NewRequest("GET", "/", nil)
	req.Host = "acme.iam.com:8080"
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req)
	if !called {
		t.Error("handler should be called")
	}
}
