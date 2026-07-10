package middleware

import (
	"context"
	"net/http"
)

// TenantContextKey is the context key for tenant ID.
const TenantContextKey contextKey = "tenant_id"

// InjectTenantContext extracts tenant_id from JWT claims and injects it
// into both the request context and X-Tenant-ID header for downstream services.
// This is a convenience middleware that combines JWT claim extraction with
// header injection, specifically for tenant resolution.
func InjectTenantContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if X-Tenant-ID already set (from TenantResolver or manual)
		existingTID := r.Header.Get("X-Tenant-ID")
		if existingTID == "" {
			// Extract from JWT if available
			claims := ExtractJWTClaims(r)
			if claims.TenantID != "" {
				r.Header.Set("X-Tenant-ID", claims.TenantID)
				r = r.WithContext(context.WithValue(r.Context(), TenantContextKey, claims.TenantID))
			}
		} else {
			r = r.WithContext(context.WithValue(r.Context(), TenantContextKey, existingTID))
		}
		next.ServeHTTP(w, r)
	})
}

// TenantIDFromContext extracts tenant ID from request context.
func TenantIDFromContext(ctx context.Context) string {
	if tid, ok := ctx.Value(TenantContextKey).(string); ok {
		return tid
	}
	return ""
}
