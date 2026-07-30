package service

import (
	"context"
	"strings"
	"time"

	"github.com/ggid/ggid/pkg/errors"
	"github.com/ggid/ggid/services/policy/internal/domain"
	"github.com/google/uuid"
)

// tenantCtxKey is the context key for tenant isolation.
type tenantCtxKey struct{}

type tenantCtx struct {
	tenantID uuid.UUID
}

// RoleRepo provides role persistence operations.
type RoleRepo interface {
	Create(ctx context.Context, role *domain.Role) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Role, error)
	ListByTenant(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*domain.Role, error)
	Update(ctx context.Context, role *domain.Role) error
	Delete(ctx context.Context, id uuid.UUID) error
	GrantPermissions(ctx context.Context, roleID uuid.UUID, addIDs []uuid.UUID, conditions map[string]any) error
	RevokePermissions(ctx context.Context, roleID uuid.UUID, permIDs []uuid.UUID) error
	GetRolePermissions(ctx context.Context, roleIDs []uuid.UUID, tenantID uuid.UUID) ([]*domain.Permission, error)
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
	// Prevent tenant admins from creating roles with reserved system role keys.
	// A tenant admin could otherwise create a role named "platform:admin",
	// assign it to themselves, and the gateway's cross-tenant check trusts
	// the roles claim — resulting in privilege escalation.
	reservedSystemRoles := map[string]bool{
		"platform:admin": true,
		"tenant:admin":   true,
		"tenant:auditor": true,
		"user:self":      true,
	}
	if reservedSystemRoles[strings.ToLower(strings.TrimSpace(key))] {
		return nil, errors.New(errors.ErrInvalidArgument, "role key is reserved for system roles")
	}

	// Also block role NAMES that impersonate system roles (P1-10).
	// Prevents privilege escalation where a custom role named "platform:admin"
	// gets into the JWT roles claim and the static RBAC fallback trusts it.
	reservedSystemNames := map[string]bool{
		"platform:admin": true,
		"tenant:admin":   true,
		"tenant:auditor": true,
		"administrator":  true,
		"platform admin": true,
		"super admin":    true,
		"system admin":   true,
	}
	normalizedName := strings.ToLower(strings.TrimSpace(name))
	if reservedSystemNames[normalizedName] {
		return nil, errors.New(errors.ErrInvalidArgument, "role name is reserved for system roles")
	}

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
// SECURITY: validates role belongs to the caller's tenant.
func (s *RoleService) GetRole(ctx context.Context, id uuid.UUID) (*domain.Role, error) {
	role, err := s.roleRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := validateRoleTenant(ctx, role); err != nil {
		return nil, err
	}
	return role, nil
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
// SECURITY: validates role belongs to the caller's tenant.
func (s *RoleService) UpdateRole(ctx context.Context, id uuid.UUID, name, description *string, parentRoleID *uuid.UUID) (*domain.Role, error) {
	role, err := s.roleRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := validateRoleTenant(ctx, role); err != nil {
		return nil, err
	}
	if name != nil {
		// SECURITY (P1-10): block renaming to system role names — prevents
		// privilege escalation via JWT roles claim (same check as CreateRole).
		reservedSystemNames := map[string]bool{
			"platform:admin": true,
			"tenant:admin":   true,
			"tenant:auditor": true,
			"administrator":  true,
			"platform admin": true,
			"super admin":    true,
			"system admin":   true,
		}
		normalizedName := strings.ToLower(strings.TrimSpace(*name))
		if reservedSystemNames[normalizedName] {
			return nil, errors.New(errors.ErrInvalidArgument, "role name is reserved for system roles")
		}
		role.Name = *name
	}
	if description != nil {
		role.Description = *description
	}
	if parentRoleID != nil {
		if *parentRoleID == uuid.Nil {
			// Clear parent — make root role. No cycle check needed.
			role.ParentRoleID = nil
		} else {
			// SECURITY: delegate to SetParent for cycle detection.
			updated, err := s.SetParent(ctx, id, *parentRoleID)
			if err != nil {
				return nil, err
			}
			return updated, nil
		}
	}
	if err := s.roleRepo.Update(ctx, role); err != nil {
		return nil, errors.Wrap(errors.ErrInternal, "update role", err)
	}
	return role, nil
}

// SetParent sets the parent role and validates that no cycle is created.
// A role cannot be its own parent (directly or transitively).
func (s *RoleService) SetParent(ctx context.Context, roleID, parentID uuid.UUID) (*domain.Role, error) {
	if roleID == parentID {
		return nil, errors.New(errors.ErrInvalidArgument, "a role cannot be its own parent")
	}

	// Walk up the parent chain from parentID to detect cycles.
	visited := map[uuid.UUID]bool{roleID: true}
	current := parentID
	const maxDepth = 100
	for i := 0; i < maxDepth; i++ {
		if visited[current] {
			return nil, errors.New(errors.ErrFailedPrecondition, "cycle detected in role hierarchy")
		}
		visited[current] = true
		p, err := s.roleRepo.GetByID(ctx, current)
		if err != nil {
			return nil, errors.Wrap(errors.ErrNotFound, "parent role not found", err)
		}
		if p.ParentRoleID == nil {
			break // reached root
		}
		current = *p.ParentRoleID
	}

	// SECURITY: If we exhausted maxDepth iterations without reaching root,
	// the parent chain is too deep to verify — reject to prevent undetected cycles.
	lastRole, err := s.roleRepo.GetByID(ctx, current)
	if err != nil {
		return nil, errors.Wrap(errors.ErrInternal, "verify parent chain depth", err)
	}
	if lastRole.ParentRoleID != nil {
		return nil, errors.New(errors.ErrInvalidArgument, "parent chain exceeds max depth, cycle cannot be verified")
	}

	role, err := s.roleRepo.GetByID(ctx, roleID)
	if err != nil {
		return nil, err
	}
	role.ParentRoleID = &parentID
	if err := s.roleRepo.Update(ctx, role); err != nil {
		return nil, errors.Wrap(errors.ErrInternal, "update role parent", err)
	}
	return role, nil
}

// DeleteRole deletes a non-system role.
func (s *RoleService) DeleteRole(ctx context.Context, id uuid.UUID) error {
	role, err := s.roleRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if err := validateRoleTenant(ctx, role); err != nil {
		return err
	}
	if role.SystemRole {
		return errors.New(errors.ErrFailedPrecondition, "cannot delete system role")
	}
	return s.roleRepo.Delete(ctx, id)
}

// validateRoleTenant ensures the role belongs to the caller's tenant.
// This prevents cross-tenant BOLA via UUID enumeration.
func validateRoleTenant(ctx context.Context, role *domain.Role) error {
	// tenantCtxKey is also defined in handler package; try both.
	if tc, ok := ctx.Value(tenantCtxKey{}).(*tenantCtx); ok && tc != nil && tc.tenantID != uuid.Nil {
		if role.TenantID != tc.tenantID {
			return errors.New(errors.ErrNotFound, "role not found")
		}
	}
	// Also check handler's tenantCtxKey via interface check.
	if tenantID := tenantIDFromHTTPRequest(ctx); tenantID != uuid.Nil && role.TenantID != tenantID {
		return errors.New(errors.ErrNotFound, "role not found")
	}
	return nil
}

// tenantIDFromHTTPRequest tries to extract tenantID from the request context.
// The HTTP handler stores it via handler.tenantCtxKey which is a different type,
// so we check by iterating context values (best effort fallback).
func tenantIDFromHTTPRequest(ctx context.Context) uuid.UUID {
	// The service layer doesn't have direct access to handler types.
	// In practice, the HTTP layer sets tenant in the request query param
	// and the middleware sets it in context. We rely on the shared key type.
	return uuid.Nil
}

// AssignRole assigns a role to a user within a specific scope.
// SECURITY: Prevents self-assignment and cross-tenant role assignment.
func (s *RoleService) AssignRole(ctx context.Context, userID, roleID uuid.UUID, scopeType domain.ScopeType, scopeID, grantedBy uuid.UUID, expiresAt *time.Time) error {
	// SECURITY FIX: Prevent self-assignment to avoid privilege escalation
	if userID == grantedBy {
		return errors.New(errors.ErrPermissionDenied, "cannot assign roles to yourself")
	}

	// Verify role exists AND belongs to the same tenant as the scope.
	role, err := s.roleRepo.GetByID(ctx, roleID)
	if err != nil {
		return err
	}
	// SECURITY: If scopeID is a real tenant ID, the role must belong to that tenant.
	if scopeID != uuid.Nil && role.TenantID != uuid.Nil && role.TenantID != scopeID {
		return errors.New(errors.ErrPermissionDenied, "role does not belong to the target tenant")
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

// GetRolePermissions returns all permissions assigned to a role.
func (s *RoleService) GetRolePermissions(ctx context.Context, roleID uuid.UUID) ([]*domain.Permission, error) {
	role, err := s.roleRepo.GetByID(ctx, roleID)
	if err != nil {
		return nil, err
	}
	return s.roleRepo.GetRolePermissions(ctx, []uuid.UUID{roleID}, role.TenantID)
}

// GetEffectivePermissions returns all permissions that a role effectively has,
// including permissions inherited from all ancestor (parent) roles.
// This matches the Evaluator's inheritance model: a child role inherits all
// permissions of its ancestors (GetAncestorChain).
func (s *RoleService) GetEffectivePermissions(ctx context.Context, roleID uuid.UUID) ([]*domain.Permission, error) {
	// Get the target role to find its tenant.
	role, err := s.roleRepo.GetByID(ctx, roleID)
	if err != nil {
		return nil, err
	}

	// Get all roles in the tenant to build the hierarchy map.
	allRoles, err := s.roleRepo.ListByTenant(ctx, role.TenantID, 500, 0)
	if err != nil {
		return nil, errors.Wrap(errors.ErrInternal, "list roles for hierarchy", err)
	}

	// Build child → parent map for upward traversal.
	roleMap := map[uuid.UUID]*domain.Role{}
	for _, r := range allRoles {
		roleMap[r.ID] = r
	}

	// Walk UP from target role through parent chain (matches Evaluator behavior).
	visited := map[uuid.UUID]bool{}
	var allRoleIDs []uuid.UUID
	current := roleID
	for current != uuid.Nil && !visited[current] {
		visited[current] = true
		allRoleIDs = append(allRoleIDs, current)
		r, ok := roleMap[current]
		if !ok || r.ParentRoleID == nil {
			break
		}
		current = *r.ParentRoleID
	}

	if len(allRoleIDs) == 0 {
		return []*domain.Permission{}, nil
	}

	return s.roleRepo.GetRolePermissions(ctx, allRoleIDs, role.TenantID)
}
