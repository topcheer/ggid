package middleware

import (
	"context"
	"encoding/base64"
	"encoding/json"
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
}

// ExtractJWTClaims retrieves JWT claims for routing decisions.
// SECURITY: Prefers context claims (set by JWTAuth after signature verification)
// over raw header parsing. Falls back to unsigned header parsing ONLY when the
// JWTAuth middleware has not run (e.g., legacy paths without JWTAuth). This
// prevents forged JWT injection on public paths where JWTAuth(required=false)
// passes through invalid tokens without setting context claims.
func ExtractJWTClaims(r *http.Request) JWTCClaims {
	// Priority 1: Use signature-verified claims from JWTAuth middleware context.
	if c, ok := r.Context().Value(claimsKey).(JWTCClaims); ok && c.Subject != "" {
		return c
	}
	// Priority 2: Parse from Authorization header (unsigned — for legacy paths).
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return JWTCClaims{}
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return JWTCClaims{}
	}

	token := parts[1]
	tokenParts := strings.Split(token, ".")
	if len(tokenParts) != 3 {
		return JWTCClaims{}
	}

	// Decode the payload (second part)
	payload, err := base64.RawURLEncoding.DecodeString(tokenParts[1])
	if err != nil {
		return JWTCClaims{}
	}

	var raw map[string]any
	if err := json.Unmarshal(payload, &raw); err != nil {
		return JWTCClaims{}
	}

	claims := JWTCClaims{}
	if v, ok := raw["sub"].(string); ok {
		claims.Subject = v
	}
	if v, ok := raw["tenant_id"].(string); ok {
		claims.TenantID = v
	}
	if v, ok := raw["email"].(string); ok {
		claims.Email = v
	}
	if v, ok := raw["iss"].(string); ok {
		claims.Issuer = v
	}
	// Scopes can be a string or array
	switch v := raw["scope"].(type) {
	case string:
		claims.Scopes = strings.Fields(v)
	case []any:
		for _, s := range v {
			if str, ok := s.(string); ok {
				claims.Scopes = append(claims.Scopes, str)
			}
		}
	}
	// Also check "scopes" (array)
	if v, ok := raw["scopes"].([]any); ok {
		for _, s := range v {
			if str, ok := s.(string); ok {
				claims.Scopes = append(claims.Scopes, str)
			}
		}
	}
	// Extract permissions claim (fine-grained authorization)
	if v, ok := raw["permissions"].([]any); ok {
		for _, p := range v {
			if str, ok := p.(string); ok {
				claims.Permissions = append(claims.Permissions, str)
			}
		}
	}
	// Extract roles claim
	if v, ok := raw["roles"].([]any); ok {
		for _, r := range v {
			if str, ok := r.(string); ok {
				claims.Roles = append(claims.Roles, str)
			}
		}
	}
	return claims
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
		// Store in context
		ctx := context.WithValue(r.Context(), claimsKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
