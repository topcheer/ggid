package middleware

import (
	"crypto/subtle"
	"net/http"
	"sync"
)

// TenantCORSStore manages per-tenant CORS allowed origins.
// In production this would be backed by a database or Redis.
type TenantCORSStore struct {
	mu       sync.RWMutex
	origins  map[string][]string // tenantID -> allowed origins
	fallback CORSConfig          // used when tenant has no custom origins
}

// NewTenantCORSStore creates a new store with the given fallback config.
func NewTenantCORSStore(fallback CORSConfig) *TenantCORSStore {
	return &TenantCORSStore{
		origins:  make(map[string][]string),
		fallback: fallback,
	}
}

// SetOrigins sets the allowed origins for a tenant.
func (s *TenantCORSStore) SetOrigins(tenantID string, origins []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.origins[tenantID] = origins
}

// GetOrigins returns the allowed origins for a tenant.
// Returns the fallback config's origins if tenant has none.
func (s *TenantCORSStore) GetOrigins(tenantID string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if origins, ok := s.origins[tenantID]; ok {
		return origins
	}
	return s.fallback.AllowedOrigins
}

// DeleteOrigins removes per-tenant origins, reverting to fallback.
func (s *TenantCORSStore) DeleteOrigins(tenantID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.origins, tenantID)
}

// originAllowed checks if the given origin is in the allowed list.
// An empty list or list containing "*" allows all origins.
func originAllowed(origin string, allowed []string) bool {
	if len(allowed) == 0 {
		return true // no restrictions
	}
	for _, a := range allowed {
		if a == "*" {
			return true
		}
		if subtle.ConstantTimeCompare([]byte(origin), []byte(a)) == 1 {
			return true
		}
	}
	return false
}

// PerTenantCORS returns middleware that resolves allowed origins per-tenant.
// It reads the tenant ID from context (set by TenantResolver middleware) and
// looks up per-tenant allowed origins from the store.
func PerTenantCORS(store *TenantCORSStore, allowCredentials bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Resolve tenant ID from context
			tenantID, _ := TenantIDFromRequest(r)
			allowedOrigins := store.GetOrigins(tenantID)

			if originAllowed(origin, allowedOrigins) {
				if origin != "" {
					// Echo the specific origin rather than wildcard when credentials are involved
					if allowCredentials || !containsWildcard(allowedOrigins) {
						w.Header().Set("Access-Control-Allow-Origin", origin)
						w.Header().Add("Vary", "Origin")
					} else {
						w.Header().Set("Access-Control-Allow-Origin", "*")
					}
				} else {
					w.Header().Set("Access-Control-Allow-Origin", "*")
				}
			}

			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Tenant-ID, X-Request-ID, X-API-Key")
			w.Header().Set("Access-Control-Expose-Headers", "X-Request-ID, X-Tenant-ID")
			w.Header().Set("Access-Control-Max-Age", "3600")

			if allowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			// Handle OPTIONS preflight
			if r.Method == http.MethodOptions {
				if origin == "" || originAllowed(origin, allowedOrigins) {
					w.WriteHeader(http.StatusNoContent)
					return
				}
				// Origin not allowed — return 403
				w.WriteHeader(http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// containsWildcard checks if the allowed origins list contains "*".
func containsWildcard(origins []string) bool {
	for _, o := range origins {
		if o == "*" {
			return true
		}
	}
	return false
}
