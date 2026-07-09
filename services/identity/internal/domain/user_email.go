package domain

import (
	"time"

	"github.com/google/uuid"
)

// UserEmail represents an email address associated with a user.
// A user can have multiple emails; exactly one should be primary.
type UserEmail struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TenantID  uuid.UUID // denormalised for RLS queries
	Email     string
	IsPrimary bool
	VerifiedAt *time.Time // nil = unverified
	CreatedAt time.Time
}

// EmailVerificationToken is a short-lived token sent to a user's email
// to verify ownership. It is stored in the database (hash only) and
// consumed on verification.
type EmailVerificationToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	EmailID   uuid.UUID
	TokenHash string // SHA-256 hash of the plaintext token
	ExpiresAt time.Time
	ConsumedAt *time.Time // nil = not yet consumed
	CreatedAt time.Time
}
