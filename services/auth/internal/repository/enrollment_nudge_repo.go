package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// EnrollmentNudge tracks per-user enrollment prompt state.
type EnrollmentNudge struct {
	ID             uuid.UUID  `json:"id"`
	TenantID       uuid.UUID  `json:"tenant_id"`
	UserID         uuid.UUID  `json:"user_id"`
	NudgeType      string     `json:"nudge_type"`
	ShownCount     int        `json:"shown_count"`
	LastShown      *time.Time `json:"last_shown,omitempty"`
	DismissedUntil *time.Time `json:"dismissed_until,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// EnrollmentNudgeRepository manages enrollment nudge persistence in PostgreSQL.
type EnrollmentNudgeRepository struct {
	pool *pgxpool.Pool
}

func NewEnrollmentNudgeRepository(pool *pgxpool.Pool) *EnrollmentNudgeRepository {
	return &EnrollmentNudgeRepository{pool: pool}
}

// EnsureSchema creates the enrollment_nudges table if it doesn't exist.
func (r *EnrollmentNudgeRepository) EnsureSchema(ctx context.Context) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS enrollment_nudges (
			id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id       UUID NOT NULL,
			user_id         UUID NOT NULL,
			nudge_type      TEXT NOT NULL DEFAULT 'passkey',
			shown_count     INT NOT NULL DEFAULT 0,
			last_shown      TIMESTAMPTZ,
			dismissed_until TIMESTAMPTZ,
			created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
			UNIQUE(tenant_id, user_id, nudge_type)
		);
		CREATE INDEX IF NOT EXISTS idx_enrollment_nudge_user ON enrollment_nudges (tenant_id, user_id);
	`)
	return err
}

// GetOrCreate retrieves the nudge state for a user+type, creating a default row if absent.
func (r *EnrollmentNudgeRepository) GetOrCreate(ctx context.Context, tenantID, userID uuid.UUID, nudgeType string) (*EnrollmentNudge, error) {
	if r.pool == nil {
		return &EnrollmentNudge{
			ID:        uuid.New(),
			TenantID:  tenantID,
			UserID:    userID,
			NudgeType: nudgeType,
		}, nil
	}
	// Try insert-if-not-exists, then read.
	_, _ = r.pool.Exec(ctx, `
		INSERT INTO enrollment_nudges (tenant_id, user_id, nudge_type)
		VALUES ($1, $2, $3)
		ON CONFLICT (tenant_id, user_id, nudge_type) DO NOTHING`,
		tenantID, userID, nudgeType,
	)
	row := r.pool.QueryRow(ctx, `
		SELECT id, tenant_id, user_id, nudge_type, shown_count, last_shown, dismissed_until, created_at, updated_at
		FROM enrollment_nudges
		WHERE tenant_id = $1 AND user_id = $2 AND nudge_type = $3`,
		tenantID, userID, nudgeType,
	)
	return scanEnrollmentNudgeRow(row)
}

// RecordShown increments the shown count and updates last_shown.
func (r *EnrollmentNudgeRepository) RecordShown(ctx context.Context, tenantID, userID uuid.UUID, nudgeType string) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		UPDATE enrollment_nudges
		SET shown_count = shown_count + 1, last_shown = now(), updated_at = now()
		WHERE tenant_id = $1 AND user_id = $2 AND nudge_type = $3`,
		tenantID, userID, nudgeType,
	)
	return err
}

// Dismiss sets dismissed_until to now + 7 days.
func (r *EnrollmentNudgeRepository) Dismiss(ctx context.Context, tenantID, userID uuid.UUID, nudgeType string, days int) error {
	if r.pool == nil {
		return nil
	}
	if days <= 0 {
		days = 7
	}
	_, err := r.pool.Exec(ctx, `
		UPDATE enrollment_nudges
		SET dismissed_until = now() + ($4 || ' days')::interval, updated_at = now()
		WHERE tenant_id = $1 AND user_id = $2 AND nudge_type = $3`,
		tenantID, userID, nudgeType, days,
	)
	return err
}

// IsDismissed checks whether the nudge is currently dismissed.
func (r *EnrollmentNudgeRepository) IsDismissed(ctx context.Context, tenantID, userID uuid.UUID, nudgeType string) (bool, error) {
	nudge, err := r.GetOrCreate(ctx, tenantID, userID, nudgeType)
	if err != nil || nudge == nil {
		return false, err
	}
	if nudge.DismissedUntil == nil {
		return false, nil
	}
	return nudge.DismissedUntil.After(time.Now()), nil
}

func scanEnrollmentNudgeRow(row interface {
	Scan(dest ...any) error
}) (*EnrollmentNudge, error) {
	var n EnrollmentNudge
	var lastShown, dismissedUntil *time.Time
	err := row.Scan(
		&n.ID, &n.TenantID, &n.UserID, &n.NudgeType, &n.ShownCount,
		&lastShown, &dismissedUntil, &n.CreatedAt, &n.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	n.LastShown = lastShown
	n.DismissedUntil = dismissedUntil
	return &n, nil
}
