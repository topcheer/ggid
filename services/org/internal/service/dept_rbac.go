package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ggid/ggid/services/org/internal/domain"
	"github.com/google/uuid"
)

// DeptRole represents a department-scoped role assignment.
type DeptRole struct {
	ID           uuid.UUID
	DeptID       uuid.UUID
	UserID       uuid.UUID
	Role         string    // role key (e.g. "dept.admin", "dept.member")
	Permissions  []string  // permission keys scoped to department
	AssignedAt   time.Time
}

// deptRoleStore is in-memory storage for department RBAC.
type deptRoleStore struct {
	mu    sync.RWMutex
	roles map[uuid.UUID]*DeptRole
}

var globalDeptRoles = &deptRoleStore{roles: make(map[uuid.UUID]*DeptRole)}

// AssignDeptRole assigns a department-scoped role to a user.
func AssignDeptRole(ctx context.Context, deptID, userID uuid.UUID, role string, perms []string) (*DeptRole, error) {
	if deptID == uuid.Nil || userID == uuid.Nil {
		return nil, fmt.Errorf("dept_id and user_id are required")
	}
	if role == "" {
		return nil, fmt.Errorf("role is required")
	}
	dr := &DeptRole{
		ID: uuid.New(), DeptID: deptID, UserID: userID,
		Role: role, Permissions: perms, AssignedAt: time.Now().UTC(),
	}
	globalDeptRoles.mu.Lock()
	globalDeptRoles.roles[dr.ID] = dr
	globalDeptRoles.mu.Unlock()
	return dr, nil
}

// ListDeptRoles lists department roles for a user.
func ListDeptRoles(ctx context.Context, userID uuid.UUID) ([]*DeptRole, error) {
	globalDeptRoles.mu.RLock()
	defer globalDeptRoles.mu.RUnlock()
	var out []*DeptRole
	for _, dr := range globalDeptRoles.roles {
		if dr.UserID == userID {
			out = append(out, dr)
		}
	}
	return out, nil
}

// CheckDeptPermission checks if a user has a permission within a department.
func CheckDeptPermission(ctx context.Context, deptID, userID uuid.UUID, permission string) bool {
	globalDeptRoles.mu.RLock()
	defer globalDeptRoles.mu.RUnlock()
	for _, dr := range globalDeptRoles.roles {
		if dr.DeptID == deptID && dr.UserID == userID {
			for _, p := range dr.Permissions {
				if p == permission || p == "*" {
					return true
				}
			}
		}
	}
	return false
}

// ListDeptMembersWithRole returns users who have a specific role in a department.
func ListDeptMembersWithRole(ctx context.Context, deptID uuid.UUID, role string) ([]uuid.UUID, error) {
	globalDeptRoles.mu.RLock()
	defer globalDeptRoles.mu.RUnlock()
	var users []uuid.UUID
	for _, dr := range globalDeptRoles.roles {
		if dr.DeptID == deptID && dr.Role == role {
			users = append(users, dr.UserID)
		}
	}
	return users, nil
}

// ResetDeptRoleStore clears all dept roles (for testing).
func ResetDeptRoleStore() {
	globalDeptRoles.mu.Lock()
	defer globalDeptRoles.mu.Unlock()
	globalDeptRoles.roles = make(map[uuid.UUID]*DeptRole)
}

// Suppress unused import warning.
var _ = domain.MembershipActive
