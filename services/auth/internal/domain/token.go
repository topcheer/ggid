package domain

import (
	"time"

	"github.com/google/uuid"
)

// RefreshToken represents an opaque refresh token stored in the database and Redis.
type RefreshToken struct {
	ID          uuid.UUID
	TenantID    uuid.UUID
	UserID      uuid.UUID
	SessionID   uuid.UUID
	ClientID    *uuid.UUID
	TokenHash   string // SHA-256 hash of the opaque token
	Scope       []string
	ExpiresAt   time.Time
	RotatedFrom *uuid.UUID // previous token in the rotation chain
	RevokedAt   *time.Time
	CreatedAt   time.Time
}

// IsActive returns true if the token has not been revoked and hasn't expired.
func (t *RefreshToken) IsActive() bool {
	return t.RevokedAt == nil && t.ExpiresAt.After(time.Now())
}

// Revoke marks the token as revoked.
func (t *RefreshToken) Revoke() {
	now := time.Now()
	t.RevokedAt = &now
}

// TokenSet is the response issued after successful authentication or token refresh.
type TokenSet struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"` // always "Bearer"
	ExpiresIn    int    `json:"expires_in"` // access token TTL in seconds
	SessionID    string `json:"session_id"`
	// MFA challenge — populated when MFA is required but not yet completed.
	MFARequired  bool   `json:"mfa_required,omitempty"`
	MFAChallenge string `json:"mfa_challenge,omitempty"`
	// Password expiration — populated when the user's password has expired.
	MustChangePassword bool `json:"must_change_password,omitempty"`
}
