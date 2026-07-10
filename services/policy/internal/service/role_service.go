package service

import (
	"context"
	"time"

	"github.com/ggid/ggid/pkg/errors"
	"github.com/ggid/ggid/services/policy/internal/domain"
	"github.com/google/uuid"
)

// RoleRepo provides role persistence operations.
type RoleRepo interface {
	Create(ctx context.Context, role *domain.Role) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Role, error)
	ListByTenant(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*domain.Role, error)
	Update(ctx context.Context, role *domain.Role) error
	Delete(ctx context.Context, id uuid.UUID) error
	GrantPermissions(ctx context.Context, roleID uuid.UUID, addIDs []uuid.UUID, conditions map[string]any) error
	RevokePermissions(ctx context.Context, roleID uuid.UUID, permIDs []uuid.UUID) error
}

// PermRepo provides permission persistence operations.
type PermRepo interface {
	Create(ctx context.Context, perm *domain.Permission) error
	ListByTenant(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*domain.Permission, error)
}

// UserRoleRepo provides user-role assignment operations.
type UserRoleRepo interface {
	Assign(ctx context.Context, ur *domain.UserRole) error
	Revoke(ctx context.Context, userID, roleID uuid.UUID, scopeType domain.ScopeType, scopeID uuid.UUID) error
	ListByUser(ctx context.Context, userID uuid.UUID) ([]*domain.UserRole, error)
}

// RoleService handles role CRUD and user-role assignment operations.
type RoleService struct {
	roleRepo     RoleRepo
	permRepo     PermRepo
	userRoleRepo UserRoleRepo
}

// NewRoleService creates a new RoleService.
func NewRoleService(
	roleRepo RoleRepo,
	permRepo PermRepo,
	userRoleRepo UserRoleRepo,
) *RoleService {
	return &RoleService{roleRepo: roleRepo, permRepo: permRepo, userRoleRepo: userRoleRepo}
}

// CreateRole creates a new role in a tenant.
func (s *RoleService) CreateRole(ctx context.Context, tenantID uuid.UUID, key, name, description string, parentRoleID *uuid.UUID) (*domain.Role, error) {
	role := &domain.Role{
		TenantID:     tenantID,
		Key:          key,
		Name:         name,
		Description:  description,
		ParentRoleID: parentRoleID,
	}
	if err := s.roleRepo.Create(ctx, role); err != nil {
		return nil, errors.Wrap(errors.ErrInternal, "create role", err)
	}
	return role, nil
}

// GetRole retrieves a role by ID.
func (s *RoleService) GetRole(ctx context.Context, id uuid.UUID) (*domain.Role, error) {
	return s.roleRepo.GetByID(ctx, id)
}

// ListRoles lists roles for a tenant with pagination.
func (s *RoleService) ListRoles(ctx context.Context, tenantID uuid.UUID, page, pageSize int) ([]*domain.Role, error) {
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 50
	}
	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}
	return s.roleRepo.ListByTenant(ctx, tenantID, pageSize, offset)
}

// UpdateRole updates a role's name, description, or parent.
func (s *RoleService) UpdateRole(ctx context.Context, id uuid.UUID, name, description *string, parentRoleID *uuid.UUID) (*domain.Role, error) {
	role, err := s.roleRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if name != nil {
		role.Name = *name
	}
	if description != nil {
		role.Description = *description
	}
	if parentRoleID != nil {
		role.ParentRoleID = parentRoleID
	}
	if err := s.roleRepo.Update(ctx, role); err != nil {
		return nil, errors.Wrap(errors.ErrInternal, "update role", err)
	}
	return role, nil
}

// DeleteRole deletes a non-system role.
func (s *RoleService) DeleteRole(ctx context.Context, id uuid.UUID) error {
	role, err := s.roleRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if role.SystemRole {
		return errors.New(errors.ErrFailedPrecondition, "cannot delete system role")
	}
	return s.roleRepo.Delete(ctx, id)
}

// AssignRole assigns a role to a user within a specific scope.
func (s *RoleService) AssignRole(ctx context.Context, userID, roleID uuid.UUID, scopeType domain.ScopeType, scopeID, grantedBy uuid.UUID, expiresAt *time.Time) error {
	// Verify role exists.
	if _, err := s.roleRepo.GetByID(ctx, roleID); err != nil {
		return err
	}
	ur := &domain.UserRole{
		UserID:    userID,
		RoleID:    roleID,
		ScopeType: scopeType,
		ScopeID:   scopeID,
		GrantedBy: grantedBy,
		ExpiresAt: expiresAt,
	}
	return s.userRoleRepo.Assign(ctx, ur)
}

// RevokeRole removes a role assignment from a user.
func (s *RoleService) RevokeRole(ctx context.Context, userID, roleID uuid.UUID, scopeType domain.ScopeType, scopeID uuid.UUID) error {
	return s.userRoleRepo.Revoke(ctx, userID, roleID, scopeType, scopeID)
}

// ListUserRoles returns all roles assigned to a user.
func (s *RoleService) ListUserRoles(ctx context.Context, userID uuid.UUID) ([]*domain.UserRole, error) {
	return s.userRoleRepo.ListByUser(ctx, userID)
}

// --- Permission management ---

// CreatePermission creates a new permission.
func (s *RoleService) CreatePermission(ctx context.Context, perm *domain.Permission) (*domain.Permission, error) {
	if err := s.permRepo.Create(ctx, perm); err != nil {
		return nil, errors.Wrap(errors.ErrInternal, "create permission", err)
	}
	return perm, nil
}

// ListPermissions lists permissions for a tenant.
func (s *RoleService) ListPermissions(ctx context.Context, tenantID uuid.UUID, page, pageSize int) ([]*domain.Permission, error) {
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 50
	}
	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}
	return s.permRepo.ListByTenant(ctx, tenantID, pageSize, offset)
}

// GrantPermissionsToRole assigns permissions to a role.
func (s *RoleService) GrantPermissionsToRole(ctx context.Context, roleID uuid.UUID, permissionIDs []uuid.UUID) error {
	return s.roleRepo.GrantPermissions(ctx, roleID, permissionIDs, nil)
}

// RevokePermissionsFromRole removes permissions from a role.
func (s *RoleService) RevokePermissionsFromRole(ctx context.Context, roleID uuid.UUID, permissionIDs []uuid.UUID) error {
	return s.roleRepo.RevokePermissions(ctx, roleID, permissionIDs)
}
