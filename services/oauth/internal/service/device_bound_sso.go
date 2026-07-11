package service

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// DeviceBoundSSO provides device-bound token issuance and verification.
//
// Device-bound SSO ties authentication sessions to a specific device via
// hardware-backed cryptographic keys (WebAuthn/TPM/Secure Enclave).
// This is a research skeleton — production implementation requires:
//   - Integration with WebAuthn credential store
//   - JWT claim injection (device_id)
//   - Token refresh with device assertion
//
// See: docs/research/device-bound-sso-design.md
type DeviceBoundSSO struct {
	// TODO: Add WebAuthn credential store reference
	// TODO: Add JWT signer reference
}

// NewDeviceBoundSSO creates a new DeviceBoundSSO instance.
func NewDeviceBoundSSO() *DeviceBoundSSO {
	return &DeviceBoundSSO{}
}

// DeviceToken represents a token bound to a specific device.
type DeviceToken struct {
	Token      string    `json:"token"`
	DeviceID   string    `json:"device_id"`
	UserID     string    `json:"user_id"`
	ExpiresAt  time.Time `json:"expires_at"`
	IssuedAt   time.Time `json:"issued_at"`
}

// IssueDeviceBoundToken creates a token bound to a specific device.
//
// The token contains a device_id claim that must match on verification.
// Production implementation should:
//  1. Verify the deviceID is registered (WebAuthn credential exists)
//  2. Sign a JWT with device_id, user_id, and expiry claims
//  3. Return the signed token
func (s *DeviceBoundSSO) IssueDeviceBoundToken(deviceID, userID string) (*DeviceToken, error) {
	if deviceID == "" {
		return nil, errors.New("device_id is required")
	}
	if userID == "" {
		return nil, errors.New("user_id is required")
	}

	// TODO: Verify device is registered via WebAuthn
	// TODO: Sign JWT with device_id claim
	now := time.Now()
	return &DeviceToken{
		Token:     fmt.Sprintf("dev-bound:%s|%s|%d", deviceID, userID, now.Unix()),
		DeviceID:  deviceID,
		UserID:    userID,
		IssuedAt:  now,
		ExpiresAt: now.Add(1 * time.Hour),
	}, nil
}

// VerifyDeviceBoundToken verifies that a token is valid AND was issued to the
// specified device. Returns an error if the token is expired, invalid, or
// bound to a different device.
func (s *DeviceBoundSSO) VerifyDeviceBoundToken(token, deviceID string) error {
	if token == "" {
		return errors.New("token is required")
	}
	if deviceID == "" {
		return errors.New("device_id is required")
	}

	// TODO: Parse JWT, extract device_id claim
	// TODO: Compare claim device_id with provided deviceID
	// TODO: Verify token signature and expiry

	// Parse the signed token: dev-bound:{deviceID}|{userID}|{timestamp}
	trimmed := strings.TrimPrefix(token, "dev-bound:")
	parts := strings.SplitN(trimmed, "|", 3)
	if len(parts) != 3 {
		return errors.New("invalid token format")
	}
	tokDeviceID := parts[0]
	var tokUnix int64
	if _, err := fmt.Sscanf(parts[2], "%d", &tokUnix); err != nil {
		return errors.New("invalid token format")
	}

	if tokDeviceID != deviceID {
		return ErrDeviceMismatch
	}

	if time.Now().After(time.Unix(tokUnix, 0).Add(1 * time.Hour)) {
		return ErrTokenExpired
	}

	return nil
}

// Sentinel errors for device-bound SSO.
var (
	ErrDeviceMismatch = errors.New("token is bound to a different device")
	ErrTokenExpired   = errors.New("device-bound token has expired")
)
