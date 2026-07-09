package repository

import (
	"context"
	"database/sql"

	"github.com/ggid/ggid/services/auth/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RefreshTokenRepository manages refresh-token persistence.
type RefreshTokenRepository struct {
	db *pgxpool.Pool
}

func NewRefreshTokenRepository(db *pgxpool.Pool) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

// Create inserts a new refresh token.
func (r *RefreshTokenRepository) Create(ctx context.Context, t *domain.RefreshToken) error {
	var clientID any
	if t.ClientID != nil {
		clientID = *t.ClientID
	}
	var rotatedFrom any
	if t.RotatedFrom != nil {
		rotatedFrom = *t.RotatedFrom
	}
	_, err := r.db.Exec(ctx, `
		INSERT INTO refresh_tokens (id, tenant_id, user_id, session_id, client_id, token_hash, scope, expires_at, rotated_from)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		t.ID, t.TenantID, t.UserID, t.SessionID, clientID, t.TokenHash, t.Scope, t.ExpiresAt, rotatedFrom,
	)
	return err
}

// FindByHash looks up a refresh token by its hash.
func (r *RefreshTokenRepository) FindByHash(ctx context.Context, tokenHash string) (*domain.RefreshToken, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, user_id, session_id, client_id, token_hash, scope,
		       expires_at, rotated_from, revoked_at, created_at
		FROM refresh_tokens
		WHERE token_hash = $1
		LIMIT 1`,
		tokenHash,
	)
	return scanRefreshToken(row)
}

// Revoke marks a token as revoked.
func (r *RefreshTokenRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE refresh_tokens SET revoked_at = NOW() WHERE id = $1`, id)
	return err
}

// RevokeAllForSession revokes all refresh tokens tied to a session.
func (r *RefreshTokenRepository) RevokeAllForSession(ctx context.Context, sessionID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE refresh_tokens SET revoked_at = NOW() WHERE session_id = $1`, sessionID)
	return err
}

// RevokeAllForUser revokes all active refresh tokens for a user.
func (r *RefreshTokenRepository) RevokeAllForUser(ctx context.Context, tenantID, userID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		UPDATE refresh_tokens SET revoked_at = NOW()
		WHERE tenant_id = $1 AND user_id = $2 AND revoked_at IS NULL`,
		tenantID, userID,
	)
	return err
}

func scanRefreshToken(row rowScanner) (*domain.RefreshToken, error) {
	var t domain.RefreshToken
	var clientID, rotatedFrom sql.NullString
	var revokedAt sql.NullTime

	err := row.Scan(
		&t.ID, &t.TenantID, &t.UserID, &t.SessionID, &clientID, &t.TokenHash, &t.Scope,
		&t.ExpiresAt, &rotatedFrom, &revokedAt, &t.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if clientID.Valid {
		if id, err := uuid.Parse(clientID.String); err == nil {
			t.ClientID = &id
		}
	}
	if rotatedFrom.Valid {
		if id, err := uuid.Parse(rotatedFrom.String); err == nil {
			t.RotatedFrom = &id
		}
	}
	if revokedAt.Valid {
		t.RevokedAt = &revokedAt.Time
	}
	return &t, nil
}
