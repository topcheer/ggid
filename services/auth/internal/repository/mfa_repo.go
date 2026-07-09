package repository

import (
	"context"
	stderrors "errors"
	"fmt"
	"time"

	"github.com/ggid/ggid/services/auth/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// MFADeviceRepository manages TOTP device registrations.
type MFADeviceRepository interface {
	CreateDevice(ctx context.Context, device *domain.MFADevice) error
	GetDeviceByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.MFADevice, error)
	ListDevicesByUser(ctx context.Context, tenantID, userID uuid.UUID) ([]*domain.MFADevice, error)
	GetEnabledDevice(ctx context.Context, tenantID, userID uuid.UUID) (*domain.MFADevice, error)
	UpdateDevice(ctx context.Context, device *domain.MFADevice) error
	DeleteDevice(ctx context.Context, tenantID, id uuid.UUID) error
}

// --- pgx implementation ---

type pgMFADeviceRepo struct {
	pool *pgxpool.Pool
}

// NewPGMFADeviceRepository creates a new MFADeviceRepository backed by pgx.
func NewPGMFADeviceRepository(pool *pgxpool.Pool) MFADeviceRepository {
	return &pgMFADeviceRepo{pool: pool}
}

func mfaSetTenant(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID) error {
	_, err := tx.Exec(ctx, "SET LOCAL app.tenant_id = $1", tenantID.String())
	return err
}

func mfaIsDup(err error) bool {
	var pgErr *pgconn.PgError
	return stderrors.As(err, &pgErr) && pgErr.Code == "23505"
}

func mfaIsNoRows(err error) bool {
	return stderrors.Is(err, pgx.ErrNoRows)
}

func scanMFA(row pgx.Row) (*domain.MFADevice, error) {
	d := &domain.MFADevice{}
	err := row.Scan(
		&d.ID, &d.TenantID, &d.UserID, &d.Name, &d.Secret,
		&d.Algorithm, &d.Digits, &d.Period, &d.Enabled,
		&d.VerifiedAt, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return d, nil
}

const mfaColumns = `id, tenant_id, user_id, name, secret, algorithm, digits, period, enabled, verified_at, created_at, updated_at`

func (r *pgMFADeviceRepo) CreateDevice(ctx context.Context, device *domain.MFADevice) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := mfaSetTenant(ctx, tx, device.TenantID); err != nil {
		return err
	}

	err = tx.QueryRow(ctx, `
		INSERT INTO mfa_devices (id, tenant_id, user_id, name, secret, algorithm, digits, period, enabled)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING created_at, updated_at`,
		device.ID, device.TenantID, device.UserID, device.Name, device.Secret,
		device.Algorithm, device.Digits, device.Period, device.Enabled,
	).Scan(&device.CreatedAt, &device.UpdatedAt)

	if err != nil {
		if mfaIsDup(err) {
			return fmt.Errorf("MFA device already exists")
		}
		return fmt.Errorf("create mfa device: %w", err)
	}
	return tx.Commit(ctx)
}

func (r *pgMFADeviceRepo) GetDeviceByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.MFADevice, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := mfaSetTenant(ctx, tx, tenantID); err != nil {
		return nil, err
	}

	query := fmt.Sprintf(`SELECT %s FROM mfa_devices WHERE id = $1`, mfaColumns)
	d, err := scanMFA(tx.QueryRow(ctx, query, id))
	if err != nil {
		if mfaIsNoRows(err) {
			return nil, fmt.Errorf("mfa device not found: %s", id)
		}
		return nil, fmt.Errorf("get mfa device: %w", err)
	}
	return d, nil
}

func (r *pgMFADeviceRepo) ListDevicesByUser(ctx context.Context, tenantID, userID uuid.UUID) ([]*domain.MFADevice, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := mfaSetTenant(ctx, tx, tenantID); err != nil {
		return nil, err
	}

	query := fmt.Sprintf(`SELECT %s FROM mfa_devices WHERE user_id = $1 ORDER BY created_at DESC`, mfaColumns)
	rows, err := tx.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list mfa devices: %w", err)
	}
	defer rows.Close()

	var devices []*domain.MFADevice
	for rows.Next() {
		d, err := scanMFA(rows)
		if err != nil {
			return nil, fmt.Errorf("scan mfa device: %w", err)
		}
		devices = append(devices, d)
	}
	return devices, nil
}

func (r *pgMFADeviceRepo) GetEnabledDevice(ctx context.Context, tenantID, userID uuid.UUID) (*domain.MFADevice, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := mfaSetTenant(ctx, tx, tenantID); err != nil {
		return nil, err
	}

	query := fmt.Sprintf(`SELECT %s FROM mfa_devices WHERE user_id = $1 AND enabled = true LIMIT 1`, mfaColumns)
	d, err := scanMFA(tx.QueryRow(ctx, query, userID))
	if err != nil {
		if mfaIsNoRows(err) {
			return nil, nil // no enabled device — not an error
		}
		return nil, fmt.Errorf("get enabled mfa device: %w", err)
	}
	return d, nil
}

func (r *pgMFADeviceRepo) UpdateDevice(ctx context.Context, device *domain.MFADevice) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := mfaSetTenant(ctx, tx, device.TenantID); err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		UPDATE mfa_devices SET enabled = $3, verified_at = $4, name = $5, updated_at = NOW()
		WHERE id = $2 AND tenant_id = $1`,
		device.TenantID, device.ID, device.Enabled, device.VerifiedAt, device.Name)
	if err != nil {
		return fmt.Errorf("update mfa device: %w", err)
	}
	return tx.Commit(ctx)
}

func (r *pgMFADeviceRepo) DeleteDevice(ctx context.Context, tenantID, id uuid.UUID) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := mfaSetTenant(ctx, tx, tenantID); err != nil {
		return err
	}

	tag, err := tx.Exec(ctx, `DELETE FROM mfa_devices WHERE id = $2 AND tenant_id = $1`, tenantID, id)
	if err != nil {
		return fmt.Errorf("delete mfa device: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("mfa device not found: %s", id)
	}
	return tx.Commit(ctx)
}

// Suppress unused import guard.
var _ = time.Now
