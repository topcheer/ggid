// Package repository provides data access for the Policy Engine.
package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ggid/ggid/services/policy/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RoleRepository manages role persistence.
type RoleRepository struct {
	db *pgxpool.Pool
}

func NewRoleRepository(db *pgxpool.Pool) *RoleRepository {
	return &RoleRepository{db: db}
}

// Create inserts a new role.
func (r *RoleRepository) Create(ctx context.Context, role *domain.Role) error {
	query := `
		INSERT INTO roles (tenant_id, key, name, description, system_role, parent_role_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at`
	return r.db.QueryRow(ctx, query,
		role.TenantID, role.Key, role.Name, role.Description,
		role.SystemRole, role.ParentRoleID,
	).Scan(&role.ID, &role.CreatedAt, &role.UpdatedAt)
}

// GetByID retrieves a role by its ID.
func (r *RoleRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Role, error) {
	role := &domain.Role{}
	query := `
		SELECT id, tenant_id, key, name, description, system_role, parent_role_id, created_at, updated_at
		FROM roles WHERE id = $1`
	err := r.db.QueryRow(ctx, query, id).Scan(
		&role.ID, &role.TenantID, &role.Key, &role.Name, &role.Description,
		&role.SystemRole, &role.ParentRoleID, &role.CreatedAt, &role.UpdatedAt,
	)
	if err != nil {
		return nil, mapErr(err, "role", id.String())
	}
	return role, nil
}

// GetByKey retrieves a role by tenant + key.
func (r *RoleRepository) GetByKey(ctx context.Context, tenantID uuid.UUID, key string) (*domain.Role, error) {
	role := &domain.Role{}
	query := `
		SELECT id, tenant_id, key, name, description, system_role, parent_role_id, created_at, updated_at
		FROM roles WHERE tenant_id = $1 AND key = $2`
	err := r.db.QueryRow(ctx, query, tenantID, key).Scan(
		&role.ID, &role.TenantID, &role.Key, &role.Name, &role.Description,
		&role.SystemRole, &role.ParentRoleID, &role.CreatedAt, &role.UpdatedAt,
	)
	if err != nil {
		return nil, mapErr(err, "role", key)
	}
	return role, nil
}

// ListByTenant returns all roles for a tenant with pagination.
func (r *RoleRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*domain.Role, error) {
	query := `
		SELECT id, tenant_id, key, name, description, system_role, parent_role_id, created_at, updated_at
		FROM roles WHERE tenant_id = $1
		ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	rows, err := r.db.Query(ctx, query, tenantID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list roles: %w", err)
	}
	defer rows.Close()

	var roles []*domain.Role
	for rows.Next() {
		role := &domain.Role{}
		if err := rows.Scan(
			&role.ID, &role.TenantID, &role.Key, &role.Name, &role.Description,
			&role.SystemRole, &role.ParentRoleID, &role.CreatedAt, &role.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan role: %w", err)
		}
		roles = append(roles, role)
	}
	return roles, nil
}

// Update modifies a role's mutable fields.
func (r *RoleRepository) Update(ctx context.Context, role *domain.Role) error {
	query := `
		UPDATE roles SET name = $2, description = $3, parent_role_id = $4, updated_at = NOW()
		WHERE id = $1 AND system_role = FALSE
		RETURNING updated_at`
	err := r.db.QueryRow(ctx, query, role.ID, role.Name, role.Description, role.ParentRoleID).Scan(&role.UpdatedAt)
	if err != nil {
		return mapErr(err, "role", role.ID.String())
	}
	return nil
}

// Delete removes a non-system role.
func (r *RoleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	cmd, err := r.db.Exec(ctx, `DELETE FROM roles WHERE id = $1 AND system_role = FALSE`, id)
	if err != nil {
		return fmt.Errorf("delete role: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return notFound("role", id.String())
	}
	return nil
}

// GetAncestorChain retrieves a role and all its ancestor roles (for inheritance resolution).
// Uses a recursive CTE to walk the parent_role_id chain.
func (r *RoleRepository) GetAncestorChain(ctx context.Context, roleID uuid.UUID) ([]uuid.UUID, error) {
	query := `
		WITH RECURSIVE role_tree AS (
			SELECT id, parent_role_id FROM roles WHERE id = $1
			UNION
			SELECT r.id, r.parent_role_id FROM roles r
			JOIN role_tree rt ON r.id = rt.parent_role_id
		)
		SELECT id FROM role_tree`
	rows, err := r.db.Query(ctx, query, roleID)
	if err != nil {
		return nil, fmt.Errorf("get role chain: %w", err)
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

// --- Role-Permission management ---

// GrantPermissions assigns permissions to a role.
func (r *RoleRepository) GrantPermissions(ctx context.Context, roleID uuid.UUID, permissionIDs []uuid.UUID, conditions map[string]any) error {
	condJSON, _ := json.Marshal(conditions)
	_, err := r.db.CopyFrom(ctx,
		pgx.Identifier{"role_permissions"},
		[]string{"role_id", "permission_id", "conditions"},
		pgx.CopyFromSlice(len(permissionIDs), func(i int) ([]any, error) {
			return []any{roleID, permissionIDs[i], condJSON}, nil
		}),
	)
	return err
}

// RevokePermissions removes permissions from a role.
func (r *RoleRepository) RevokePermissions(ctx context.Context, roleID uuid.UUID, permissionIDs []uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM role_permissions WHERE role_id = $1 AND permission_id = ANY($2)`,
		roleID, permissionIDs)
	return err
}

// GetRolePermissions retrieves all permissions for a role (including inherited).
// roleIDs should include the role and all its ancestors.
func (r *RoleRepository) GetRolePermissions(ctx context.Context, roleIDs []uuid.UUID) ([]*domain.Permission, error) {
	query := `
		SELECT p.id, p.tenant_id, p.key, p.name, p.resource_type, p.action, p.description, p.system_perm
		FROM permissions p
		JOIN role_permissions rp ON p.id = rp.permission_id
		WHERE rp.role_id = ANY($1)`
	rows, err := r.db.Query(ctx, query, roleIDs)
	if err != nil {
		return nil, fmt.Errorf("get role permissions: %w", err)
	}
	defer rows.Close()

	var perms []*domain.Permission
	for rows.Next() {
		p := &domain.Permission{}
		if err := rows.Scan(&p.ID, &p.TenantID, &p.Key, &p.Name, &p.ResourceType, &p.Action, &p.Description, &p.SystemPerm); err != nil {
			return nil, err
		}
		perms = append(perms, p)
	}
	return perms, nil
}
