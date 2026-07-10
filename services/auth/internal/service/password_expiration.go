package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// ErrPasswordExpired indicates the user's password has exceeded its maximum age
// and must be changed before they can proceed.
var ErrPasswordExpired = errors.New("password has expired and must be changed")

// CheckPasswordExpiration checks the most recent password_history entry's created_at
// against the configured MaxAgeDays policy. If MaxAgeDays <= 0, no check is performed.
// Returns ErrPasswordExpired if the password age exceeds the policy.
func (ps *PasswordService) CheckPasswordExpiration(ctx context.Context, tenantID, userID uuid.UUID) error {
	if ps.policy.MaxAgeDays <= 0 {
		return nil
	}

	// Get the most recent history entry (the current password's predecessor).
	// We also check the credential's UpdatedAt as the definitive password change timestamp.
	history, err := ps.credentialRepo.GetHistory(ctx, tenantID, userID, 1)
	if err != nil {
		return err
	}

	// The current password's age is determined by the credential's UpdatedAt field.
	// Password history entries are previous passwords; the current one was set at UpdatedAt.
	cred, err := ps.credentialRepo.FindByUserID(ctx, tenantID, userID)
	if err != nil {
		return err
	}
	if cred == nil {
		return nil // no credential — can't check expiration
	}

	// If the credential was recently changed, it's fresh.
	maxAge := time.Duration(ps.policy.MaxAgeDays) * 24 * time.Hour
	if time.Since(cred.UpdatedAt) > maxAge {
		return ErrPasswordExpired
	}

	// Also check history entries if credential UpdatedAt is unreliable.
	if len(history) > 0 {
		if time.Since(history[0].CreatedAt) > maxAge && time.Since(cred.UpdatedAt) > maxAge {
			return ErrPasswordExpired
		}
	}

	return nil
}

// MustChangePassword returns true if the user's password has expired.
// Convenience method for callers that want a boolean rather than an error.
func (ps *PasswordService) MustChangePassword(ctx context.Context, tenantID, userID uuid.UUID) bool {
	return errors.Is(ps.CheckPasswordExpiration(ctx, tenantID, userID), ErrPasswordExpired)
}
