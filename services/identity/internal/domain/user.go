// Package domain defines the core domain entities for the Identity Service.
// These are pure Go types with no external dependencies (no ORM, no proto).
package domain

import (
	"net/netip"
	"time"

	"github.com/google/uuid"
)

// UserStatus represents the lifecycle state of a user account.
type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusLocked   UserStatus = "locked"
	UserStatusDisabled UserStatus = "disabled"
	UserStatusDeleted  UserStatus = "deleted"
)

// IsValid returns true if the status is a recognised value.
func (s UserStatus) IsValid() bool {
	switch s {
	case UserStatusActive, UserStatusLocked, UserStatusDisabled, UserStatusDeleted:
		return true
	}
	return false
}

// CanAuthenticate returns true if a user in this status should be allowed to authenticate.
func (s UserStatus) CanAuthenticate() bool {
	return s == UserStatusActive
}

// User is the central identity entity.
type User struct {
	ID             uuid.UUID
	TenantID       uuid.UUID
	Username       string
	Email          string // denormalised primary email for quick lookups
	Phone          string
	Status         UserStatus
	EmailVerified  bool
	PhoneVerified  bool
	PrimaryEmailID *uuid.UUID
	DisplayName    string
	AvatarURL      string
	Locale         string
	Timezone       string
	LastLoginAt    *time.Time
	LastLoginIP    *netip.Addr
	PasswordHash   string // Argon2id encoded hash; empty for external-only users
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      *time.Time // soft delete
}

// CreateUserInput holds the parameters for creating a new user.
type CreateUserInput struct {
	TenantID    uuid.UUID
	Username    string
	Email       string
	Phone       string
	Password    string // plaintext; will be hashed by the service layer
	DisplayName string
	Locale      string
	Timezone    string
}

// UpdateUserInput holds optional fields for updating a user.
// Only non-nil fields will be applied.
type UpdateUserInput struct {
	Phone       *string
	DisplayName *string
	AvatarURL   *string
	Locale      *string
	Timezone    *string
}

// ListUsersFilter holds query parameters for listing users.
type ListUsersFilter struct {
	TenantID uuid.UUID
	Search   string // matches username or email (ILIKE)
	Status   *UserStatus
	PageSize int
	Offset   int
	SortBy   string // username, email, created_at
	SortDesc bool
}

// ListUsersResult holds paginated results.
type ListUsersResult struct {
	Users      []*User
	Total      int
	NextOffset int // 0 if no more pages
}
