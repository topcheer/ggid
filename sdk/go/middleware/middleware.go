// Package middleware provides HTTP middleware for integrating GGID authentication
// into Go backend applications.
//
// Usage:
//
//	import ggidmw "github.com/ggid/ggid/sdk/go/middleware"
//
//	mux := http.NewServeMux()
//	mux.HandleFunc("/api/protected", myHandler)
//
//	// Wrap with GGID auth — verifies JWT on every request
//	handler := ggidmw.Auth("https://iam.example.com", ggidmw.Options{
//		SkipPaths: []string{"/health", "/public"},
//	})(mux)
package middleware

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Options configures the Auth middleware.
type Options struct {
	// SkipPaths are URL paths that bypass JWT verification (e.g. health checks).
	SkipPaths []string
	// TenantHeader is the header name for tenant ID (default: X-Tenant-ID).
	TenantHeader string
	// OnUnauthorized is called when auth fails (default: writes JSON error).
	OnUnauthorized http.HandlerFunc
	// JWKSURL overrides the auto-detected JWKS URL (default: baseURL+/.well-known/jwks.json).
	JWKSURL string
	// Issuer is the expected JWT issuer (iss claim). If set, tokens from other issuers are rejected.
	Issuer string
	// Audience is the expected JWT audience (aud claim). If set, tokens for other audiences are rejected.
	Audience string
}

// UserInfo holds the authenticated user information extracted from the JWT.
type UserInfo struct {
	UserID      string
	TenantID    string
	Username    string
	Email       string
	Roles       []string
	Scopes      []string // OAuth scopes (openid, profile, email)
	Permissions []string // Fine-grained permissions (inventory:read, orders:write)
	Claims      map[string]any
}

type contextKey struct{}

// FromContext extracts UserInfo from the request context.
func FromContext(ctx context.Context) (*UserInfo, bool) {
	info, ok := ctx.Value(contextKey{}).(*UserInfo)
	return info, ok
}

// Auth returns an HTTP middleware that verifies GGID JWT tokens.
// The baseURL should point to the GGID Gateway (e.g. http://localhost:8080).
// JWT signatures are verified against the JWKS endpoint at baseURL/.well-known/jwks.json.
func Auth(baseURL string, opts Options) func(http.Handler) http.Handler {
	if opts.TenantHeader == "" {
		opts.TenantHeader = "X-Tenant-ID"
	}
	if opts.OnUnauthorized == nil {
		opts.OnUnauthorized = defaultUnauthorized
	}
	if opts.JWKSURL == "" {
		opts.JWKSURL = strings.TrimRight(baseURL, "/") + "/.well-known/jwks.json"
	}

	verifier := newJWKSCache(opts.JWKSURL)
	verifier.issuer = opts.Issuer
	verifier.audience = opts.Audience

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip whitelisted paths
			for _, p := range opts.SkipPaths {
				// SECURITY: Match exact path or path segment boundary to prevent
				// prefix bypass (e.g. "/public" shouldn't match "/publicadmin").
				if r.URL.Path == p || strings.HasPrefix(r.URL.Path, p+"/") {
					next.ServeHTTP(w, r)
					return
				}
			}

			// Extract Bearer token
			token := extractBearer(r)
			if token == "" {
				opts.OnUnauthorized(w, r)
				return
			}

			// Verify JWT signature against JWKS
			info, err := verifier.verify(token)
			if err != nil {
				opts.OnUnauthorized(w, r)
				return
			}

			// Inject user info into context
			ctx := context.WithValue(r.Context(), contextKey{}, info)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole returns middleware that checks if the user has the given role.
func RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			info, ok := FromContext(r.Context())
			if !ok {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			for _, userRole := range info.Roles {
				if userRole == role || userRole == "admin" {
					next.ServeHTTP(w, r)
					return
				}
			}
			http.Error(w, fmt.Sprintf(`{"error":"forbidden: requires role '%s'"}`, role), http.StatusForbidden)
		})
	}
}

// RequirePermission returns middleware that checks if the user has the given
// fine-grained permission (e.g. "inventory:read"). Users with the "admin"
// permission bypass the check.
func RequirePermission(permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			info, ok := FromContext(r.Context())
			if !ok {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			for _, p := range info.Permissions {
				if p == permission || p == "admin" {
					next.ServeHTTP(w, r)
					return
				}
			}
			http.Error(w, fmt.Sprintf(`{"error":"forbidden: requires permission '%s'"}`, permission), http.StatusForbidden)
		})
	}
}

// extractBearer extracts the Bearer token from the Authorization header.
func extractBearer(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return ""
	}
	if !strings.HasPrefix(auth, "Bearer ") {
		return ""
	}
	return strings.TrimPrefix(auth, "Bearer ")
}

// --- JWKS verification ---

// jwk represents a JSON Web Key.
type jwk struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// jwksCache caches JWKS keys with periodic refresh.
type jwksCache struct {
	url      string
	issuer   string
	audience string
	mu       sync.RWMutex
	keys     map[string]*rsa.PublicKey
	updated  time.Time
	client   *http.Client
}

func newJWKSCache(jwksURL string) *jwksCache {
	return &jwksCache{
		url:    jwksURL,
		client: &http.Client{Timeout: 5 * time.Second},
	}
}

// invalidate forces a JWKS refresh on next getKeys() call.
func (c *jwksCache) invalidate() {
	c.mu.Lock()
	c.updated = time.Time{}
	c.mu.Unlock()
}

func (c *jwksCache) getKeys() (map[string]*rsa.PublicKey, error) {
	c.mu.RLock()
	if c.keys != nil && time.Since(c.updated) < 15*time.Minute {
		keys := c.keys
		c.mu.RUnlock()
		return keys, nil
	}
	c.mu.RUnlock()

	// Check again under write lock — another goroutine may have refreshed.
	c.mu.Lock()
	if c.keys != nil && time.Since(c.updated) < 15*time.Minute {
		keys := c.keys
		c.mu.Unlock()
		return keys, nil
	}
	c.mu.Unlock()

	// SECURITY: Fetch JWKS outside the lock to avoid blocking all JWT
	// verification during HTTP latency (up to 5s timeout).
	resp, err := c.client.Get(c.url)
	if err != nil {
		return nil, fmt.Errorf("fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	var keySet struct {
		Keys []jwk `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&keySet); err != nil {
		return nil, fmt.Errorf("decode JWKS: %w", err)
	}

	keys := make(map[string]*rsa.PublicKey)
	for _, k := range keySet.Keys {
		if k.Kty != "RSA" || k.Use != "sig" {
			continue
		}
		nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
		if err != nil {
			continue
		}
		eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
		if err != nil {
			continue
		}
		e := 0
		for _, b := range eBytes {
			e = e<<8 + int(b)
		}
		pub := &rsa.PublicKey{
			N: new(big.Int).SetBytes(nBytes),
			E: e,
		}
		keys[k.Kid] = pub
	}

	// Swap under write lock to protect concurrent readers
	c.mu.Lock()
	c.keys = keys
	c.updated = time.Now()
	c.mu.Unlock()
	return keys, nil
}

// verify validates the JWT signature and extracts user info.
func (c *jwksCache) verify(tokenString string) (*UserInfo, error) {
	keys, err := c.getKeys()
	if err != nil {
		return nil, fmt.Errorf("jwks: %w", err)
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method against the GGID algorithm whitelist
		if !isSupportedAlg(token.Method.Alg()) {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		kid, _ := token.Header["kid"].(string)
		key, ok := keys[kid]
		if !ok {
			// SECURITY: Key may have been rotated — force a JWKS refresh.
			c.invalidate()
			keys, err = c.getKeys()
			if err != nil {
				return nil, fmt.Errorf("jwks refresh: %w", err)
			}
			key, ok = keys[kid]
			if !ok {
				return nil, fmt.Errorf("key not found for kid: %s", kid)
			}
		}
		return key, nil
	})
	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid claims")
	}

	// Validate issuer if configured.
	if c.issuer != "" {
		if iss, _ := claims["iss"].(string); iss != c.issuer {
			return nil, fmt.Errorf("invalid token issuer")
		}
	}
	// Validate audience if configured.
	if c.audience != "" {
		aud, _ := claims["aud"].(string)
		if aud != c.audience {
			// Also check if aud is a slice (JWT spec allows multiple audiences).
			if audSlice, ok := claims["aud"].([]any); ok {
				found := false
				for _, a := range audSlice {
					if fmt.Sprintf("%v", a) == c.audience {
						found = true
						break
					}
				}
				if !found {
					return nil, fmt.Errorf("invalid token audience")
				}
			} else {
				return nil, fmt.Errorf("invalid token audience")
			}
		}
	}

	info := &UserInfo{}
	if v, ok := claims["sub"]; ok {
		info.UserID = fmt.Sprintf("%v", v)
	}
	if v, ok := claims["tenant_id"]; ok {
		info.TenantID = fmt.Sprintf("%v", v)
	}
	if v, ok := claims["username"]; ok {
		info.Username = fmt.Sprintf("%v", v)
	}
	if v, ok := claims["email"]; ok {
		info.Email = fmt.Sprintf("%v", v)
	}
	if roles, ok := claims["roles"].([]any); ok {
		for _, r := range roles {
			info.Roles = append(info.Roles, fmt.Sprintf("%v", r))
		}
	}
	if scope, ok := claims["scope"].(string); ok {
		info.Scopes = strings.Split(scope, " ")
	}
	// Extract fine-grained permissions claim (array of strings)
	if perms, ok := claims["permissions"].([]any); ok {
		for _, p := range perms {
			info.Permissions = append(info.Permissions, fmt.Sprintf("%v", p))
		}
	}

	return info, nil
}

func defaultUnauthorized(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]string{
		"error": "missing or invalid token",
	})
}
