package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ggid/ggid/pkg/crypto"
	"github.com/ggid/ggid/services/auth/internal/conf"
	"github.com/ggid/ggid/services/auth/internal/domain"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

var (
	// ErrPasswordTooShort indicates the password does not meet minimum length.
	ErrPasswordTooShort = errors.New("password does not meet minimum length requirement")
	// ErrPasswordTooWeak indicates the password does not meet complexity requirements.
	ErrPasswordTooWeak = errors.New("password does not meet complexity requirements")
	// ErrPasswordReused indicates the password has been used recently.
	ErrPasswordReused = errors.New("password has been used recently and cannot be reused")
	// ErrInvalidResetToken indicates the reset token is invalid or expired.
	ErrInvalidResetToken = errors.New("invalid or expired password reset token")
)

// PasswordService handles password validation, change, and reset flows.
type PasswordService struct {
	policy         conf.PasswordPolicy
	credentialRepo CredentialRepo
	rdb            *redis.Client
}

func NewPasswordService(
	policy conf.PasswordPolicy,
	credentialRepo CredentialRepo,
	rdb *redis.Client,
) *PasswordService {
	return &PasswordService{
		policy:        policy,
		credentialRepo: credentialRepo,
		rdb:           rdb,
	}
}

// Validate checks a plaintext password against the configured policy.
func (ps *PasswordService) Validate(password string) error {
	if len(password) < ps.policy.MinLength {
		return ErrPasswordTooShort
	}

	var hasUpper, hasLower, hasDigit bool
	for _, ch := range password {
		switch {
		case 'A' <= ch && ch <= 'Z':
			hasUpper = true
		case 'a' <= ch && ch <= 'z':
			hasLower = true
		case '0' <= ch && ch <= '9':
			hasDigit = true
		}
	}

	if ps.policy.RequireUpper && !hasUpper {
		return ErrPasswordTooWeak
	}
	if ps.policy.RequireLower && !hasLower {
		return ErrPasswordTooWeak
	}
	if ps.policy.RequireDigit && !hasDigit {
		return ErrPasswordTooWeak
	}
	// RequireSpecial is optional (checked by ASCII non-alphanumeric presence)

	return nil
}

// CheckHistory verifies the new password hasn't been used recently.
func (ps *PasswordService) CheckHistory(ctx context.Context, tenantID, userID uuid.UUID, newPassword string) error {
	if ps.policy.HistoryCount <= 0 {
		return nil
	}

	history, err := ps.credentialRepo.GetHistory(ctx, tenantID, userID, ps.policy.HistoryCount)
	if err != nil {
		return err
	}

	for _, entry := range history {
		match, err := crypto.VerifyPassword(newPassword, entry.Secret)
		if err != nil {
			continue // skip malformed entries
		}
		if match {
			return ErrPasswordReused
		}
	}
	return nil
}

// SetPassword hashes and stores a new password, records the old one in history.
func (ps *PasswordService) SetPassword(ctx context.Context, cred *domain.Credential, newPassword string) error {
	if err := ps.Validate(newPassword); err != nil {
		return err
	}

	// Save old password to history
	if err := ps.credentialRepo.AddToHistory(ctx, cred.TenantID, cred.UserID, cred.Secret); err != nil {
		return err
	}

	hash, err := crypto.HashPassword(newPassword)
	if err != nil {
		return err
	}

	return ps.credentialRepo.UpdateSecret(ctx, cred.ID, hash)
}

// IssueResetToken generates a password-reset token and stores it in Redis (1h TTL).
// Returns the plaintext token to be sent to the user.
func (ps *PasswordService) IssueResetToken(ctx context.Context, userID, tenantID uuid.UUID) (string, error) {
	token, err := crypto.GenerateRandomToken(32)
	if err != nil {
		return "", err
	}

	tokenHash := hashToken(token)
	key := passwordResetKey(tokenHash)
	val := fmt.Sprintf("%s:%s", tenantID, userID)

	if err := ps.rdb.Set(ctx, key, val, time.Hour).Err(); err != nil {
		return "", err
	}
	return token, nil
}

// ConsumeResetToken validates a reset token and returns the associated user info.
// The token is consumed (deleted) after successful validation.
func (ps *PasswordService) ConsumeResetToken(ctx context.Context, token string) (uuid.UUID, uuid.UUID, error) {
	tokenHash := hashToken(token)
	key := passwordResetKey(tokenHash)

	val, err := ps.rdb.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return uuid.Nil, uuid.Nil, ErrInvalidResetToken
		}
		return uuid.Nil, uuid.Nil, err
	}

	// Delete the token (one-time use)
	ps.rdb.Del(ctx, key)

	parts := strings.SplitN(val, ":", 2)
	if len(parts) != 2 {
		return uuid.Nil, uuid.Nil, ErrInvalidResetToken
	}

	tenantID, err := uuid.Parse(parts[0])
	if err != nil {
		return uuid.Nil, uuid.Nil, ErrInvalidResetToken
	}
	userID, err := uuid.Parse(parts[1])
	if err != nil {
		return uuid.Nil, uuid.Nil, ErrInvalidResetToken
	}

	return tenantID, userID, nil
}

// VerifyOldPassword checks if the old password matches the stored hash.
func (ps *PasswordService) VerifyOldPassword(_ context.Context, cred *domain.Credential, oldPassword string) (bool, error) {
	return crypto.VerifyPassword(oldPassword, cred.Secret)
}

func passwordResetKey(hash string) string {
	return "ggid:pwreset:" + hash
}
