package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"
)

const (
	phoneOTPTTL       = 5 * time.Minute
	phoneOTPMaxRetry  = 5 // max resend before rate limiting window resets
)

// SendPhoneOTP generates a 6-digit OTP, stores it in Redis keyed by phone number hash,
// and returns the plaintext OTP (in production, this would be sent via SMS).
// The OTP is valid for 5 minutes.
func (s *AuthService) SendPhoneOTP(ctx context.Context, tenantID, userID uuid.UUID, phone string) (string, error) {
	// Rate limit: prevent OTP spam for the same phone number.
	rlKey := fmt.Sprintf("phoneotp:rl:%s", hashToken(phone))
	count, err := s.rateLimiter.rdb.Incr(ctx, rlKey).Result()
	if err != nil {
		return "", fmt.Errorf("rate limit check: %w", err)
	}
	if count == 1 {
		s.rateLimiter.rdb.Expire(ctx, rlKey, phoneOTPTTL)
	}
	if count > phoneOTPMaxRetry {
		return "", ErrRateLimited
	}

	// Generate 6-digit OTP.
	otp, err := generateNumericOTP(6)
	if err != nil {
		return "", fmt.Errorf("generate OTP: %w", err)
	}

	// Store OTP in Redis with 5min TTL.
	otpKey := fmt.Sprintf("ggid:phoneotp:%s", hashToken(phone))
	val := fmt.Sprintf("%s:%s:%s", tenantID, userID, otp)
	if err := s.rateLimiter.rdb.Set(ctx, otpKey, val, phoneOTPTTL).Err(); err != nil {
		return "", fmt.Errorf("store OTP: %w", err)
	}

	return otp, nil
}

// VerifyPhoneOTP validates the OTP for the given phone number and, if valid,
// creates a session and issues JWT tokens (passwordless login via SMS OTP).
func generateNumericOTP(n int) (string, error) {
	max := big.NewInt(0).Exp(big.NewInt(10), big.NewInt(int64(n)), nil)
	num, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%0*d", n, num), nil
}
