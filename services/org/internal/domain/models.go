// Package domain defines the core domain models for the Org Service.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// Plan defines the subscription tier for a tenant.
type Plan string

const (
	PlanFree       Plan = "free"
	PlanPro        Plan = "pro"
	PlanEnterprise Plan = "enterprise"
)

// TenantStatus defines the lifecycle state of a tenant.
type TenantStatus string

const (
	TenantActive    TenantStatus = "active"
	TenantSuspended TenantStatus = "suspended"
	TenantDeleted   TenantStatus = "deleted"
)

// MembershipStatus defines the state of a membership.
type MembershipStatus string

const (
	MembershipActive  MembershipStatus = "active"
	MembershipInvited MembershipStatus = "invited"
	MembershipRemoved MembershipStatus = "removed"
)

// Tenant represents a customer organization (the top-level multi-tenant entity).
type Tenant struct {
	ID        uuid.UUID
	Name      string
	Slug      string
	Plan      Plan
	Status    TenantStatus
	Settings  map[string]any
	MaxUsers  int
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Organization represents a node in the organizational tree within a tenant.
type Organization struct {
	ID       uuid.UUID
	TenantID uuid.UUID
	ParentID *uuid.UUID
	Name     string
	Path     string // LTREE path for hierarchical queries
	Metadata map[string]any
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Department represents a department within an organization.
type Department struct {
	ID        uuid.UUID
	OrgID     uuid.UUID
	ParentID  *uuid.UUID
	Name      string
	Path      string // LTREE path
	ManagerID *uuid.UUID
	Metadata  map[string]any
	CreatedAt time.Time
}

// Team represents a cross-functional team within an organization.
type Team struct {
	ID          uuid.UUID
	OrgID       uuid.UUID
	Name        string
	Description string
	CreatedBy   uuid.UUID
	CreatedAt   time.Time
}

// Membership represents a user's membership in an org/dept/team.
type Membership struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TenantID  uuid.UUID
	OrgID     uuid.UUID
	DeptID    *uuid.UUID
	TeamID    *uuid.UUID
	Title     string
	Status    MembershipStatus
	JoinedAt  time.Time
	Metadata  map[string]any
}
