// Package domain defines the Group entity for SCIM 2.0 group persistence.
package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Group represents a SCIM 2.0 Group resource backed by database persistence.
type Group struct {
	ID          uuid.UUID `json:"id"`
	TenantID    uuid.UUID `json:"-"`
	DisplayName string    `json:"displayName"`
	ExternalID  string    `json:"externalId,omitempty"`
	CreatedAt   time.Time `json:"-"`
	UpdatedAt   time.Time `json:"-"`
}

// GroupMember represents a user's membership in a group.
type GroupMember struct {
	ID        uuid.UUID `json:"-"`
	GroupID   uuid.UUID `json:"-"`
	UserID    uuid.UUID `json:"value"`
	UserRef   string    `json:"$ref"`
	UserType  string    `json:"type"` // "User" or "Group"
	Display   string    `json:"display"`
	CreatedAt time.Time `json:"-"`
}

// CreateGroupInput holds parameters for creating a new group.
type CreateGroupInput struct {
	TenantID    uuid.UUID
	DisplayName string
	ExternalID  string
	Members     []GroupMemberInput
}

// GroupMemberInput is used when creating/updating group memberships.
type GroupMemberInput struct {
	UserID uuid.UUID
	Type   string // default "User"
}

// UpdateGroupInput holds optional parameters for updating a group.
type UpdateGroupInput struct {
	DisplayName *string
	ExternalID  *string
}

// GroupListFilter holds query parameters for listing groups.
type GroupListFilter struct {
	TenantID uuid.UUID
	Search   string // matches displayName (ILIKE)
	PageSize int
	Offset   int
}

// GroupListResult holds paginated results for group queries.
type GroupListResult struct {
	Groups     []*Group
	Total      int
	NextOffset int
}

// PatchGroupMembersInput holds parameters for bulk membership changes.
type PatchGroupMembersInput struct {
	GroupID       uuid.UUID
	AddMembers    []GroupMemberInput
	RemoveMembers []uuid.UUID // user IDs to remove
}

// PatchGroupResult contains the outcome of a membership patch.
type PatchGroupResult struct {
	Added   int
	Removed int
}

// String returns a human-readable identifier for the group.
func (g *Group) String() string {
	return g.DisplayName
}

// IsValid checks if the group has required fields set.
func (g *Group) IsValid() bool {
	return g.TenantID != uuid.Nil && g.DisplayName != ""
}

// IsValid checks if the member has required fields.
func (m *GroupMember) IsValid() bool {
	return m.GroupID != uuid.Nil && m.UserID != uuid.Nil
}

// String returns a human-readable identifier for the member.
func (m *GroupMember) String() string {
	return m.Display
}

// PatchOpType represents the type of membership change.
type PatchOpType string

const (
	PatchOpAdd    PatchOpType = "add"
	PatchOpRemove PatchOpType = "remove"
)

// MembershipPatch represents a single membership change operation.
type MembershipPatch struct {
	Op     PatchOpType
	UserID uuid.UUID
	Type   string
}

// MembershipPatchResult summarizes how many memberships were affected.
type MembershipPatchResult struct {
	Added   int
	Removed int
}

// GroupRepository defines the data-access interface for SCIM groups.
// This interface is defined in the domain package to keep dependencies clean.
type GroupRepository interface {
	CreateGroup(ctx context.Context, group *Group) (*Group, error)
	GetGroupByID(ctx context.Context, tenantID, id uuid.UUID) (*Group, error)
	GetGroupByDisplayName(ctx context.Context, tenantID uuid.UUID, displayName string) (*Group, error)
	UpdateGroup(ctx context.Context, tenantID, id uuid.UUID, input *UpdateGroupInput) (*Group, error)
	DeleteGroup(ctx context.Context, tenantID, id uuid.UUID) error
	ListGroups(ctx context.Context, filter *GroupListFilter) (*GroupListResult, error)

	// Membership operations
	ListMembers(ctx context.Context, tenantID, groupID uuid.UUID) ([]*GroupMember, error)
	AddMembers(ctx context.Context, tenantID, groupID uuid.UUID, members []GroupMemberInput) error
	RemoveMembers(ctx context.Context, tenantID, groupID uuid.UUID, userIDs []uuid.UUID) error
	GetMemberGroups(ctx context.Context, tenantID, userID uuid.UUID) ([]*Group, error)
}
