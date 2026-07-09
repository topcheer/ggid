package repository

import (
	"context"
	"fmt"

	"github.com/ggid/ggid/services/policy/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PermissionRepository manages permission persistence.
type PermissionRepository struct {
	db *pgxpool.Pool
}

func NewPermissionRepository(db *pgxpool.Pool) *PermissionRepository {
	return &PermissionRepository{db: db}
}

// Create inserts a new permission.
func (r *PermissionRepository) Create(ctx context.Context, perm *domain.Permission) error {
	query := `
		INSERT INTO permissions (tenant_id, key, name, resource_type, action, description, system_perm)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id`
	return r.db.QueryRow(ctx, query,
		perm.TenantID, perm.Key, perm.Name, perm.ResourceType, perm.Action, perm.Description, perm.SystemPerm,
	).Scan(&perm.ID)
}

// ListByTenant returns permissions for a tenant with pagination.
func (r *PermissionRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*domain.Permission, error) {
	query := `
		SELECT id, tenant_id, key, name, resource_type, action, description, system_perm
		FROM permissions WHERE tenant_id = $1
		ORDER BY resource_type, action LIMIT $2 OFFSET $3`
	rows, err := r.db.Query(ctx, query, tenantID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list permissions: %w", err)
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

// FindByResourceAction retrieves permissions matching a resource type and action.
func (r *PermissionRepository) FindByResourceAction(ctx context.Context, tenantID uuid.UUID, resourceType, action string) ([]*domain.Permission, error) {
	query := `
		SELECT id, tenant_id, key, name, resource_type, action, description, system_perm
		FROM permissions
		WHERE tenant_id = $1 AND resource_type = $2 AND action = $3`
	rows, err := r.db.Query(ctx, query, tenantID, resourceType, action)
	if err != nil {
		return nil, fmt.Errorf("find permissions: %w", err)
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
