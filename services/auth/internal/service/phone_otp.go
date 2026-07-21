package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ggid/ggid/services/auth/internal/domain"
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
func (s *AuthService) VerifyPhoneOTP(ctx context.Context, phone, otp, ip, userAgent string) (*domain.TokenSet, error) {
	otpKey := fmt.Sprintf("ggid:phoneotp:%s", hashToken(phone))

	val, err := s.rateLimiter.rdb.Get(ctx, otpKey).Result()
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	parts := strings.SplitN(val, ":", 3)
	if len(parts) != 3 {
		return nil, ErrInvalidCredentials
	}

	tenantID, err := uuid.Parse(parts[0])
	if err != nil {
		return nil, ErrInvalidCredentials
	}
	userID, err := uuid.Parse(parts[1])
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if parts[2] != otp {
		return nil, ErrInvalidCredentials
	}

	// OTP verified — delete it (one-time use).
	s.rateLimiter.rdb.Del(ctx, otpKey)

	// Create session.
	_, session, err := s.sessionService.Create(ctx, CreateSessionParams{
		TenantID:  tenantID,
		UserID:    userID,
		IPAddress: ip,
		UserAgent: userAgent,
		TTL:       24 * time.Hour,
	})
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	accessToken, jti, expiresIn, err := s.tokenService.IssueAccessTokenWithJTI(tenantID, userID, []string{"admin"}, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("issue access token: %w", err)
	}
	// Write JTI back to session for CAE revocation (Phase 2).
	s.writeJTI(ctx, session.ID, jti, expiresIn)

	refreshToken, err := s.tokenService.IssueRefreshToken(ctx, tenantID, userID, session.ID)
	if err != nil {
		return nil, fmt.Errorf("issue refresh token: %w", err)
	}

	return &domain.TokenSet{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    expiresIn,
		SessionID:    session.ID.String(),
	}, nil
}

// generateNumericOTP generates a random n-digit numeric OTP.
func generateNumericOTP(n int) (string, error) {
	max := big.NewInt(0).Exp(big.NewInt(10), big.NewInt(int64(n)), nil)
	num, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%0*d", n, num), nil
}
