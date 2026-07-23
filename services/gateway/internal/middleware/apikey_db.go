package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	ggidcrypto "github.com/ggid/ggid/pkg/crypto"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DBAPIKeyValidator validates API keys against the api_keys database table.
// Keys are stored as Argon2id hashes. The key format embeds a UUID for O(1)
// lookup: ggid_sk_<uuid>_<random_secret>. The validator extracts the UUID,
// fetches the stored hash, and verifies the full secret via Argon2id.
//
// Results are cached for 30s per key ID to reduce DB load.
type DBAPIKeyValidator struct {
	pool  *pgxpool.Pool
	cache sync.Map // keyID string → *cachedKey
	ttl   time.Duration
}

type cachedKey struct {
	tenantID string
	scopes   []string
	status   string
	cachedAt time.Time
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

// Validate implements APIKeyValidator. It extracts the key ID from the
// plaintext key, looks up the stored Argon2id hash, and verifies the secret.
func (v *DBAPIKeyValidator) Validate(ctx context.Context, key string) (string, string, []string, error) {
	// Parse key format: ggid_sk_<uuid>_<random_secret>
	keyID, ok := parseAPIKeyID(key)
	if !ok {
		return "", "", nil, fmt.Errorf("invalid api key format")
	}

	// Check cache first
	if cached, ok := v.cache.Load(keyID); ok {
		ck := cached.(*cachedKey)
		if time.Since(ck.cachedAt) < v.ttl {
			if ck.status != "active" {
				return "", "", nil, fmt.Errorf("api key is %s", ck.status)
			}
			return ck.tenantID, "", ck.scopes, nil
		}
	}

	// Query DB for the stored hash + metadata
	var tenantID, keyHash, status string
	var scopes []string
	var expiresAt time.Time

	err := v.pool.QueryRow(ctx, `
		SELECT tenant_id::text, key_hash, scopes, status, COALESCE(expires_at, 'epoch')
		FROM api_keys
		WHERE id = $1`,
		keyID,
	).Scan(&tenantID, &keyHash, &scopes, &status, &expiresAt)

	if err == pgx.ErrNoRows {
		return "", "", nil, fmt.Errorf("invalid api key")
	}
	if err != nil {
		return "", "", nil, fmt.Errorf("api key lookup failed: %w", err)
	}

	if status != "active" {
		return "", "", nil, fmt.Errorf("api key is %s", status)
	}
	if !expiresAt.IsZero() && time.Now().After(expiresAt) {
		return "", "", nil, fmt.Errorf("api key expired")
	}

	// Verify the full secret against the stored Argon2id hash.
	match, err := ggidcrypto.VerifyPassword(key, keyHash)
	if err != nil || !match {
		return "", "", nil, fmt.Errorf("invalid api key")
	}

	// Cache the result (key verified)
	v.cache.Store(keyID, &cachedKey{
		tenantID: tenantID,
		scopes:   scopes,
		status:   status,
		cachedAt: time.Now(),
	})

	// Async: update last_used_at (best-effort)
	go func() {
		v.pool.Exec(context.Background(),
			`UPDATE api_keys SET last_used_at = NOW() WHERE id = $1`,
			keyID)
	}()

	return tenantID, "", scopes, nil
}

// parseAPIKeyID extracts the UUID from a key of format: ggid_sk_<uuid>_<rest>
func parseAPIKeyID(key string) (string, bool) {
	// Expected: ggid_sk_<uuid>_<random>
	if !strings.HasPrefix(key, "ggid_sk_") {
		return "", false
	}
	rest := key[len("ggid_sk_"):]
	// UUID is 36 chars (with dashes); followed by underscore + random
	if len(rest) < 38 { // 36 (uuid) + 1 (_) + at least 1 char
		return "", false
	}
	// Try to parse the first 36 chars as UUID
	uuidStr := rest[:36]
	if _, err := uuid.Parse(uuidStr); err != nil {
		return "", false
	}
	// Must be followed by underscore
	if rest[36] != '_' {
		return "", false
	}
	return uuidStr, true
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
