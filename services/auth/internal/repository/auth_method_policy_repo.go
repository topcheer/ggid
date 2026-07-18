package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AuthMethodPolicy represents a rule governing which auth methods are required or forbidden for a group.
type AuthMethodPolicy struct {
	ID               uuid.UUID  `json:"id"`
	TenantID         uuid.UUID  `json:"tenant_id"`
	GroupID          string     `json:"group_id"`
	RequiredMethods  []string   `json:"required_methods"`
	ForbiddenMethods []string   `json:"forbidden_methods"`
	Priority         int        `json:"priority"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// AuthMethodPolicyRepository manages auth method policy persistence in PostgreSQL.
type AuthMethodPolicyRepository struct {
	pool *pgxpool.Pool
}

func NewAuthMethodPolicyRepository(pool *pgxpool.Pool) *AuthMethodPolicyRepository {
	return &AuthMethodPolicyRepository{pool: pool}
}

// EnsureSchema creates the auth_method_policies table if it doesn't exist.
func (r *AuthMethodPolicyRepository) EnsureSchema(ctx context.Context) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS auth_method_policies (
			id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id         UUID NOT NULL,
			group_id          TEXT NOT NULL,
			required_methods  TEXT[] NOT NULL DEFAULT '{}',
			forbidden_methods TEXT[] NOT NULL DEFAULT '{}',
			priority          INT NOT NULL DEFAULT 0,
			created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
			UNIQUE(tenant_id, group_id)
		);
		CREATE INDEX IF NOT EXISTS idx_auth_method_pol_tenant ON auth_method_policies (tenant_id);
		CREATE INDEX IF NOT EXISTS idx_auth_method_pol_priority ON auth_method_policies (tenant_id, priority DESC);
	`)
	return err
}

// Create inserts a new auth method policy.
func (r *AuthMethodPolicyRepository) Create(ctx context.Context, p *AuthMethodPolicy) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO auth_method_policies (id, tenant_id, group_id, required_methods, forbidden_methods, priority)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		p.ID, p.TenantID, p.GroupID, p.RequiredMethods, p.ForbiddenMethods, p.Priority,
	)
	return err
}

// ListByTenant returns all policies for a tenant, ordered by priority descending.
func (r *AuthMethodPolicyRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*AuthMethodPolicy, error) {
	if r.pool == nil {
		return []*AuthMethodPolicy{}, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, tenant_id, group_id, required_methods, forbidden_methods, priority, created_at, updated_at
		FROM auth_method_policies
		WHERE tenant_id = $1
		ORDER BY priority DESC`,
		tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var policies []*AuthMethodPolicy
	for rows.Next() {
		p, err := scanAuthMethodPolicyRow(rows)
		if err != nil {
			continue
		}
		policies = append(policies, p)
	}
	return policies, nil
}

// GetByTenantAndGroup returns the policy for a specific tenant+group, or nil if none.
func (r *AuthMethodPolicyRepository) GetByTenantAndGroup(ctx context.Context, tenantID uuid.UUID, groupID string) (*AuthMethodPolicy, error) {
	if r.pool == nil {
		return nil, nil
	}
	row := r.pool.QueryRow(ctx, `
		SELECT id, tenant_id, group_id, required_methods, forbidden_methods, priority, created_at, updated_at
		FROM auth_method_policies
		WHERE tenant_id = $1 AND group_id = $2`,
		tenantID, groupID,
	)
	return scanAuthMethodPolicyRow(row)
}

// Update modifies an existing policy.
func (r *AuthMethodPolicyRepository) Update(ctx context.Context, p *AuthMethodPolicy) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		UPDATE auth_method_policies
		SET required_methods = $3, forbidden_methods = $4, priority = $5, updated_at = now()
		WHERE id = $1 AND tenant_id = $2`,
		p.ID, p.TenantID, p.RequiredMethods, p.ForbiddenMethods, p.Priority,
	)
	return err
}

// Delete removes a policy by ID.
func (r *AuthMethodPolicyRepository) Delete(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `DELETE FROM auth_method_policies WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	return err
}

// CheckMethodAllowed evaluates whether a given auth method is allowed for a user's groups.
// Returns (allowed, reason). If forbidden, reason explains which policy blocked it.
// If required methods are set, the method must be in the required list.
func (r *AuthMethodPolicyRepository) CheckMethodAllowed(ctx context.Context, tenantID uuid.UUID, groups []string, method string) (bool, string) {
	if r.pool == nil {
		return true, ""
	}
	for _, groupID := range groups {
		p, err := r.GetByTenantAndGroup(ctx, tenantID, groupID)
		if err != nil || p == nil {
			continue
		}
		// Check forbidden list first.
		for _, fm := range p.ForbiddenMethods {
			if fm == method {
				return false, "method '" + method + "' is forbidden for group '" + groupID + "'"
			}
		}
		// Check required list: if required_methods is non-empty, method must be in it.
		if len(p.RequiredMethods) > 0 {
			found := false
			for _, rm := range p.RequiredMethods {
				if rm == method {
					found = true
					break
				}
			}
			if !found {
				return false, "method '" + method + "' is not in required methods for group '" + groupID + "'"
			}
		}
	}
	return true, ""
}

func scanAuthMethodPolicyRow(row interface {
	Scan(dest ...any) error
}) (*AuthMethodPolicy, error) {
	var p AuthMethodPolicy
	err := row.Scan(
		&p.ID, &p.TenantID, &p.GroupID, &p.RequiredMethods, &p.ForbiddenMethods,
		&p.Priority, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}
