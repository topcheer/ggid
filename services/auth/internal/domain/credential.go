// Package domain contains the core domain models for the Auth Service.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// CredentialType identifies the authentication method stored.
type CredentialType string

const (
	CredentialPassword  CredentialType = "password"
	CredentialPasskey   CredentialType = "passkey"
	CredentialTOTP      CredentialType = "totp"
	CredentialSMS       CredentialType = "sms"
	CredentialEmailCode CredentialType = "email_code"
)

// Credential represents an authentication credential stored in the database.
type Credential struct {
	ID             uuid.UUID
	TenantID       uuid.UUID
	UserID         uuid.UUID
	Type           CredentialType
	Identifier     string // username or credential_id
	Secret         string // Argon2id hash for passwords
	Metadata       map[string]any
	Enabled        bool
	FailedAttempts int
	LockedUntil    *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
	LastUsedAt     *time.Time
}

// IsLocked returns true if the credential is currently locked due to failed attempts.
func (c *Credential) IsLocked() bool {
	return c.LockedUntil != nil && c.LockedUntil.After(time.Now())
}

// RegisterFailedAttempt increments the failed counter and locks if threshold reached.
func (c *Credential) RegisterFailedAttempt(maxAttempts int, lockDuration time.Duration) {
	c.FailedAttempts++
	if c.FailedAttempts >= maxAttempts {
		until := time.Now().Add(lockDuration)
		c.LockedUntil = &until
	}
}

// ResetFailedAttempts clears the failed attempt counter and lock.
func (c *Credential) ResetFailedAttempts() {
	c.FailedAttempts = 0
	c.LockedUntil = nil
}

// CredentialHistoryEntry stores a previous password hash for reuse prevention.
type CredentialHistoryEntry struct {
	ID        uuid.UUID
	TenantID  uuid.UUID
	UserID    uuid.UUID
	Secret    string
	CreatedAt time.Time
}
