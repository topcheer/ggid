package server

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	ggidcrypto "github.com/ggid/ggid/pkg/crypto"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// APIKeyRecord is the persisted representation of an API key.
// KeyHash holds the Argon2id-encoded hash; the plaintext secret is never stored.
type APIKeyRecord struct {
	ID         uuid.UUID
	TenantID   uuid.UUID
	Name       string
	KeyHash    string
	Scopes     []string
	Status     string
	CreatedAt  time.Time
	ExpiresAt  *time.Time
	LastUsedAt *time.Time
}

// apiKeyRepo provides PG-backed CRUD for api_keys.
type apiKeyRepo struct {
	pool *pgxpool.Pool
}

func newAPIKeyRepo(pool *pgxpool.Pool) *apiKeyRepo {
	return &apiKeyRepo{pool: pool}
}

// Create inserts a new API key. The secret is hashed with Argon2id before storage.
// Returns the created record (without the plaintext secret).
func (r *apiKeyRepo) Create(ctx context.Context, tenantID uuid.UUID, name, plaintextSecret string, scopes []string, expiresAt *time.Time) (*APIKeyRecord, error) {
	return r.CreateWithID(ctx, tenantID, uuid.New(), name, plaintextSecret, scopes, expiresAt)
}

// CreateWithID is like Create but uses a caller-provided UUID (needed when the
// ID is embedded in the plaintext key for lookup).
func (r *apiKeyRepo) CreateWithID(ctx context.Context, tenantID, keyID uuid.UUID, name, plaintextSecret string, scopes []string, expiresAt *time.Time) (*APIKeyRecord, error) {
	keyHash, err := ggidcrypto.HashPassword(plaintextSecret)
	if err != nil {
		return nil, fmt.Errorf("hash api key: %w", err)
	}

	rec := &APIKeyRecord{
		ID:        keyID,
		TenantID:  tenantID,
		Name:      name,
		KeyHash:   keyHash,
		Scopes:    scopes,
		Status:    "active",
		CreatedAt: time.Now(),
		ExpiresAt: expiresAt,
	}

	_, err = r.pool.Exec(ctx, `
		INSERT INTO api_keys (id, tenant_id, name, key_hash, scopes, status, created_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		rec.ID, rec.TenantID, rec.Name, rec.KeyHash, rec.Scopes, rec.Status, rec.CreatedAt, rec.ExpiresAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert api key: %w", err)
	}
	return rec, nil
}

// ListByTenant returns all API keys for a tenant (excluding key_hash).
func (r *apiKeyRepo) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]APIKeyRecord, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, tenant_id, name, scopes, status, created_at, expires_at, last_used_at
		FROM api_keys
		WHERE tenant_id = $1
		ORDER BY created_at DESC`,
		tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("list api keys: %w", err)
	}
	defer rows.Close()

	var records []APIKeyRecord
	for rows.Next() {
		rec, err := scanAPIKeyPublic(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, *rec)
	}
	return records, rows.Err()
}

// GetByID retrieves a single API key by ID and tenant (excluding key_hash).
func (r *apiKeyRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*APIKeyRecord, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, tenant_id, name, scopes, status, created_at, expires_at, last_used_at
		FROM api_keys
		WHERE id = $1 AND tenant_id = $2`,
		id, tenantID,
	)
	return scanAPIKeyPublic(row)
}

// FindForKeyValidation retrieves everything the gateway needs to validate an API key.
// The gateway calls this with the keyID extracted from the plaintext key, then
// verifies the full secret against key_hash using Argon2id.
func (r *apiKeyRepo) FindForKeyValidation(ctx context.Context, id uuid.UUID) (*APIKeyRecord, error) {
	var rec APIKeyRecord
	var expiresAt, lastUsedAt sql.NullTime
	err := r.pool.QueryRow(ctx, `
		SELECT id, tenant_id, name, key_hash, scopes, status, created_at, expires_at, last_used_at
		FROM api_keys
		WHERE id = $1`, id,
	).Scan(&rec.ID, &rec.TenantID, &rec.Name, &rec.KeyHash, &rec.Scopes, &rec.Status, &rec.CreatedAt, &expiresAt, &lastUsedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if expiresAt.Valid {
		rec.ExpiresAt = &expiresAt.Time
	}
	if lastUsedAt.Valid {
		rec.LastUsedAt = &lastUsedAt.Time
	}
	return &rec, nil
}

// UpdateStatus sets the status of an API key (e.g. "revoked", "active").
func (r *apiKeyRepo) UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE api_keys SET status = $3 WHERE id = $1 AND tenant_id = $2`,
		id, tenantID, status,
	)
	return err
}

// Rotate generates a new hash for an existing key, replacing key_hash and resetting status to active.
// Returns the new plaintext secret (returned exactly once).
func (r *apiKeyRepo) Rotate(ctx context.Context, tenantID, id uuid.UUID, plaintextSecret string) error {
	keyHash, err := ggidcrypto.HashPassword(plaintextSecret)
	if err != nil {
		return fmt.Errorf("hash rotated api key: %w", err)
	}
	_, err = r.pool.Exec(ctx, `
		UPDATE api_keys SET key_hash = $3, status = 'active', last_used_at = NULL
		WHERE id = $1 AND tenant_id = $2`,
		id, tenantID, keyHash,
	)
	return err
}

// TouchLastUsed updates last_used_at for a key (best-effort, non-fatal on error).
func (r *apiKeyRepo) TouchLastUsed(ctx context.Context, id uuid.UUID) {
	_, _ = r.pool.Exec(ctx, `UPDATE api_keys SET last_used_at = now() WHERE id = $1`, id)
}

// VerifySecret compares a plaintext secret against the stored Argon2id hash for the given key ID.
func (r *apiKeyRepo) VerifySecret(ctx context.Context, id uuid.UUID, plaintextSecret string) (*APIKeyRecord, error) {
	var rec APIKeyRecord
	var expiresAt, lastUsedAt sql.NullTime
	err := r.pool.QueryRow(ctx, `
		SELECT id, tenant_id, name, key_hash, scopes, status, created_at, expires_at, last_used_at
		FROM api_keys
		WHERE id = $1 AND status = 'active'`,
		id,
	).Scan(&rec.ID, &rec.TenantID, &rec.Name, &rec.KeyHash, &rec.Scopes, &rec.Status, &rec.CreatedAt, &expiresAt, &lastUsedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if expiresAt.Valid {
		rec.ExpiresAt = &expiresAt.Time
	}
	if lastUsedAt.Valid {
		rec.LastUsedAt = &lastUsedAt.Time
	}

	match, err := ggidcrypto.VerifyPassword(plaintextSecret, rec.KeyHash)
	if err != nil || !match {
		return nil, nil
	}
	return &rec, nil
}

// --- scanner ---

type apiKeyRowScanner interface {
	Scan(dest ...any) error
}

func scanAPIKeyPublic(row apiKeyRowScanner) (*APIKeyRecord, error) {
	var rec APIKeyRecord
	var expiresAt, lastUsedAt sql.NullTime
	err := row.Scan(&rec.ID, &rec.TenantID, &rec.Name, &rec.Scopes, &rec.Status, &rec.CreatedAt, &expiresAt, &lastUsedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if expiresAt.Valid {
		rec.ExpiresAt = &expiresAt.Time
	}
	if lastUsedAt.Valid {
		rec.LastUsedAt = &lastUsedAt.Time
	}
	return &rec, nil
}
