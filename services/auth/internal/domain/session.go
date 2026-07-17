package domain

import (
	"time"

	"github.com/google/uuid"
)

// SessionJTI holds the JTI and token expiry for an active session.
// Used by SessionRevocationManager to revoke access tokens via the Redis blocklist.
type SessionJTI struct {
	SessionID uuid.UUID
	JTI       string
	TokenExp  time.Time
}

// Session represents an active user session.
type Session struct {
	ID         uuid.UUID
	TenantID   uuid.UUID
	UserID     uuid.UUID
	TokenHash  string          // SHA-256 hash of the session token
	DeviceInfo map[string]any  // browser, os, device type
	IPAddress  string
	UserAgent  string
	ExpiresAt  time.Time
	RevokedAt  *time.Time
	CreatedAt  time.Time
	Metadata   map[string]any  // MFA verified, auth context, etc.
	JTI        string          // JWT ID of the access token (CAE Phase 2)
	TokenExp   time.Time       // Expiry of the access token (for Redis ZSET score)
}

// IsActive returns true if the session has not been revoked and hasn't expired.
func (s *Session) IsActive() bool {
	return s.RevokedAt == nil && s.ExpiresAt.After(time.Now())
}

// Revoke marks the session as revoked at the given time.
func (s *Session) Revoke() {
	now := time.Now()
	s.RevokedAt = &now
}

// DeviceInfo represents parsed client device information.
type DeviceInfo struct {
	Browser string `json:"browser"`
	OS      string `json:"os"`
	Device  string `json:"device"`
}
