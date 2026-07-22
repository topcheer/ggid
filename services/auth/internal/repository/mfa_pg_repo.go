package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ggid/ggid/services/auth/internal/domain"
	ggidcrypto "github.com/ggid/ggid/pkg/crypto"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// pgMFADeviceRepository implements MFADeviceRepository using pgx.
type pgMFADeviceRepository struct {
	db *pgxpool.Pool
}

// NewMFADeviceRepository creates a new MFADeviceRepository.
func NewMFADeviceRepository(db *pgxpool.Pool) *pgMFADeviceRepository {
	return &pgMFADeviceRepository{db: db}
}

func (r *pgMFADeviceRepository) setTenantRLS(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID) error {
	_, err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL app.tenant_id = '%s'", tenantID.String()))
	return err
}

func (r *pgMFADeviceRepository) CreateDevice(ctx context.Context, device *domain.MFADevice) error {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := r.setTenantRLS(ctx, tx, device.TenantID); err != nil {
		return err
	}

	encryptedSecret, encErr := ggidcrypto.EncryptTOTPSecret(device.Secret)
	if encErr != nil {
		return fmt.Errorf("encrypt totp secret: %w", encErr)
	}

	err = tx.QueryRow(ctx, `
		INSERT INTO mfa_devices (id, tenant_id, user_id, name, secret, algorithm, digits, period, enabled, verified_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING created_at, updated_at`,
		device.ID, device.TenantID, device.UserID, device.Name, encryptedSecret,
		device.Algorithm, device.Digits, device.Period, device.Enabled, device.VerifiedAt,
	).Scan(&device.CreatedAt, &device.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			return fmt.Errorf("postgres error: %s: %w", pgErr.Code, err)
		}
		return fmt.Errorf("create mfa device: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *pgMFADeviceRepository) GetDeviceByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.MFADevice, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := r.setTenantRLS(ctx, tx, tenantID); err != nil {
		return nil, err
	}

	device := &domain.MFADevice{}
	err = tx.QueryRow(ctx, `
		SELECT id, tenant_id, user_id, name, secret, algorithm, digits, period, enabled, verified_at, created_at, updated_at
		FROM mfa_devices WHERE id = $1`, id,
	).Scan(&device.ID, &device.TenantID, &device.UserID, &device.Name, &device.Secret,
		&device.Algorithm, &device.Digits, &device.Period, &device.Enabled, &device.VerifiedAt,
		&device.CreatedAt, &device.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("mfa device not found: %s", id)
		}
		return nil, fmt.Errorf("get mfa device: %w", err)
	}
	device.Secret, _ = ggidcrypto.DecryptTOTPSecret(device.Secret)

	tx.Commit(ctx)
	return device, nil
}

func (r *pgMFADeviceRepository) ListDevicesByUser(ctx context.Context, tenantID, userID uuid.UUID) ([]*domain.MFADevice, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := r.setTenantRLS(ctx, tx, tenantID); err != nil {
		return nil, err
	}

	rows, err := tx.Query(ctx, `
		SELECT id, tenant_id, user_id, name, secret, algorithm, digits, period, enabled, verified_at, created_at, updated_at
		FROM mfa_devices WHERE user_id = $1 ORDER BY created_at`, userID)
	if err != nil {
		return nil, fmt.Errorf("list mfa devices: %w", err)
	}
	defer rows.Close()

	var devices []*domain.MFADevice
	for rows.Next() {
		device := &domain.MFADevice{}
		if err := rows.Scan(&device.ID, &device.TenantID, &device.UserID, &device.Name, &device.Secret,
			&device.Algorithm, &device.Digits, &device.Period, &device.Enabled, &device.VerifiedAt,
			&device.CreatedAt, &device.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan mfa device: %w", err)
		}
		device.Secret, _ = ggidcrypto.DecryptTOTPSecret(device.Secret)
		devices = append(devices, device)
	}

	tx.Commit(ctx)
	return devices, nil
}

func (r *pgMFADeviceRepository) GetEnabledDevice(ctx context.Context, tenantID, userID uuid.UUID) (*domain.MFADevice, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := r.setTenantRLS(ctx, tx, tenantID); err != nil {
		return nil, err
	}

	device := &domain.MFADevice{}
	err = tx.QueryRow(ctx, `
		SELECT id, tenant_id, user_id, name, secret, algorithm, digits, period, enabled, verified_at, created_at, updated_at
		FROM mfa_devices WHERE user_id = $1 AND enabled = true LIMIT 1`, userID,
	).Scan(&device.ID, &device.TenantID, &device.UserID, &device.Name, &device.Secret,
		&device.Algorithm, &device.Digits, &device.Period, &device.Enabled, &device.VerifiedAt,
		&device.CreatedAt, &device.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // no enabled device — not an error
		}
		return nil, fmt.Errorf("get enabled mfa device: %w", err)
	}

	tx.Commit(ctx)
	return device, nil
}

func (r *pgMFADeviceRepository) UpdateDevice(ctx context.Context, device *domain.MFADevice) error {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := r.setTenantRLS(ctx, tx, device.TenantID); err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		UPDATE mfa_devices
		SET name = $3, enabled = $4, verified_at = $5, updated_at = NOW()
		WHERE id = $1 AND tenant_id = $2`,
		device.ID, device.TenantID, device.Name, device.Enabled, device.VerifiedAt)
	if err != nil {
		return fmt.Errorf("update mfa device: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *pgMFADeviceRepository) DeleteDevice(ctx context.Context, tenantID, id uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := r.setTenantRLS(ctx, tx, tenantID); err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `DELETE FROM mfa_devices WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	if err != nil {
		return fmt.Errorf("delete mfa device: %w", err)
	}

	return tx.Commit(ctx)
}

// Suppress unused import.
var _ = time.Now
