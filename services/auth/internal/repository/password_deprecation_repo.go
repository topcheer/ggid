package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Password deprecation levels.
const (
	DeprecationOff               = "off"
	DeprecationReadOnly          = "read_only"
	DeprecationMigrationRequired = "migration_required"
	DeprecationDisabled          = "disabled"
)

// PasswordDeprecationConfig represents the tenant-wide password deprecation policy.
type PasswordDeprecationConfig struct {
	TenantID          uuid.UUID  `json:"tenant_id"`
	Level             string     `json:"level"`
	EnforcementDate   *time.Time `json:"enforcement_date,omitempty"`
	GracePeriodDays   int        `json:"grace_period_days"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// PasswordDeprecationRepository manages password deprecation config in PostgreSQL.
type PasswordDeprecationRepository struct {
	pool *pgxpool.Pool
}

func NewPasswordDeprecationRepository(pool *pgxpool.Pool) *PasswordDeprecationRepository {
	return &PasswordDeprecationRepository{pool: pool}
}

// EnsureSchema creates the password_deprecation_config table if it doesn't exist.
func (r *PasswordDeprecationRepository) EnsureSchema(ctx context.Context) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS password_deprecation_config (
			tenant_id          UUID PRIMARY KEY,
			level              TEXT NOT NULL DEFAULT 'off',
			enforcement_date   TIMESTAMPTZ,
			grace_period_days  INT NOT NULL DEFAULT 30,
			updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
		);
	`)
	return err
}

// Get retrieves the password deprecation config for a tenant.
// Returns a default config if none exists.
func (r *PasswordDeprecationRepository) Get(ctx context.Context, tenantID uuid.UUID) (*PasswordDeprecationConfig, error) {
	if r.pool == nil {
		return &PasswordDeprecationConfig{TenantID: tenantID, Level: DeprecationOff, GracePeriodDays: 30}, nil
	}
	row := r.pool.QueryRow(ctx, `
		SELECT tenant_id, level, enforcement_date, grace_period_days, updated_at
		FROM password_deprecation_config
		WHERE tenant_id = $1`,
		tenantID,
	)
	var cfg PasswordDeprecationConfig
	var enforcementDate *time.Time
	err := row.Scan(&cfg.TenantID, &cfg.Level, &enforcementDate, &cfg.GracePeriodDays, &cfg.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return &PasswordDeprecationConfig{TenantID: tenantID, Level: DeprecationOff, GracePeriodDays: 30}, nil
		}
		return nil, err
	}
	cfg.EnforcementDate = enforcementDate
	return &cfg, nil
}

// Upsert creates or updates the password deprecation config for a tenant.
func (r *PasswordDeprecationRepository) Upsert(ctx context.Context, cfg *PasswordDeprecationConfig) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO password_deprecation_config (tenant_id, level, enforcement_date, grace_period_days, updated_at)
		VALUES ($1, $2, $3, $4, now())
		ON CONFLICT (tenant_id) DO UPDATE SET
			level = EXCLUDED.level,
			enforcement_date = EXCLUDED.enforcement_date,
			grace_period_days = EXCLUDED.grace_period_days,
			updated_at = now()`,
		cfg.TenantID, cfg.Level, cfg.EnforcementDate, cfg.GracePeriodDays,
	)
	return err
}

// CheckPasswordLoginAllowed evaluates whether password login is allowed.
// Returns (allowed, mustEnrollPasswordless, reason).
func (r *PasswordDeprecationRepository) CheckPasswordLoginAllowed(ctx context.Context, tenantID uuid.UUID) (bool, bool, string) {
	cfg, err := r.Get(ctx, tenantID)
	if err != nil || cfg == nil {
		return true, false, ""
	}
	switch cfg.Level {
	case DeprecationDisabled:
		return false, false, "password_auth_disabled"
	case DeprecationMigrationRequired:
		return true, true, "passwordless_enrollment_required"
	case DeprecationReadOnly:
		return true, false, ""
	default:
		return true, false, ""
	}
}

// IsPasswordChangeAllowed checks whether password create/change is allowed.
// In read_only mode, password changes are forbidden.
func (r *PasswordDeprecationRepository) IsPasswordChangeAllowed(ctx context.Context, tenantID uuid.UUID) bool {
	cfg, err := r.Get(ctx, tenantID)
	if err != nil || cfg == nil {
		return true
	}
	return cfg.Level != DeprecationReadOnly && cfg.Level != DeprecationDisabled
}

// ValidDeprecationLevels is the set of valid deprecation levels.
var ValidDeprecationLevels = map[string]bool{
	DeprecationOff:               true,
	DeprecationReadOnly:          true,
	DeprecationMigrationRequired: true,
	DeprecationDisabled:          true,
}
