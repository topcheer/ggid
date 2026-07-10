// Package middleware implements HTTP middleware for the API Gateway.
package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/ggid/ggid/pkg/tenant"
	"github.com/google/uuid"
)

// TenantResolverEnhanced is like TenantResolver but also extracts tenant ID
// from JWT claims and from subdomain (alias-based tenants).
//
// Resolution order (highest priority first):
//  1. X-Tenant-ID header (explicit override)
//  2. tenant_id JWT claim (set by auth service)
//  3. Subdomain prefix (e.g. acme.iam.example.com → tenant alias "acme")
//
// For subdomain-based tenants, an alias-to-UUID mapping function can be
// supplied to resolve the alias to a concrete tenant UUID.
type TenantAliasResolver func(ctx context.Context, alias string) (uuid.UUID, error)

// EnhancedTenantConfig configures the enhanced tenant resolver.
type EnhancedTenantConfig struct {
	DomainSuffix  string              // e.g. ".iam.example.com"
	AliasResolver TenantAliasResolver // optional: resolve alias → UUID
}

// EnhancedTenantResolver creates a middleware that resolves tenant from
// header, JWT claim, or subdomain.
func EnhancedTenantResolver(cfg EnhancedTenantConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var tenantID uuid.UUID
			var source string

			// 1. X-Tenant-ID header (highest priority)
			if tidStr := r.Header.Get("X-Tenant-ID"); tidStr != "" {
				if id, err := uuid.Parse(tidStr); err == nil {
					tenantID = id
					source = "header"
				}
			}

			// 2. JWT claim tenant_id (set by JWTAuth middleware, stored in context)
			if tenantID == uuid.Nil {
				if tidStr, ok := r.Context().Value(TenantIDKey).(string); ok && tidStr != "" {
					if id, err := uuid.Parse(tidStr); err == nil {
						tenantID = id
						source = "jwt"
					}
				}
			}

			// 3. Subdomain extraction
			if tenantID == uuid.Nil && cfg.DomainSuffix != "" {
				host := r.Host
				// Strip port if present
				if idx := strings.LastIndex(host, ":"); idx != -1 {
					host = host[:idx]
				}
				if strings.HasSuffix(host, cfg.DomainSuffix) {
					sub := strings.TrimSuffix(host, cfg.DomainSuffix)
					parts := strings.Split(sub, ".")
					if len(parts) > 0 && parts[0] != "" && parts[0] != "www" {
						alias := parts[0]
						if cfg.AliasResolver != nil {
							if id, err := cfg.AliasResolver(r.Context(), alias); err == nil {
								tenantID = id
								source = "subdomain"
							}
						} else {
							// Without resolver, try parsing as UUID directly
							if id, err := uuid.Parse(alias); err == nil {
								tenantID = id
								source = "subdomain"
							}
						}
					}
				}
			}

			if tenantID != uuid.Nil {
				tc := &tenant.Context{
					TenantID:       tenantID,
					IsolationLevel: tenant.IsolationShared,
				}
				ctx := tenant.WithContext(r.Context(), tc)
				ctx = context.WithValue(ctx, TenantIDKey, tenantID.String())
				ctx = context.WithValue(ctx, tenantSourceKey{}, source)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

type tenantSourceKey struct{}

// TenantSourceFromRequest returns the source of the tenant resolution
// ("header", "jwt", "subdomain", or "" if not resolved).
func TenantSourceFromRequest(r *http.Request) string {
	s, _ := r.Context().Value(tenantSourceKey{}).(string)
	return s
}

// ResolveTenantFromSubdomain extracts the tenant alias from a subdomain.
// Returns empty string if no valid subdomain is found.
func ResolveTenantFromSubdomain(host, domainSuffix string) string {
	// Strip port
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		host = host[:idx]
	}
	if domainSuffix == "" {
		return ""
	}
	if !strings.HasSuffix(host, domainSuffix) {
		return ""
	}
	sub := strings.TrimSuffix(host, domainSuffix)
	parts := strings.Split(sub, ".")
	if len(parts) == 0 || parts[0] == "" || parts[0] == "www" {
		return ""
	}
	return parts[0]
}

// ResolveTenantFromJWTClaim extracts tenant_id from a JWT claims map.
func ResolveTenantFromJWTClaim(claims map[string]any) (uuid.UUID, error) {
	tid, ok := claims["tenant_id"]
	if !ok {
		return uuid.Nil, fmt.Errorf("no tenant_id claim")
	}
	switch v := tid.(type) {
	case string:
		return uuid.Parse(v)
	default:
		return uuid.Nil, fmt.Errorf("invalid tenant_id claim type")
	}
}
