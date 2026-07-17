package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// JITRequest represents a PAM JIT elevation request.
type JITRequest struct {
	ID            uuid.UUID  `json:"id"`
	TenantID      uuid.UUID  `json:"tenant_id"`
	UserID        uuid.UUID  `json:"user_id"`
	RoleID        uuid.UUID  `json:"role_id"`
	ScopeType     string     `json:"scope_type"`
	ScopeID       *uuid.UUID `json:"scope_id,omitempty"`
	Reason        string     `json:"reason"`
	DurationMin   int        `json:"duration_min"`
	Status        string     `json:"status"`
	ApproverID    *uuid.UUID `json:"approver_id,omitempty"`
	ApprovedAt    *time.Time `json:"approved_at,omitempty"`
	ActivatedAt   *time.Time `json:"activated_at,omitempty"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty"`
	RevokedAt     *time.Time `json:"revoked_at,omitempty"`
	RevokedReason string     `json:"revoked_reason,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// JITRequestRepository manages jit_requests persistence.
type JITRequestRepository struct {
	pool *pgxpool.Pool
}

func NewJITRequestRepository(pool *pgxpool.Pool) *JITRequestRepository {
	return &JITRequestRepository{pool: pool}
}

func (r *JITRequestRepository) EnsureSchema(ctx context.Context) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS jit_requests (
			id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id       UUID NOT NULL,
			user_id         UUID NOT NULL,
			role_id         UUID NOT NULL,
			scope_type      TEXT NOT NULL DEFAULT 'tenant',
			scope_id        UUID,
			reason          TEXT NOT NULL,
			duration_min    INT NOT NULL,
			status          TEXT NOT NULL DEFAULT 'pending',
			approver_id     UUID,
			approved_at     TIMESTAMPTZ,
			activated_at    TIMESTAMPTZ,
			expires_at      TIMESTAMPTZ,
			revoked_at      TIMESTAMPTZ,
			revoked_reason  TEXT,
			created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_jit_req_user   ON jit_requests (tenant_id, user_id, status);
		CREATE INDEX IF NOT EXISTS idx_jit_req_expiry ON jit_requests (status, expires_at) WHERE status = 'active';
		CREATE INDEX IF NOT EXISTS idx_jit_req_status ON jit_requests (tenant_id, status);
	`)
	return err
}

func (r *JITRequestRepository) Create(ctx context.Context, req *JITRequest) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO jit_requests (id, tenant_id, user_id, role_id, scope_type, scope_id, reason, duration_min, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, now(), now())`,
		req.ID, req.TenantID, req.UserID, req.RoleID, req.ScopeType, req.ScopeID,
		req.Reason, req.DurationMin, req.Status,
	)
	return err
}

func (r *JITRequestRepository) List(ctx context.Context, tenantID uuid.UUID, status string, userID *uuid.UUID) ([]*JITRequest, error) {
	if r.pool == nil {
		return []*JITRequest{}, nil
	}
	query := `SELECT id, tenant_id, user_id, role_id, scope_type, scope_id, reason, duration_min, status, approver_id, approved_at, activated_at, expires_at, revoked_at, revoked_reason, created_at, updated_at
		FROM jit_requests WHERE tenant_id = $1`
	args := []any{tenantID}
	argIdx := 2
	if status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}
	if userID != nil {
		query += fmt.Sprintf(" AND user_id = $%d", argIdx)
		args = append(args, *userID)
		argIdx++
	}
	_ = argIdx
	query += " ORDER BY created_at DESC LIMIT 50"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*JITRequest
	for rows.Next() {
		req, err := scanJITRequest(rows)
		if err != nil {
			continue
		}
		result = append(result, req)
	}
	return result, nil
}

func (r *JITRequestRepository) GetByID(ctx context.Context, id uuid.UUID) (*JITRequest, error) {
	if r.pool == nil {
		return nil, fmt.Errorf("not found")
	}
	row := r.pool.QueryRow(ctx, `
		SELECT id, tenant_id, user_id, role_id, scope_type, scope_id, reason, duration_min, status, approver_id, approved_at, activated_at, expires_at, revoked_at, revoked_reason, created_at, updated_at
		FROM jit_requests WHERE id = $1`, id)
	return scanJITRequest(row)
}

func (r *JITRequestRepository) Approve(ctx context.Context, id, approverID uuid.UUID, expiresAt time.Time) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		UPDATE jit_requests SET status = 'active', approver_id = $2, approved_at = now(), activated_at = now(), expires_at = $3, updated_at = now()
		WHERE id = $1 AND status = 'pending'`, id, approverID, expiresAt)
	return err
}

func (r *JITRequestRepository) Reject(ctx context.Context, id, approverID uuid.UUID) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		UPDATE jit_requests SET status = 'rejected', approver_id = $2, updated_at = now()
		WHERE id = $1 AND status = 'pending'`, id, approverID)
	return err
}

func (r *JITRequestRepository) Revoke(ctx context.Context, id uuid.UUID, reason string) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		UPDATE jit_requests SET status = 'revoked', revoked_at = now(), revoked_reason = $2, updated_at = now()
		WHERE id = $1 AND status = 'active'`, id, reason)
	return err
}

func (r *JITRequestRepository) ListExpired(ctx context.Context) ([]*JITRequest, error) {
	if r.pool == nil {
		return []*JITRequest{}, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, tenant_id, user_id, role_id, scope_type, scope_id, reason, duration_min, status, approver_id, approved_at, activated_at, expires_at, revoked_at, revoked_reason, created_at, updated_at
		FROM jit_requests WHERE status = 'active' AND expires_at < now()`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*JITRequest
	for rows.Next() {
		req, err := scanJITRequest(rows)
		if err != nil {
			continue
		}
		result = append(result, req)
	}
	return result, nil
}

func (r *JITRequestRepository) MarkExpired(ctx context.Context, id uuid.UUID) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `UPDATE jit_requests SET status = 'expired', updated_at = now() WHERE id = $1 AND status = 'active'`, id)
	return err
}

type jitScanner interface {
	Scan(dest ...any) error
}

func scanJITRequest(s jitScanner) (*JITRequest, error) {
	var req JITRequest
	var scopeID, approverID *uuid.UUID
	var approvedAt, activatedAt, expiresAt, revokedAt *time.Time
	err := s.Scan(&req.ID, &req.TenantID, &req.UserID, &req.RoleID, &req.ScopeType, &scopeID,
		&req.Reason, &req.DurationMin, &req.Status, &approverID, &approvedAt, &activatedAt, &expiresAt,
		&revokedAt, &req.RevokedReason, &req.CreatedAt, &req.UpdatedAt)
	if err != nil {
		return nil, err
	}
	req.ScopeID = scopeID
	req.ApproverID = approverID
	req.ApprovedAt = approvedAt
	req.ActivatedAt = activatedAt
	req.ExpiresAt = expiresAt
	req.RevokedAt = revokedAt
	return &req, nil
}
