package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ggid/ggid/pkg/crypto"
	"github.com/ggid/ggid/services/auth/internal/conf"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// EmailService handles email verification tokens via Redis.
type EmailService struct {
	rdb    *redis.Client
	policy conf.PasswordPolicy
}

// NewEmailService creates a new EmailService.
func NewEmailService(rdb *redis.Client) *EmailService {
	return &EmailService{rdb: rdb}
}

// IssueVerificationToken generates a one-time email verification token.
// Stored in Redis with 24h TTL. Returns the plaintext token.
func (s *EmailService) IssueVerificationToken(ctx context.Context, tenantID, userID uuid.UUID, email string) (string, error) {
	token, err := crypto.GenerateRandomToken(32)
	if err != nil {
		return "", fmt.Errorf("generate verification token: %w", err)
	}

	key := fmt.Sprintf("ggid:emailverify:%s", hashToken(token))
	val := fmt.Sprintf("%s:%s:%s", tenantID, userID, email)

	if err := s.rdb.Set(ctx, key, val, 24*time.Hour).Err(); err != nil {
		return "", fmt.Errorf("store verification token: %w", err)
	}

	return token, nil
}

// VerifyEmailToken validates a verification token and returns the stored data.
// The token is consumed (deleted) after successful verification.
func (s *EmailService) VerifyEmailToken(ctx context.Context, token string) (tenantID, userID uuid.UUID, email string, err error) {
	key := fmt.Sprintf("ggid:emailverify:%s", hashToken(token))
	// Use GetDel for atomic get+delete (prevents TOCTOU token replay).
	// Consistent with ConsumeResetToken and OTP service.
	val, err := s.rdb.GetDel(ctx, key).Result()
	if err != nil {
		return uuid.Nil, uuid.Nil, "", fmt.Errorf("invalid or expired verification token")
	}

	parts := strings.SplitN(val, ":", 3)
	if len(parts) != 3 {
		return uuid.Nil, uuid.Nil, "", fmt.Errorf("corrupted verification token")
	}

	tenantID, err = uuid.Parse(parts[0])
	if err != nil {
		return uuid.Nil, uuid.Nil, "", fmt.Errorf("invalid tenant ID")
	}
	userID, err = uuid.Parse(parts[1])
	if err != nil {
		return uuid.Nil, uuid.Nil, "", fmt.Errorf("invalid user ID")
	}
	email = parts[2]
	return
}

// AccountLockoutService handles brute-force protection via Redis counters.
type AccountLockoutService struct {
	rdb          *redis.Client
	maxAttempts  int
	lockDuration time.Duration
}

// NewAccountLockoutService creates a new AccountLockoutService.
func NewAccountLockoutService(rdb *redis.Client, maxAttempts int, lockDuration time.Duration) *AccountLockoutService {
	if maxAttempts <= 0 {
		maxAttempts = 5
	}
	if lockDuration <= 0 {
		lockDuration = 15 * time.Minute
	}
	return &AccountLockoutService{
		rdb:          rdb,
		maxAttempts:  maxAttempts,
		lockDuration: lockDuration,
	}
}

// IsLocked checks if an account is currently locked.
func (s *AccountLockoutService) IsLocked(ctx context.Context, tenantID uuid.UUID, identifier string) bool {
	key := fmt.Sprintf("ggid:lockout:%s:%s", tenantID, identifier)
	count, err := s.rdb.Get(ctx, key).Int()
	if err != nil {
		return false
	}
	return count >= s.maxAttempts
}

// RecordFailedAttempt increments the failed attempt counter.
// If the threshold is reached, the account is locked for lockDuration.
func (s *AccountLockoutService) RecordFailedAttempt(ctx context.Context, tenantID uuid.UUID, identifier string) error {
	key := fmt.Sprintf("ggid:lockout:%s:%s", tenantID, identifier)
	// SECURITY: Use pipeline for atomic INCR+EXPIRE. Separate calls risk
	// INCR succeeding but EXPIRE failing (e.g., Redis failover) → permanent lock.
	pipe := s.rdb.Pipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, s.lockDuration)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("increment lockout counter: %w", err)
	}
	_ = incr
	return nil
}

// ResetAttempts clears the failed attempt counter after successful login.
func (s *AccountLockoutService) ResetAttempts(ctx context.Context, tenantID uuid.UUID, identifier string) {
	key := fmt.Sprintf("ggid:lockout:%s:%s", tenantID, identifier)
	s.rdb.Del(ctx, key)
}

// MaxAttempts returns the configured max attempts.
func (s *AccountLockoutService) MaxAttempts() int { return s.maxAttempts }

// LockDuration returns the configured lock duration.
func (s *AccountLockoutService) LockDuration() time.Duration { return s.lockDuration }
