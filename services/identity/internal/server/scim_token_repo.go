package server

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SCIMToken represents a SCIM bearer token record.
type SCIMToken struct {
	ID         uuid.UUID  `json:"id"`
	TenantID   uuid.UUID  `json:"tenant_id"`
	Name       string     `json:"name"`
	TokenHash  string     `json:"-"` // never expose hash
	Scopes     []string   `json:"scopes"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
	CreatedBy  uuid.UUID  `json:"created_by"`
	CreatedAt  time.Time  `json:"created_at"`
}

const scimTokenPrefix = "ggid_scim_"

// scimTokenRepo manages SCIM token persistence.
type scimTokenRepo struct {
	pool *pgxpool.Pool
}

func newSCIMTokenRepo(pool *pgxpool.Pool) *scimTokenRepo {
	return &scimTokenRepo{pool: pool}
}

// EnsureSchema creates the scim_tokens table if it doesn't exist.
func (r *scimTokenRepo) EnsureSchema(ctx context.Context) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS scim_tokens (
			id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id     UUID NOT NULL,
			name          TEXT NOT NULL,
			token_hash    TEXT NOT NULL,
			scopes        TEXT[] NOT NULL DEFAULT '{scim}',
			expires_at    TIMESTAMPTZ,
			last_used_at  TIMESTAMPTZ,
			revoked_at    TIMESTAMPTZ,
			created_by    UUID NOT NULL,
			created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_scim_tokens_tenant ON scim_tokens (tenant_id) WHERE revoked_at IS NULL;
		CREATE INDEX IF NOT EXISTS idx_scim_tokens_hash  ON scim_tokens (token_hash) WHERE revoked_at IS NULL;
	`)
	return err
}

// generateSCIMTokenPlaintext generates a random token: ggid_scim_<base64url(32 bytes)>.
func generateSCIMTokenPlaintext() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate scim token: %w", err)
	}
	return scimTokenPrefix + base64.RawURLEncoding.EncodeToString(raw), nil
}

// Create creates a new SCIM token and returns the plaintext (shown once).
func (r *scimTokenRepo) Create(ctx context.Context, tenantID uuid.UUID, name string, createdBy uuid.UUID, hashFn func(string) string) (*SCIMToken, string, error) {
	plaintext, err := generateSCIMTokenPlaintext()
	if err != nil {
		return nil, "", err
	}
	tokenHash := hashFn(plaintext)

	token := &SCIMToken{
		ID:        uuid.New(),
		TenantID:  tenantID,
		Name:      name,
		TokenHash: tokenHash,
		Scopes:    []string{"scim"},
		CreatedBy: createdBy,
		CreatedAt: time.Now(),
	}

	if r.pool != nil {
		_, err := r.pool.Exec(ctx, `
			INSERT INTO scim_tokens (id, tenant_id, name, token_hash, scopes, created_by, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			token.ID, token.TenantID, token.Name, token.TokenHash, token.Scopes, token.CreatedBy, token.CreatedAt,
		)
		if err != nil {
			return nil, "", fmt.Errorf("create scim token: %w", err)
		}
	}

	return token, plaintext, nil
}

// ListByTenant returns all non-revoked SCIM tokens for a tenant.
func (r *scimTokenRepo) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*SCIMToken, error) {
	if r.pool == nil {
		return []*SCIMToken{}, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, tenant_id, name, scopes, expires_at, last_used_at, revoked_at, created_by, created_at
		FROM scim_tokens
		WHERE tenant_id = $1 AND revoked_at IS NULL
		ORDER BY created_at DESC`,
		tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []*SCIMToken
	for rows.Next() {
		t, err := scanSCIMToken(rows)
		if err != nil {
			continue
		}
		tokens = append(tokens, t)
	}
	return tokens, nil
}

// Revoke soft-deletes a token by setting revoked_at.
func (r *scimTokenRepo) Revoke(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		UPDATE scim_tokens SET revoked_at = now()
		WHERE id = $1 AND tenant_id = $2 AND revoked_at IS NULL`,
		id, tenantID,
	)
	return err
}

// FindByHash looks up a token by its hash for authentication verification.
func (r *scimTokenRepo) FindByHash(ctx context.Context, hash string) (*SCIMToken, error) {
	if r.pool == nil {
		return nil, nil
	}
	row := r.pool.QueryRow(ctx, `
		SELECT id, tenant_id, name, scopes, expires_at, last_used_at, revoked_at, created_by, created_at
		FROM scim_tokens
		WHERE token_hash = $1 AND revoked_at IS NULL
		LIMIT 1`,
		hash,
	)
	return scanSCIMToken(row)
}

// UpdateLastUsed records the last authentication time.
func (r *scimTokenRepo) UpdateLastUsed(ctx context.Context, id uuid.UUID) {
	if r.pool == nil {
		return
	}
	_, _ = r.pool.Exec(ctx, `UPDATE scim_tokens SET last_used_at = now() WHERE id = $1`, id)
}

func scanSCIMToken(row interface {
	Scan(dest ...any) error
}) (*SCIMToken, error) {
	var t SCIMToken
	if err := row.Scan(&t.ID, &t.TenantID, &t.Name, &t.Scopes, &t.ExpiresAt, &t.LastUsedAt, &t.RevokedAt, &t.CreatedBy, &t.CreatedAt); err != nil {
		return nil, err
	}
	return &t, nil
}
