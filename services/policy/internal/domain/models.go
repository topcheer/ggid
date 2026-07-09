// Package domain defines the core domain models for the Policy Engine.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// ScopeType defines the scope at which a role is assigned to a user.
type ScopeType string

const (
	ScopeGlobal       ScopeType = "global"
	ScopeOrganization ScopeType = "organization"
	ScopeDepartment   ScopeType = "department"
	ScopeTeam         ScopeType = "team"
	ScopeResource     ScopeType = "resource"
)

// Effect defines whether an ABAC policy allows or denies access.
type Effect string

const (
	EffectAllow Effect = "allow"
	EffectDeny  Effect = "deny"
)

// PrincipalType defines the kind of entity a policy is attached to.
type PrincipalType string

const (
	PrincipalUser  PrincipalType = "user"
	PrincipalRole  PrincipalType = "role"
	PrincipalGroup PrincipalType = "group"
)

// Role represents an RBAC role with optional inheritance.
type Role struct {
	ID           uuid.UUID
	TenantID     uuid.UUID
	Key          string
	Name         string
	Description  string
	SystemRole   bool
	ParentRoleID *uuid.UUID
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Permission represents a single permission (resource_type + action).
type Permission struct {
	ID           uuid.UUID
	TenantID     uuid.UUID
	Key          string
	Name         string
	ResourceType string
	Action       string
	Description  string
	SystemPerm   bool
}

// RolePermission links a role to a permission with optional ABAC conditions.
type RolePermission struct {
	RoleID       uuid.UUID
	PermissionID uuid.UUID
	Conditions   map[string]any
}

// UserRole represents a role assigned to a user within a scope.
type UserRole struct {
	UserID    uuid.UUID
	RoleID    uuid.UUID
	ScopeType ScopeType
	ScopeID   uuid.UUID
	GrantedBy uuid.UUID
	ExpiresAt *time.Time
	CreatedAt time.Time
}

// Policy represents an ABAC policy in AWS IAM style.
type Policy struct {
	ID          uuid.UUID
	TenantID    uuid.UUID
	Name        string
	Description string
	Effect      Effect
	Actions     []string
	Resources   []string
	Conditions  map[string]any
	Priority    int
	CreatedAt   time.Time
}

// PolicyAttachment links a policy to a principal (user/role/group).
type PolicyAttachment struct {
	PolicyID      uuid.UUID
	PrincipalType PrincipalType
	PrincipalID   uuid.UUID
}

// --- Request/Response DTOs for evaluation ---

// CheckRequest is the input for a permission check.
type CheckRequest struct {
	UserID       uuid.UUID
	TenantID     uuid.UUID
	ResourceType string
	Action       string
	Resource     string
	Conditions   map[string]any
}

// CheckResult is the output of a permission check.
type CheckResult struct {
	Allowed   bool
	Reason    string
	MatchedBy string // e.g. "role:admin" or "policy:deny-sensitive"
}
