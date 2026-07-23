package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DBAPIKeyValidator validates API keys against the api_keys database table.
// Keys are stored as SHA-256 hashes. The validator caches lookups for 30s
// per key hash to reduce DB load.
type DBAPIKeyValidator struct {
	pool   *pgxpool.Pool
	cache  sync.Map // keyHash → *cachedKey
	ttl    time.Duration
}

type cachedKey struct {
	tenantID  string
	userID    string
	scopes    []string
	expiresAt time.Time
	status    string
	cachedAt  time.Time
}

// NewDBAPIKeyValidator creates a DB-backed API key validator.
// Returns nil if dbURL is empty (API key auth disabled).
func NewDBAPIKeyValidator(ctx context.Context, dbURL string) *DBAPIKeyValidator {
	if dbURL == "" {
		return nil
	}
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return nil
	}
	return &DBAPIKeyValidator{
		pool: pool,
		ttl:  30 * time.Second,
	}
}

// Validate implements APIKeyValidator. It hashes the incoming key with SHA-256
// and looks up the hash in the api_keys table.
func (v *DBAPIKeyValidator) Validate(ctx context.Context, key string) (string, string, []string, error) {
	// Hash the key
	sum := sha256.Sum256([]byte(key))
	keyHash := hex.EncodeToString(sum[:])

	// Check cache first
	if cached, ok := v.cache.Load(keyHash); ok {
		ck := cached.(*cachedKey)
		if time.Since(ck.cachedAt) < v.ttl {
			if ck.status != "active" {
				return "", "", nil, fmt.Errorf("api key is %s", ck.status)
			}
			if !ck.expiresAt.IsZero() && time.Now().After(ck.expiresAt) {
				return "", "", nil, fmt.Errorf("api key expired")
			}
			return ck.tenantID, ck.userID, ck.scopes, nil
		}
	}

	// Query DB
	row := v.pool.QueryRow(ctx, `
		SELECT tenant_id, COALESCE(user_id::text, ''), scopes, status, expires_at
		FROM api_keys
		WHERE key_hash = $1`,
		keyHash)

	var tenantID, userID, status string
	var scopes []string
	var expiresAt time.Time

	if err := row.Scan(&tenantID, &userID, &scopes, &status, &expiresAt); err != nil {
		return "", "", nil, fmt.Errorf("invalid api key")
	}

	// Cache the result
	v.cache.Store(keyHash, &cachedKey{
		tenantID:  tenantID,
		userID:    userID,
		scopes:    scopes,
		expiresAt: expiresAt,
		status:    status,
		cachedAt:  time.Now(),
	})

	// Async: update last_used + usage_count
	go func() {
		v.pool.Exec(context.Background(),
			`UPDATE api_keys SET last_used = NOW(), usage_count = usage_count + 1 WHERE key_hash = $1`,
			keyHash)
	}()

	if status != "active" {
		return "", "", nil, fmt.Errorf("api key is %s", status)
	}
	if !expiresAt.IsZero() && time.Now().After(expiresAt) {
		return "", "", nil, fmt.Errorf("api key expired")
	}

	return tenantID, userID, scopes, nil
}

// WithDBAPIKeyAuth wraps the given handler with DB-backed API key authentication.
// If the validator is nil (no DB configured), the handler is returned as-is.
func WithDBAPIKeyAuth(validator *DBAPIKeyValidator) func(http.Handler) http.Handler {
	if validator == nil {
		return func(next http.Handler) http.Handler { return next }
	}
	return APIKeyAuth(validator)
}

// extractAPIKeyFromRequest gets the API key from header or query param.
func extractAPIKeyFromRequest(r *http.Request) string {
	key := r.Header.Get("X-API-Key")
	if key == "" {
		key = r.URL.Query().Get("api_key")
	}
	// Also check Authorization: ApiKey ggid_sk_*
	auth := r.Header.Get("Authorization")
	if key == "" && strings.HasPrefix(auth, "ApiKey ") {
		key = strings.TrimPrefix(auth, "ApiKey ")
	}
	return key
}
