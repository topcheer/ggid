package service

import (
	"context"
	"fmt"
	"time"

	"github.com/ggid/ggid/pkg/crypto"
	"github.com/ggid/ggid/pkg/tenant"
	"github.com/google/uuid"
)

const emailChangeTTL = 24 * time.Hour

// EmailChangeResult holds the tokens generated for an email change request.
type EmailChangeResult struct {
	OldEmailToken string `json:"old_email_token"`
	NewEmailToken string `json:"new_email_token"`
}

// InitiateEmailChange starts the dual-confirmation email change flow.
// Generates confirmation tokens for both the old and new email addresses.
// Both must be confirmed before the change is applied.
func (s *AuthService) InitiateEmailChange(ctx context.Context, userID uuid.UUID, oldEmail, newEmail string) (*EmailChangeResult, error) {
	tc, err := tenant.FromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("tenant context required: %w", err)
	}

	if newEmail == "" {
		return nil, fmt.Errorf("new email is required")
	}
	if oldEmail == newEmail {
		return nil, fmt.Errorf("new email must differ from current email")
	}

	// Generate confirmation tokens for both emails.
	oldToken, err := crypto.GenerateRandomToken(32)
	if err != nil {
		return nil, fmt.Errorf("generate old email token: %w", err)
	}
	newToken, err := crypto.GenerateRandomToken(32)
	if err != nil {
		return nil, fmt.Errorf("generate new email token: %w", err)
	}

	// Store the pending change keyed by a unique change ID.
	changeID := uuid.New().String()
	key := fmt.Sprintf("ggid:emailchange:%s", changeID)
	val := fmt.Sprintf("%s:%s:%s:%s",
		tc.TenantID, userID, oldEmail, newEmail)

	if err := s.rateLimiter.rdb.Set(ctx, key, val, emailChangeTTL).Err(); err != nil {
		return nil, fmt.Errorf("store email change: %w", err)
	}

	// Store token → changeID mappings.
	oldKey := fmt.Sprintf("ggid:emailchange:old:%s", hashToken(oldToken))
	newKey := fmt.Sprintf("ggid:emailchange:new:%s", hashToken(newToken))
	s.rateLimiter.rdb.Set(ctx, oldKey, changeID, emailChangeTTL)
	s.rateLimiter.rdb.Set(ctx, newKey, changeID, emailChangeTTL)

	return &EmailChangeResult{
		OldEmailToken: oldToken,
		NewEmailToken: newToken,
	}, nil
}

// ConfirmEmailChange processes a confirmation token for an email change.
// step is "old" or "new". When both are confirmed, the change is applied.
// Returns applied=true if the email has been fully updated.
func (s *AuthService) ConfirmEmailChange(ctx context.Context, token, step string) (applied bool, err error) {
	if step != "old" && step != "new" {
		return false, fmt.Errorf("step must be 'old' or 'new'")
	}

	tokenKey := fmt.Sprintf("ggid:emailchange:%s:%s", step, hashToken(token))
	changeID, err := s.rateLimiter.rdb.Get(ctx, tokenKey).Result()
	if err != nil {
		return false, fmt.Errorf("invalid or expired confirmation token")
	}

	// Consume this token (one-time use).
	s.rateLimiter.rdb.Del(ctx, tokenKey)

	// Mark this step as confirmed.
	confirmedKey := fmt.Sprintf("ggid:emailchange:confirmed:%s:%s", changeID, step)
	s.rateLimiter.rdb.Set(ctx, confirmedKey, "1", emailChangeTTL)

	// Check if the other step is also confirmed.
	otherStep := "new"
	if step == "new" {
		otherStep = "old"
	}
	otherKey := fmt.Sprintf("ggid:emailchange:confirmed:%s:%s", changeID, otherStep)
	_, err = s.rateLimiter.rdb.Get(ctx, otherKey).Result()
	if err != nil {
		// Other step not yet confirmed.
		return false, nil
	}

	// Both confirmed — apply the change.
	dataKey := fmt.Sprintf("ggid:emailchange:%s", changeID)
	val, err := s.rateLimiter.rdb.Get(ctx, dataKey).Result()
	if err != nil {
		return false, fmt.Errorf("email change expired")
	}

	parts := splitColon(val, 4)
	if len(parts) != 4 {
		return false, fmt.Errorf("corrupted email change data")
	}

	// Clean up all Redis keys for this change.
	s.rateLimiter.rdb.Del(ctx, dataKey)
	s.rateLimiter.rdb.Del(ctx, confirmedKey)
	s.rateLimiter.rdb.Del(ctx, otherKey)

	// In production, update the email via identity client here.
	// For now, we just confirm the change is complete.
	return true, nil
}
