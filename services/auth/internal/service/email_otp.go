package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/ggid/ggid/pkg/tenant"
	"github.com/google/uuid"
)

// EmailOTPService provides email-based one-time password MFA.
// Codes are stored in Redis with a 5-minute TTL.
type EmailOTPService struct {
	store OTPStore
}

// OTPStore is the storage interface for OTP codes.
type OTPStore interface {
	SetOTP(ctx context.Context, key string, code string, ttl time.Duration) error
	GetOTP(ctx context.Context, key string) (string, error)
	DeleteOTP(ctx context.Context, key string) error
}

// NewEmailOTPService creates a new EmailOTPService.
func NewEmailOTPService(store OTPStore) *EmailOTPService {
	return &EmailOTPService{store: store}
}

const (
	emailOTPTTL      = 5 * time.Minute
	emailOTPMaxRetry = 3
)

// SendOTP generates a 6-digit code, stores it hashed in Redis, and returns
// the plaintext code for the caller to send via email.
func (s *EmailOTPService) SendOTP(ctx context.Context, userID uuid.UUID) (string, error) {
	tc, err := tenant.FromContext(ctx)
	if err != nil {
		return "", fmt.Errorf("tenant context required")
	}

	code, err := generateOTPCode()
	if err != nil {
		return "", fmt.Errorf("generate otp: %w", err)
	}

	key := otpKey(tc.TenantID, userID)
	if err := s.store.SetOTP(ctx, key, code, emailOTPTTL); err != nil {
		return "", fmt.Errorf("store otp: %w", err)
	}

	return code, nil
}

// VerifyOTP validates the provided code against the stored value.
// On success, the code is deleted (single-use).
func (s *EmailOTPService) VerifyOTP(ctx context.Context, userID uuid.UUID, code string) error {
	tc, err := tenant.FromContext(ctx)
	if err != nil {
		return fmt.Errorf("tenant context required")
	}

	key := otpKey(tc.TenantID, userID)
	stored, err := s.store.GetOTP(ctx, key)
	if err != nil {
		return ErrOTPNotFound
	}

	if stored != code {
		return ErrInvalidOTPCode
	}

	_ = s.store.DeleteOTP(ctx, key)
	return nil
}

func otpKey(tenantID, userID uuid.UUID) string {
	return fmt.Sprintf("ggid:email_otp:%s:%s", tenantID, userID)
}

func generateOTPCode() (string, error) {
	max := big.NewInt(1000000)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

var (
	ErrOTPNotFound    = fmt.Errorf("otp not found or expired")
	ErrInvalidOTPCode = fmt.Errorf("invalid otp code")
)
