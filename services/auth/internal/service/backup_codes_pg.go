package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// pgBackupCodeRepo implements BackupCodeRepository using PostgreSQL.
type pgBackupCodeRepo struct {
	pool *pgxpool.Pool
}

// NewPgBackupCodeRepo creates a PostgreSQL-backed backup code repository.
// Falls back to in-memory if pool is nil.
func NewPgBackupCodeRepo(pool *pgxpool.Pool) BackupCodeRepository {
	if pool == nil {
		return NewInMemBackupCodeRepo()
	}
	return &pgBackupCodeRepo{pool: pool}
}

// EnsureSchema creates the backup_codes table if it doesn't exist.
func (r *pgBackupCodeRepo) EnsureSchema(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS backup_codes (
			id UUID PRIMARY KEY,
			tenant_id UUID NOT NULL,
			user_id UUID NOT NULL,
			code_hash TEXT NOT NULL,
			used_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS idx_backup_codes_user ON backup_codes(tenant_id, user_id);
		CREATE INDEX IF NOT EXISTS idx_backup_codes_unused ON backup_codes(user_id) WHERE used_at IS NULL;
	`)
	return err
}

func (r *pgBackupCodeRepo) Create(ctx context.Context, codes []*BackupCode) error {
	for _, c := range codes {
		_, err := r.pool.Exec(ctx,
			`INSERT INTO backup_codes (id, tenant_id, user_id, code_hash, created_at) VALUES ($1, $2, $3, $4, $5)`,
			c.ID, c.TenantID, c.UserID, c.CodeHash, c.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("insert backup code: %w", err)
		}
	}
	return nil
}

func (r *pgBackupCodeRepo) ListUnused(ctx context.Context, tenantID, userID uuid.UUID) ([]*BackupCode, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, user_id, code_hash, used_at, created_at FROM backup_codes WHERE tenant_id = $1 AND user_id = $2 AND used_at IS NULL`,
		tenantID, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list backup codes: %w", err)
	}
	defer rows.Close()

	var result []*BackupCode
	for rows.Next() {
		bc := &BackupCode{}
		if err := rows.Scan(&bc.ID, &bc.TenantID, &bc.UserID, &bc.CodeHash, &bc.UsedAt, &bc.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, bc)
	}
	return result, nil
}

func (r *pgBackupCodeRepo) MarkUsed(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	tag, err := r.pool.Exec(ctx, `UPDATE backup_codes SET used_at = $1 WHERE id = $2 AND used_at IS NULL`, now, id)
	if err != nil {
		return fmt.Errorf("mark backup code used: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("backup code not found or already used")
	}
	return nil
}

func (r *pgBackupCodeRepo) DeleteAll(ctx context.Context, tenantID, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM backup_codes WHERE tenant_id = $1 AND user_id = $2`, tenantID, userID)
	return err
}
