package domain

import (
	"time"

	"github.com/google/uuid"
)

// MFADevice represents a registered TOTP authenticator device.
type MFADevice struct {
	ID         uuid.UUID
	TenantID   uuid.UUID
	UserID     uuid.UUID
	Name       string // user-assigned device name, e.g. "iPhone"
	Secret     string // TOTP shared secret (encrypted at rest in production)
	Algorithm  string // SHA1, SHA256, SHA512
	Digits     int    // 6 or 8
	Period     int    // seconds (typically 30)
	Enabled    bool
	VerifiedAt *time.Time // nil until user verifies first code
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// MFAChallenge is a short-lived token issued during login when MFA is required.
// The client must submit a valid TOTP code with this challenge to complete login.
type MFAChallenge struct {
	Token     string
	TenantID  uuid.UUID
	UserID    uuid.UUID
	DeviceID  uuid.UUID
	ExpiresAt time.Time
}

// IsExpired returns true if the challenge has expired.
func (c *MFAChallenge) IsExpired() bool {
	return time.Now().After(c.ExpiresAt)
}
