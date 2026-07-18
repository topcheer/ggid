package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TAPPolicy represents the tenant-wide Temporary Access Pass usage policy.
type TAPPolicy struct {
	TenantID       uuid.UUID `json:"tenant_id"`
	AllowedGroups  []string  `json:"allowed_groups"`
	MaxPerDay      int       `json:"max_per_day"`
	TTLMinutes     int       `json:"ttl_minutes"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// TAPPolicyRepository manages TAP policy config in PostgreSQL.
type TAPPolicyRepository struct {
	pool *pgxpool.Pool
}

func NewTAPPolicyRepository(pool *pgxpool.Pool) *TAPPolicyRepository {
	return &TAPPolicyRepository{pool: pool}
}

func (r *TAPPolicyRepository) EnsureSchema(ctx context.Context) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS tap_policy (
			tenant_id      UUID PRIMARY KEY,
			allowed_groups TEXT[] NOT NULL DEFAULT '{}',
			max_per_day    INT NOT NULL DEFAULT 10,
			ttl_minutes   INT NOT NULL DEFAULT 15,
			updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
		);
	`)
	return err
}

func (r *TAPPolicyRepository) Get(ctx context.Context, tenantID uuid.UUID) (*TAPPolicy, error) {
	if r.pool == nil {
		return &TAPPolicy{TenantID: tenantID, AllowedGroups: []string{}, MaxPerDay: 10, TTLMinutes: 15}, nil
	}
	row := r.pool.QueryRow(ctx, `
		SELECT tenant_id, allowed_groups, max_per_day, ttl_minutes, updated_at
		FROM tap_policy WHERE tenant_id = $1`, tenantID)
	var p TAPPolicy
	err := row.Scan(&p.TenantID, &p.AllowedGroups, &p.MaxPerDay, &p.TTLMinutes, &p.UpdatedAt)
	if err != nil {
		return &TAPPolicy{TenantID: tenantID, AllowedGroups: []string{}, MaxPerDay: 10, TTLMinutes: 15}, nil
	}
	return &p, nil
}

func (r *TAPPolicyRepository) Upsert(ctx context.Context, p *TAPPolicy) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO tap_policy (tenant_id, allowed_groups, max_per_day, ttl_minutes, updated_at)
		VALUES ($1, $2, $3, $4, now())
		ON CONFLICT (tenant_id) DO UPDATE SET
			allowed_groups = EXCLUDED.allowed_groups,
			max_per_day = EXCLUDED.max_per_day,
			ttl_minutes = EXCLUDED.ttl_minutes,
			updated_at = now()`,
		p.TenantID, p.AllowedGroups, p.MaxPerDay, p.TTLMinutes)
	return err
}

// IsGroupAllowed checks whether a group is permitted to use TAP.
// Empty allowed_groups = allow all groups.
func (r *TAPPolicyRepository) IsGroupAllowed(ctx context.Context, tenantID uuid.UUID, group string) bool {
	p, err := r.Get(ctx, tenantID)
	if err != nil || p == nil {
		return true
	}
	if len(p.AllowedGroups) == 0 {
		return true
	}
	for _, g := range p.AllowedGroups {
		if g == group {
			return true
		}
	}
	return false
}
