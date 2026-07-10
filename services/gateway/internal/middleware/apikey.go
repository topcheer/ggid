package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// APIKeyAuth validates API keys for machine-to-machine authentication.
// API keys use the prefix "ggid_" and are validated against an APIKeyValidator.
type APIKeyValidator interface {
	Validate(ctx context.Context, key string) (tenantID string, userID string, scopes []string, err error)
}

// APIKeyAuth middleware accepts API keys as an alternative to JWT Bearer tokens.
// If no API key is present, the request passes through (JWTAuth handles it).
// If an API key is present but invalid, returns 401.
func APIKeyAuth(validator APIKeyValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract API key from header or query param
			apiKey := r.Header.Get("X-API-Key")
			if apiKey == "" {
				apiKey = r.URL.Query().Get("api_key")
			}

			// No API key → pass through to JWT auth
			if apiKey == "" {
				next.ServeHTTP(w, r)
				return
			}

			tenantID, userID, scopes, err := validator.Validate(r.Context(), apiKey)
			if err != nil {
				writeAPIKeyError(w, "invalid API key")
				return
			}

			// Inject identity into context
			ctx := r.Context()
			ctx = context.WithValue(ctx, TenantIDKey, tenantID)
			ctx = context.WithValue(ctx, UserIDKey, userID)
			ctx = context.WithValue(ctx, APIKeyScopesKey, scopes)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// IsAPIKeyRequest checks if the request carries an API key.
func IsAPIKeyRequest(r *http.Request) bool {
	return r.Header.Get("X-API-Key") != "" || r.URL.Query().Get("api_key") != ""
}

// HasScope checks if the request context contains a specific API key scope.
// HasScope checks if the request context has the required scope.
// If no scopes are in context, returns false (deny by default).
func HasScope(ctx context.Context, scope string) bool {
	scopes, ok := ctx.Value(APIKeyScopesKey).([]string)
	if !ok {
		// P0 Security: deny by default when no scopes in context.
		// Check JWT scopes if available.
		jwtScopes, jwtOk := ctx.Value(jwtScopesKey).([]string)
		if !jwtOk {
			return false
		}
		scopes = jwtScopes
	}
	for _, s := range scopes {
		if s == scope || s == "*" {
			return true
		}
	}
	return false
}

// MemoryAPIKeyValidator is a simple in-memory validator for testing.
type MemoryAPIKeyValidator struct {
	keys map[string]*apiKeyEntry
}

type apiKeyEntry struct {
	tenantID string
	userID   string
	scopes   []string
	active   bool
	expires  time.Time
}

func NewMemoryAPIKeyValidator() *MemoryAPIKeyValidator {
	return &MemoryAPIKeyValidator{keys: make(map[string]*apiKeyEntry)}
}

// AddKey registers an API key for testing.
func (v *MemoryAPIKeyValidator) AddKey(key, tenantID, userID string, scopes []string) {
	v.keys[key] = &apiKeyEntry{
		tenantID: tenantID,
		userID:   userID,
		scopes:   scopes,
		active:   true,
		expires:  time.Now().Add(24 * time.Hour),
	}
}

func (v *MemoryAPIKeyValidator) Validate(_ context.Context, key string) (string, string, []string, error) {
	entry, ok := v.keys[key]
	if !ok || !entry.active || time.Now().After(entry.expires) {
		return "", "", nil, errInvalidAPIKey
	}
	return entry.tenantID, entry.userID, entry.scopes, nil
}

var errInvalidAPIKey = &apiKeyError{"invalid or expired API key"}

type apiKeyError struct{ msg string }

func (e *apiKeyError) Error() string { return e.msg }

func writeAPIKeyError(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

var _ = strings.Contains // keep import
var _ = uuid.New          // keep import

// context key for API key scopes
var APIKeyScopesKey apiScopeCtxKey = "api_key_scopes"

// jwtScopesKey is for JWT scopes extracted from token claims.
var jwtScopesKey apiScopeCtxKey = "jwt_scopes"

type apiScopeCtxKey string

func (k apiScopeCtxKey) String() string { return string(k) }
