package service

import (
	"context"
	"fmt"
	"time"

	"github.com/ggid/ggid/pkg/tenant"
	"github.com/ggid/ggid/services/auth/internal/domain"
	"github.com/ggid/ggid/services/auth/internal/repository"
	"github.com/google/uuid"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

// MFAService handles TOTP device registration and verification.
type MFAService struct {
	repo repository.MFADeviceRepository
}

// NewMFAService creates a new MFAService.
func NewMFAService(repo repository.MFADeviceRepository) *MFAService {
	return &MFAService{repo: repo}
}

// SetupResponse is returned by SetupMFA.
type SetupResponse struct {
	DeviceID  string `json:"device_id"`
	Secret    string `json:"secret"`
	QRCodeURI string `json:"qr_code_uri"` // otpauth:// URI for QR code generation
}

// SetupMFA generates a new TOTP secret for the user and returns it for QR code display.
// The device is created in disabled state — it must be verified before activation.
func (s *MFAService) SetupMFA(ctx context.Context, userID uuid.UUID, deviceName string) (*SetupResponse, error) {
	tc, err := tenant.FromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("tenant context required")
	}

	// Check if user already has an enabled device.
	existing, _ := s.repo.GetEnabledDevice(ctx, tc.TenantID, userID)
	if existing != nil {
		return nil, fmt.Errorf("MFA already enabled — disable first to reconfigure")
	}

	// Generate TOTP key.
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "GGID",
		AccountName: userID.String(),
		Period:      30,
		Digits:      otp.DigitsSix,
		Algorithm:   otp.AlgorithmSHA1,
	})
	if err != nil {
		return nil, fmt.Errorf("generate totp key: %w", err)
	}

	if deviceName == "" {
		deviceName = "default"
	}

	device := &domain.MFADevice{
		ID:        uuid.New(),
		TenantID:  tc.TenantID,
		UserID:    userID,
		Name:      deviceName,
		Secret:    key.Secret(),
		Algorithm: "SHA1",
		Digits:    6,
		Period:    30,
		Enabled:   false, // not enabled until verified
	}

	if err := s.repo.CreateDevice(ctx, device); err != nil {
		return nil, fmt.Errorf("create mfa device: %w", err)
	}

	return &SetupResponse{
		DeviceID:  device.ID.String(),
		Secret:    key.Secret(),
		QRCodeURI: key.URL(),
	}, nil
}

// VerifyMFA verifies a TOTP code. On first verification, the device is enabled.
// On subsequent verifications, it returns success if the code is valid.
func (s *MFAService) VerifyMFA(ctx context.Context, deviceID uuid.UUID, code string) error {
	tc, err := tenant.FromContext(ctx)
	if err != nil {
		return fmt.Errorf("tenant context required")
	}

	device, err := s.repo.GetDeviceByID(ctx, tc.TenantID, deviceID)
	if err != nil {
		return fmt.Errorf("device not found: %w", err)
	}

	// Validate TOTP code.
	valid := totp.Validate(code, device.Secret)
	if !valid {
		return ErrInvalidMFACode
	}

	// If not yet verified, enable the device.
	if !device.Enabled {
		now := time.Now()
		device.Enabled = true
		device.VerifiedAt = &now
		if err := s.repo.UpdateDevice(ctx, device); err != nil {
			return fmt.Errorf("enable device: %w", err)
		}
	}

	return nil
}

// VerifyUserCode finds the user's enabled device and validates the code.
// Used during login MFA challenge.
func (s *MFAService) VerifyUserCode(ctx context.Context, tenantID, userID uuid.UUID, code string) error {
	device, err := s.repo.GetEnabledDevice(ctx, tenantID, userID)
	if err != nil {
		return fmt.Errorf("no enabled MFA device: %w", err)
	}
	if device == nil {
		return fmt.Errorf("no enabled MFA device")
	}

	valid := totp.Validate(code, device.Secret)
	if !valid {
		return ErrInvalidMFACode
	}
	return nil
}

// DisableMFA removes the user's MFA device.
func (s *MFAService) DisableMFA(ctx context.Context, deviceID uuid.UUID) error {
	tc, err := tenant.FromContext(ctx)
	if err != nil {
		return fmt.Errorf("tenant context required")
	}

	return s.repo.DeleteDevice(ctx, tc.TenantID, deviceID)
}

// HasMFAEnabled returns true if the user has any enabled MFA device.
func (s *MFAService) HasMFAEnabled(ctx context.Context, tenantID, userID uuid.UUID) bool {
	device, _ := s.repo.GetEnabledDevice(ctx, tenantID, userID)
	return device != nil
}

// ListDevices returns all MFA devices for a user.
func (s *MFAService) ListDevices(ctx context.Context, userID uuid.UUID) ([]*domain.MFADevice, error) {
	tc, err := tenant.FromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("tenant context required")
	}
	return s.repo.ListDevicesByUser(ctx, tc.TenantID, userID)
}

// ErrInvalidMFACode is returned when a TOTP code is invalid.
var ErrInvalidMFACode = fmt.Errorf("invalid MFA code")
