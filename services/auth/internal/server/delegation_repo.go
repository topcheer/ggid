package server

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserDelegation represents a delegated permission from one user to another.
type UserDelegation struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	DelegatorID string    `json:"delegator_id"`
	DelegateeID string    `json:"delegatee_id"`
	Scopes      []string  `json:"scopes"`
	ResourceID  string    `json:"resource_id,omitempty"`
	ExpiresAt   time.Time `json:"expires_at"`
	RevokedAt   *time.Time `json:"revoked_at,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// delegationRepo manages user_delegations in PostgreSQL.
type delegationRepo struct {
	pool *pgxpool.Pool
}

func newDelegationRepo(pool *pgxpool.Pool) *delegationRepo {
	return &delegationRepo{pool: pool}
}

// NewDelegationRepo is the exported constructor.
func NewDelegationRepo(pool *pgxpool.Pool) *delegationRepo {
	return newDelegationRepo(pool)
}

func (r *delegationRepo) EnsureSchema(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS user_delegations (
			id            TEXT PRIMARY KEY,
			tenant_id     UUID NOT NULL,
			delegator_id  UUID NOT NULL,
			delegatee_id  UUID NOT NULL,
			scopes        TEXT[] NOT NULL DEFAULT '{}',
			resource_id   TEXT DEFAULT '',
			expires_at    TIMESTAMPTZ NOT NULL,
			revoked_at    TIMESTAMPTZ,
			created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_delegations_delegator ON user_delegations(delegator_id);
		CREATE INDEX IF NOT EXISTS idx_delegations_delegatee ON user_delegations(delegatee_id);
		CREATE INDEX IF NOT EXISTS idx_delegations_tenant ON user_delegations(tenant_id);
	`)
	return err
}

// forbiddenScopes are scopes that cannot be delegated.
var forbiddenScopes = map[string]bool{
	"admin": true, "root": true, "superuser": true, "sudo": true,
}

// ValidateDelegation checks delegation rules.
func ValidateDelegation(d *UserDelegation) error {
	if d.DelegatorID == "" || d.DelegateeID == "" {
		return fmt.Errorf("delegator_id and delegatee_id are required")
	}
	if d.DelegatorID == d.DelegateeID {
		return fmt.Errorf("cannot delegate to yourself")
	}
	if len(d.Scopes) == 0 {
		return fmt.Errorf("at least one scope is required")
	}
	for _, s := range d.Scopes {
		if forbiddenScopes[strings.ToLower(s)] {
			return fmt.Errorf("scope %q cannot be delegated (admin-level)", s)
		}
	}
	if d.ExpiresAt.IsZero() || d.ExpiresAt.Before(time.Now()) {
		return fmt.Errorf("expires_at must be in the future")
	}
	return nil
}

func (r *delegationRepo) Create(ctx context.Context, d *UserDelegation) error {
	if d.ID == "" {
		d.ID = "dlg-" + uuid.New().String()[:8]
	}
	if d.CreatedAt.IsZero() {
		d.CreatedAt = time.Now().UTC()
	}
	_, err := r.pool.Exec(ctx,
		`INSERT INTO user_delegations (id, tenant_id, delegator_id, delegatee_id, scopes, resource_id, expires_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		d.ID, d.TenantID, d.DelegatorID, d.DelegateeID, d.Scopes, d.ResourceID, d.ExpiresAt, d.CreatedAt)
	return err
}

func (r *delegationRepo) ListByUser(ctx context.Context, tenantID, userID uuid.UUID) ([]*UserDelegation, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id::text, delegator_id::text, delegatee_id::text, scopes, resource_id,
		        expires_at, revoked_at, created_at
		 FROM user_delegations
		 WHERE tenant_id = $1 AND (delegator_id = $2 OR delegatee_id = $2)
		 ORDER BY created_at DESC`, tenantID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*UserDelegation
	for rows.Next() {
		d := &UserDelegation{}
		if err := rows.Scan(&d.ID, &d.TenantID, &d.DelegatorID, &d.DelegateeID,
			&d.Scopes, &d.ResourceID, &d.ExpiresAt, &d.RevokedAt, &d.CreatedAt); err != nil {
			slog.Warn("delegation scan error", "error", err)
			continue
		}
		result = append(result, d)
	}
	return result, nil
}

func (r *delegationRepo) Revoke(ctx context.Context, id string) error {
	now := time.Now().UTC()
	_, err := r.pool.Exec(ctx,
		`UPDATE user_delegations SET revoked_at = $2 WHERE id = $1 AND revoked_at IS NULL`, id, now)
	return err
}

// CheckDelegation verifies whether a delegatee has a valid delegation for the given scopes.
func (r *delegationRepo) CheckDelegation(ctx context.Context, tenantID, delegatorID, delegateeID uuid.UUID, requiredScope string) (bool, *UserDelegation) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id::text, delegator_id::text, delegatee_id::text, scopes, resource_id,
		        expires_at, revoked_at, created_at
		 FROM user_delegations
		 WHERE tenant_id = $1 AND delegator_id = $2 AND delegatee_id = $3
		   AND revoked_at IS NULL AND expires_at > now()`,
		tenantID, delegatorID, delegateeID)
	if err != nil {
		return false, nil
	}
	defer rows.Close()

	for rows.Next() {
		d := &UserDelegation{}
		if err := rows.Scan(&d.ID, &d.TenantID, &d.DelegatorID, &d.DelegateeID,
			&d.Scopes, &d.ResourceID, &d.ExpiresAt, &d.RevokedAt, &d.CreatedAt); err != nil {
			continue
		}
		// Check if the required scope is in the delegated scopes.
		for _, s := range d.Scopes {
			if s == requiredScope || s == "*" {
				return true, d
			}
		}
	}
	return false, nil
}
