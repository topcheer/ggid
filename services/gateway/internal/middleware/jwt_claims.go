package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// jwtClaimsCtxKey is the context key for extracted JWT claims.
type jwtClaimsCtxKey string

const claimsKey jwtClaimsCtxKey = "jwt_claims"

// JWTCClaims holds extracted JWT claims relevant to routing.
type JWTCClaims struct {
	Subject     string   `json:"sub"`
	TenantID    string   `json:"tenant_id"`
	Scopes      []string `json:"scopes"`      // OAuth scopes (openid, profile, email)
	Permissions []string `json:"permissions"` // Fine-grained permissions (inventory:read)
	Roles       []string `json:"roles"`       // Role names (ERP Manager)
	Email       string   `json:"email"`
	Issuer      string   `json:"iss"`
	Imp         bool     `json:"imp"` // impersonation marker
}

// ExtractJWTClaims retrieves JWT claims for routing decisions.
// SECURITY: Prefers context claims (set by JWTAuth after signature verification)
// over raw header parsing. Falls back to unsigned header parsing ONLY when the
// JWTAuth middleware has not run (e.g., legacy paths without JWTAuth). This
// prevents forged JWT injection on public paths where JWTAuth(required=false)
// passes through invalid tokens without setting context claims.
func ExtractJWTClaims(r *http.Request) JWTCClaims {
	// SECURITY (R226 P0): Only use signature-verified claims from JWTAuth
	// middleware context. Never parse unsigned JWT payload from Authorization
	// header — forged tokens on public paths (no JWTAuth) would inject
	// arbitrary sub/scopes into downstream headers.
	if c, ok := r.Context().Value(claimsKey).(JWTCClaims); ok && c.Subject != "" {
		return c
	}
	// No verified claims available — return empty (fail-closed).
	return JWTCClaims{}
}

// buildVerifiedClaims constructs JWTCClaims from jwt.MapClaims (already signature-verified).
func buildVerifiedClaims(claims jwt.MapClaims) JWTCClaims {
	c := JWTCClaims{}
	if v, ok := claims["sub"].(string); ok {
		c.Subject = v
	}
	if v, ok := claims["tenant_id"].(string); ok {
		c.TenantID = v
	}
	if v, ok := claims["email"].(string); ok {
		c.Email = v
	}
	if v, ok := claims["iss"].(string); ok {
		c.Issuer = v
	}
	switch v := claims["scope"].(type) {
	case string:
		c.Scopes = strings.Fields(v)
	case []any:
		for _, s := range v {
			if str, ok := s.(string); ok {
				c.Scopes = append(c.Scopes, str)
			}
		}
	}
	if v, ok := claims["scopes"].([]any); ok {
		for _, s := range v {
			if str, ok := s.(string); ok {
				c.Scopes = append(c.Scopes, str)
			}
		}
	}
	if v, ok := claims["permissions"].([]any); ok {
		for _, p := range v {
			if str, ok := p.(string); ok {
				c.Permissions = append(c.Permissions, str)
			}
		}
	}
	if v, ok := claims["roles"].([]any); ok {
		for _, r := range v {
			if str, ok := r.(string); ok {
				c.Roles = append(c.Roles, str)
			}
		}
	}
	return c
}

// ClaimsFromContext retrieves JWT claims from context.
func ClaimsFromContext(ctx context.Context) JWTCClaims {
	if ctx == nil {
		return JWTCClaims{}
	}
	if c, ok := ctx.Value(claimsKey).(JWTCClaims); ok {
		return c
	}
	return JWTCClaims{}
}

// WithVerifiedClaims returns a context carrying signature-verified JWT
// claims, the same representation JWTAuth stores on successful validation.
// Used by callers that run their own token verification (e.g. tests,
// alternate routers) before routing decisions call ExtractJWTClaims.
func WithVerifiedClaims(ctx context.Context, claims JWTCClaims) context.Context {
	return context.WithValue(ctx, claimsKey, claims)
}

// JWTClaimExtraction middleware extracts JWT claims and sets downstream headers
// (X-User-ID, X-Tenant-ID, X-Scopes) for backend services.
func JWTClaimExtraction(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := ExtractJWTClaims(r)
		if claims.Subject != "" {
			r.Header.Set("X-User-ID", claims.Subject)
		}
		// Only set X-Tenant-ID from JWT if not already set by TenantResolver
		// or explicit client request. This allows platform admins to target
		// other tenants via X-Tenant-ID header.
		if claims.TenantID != "" && r.Header.Get("X-Tenant-ID") == "" {
			r.Header.Set("X-Tenant-ID", claims.TenantID)
		}
		if len(claims.Scopes) > 0 {
			r.Header.Set("X-Scopes", strings.Join(claims.Scopes, ","))
			isAdmin := false
			for _, sc := range claims.Scopes {
				if sc == "platform:admin" || sc == "tenant:admin" {
					r.Header.Set("X-User-Role", sc)
					r.Header.Set("X-Is-Admin", "true")
					isAdmin = true
					break
				}
			}
			// SECURITY: always clear spoofed admin headers for non-admin users.
			if !isAdmin {
				r.Header.Del("X-Is-Admin")
				r.Header.Del("X-User-Role")
			}
		} else {
			// SECURITY: no scopes → definitely not admin, clear headers.
			r.Header.Del("X-Is-Admin")
			r.Header.Del("X-User-Role")
			r.Header.Del("X-Scopes")
		}
		// SECURITY: Clear spoofed X-Impersonated header, then set from verified JWT.
		r.Header.Del("X-Impersonated")
		if claims.Imp {
			r.Header.Set("X-Impersonated", "true")
		}
		// Store in context
		ctx := context.WithValue(r.Context(), claimsKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
