// Package repository implements data-access for the Auth Service.
package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ggid/ggid/services/auth/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CredentialRepository manages credential persistence.
type CredentialRepository struct {
	db *pgxpool.Pool
}

func NewCredentialRepository(db *pgxpool.Pool) *CredentialRepository {
	return &CredentialRepository{db: db}
}

// FindByIDentifier looks up a credential by tenant + identifier (username or email).
func (r *CredentialRepository) FindByIDentifier(ctx context.Context, tenantID uuid.UUID, identifier string) (*domain.Credential, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, user_id, type, identifier, secret, metadata,
		       enabled, failed_attempts, locked_until, created_at, updated_at, last_used_at
		FROM credentials
		WHERE tenant_id = $1 AND identifier = $2 AND type = 'password'
		LIMIT 1`,
		tenantID, identifier,
	)
	return scanCredential(row)
}

// FindByUserID retrieves the password credential for a user.
func (r *CredentialRepository) FindByUserID(ctx context.Context, tenantID, userID uuid.UUID) (*domain.Credential, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, user_id, type, identifier, secret, metadata,
		       enabled, failed_attempts, locked_until, created_at, updated_at, last_used_at
		FROM credentials
		WHERE tenant_id = $1 AND user_id = $2 AND type = 'password'
		LIMIT 1`,
		tenantID, userID,
	)
	return scanCredential(row)
}

// Create inserts a new credential.
func (r *CredentialRepository) Create(ctx context.Context, c *domain.Credential) error {
	metadata, _ := json.Marshal(c.Metadata)
	_, err := r.db.Exec(ctx, `
		INSERT INTO credentials (tenant_id, user_id, type, identifier, secret, metadata, enabled)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		c.TenantID, c.UserID, c.Type, c.Identifier, c.Secret, metadata, c.Enabled,
	)
	if err != nil {
		return fmt.Errorf("create credential: %w", err)
	}
	return nil
}

// UpdateFailedAttempts persists failed attempt count and lock state.
func (r *CredentialRepository) UpdateFailedAttempts(ctx context.Context, id uuid.UUID, attempts int, lockedUntil *time.Time) error {
	_, err := r.db.Exec(ctx, `
		UPDATE credentials SET failed_attempts = $2, locked_until = $3, updated_at = NOW()
		WHERE id = $1`,
		id, attempts, lockedUntil,
	)
	return err
}

// UpdateSecret updates the password hash and resets failed attempts.
func (r *CredentialRepository) UpdateSecret(ctx context.Context, id uuid.UUID, secret string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE credentials
		SET secret = $2, failed_attempts = 0, locked_until = NULL, updated_at = NOW(), last_used_at = NOW()
		WHERE id = $1`,
		id, secret,
	)
	return err
}

// AddToHistory stores a password hash in the history table.
func (r *CredentialRepository) AddToHistory(ctx context.Context, tenantID, userID uuid.UUID, secret string) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO credential_history (tenant_id, user_id, secret)
		VALUES ($1, $2, $3)`,
		tenantID, userID, secret,
	)
	return err
}

// GetHistory retrieves the last N password hashes for reuse checking.
func (r *CredentialRepository) GetHistory(ctx context.Context, tenantID, userID uuid.UUID, limit int) ([]domain.CredentialHistoryEntry, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, user_id, secret, created_at
		FROM credential_history
		WHERE tenant_id = $1 AND user_id = $2
		ORDER BY created_at DESC
		LIMIT $3`,
		tenantID, userID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []domain.CredentialHistoryEntry
	for rows.Next() {
		var e domain.CredentialHistoryEntry
		if err := rows.Scan(&e.ID, &e.TenantID, &e.UserID, &e.Secret, &e.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// --- scanner ---

type rowScanner interface {
	Scan(dest ...any) error
}

func scanCredential(row rowScanner) (*domain.Credential, error) {
	var c domain.Credential
	var metadata []byte
	var lockedUntil sql.NullTime
	var lastUsedAt sql.NullTime

	err := row.Scan(
		&c.ID, &c.TenantID, &c.UserID, &c.Type, &c.Identifier, &c.Secret, &metadata,
		&c.Enabled, &c.FailedAttempts, &lockedUntil, &c.CreatedAt, &c.UpdatedAt, &lastUsedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if metadata != nil {
		_ = json.Unmarshal(metadata, &c.Metadata)
	}
	if lockedUntil.Valid {
		c.LockedUntil = &lockedUntil.Time
	}
	if lastUsedAt.Valid {
		c.LastUsedAt = &lastUsedAt.Time
	}
	return &c, nil
}
