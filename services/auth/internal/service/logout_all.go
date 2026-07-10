package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// LogoutAll revokes all sessions and refresh tokens for a user across all devices.
// It also adds all active JWT jti values to a blacklist (if tracked).
func (s *AuthService) LogoutAll(ctx context.Context, tenantID, userID uuid.UUID, exceptSessionID uuid.UUID) error {
	// 1. Revoke all sessions for the user (except the current one if specified).
	if err := s.sessionService.RevokeAllForUser(ctx, tenantID, userID, exceptSessionID); err != nil {
		return fmt.Errorf("revoke all sessions: %w", err)
	}

	// 2. Revoke all refresh tokens for the user.
	if err := s.tokenService.RevokeAllForUser(ctx, tenantID, userID); err != nil {
		return fmt.Errorf("revoke all refresh tokens: %w", err)
	}

	return nil
}
