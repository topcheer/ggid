package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/ggid/ggid/services/auth/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SessionRepository manages session persistence.
type SessionRepository struct {
	db *pgxpool.Pool
}

func NewSessionRepository(db *pgxpool.Pool) *SessionRepository {
	return &SessionRepository{db: db}
}

// Create inserts a new session.
func (r *SessionRepository) Create(ctx context.Context, s *domain.Session) error {
	deviceInfo, _ := json.Marshal(s.DeviceInfo)
	metadata, _ := json.Marshal(s.Metadata)
	_, err := r.db.Exec(ctx, `
		INSERT INTO sessions (id, tenant_id, user_id, token_hash, device_info, ip_address, user_agent, expires_at, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		s.ID, s.TenantID, s.UserID, s.TokenHash, deviceInfo, s.IPAddress, s.UserAgent, s.ExpiresAt, metadata,
	)
	return err
}

// FindByTokenHash looks up an active session by its token hash.
func (r *SessionRepository) FindByTokenHash(ctx context.Context, tokenHash string) (*domain.Session, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, user_id, token_hash, device_info, ip_address, user_agent,
		       expires_at, revoked_at, created_at, metadata
		FROM sessions
		WHERE token_hash = $1
		LIMIT 1`,
		tokenHash,
	)
	return scanSession(row)
}

// FindByID looks up a session by ID.
func (r *SessionRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Session, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, user_id, token_hash, device_info, ip_address, user_agent,
		       expires_at, revoked_at, created_at, metadata
		FROM sessions
		WHERE id = $1
		LIMIT 1`,
		id,
	)
	return scanSession(row)
}

// ListByUser returns all active sessions for a user within a tenant.
func (r *SessionRepository) ListByUser(ctx context.Context, tenantID, userID uuid.UUID) ([]*domain.Session, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, user_id, token_hash, device_info, ip_address, user_agent,
		       expires_at, revoked_at, created_at, metadata
		FROM sessions
		WHERE tenant_id = $1 AND user_id = $2 AND revoked_at IS NULL
		ORDER BY created_at DESC`,
		tenantID, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*domain.Session
	for rows.Next() {
		s, err := scanSession(rows)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}

// Revoke marks a session as revoked.
func (r *SessionRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE sessions SET revoked_at = NOW() WHERE id = $1`, id)
	return err
}

// RevokeAllForUser revokes all active sessions for a user except the given one.
func (r *SessionRepository) RevokeAllForUser(ctx context.Context, tenantID, userID, exceptSessionID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		UPDATE sessions SET revoked_at = NOW()
		WHERE tenant_id = $1 AND user_id = $2 AND revoked_at IS NULL AND id != $3`,
		tenantID, userID, exceptSessionID,
	)
	return err
}

// UpdateJTI writes the JTI and token expiry for a session (CAE Phase 2).
func (r *SessionRepository) UpdateJTI(ctx context.Context, sessionID uuid.UUID, jti string, tokenExp time.Time) error {
	_, err := r.db.Exec(ctx, `
		UPDATE sessions SET jti = $2, token_exp = $3 WHERE id = $1`,
		sessionID, jti, tokenExp,
	)
	return err
}

// ListActiveJTIForUser returns JTI + token expiry for all active (non-revoked) sessions of a user.
func (r *SessionRepository) ListActiveJTIForUser(ctx context.Context, tenantID, userID uuid.UUID) ([]domain.SessionJTI, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, jti, token_exp FROM sessions
		WHERE tenant_id = $1 AND user_id = $2 AND revoked_at IS NULL AND jti IS NOT NULL`,
		tenantID, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.SessionJTI
	for rows.Next() {
		var s domain.SessionJTI
		if err := rows.Scan(&s.SessionID, &s.JTI, &s.TokenExp); err != nil {
			return nil, err
		}
		result = append(result, s)
	}
	return result, rows.Err()
}

// DeleteExpired removes expired and revoked sessions older than the cutoff.
func (r *SessionRepository) DeleteExpired(ctx context.Context, cutoff time.Time) (int64, error) {
	tag, err := r.db.Exec(ctx, `
		DELETE FROM sessions
		WHERE (expires_at < $1 OR revoked_at IS NOT NULL) AND created_at < $1`,
		cutoff,
	)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func scanSession(row rowScanner) (*domain.Session, error) {
	var s domain.Session
	var deviceInfo, metadata []byte
	var revokedAt sql.NullTime

	err := row.Scan(
		&s.ID, &s.TenantID, &s.UserID, &s.TokenHash, &deviceInfo, &s.IPAddress, &s.UserAgent,
		&s.ExpiresAt, &revokedAt, &s.CreatedAt, &metadata,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if deviceInfo != nil {
		_ = json.Unmarshal(deviceInfo, &s.DeviceInfo)
	}
	if metadata != nil {
		_ = json.Unmarshal(metadata, &s.Metadata)
	}
	if revokedAt.Valid {
		s.RevokedAt = &revokedAt.Time
	}
	return &s, nil
}
