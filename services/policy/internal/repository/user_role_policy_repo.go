package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ggid/ggid/services/policy/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserRoleRepository manages user-role assignments.
type UserRoleRepository struct {
	db *pgxpool.Pool
}

func NewUserRoleRepository(db *pgxpool.Pool) *UserRoleRepository {
	return &UserRoleRepository{db: db}
}

// Assign creates or updates a user-role assignment.
func (r *UserRoleRepository) Assign(ctx context.Context, ur *domain.UserRole) error {
	query := `
		INSERT INTO user_roles (user_id, role_id, scope_type, scope_id, granted_by, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id, role_id, scope_type, scope_id) DO UPDATE
			SET granted_by = EXCLUDED.granted_by, expires_at = EXCLUDED.expires_at
		RETURNING created_at`
	return r.db.QueryRow(ctx, query,
		ur.UserID, ur.RoleID, ur.ScopeType, ur.ScopeID, ur.GrantedBy, ur.ExpiresAt,
	).Scan(&ur.CreatedAt)
}

// Revoke removes a user-role assignment.
func (r *UserRoleRepository) Revoke(ctx context.Context, userID, roleID uuid.UUID, scopeType domain.ScopeType, scopeID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM user_roles WHERE user_id = $1 AND role_id = $2 AND scope_type = $3 AND scope_id = $4`,
		userID, roleID, scopeType, scopeID)
	return err
}

// ListByUser retrieves all active roles for a user.
func (r *UserRoleRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]*domain.UserRole, error) {
	query := `
		SELECT user_id, role_id, scope_type, scope_id, granted_by, expires_at, created_at
		FROM user_roles
		WHERE user_id = $1 AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY created_at DESC`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list user roles: %w", err)
	}
	defer rows.Close()

	var assignments []*domain.UserRole
	for rows.Next() {
		ur := &domain.UserRole{}
		if err := rows.Scan(&ur.UserID, &ur.RoleID, &ur.ScopeType, &ur.ScopeID, &ur.GrantedBy, &ur.ExpiresAt, &ur.CreatedAt); err != nil {
			return nil, err
		}
		assignments = append(assignments, ur)
	}
	return assignments, nil
}

// GetRoleIDsForUser returns all role IDs assigned to a user (not resolved for inheritance).
func (r *UserRoleRepository) GetRoleIDsForUser(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	query := `
		SELECT role_id FROM user_roles
		WHERE user_id = $1 AND (expires_at IS NULL OR expires_at > NOW())`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("get user role ids: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// GetUserRoles returns all role assignments for a user including ExpiresAt metadata.
// The evaluator uses this to enforce role expiration independently of the SQL filter.
func (r *UserRoleRepository) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]*domain.UserRole, error) {
	query := `
		SELECT user_id, role_id, scope_type, scope_id, granted_by, expires_at, created_at
		FROM user_roles
		WHERE user_id = $1`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("get user roles: %w", err)
	}
	defer rows.Close()

	var assignments []*domain.UserRole
	for rows.Next() {
		ur := &domain.UserRole{}
		if err := rows.Scan(&ur.UserID, &ur.RoleID, &ur.ScopeType, &ur.ScopeID, &ur.GrantedBy, &ur.ExpiresAt, &ur.CreatedAt); err != nil {
			return nil, err
		}
		assignments = append(assignments, ur)
	}
	return assignments, nil
}

// --- Policy repository ---

// PolicyRepository manages ABAC policy persistence.
type PolicyRepository struct {
	db *pgxpool.Pool
}

func NewPolicyRepository(db *pgxpool.Pool) *PolicyRepository {
	return &PolicyRepository{db: db}
}

// Create inserts a new ABAC policy.
func (r *PolicyRepository) Create(ctx context.Context, policy *domain.Policy) error {
	query := `
		INSERT INTO policies (tenant_id, name, description, effect, actions, resources, conditions, priority)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at`
	condJSON, _ := json.Marshal(policy.Conditions)
	return r.db.QueryRow(ctx, query,
		policy.TenantID, policy.Name, policy.Description, policy.Effect,
		policy.Actions, policy.Resources, condJSON, policy.Priority,
	).Scan(&policy.ID, &policy.CreatedAt)
}

// GetByID retrieves a policy by ID.
func (r *PolicyRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Policy, error) {
	policy := &domain.Policy{}
	var condBytes []byte
	query := `
		SELECT id, tenant_id, name, description, effect, actions, resources, conditions, priority, created_at
		FROM policies WHERE id = $1`
	err := r.db.QueryRow(ctx, query, id).Scan(
		&policy.ID, &policy.TenantID, &policy.Name, &policy.Description, &policy.Effect,
		&policy.Actions, &policy.Resources, &condBytes, &policy.Priority, &policy.CreatedAt,
	)
	if err != nil {
		return nil, mapErr(err, "policy", id.String())
	}
	if len(condBytes) > 0 {
		json.Unmarshal(condBytes, &policy.Conditions)
	}
	return policy, nil
}

// ListByTenant returns policies for a tenant.
func (r *PolicyRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*domain.Policy, error) {
	query := `
		SELECT id, tenant_id, name, description, effect, actions, resources, conditions, priority, created_at
		FROM policies WHERE tenant_id = $1
		ORDER BY priority DESC, created_at DESC LIMIT $2 OFFSET $3`
	rows, err := r.db.Query(ctx, query, tenantID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list policies: %w", err)
	}
	defer rows.Close()

	var policies []*domain.Policy
	for rows.Next() {
		p := &domain.Policy{}
		var condBytes []byte
		if err := rows.Scan(&p.ID, &p.TenantID, &p.Name, &p.Description, &p.Effect, &p.Actions, &p.Resources, &condBytes, &p.Priority, &p.CreatedAt); err != nil {
			return nil, err
		}
		if len(condBytes) > 0 {
			json.Unmarshal(condBytes, &p.Conditions)
		}
		policies = append(policies, p)
	}
	return policies, nil
}
// Delete removes a policy and its attachments (cascade).
func (r *PolicyRepository) Delete(ctx context.Context, id uuid.UUID) error {
	cmd, err := r.db.Exec(ctx, `DELETE FROM policies WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete policy: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return notFound("policy", id.String())
	}
	return nil
}

// AttachPolicy links a policy to a principal.
func (r *PolicyRepository) AttachPolicy(ctx context.Context, attachment *domain.PolicyAttachment) error {
	query := `
		INSERT INTO policy_attachments (policy_id, principal_type, principal_id)
		VALUES ($1, $2, $3)
		ON CONFLICT DO NOTHING`
	_, err := r.db.Exec(ctx, query, attachment.PolicyID, attachment.PrincipalType, attachment.PrincipalID)
	return err
}

// DetachPolicy removes a policy-principal link.
func (r *PolicyRepository) DetachPolicy(ctx context.Context, policyID uuid.UUID, principalType domain.PrincipalType, principalID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM policy_attachments WHERE policy_id = $1 AND principal_type = $2 AND principal_id = $3`,
		policyID, principalType, principalID)
	return err
}

// GetPoliciesForPrincipal returns all policies attached to a principal (user or role).
func (r *PolicyRepository) GetPoliciesForPrincipal(ctx context.Context, principalType domain.PrincipalType, principalID uuid.UUID) ([]*domain.Policy, error) {
	query := `
		SELECT p.id, p.tenant_id, p.name, p.description, p.effect, p.actions, p.resources, p.conditions, p.priority, p.created_at
		FROM policies p
		JOIN policy_attachments pa ON p.id = pa.policy_id
		WHERE pa.principal_type = $1 AND pa.principal_id = $2`
	rows, err := r.db.Query(ctx, query, principalType, principalID)
	if err != nil {
		return nil, fmt.Errorf("get policies for principal: %w", err)
	}
	defer rows.Close()

	var policies []*domain.Policy
	for rows.Next() {
		p := &domain.Policy{}
		var condBytes []byte
		if err := rows.Scan(&p.ID, &p.TenantID, &p.Name, &p.Description, &p.Effect, &p.Actions, &p.Resources, &condBytes, &p.Priority, &p.CreatedAt); err != nil {
			return nil, err
		}
		if len(condBytes) > 0 {
			json.Unmarshal(condBytes, &p.Conditions)
		}
		policies = append(policies, p)
	}
	return policies, nil
}

// GetPoliciesForUserAndRoles returns policies attached to the user directly
// and to any of the user's roles.
func (r *PolicyRepository) GetPoliciesForUserAndRoles(ctx context.Context, userID uuid.UUID, roleIDs []uuid.UUID) ([]*domain.Policy, error) {
	query := `
		SELECT DISTINCT p.id, p.tenant_id, p.name, p.description, p.effect, p.actions, p.resources, p.conditions, p.priority, p.created_at
		FROM policies p
		JOIN policy_attachments pa ON p.id = pa.policy_id
		WHERE (pa.principal_type = 'user' AND pa.principal_id = $1)
		   OR (pa.principal_type = 'role' AND pa.principal_id = ANY($2))`
	rows, err := r.db.Query(ctx, query, userID, roleIDs)
	if err != nil {
		return nil, fmt.Errorf("get policies for user+roles: %w", err)
	}
	defer rows.Close()

	var policies []*domain.Policy
	for rows.Next() {
		p := &domain.Policy{}
		var condBytes []byte
		if err := rows.Scan(&p.ID, &p.TenantID, &p.Name, &p.Description, &p.Effect, &p.Actions, &p.Resources, &condBytes, &p.Priority, &p.CreatedAt); err != nil {
			return nil, err
		}
		if len(condBytes) > 0 {
			json.Unmarshal(condBytes, &p.Conditions)
		}
		policies = append(policies, p)
	}
	return policies, nil
}

// GetTenantPolicies returns policies for a tenant that are NOT attached to any
// specific principal (user/role). These are tenant-global policies that apply
// to all users in the tenant. A policy is considered tenant-global if it has
// no entry in policy_attachments.
func (r *PolicyRepository) GetTenantPolicies(ctx context.Context, tenantID uuid.UUID) ([]*domain.Policy, error) {
	query := `
		SELECT p.id, p.tenant_id, p.name, p.description, p.effect, p.actions, p.resources, p.conditions, p.priority, p.created_at
		FROM policies p
		LEFT JOIN policy_attachments pa ON p.id = pa.policy_id
		WHERE p.tenant_id = $1 AND pa.policy_id IS NULL`
	rows, err := r.db.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get tenant policies: %w", err)
	}
	defer rows.Close()

	var policies []*domain.Policy
	for rows.Next() {
		p := &domain.Policy{}
		var condBytes []byte
		if err := rows.Scan(&p.ID, &p.TenantID, &p.Name, &p.Description, &p.Effect, &p.Actions, &p.Resources, &condBytes, &p.Priority, &p.CreatedAt); err != nil {
			return nil, err
		}
		if len(condBytes) > 0 {
			json.Unmarshal(condBytes, &p.Conditions)
		}
		policies = append(policies, p)
	}
	return policies, nil
}