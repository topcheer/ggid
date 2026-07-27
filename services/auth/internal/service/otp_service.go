package service

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"
	"time"

	"github.com/ggid/ggid/pkg/crypto"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// OTPChannel defines the delivery channel for one-time passwords.
type OTPChannel string

const (
	OTPSMS   OTPChannel = "sms"
	OTPEmail OTPChannel = "email"
)

// OTPService handles passwordless OTP authentication (SMS/Email).
// OTPs are stored in Redis with a short TTL and rate-limited per identifier.
type OTPService struct {
	rdb          *redis.Client
	smsSender    SMSSender
	emailSender  EmailSender
	credRepo     CredentialRepo // for user lookup by email/phone
}

// NewOTPService creates a new OTP service.
func NewOTPService(rdb *redis.Client, sms SMSSender, email EmailSender, credRepo CredentialRepo) *OTPService {
	return &OTPService{
		rdb:         rdb,
		smsSender:   sms,
		emailSender: email,
		credRepo:    credRepo,
	}
}

// SendOTP generates a 6-digit code, stores it in Redis, and sends via the specified channel.
// Rate limit: 60s between sends to the same identifier (prevents abuse).
func (s *OTPService) SendOTP(ctx context.Context, tenantID uuid.UUID, identifier string, channel OTPChannel) error {
	if s.rdb == nil {
		return fmt.Errorf("redis not configured")
	}
	if identifier == "" {
		return fmt.Errorf("identifier is required")
	}

	// Rate limit: check if OTP was recently sent
	rateKey := fmt.Sprintf("otp_rate:%s:%s", channel, identifier)
	if exists, _ := s.rdb.Exists(ctx, rateKey).Result(); exists > 0 {
		return fmt.Errorf("please wait before requesting another code")
	}

	// Generate 6-digit code using crypto/rand (secure)
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return fmt.Errorf("generate otp code: %w", err)
	}
	code := fmt.Sprintf("%06d", n.Int64())

	// Store OTP in Redis: key=otp:{channel}:{identifier}, TTL=5min
	otpKey := fmt.Sprintf("otp:%s:%s", channel, identifier)
	if err := s.rdb.Set(ctx, otpKey, code, 5*time.Minute).Err(); err != nil {
		return fmt.Errorf("store otp: %w", err)
	}

	// Set rate limit key (60s TTL)
	s.rdb.Set(ctx, rateKey, "1", 60*time.Second)

	// Send via channel
	switch channel {
	case OTPSMS:
		if s.smsSender != nil {
			if err := s.smsSender.SendSMS(identifier, fmt.Sprintf("Your verification code: %s", code)); err != nil {
				slog.Error("OTP SMS send failed", "error", err)
				return fmt.Errorf("failed to send SMS")
			}
		}
	case OTPEmail:
		if s.emailSender != nil {
			if err := s.emailSender.Send([]string{identifier}, "Verification Code", []byte(fmt.Sprintf("Your verification code: %s\n\nThis code expires in 5 minutes.", code))); err != nil {
				slog.Error("OTP email send failed", "error", err)
				return fmt.Errorf("failed to send email")
			}
		}
	default:
		return fmt.Errorf("unsupported channel: %s", channel)
	}

	slog.Info("OTP sent", "channel", channel, "identifier_len", len(identifier))
	return nil
}

// VerifyOTP validates the code and returns an auth ticket for the verified user.
// The ticket is exchanged at the OAuth authorize endpoint for a JWT.
func (s *OTPService) VerifyOTP(ctx context.Context, tenantID uuid.UUID, identifier, code string, channel OTPChannel) (string, error) {
	if s.rdb == nil {
		return "", fmt.Errorf("redis not configured")
	}

	otpKey := fmt.Sprintf("otp:%s:%s", channel, identifier)
	storedCode, err := s.rdb.Get(ctx, otpKey).Result()
	if err != nil {
		return "", fmt.Errorf("invalid or expired code")
	}

	// SECURITY: brute-force protection — track failed attempts per identifier.
	// After 5 failures, delete the OTP and force re-request.
	attemptKey := fmt.Sprintf("otp_attempts:%s:%s", channel, identifier)
	if storedCode != code {
		attempts, _ := s.rdb.Incr(ctx, attemptKey).Result()
		if attempts == 1 {
			s.rdb.Expire(ctx, attemptKey, 5*time.Minute)
		}
		if attempts >= 5 {
			s.rdb.Del(ctx, otpKey, attemptKey)
			return "", fmt.Errorf("too many failed attempts, please request a new code")
		}
		return "", fmt.Errorf("invalid code")
	}

	// Delete OTP after successful verification (single use)
	s.rdb.Del(ctx, otpKey)

	// Look up user by identifier
	cred, err := s.credRepo.FindByIDentifier(ctx, tenantID, identifier)
	if err != nil {
		return "", fmt.Errorf("user lookup failed")
	}
	if cred == nil {
		return "", fmt.Errorf("user not found")
	}

	// Generate auth ticket (same as passkey flow)
	ticket, err := crypto.GenerateRandomToken(32)
	if err != nil {
		return "", fmt.Errorf("generate ticket: %w", err)
	}

	ticketData, _ := json.Marshal(map[string]any{
		"tenant_id": tenantID.String(),
		"user_id":   cred.UserID.String(),
		"scopes":    []string{"openid", "profile"},
		"issued_at": time.Now().Unix(),
		"method":    fmt.Sprintf("otp_%s", channel),
	})

	ticketKey := "auth_ticket:" + ticket
	if err := s.rdb.Set(ctx, ticketKey, ticketData, 30*time.Second).Err(); err != nil {
		return "", fmt.Errorf("store ticket: %w", err)
	}

	return ticket, nil
}
