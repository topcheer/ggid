package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// DeviceBoundSSO provides device-bound token issuance and verification.
//
// Device-bound SSO ties authentication sessions to a specific device via
// a signed token containing a device_id claim. The token is signed with
// an HMAC-SHA256 secret, ensuring it cannot be tampered with.
type DeviceBoundSSO struct {
	signingKey []byte
}

// NewDeviceBoundSSO creates a new DeviceBoundSSO instance.
// The signingKey is used to HMAC-sign tokens. If empty, a default key
// derived from the package is used (production should set a proper key).
func NewDeviceBoundSSO(signingKey ...[]byte) *DeviceBoundSSO {
	if len(signingKey) > 0 && len(signingKey[0]) > 0 {
		return &DeviceBoundSSO{signingKey: signingKey[0]}
	}
	// Default key — production should always provide a proper key
	return &DeviceBoundSSO{signingKey: []byte("ggid-device-bound-sso-default-key-change-in-prod")}
}

// DeviceToken represents a token bound to a specific device.
type DeviceToken struct {
	Token     string    `json:"token"`
	DeviceID  string    `json:"device_id"`
	UserID    string    `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
	IssuedAt  time.Time `json:"issued_at"`
}

// deviceTokenClaims is the internal JWT-like payload for device-bound tokens.
type deviceTokenClaims struct {
	DeviceID string `json:"device_id"`
	UserID   string `json:"user_id"`
	IssuedAt int64  `json:"iat"`
	ExpiresAt int64 `json:"exp"`
}

// IssueDeviceBoundToken creates a token bound to a specific device.
//
// The token is a signed payload (HMAC-SHA256) containing device_id, user_id,
// and expiry. It can only be verified by VerifyDeviceBoundToken with the
// same signing key.
func (s *DeviceBoundSSO) IssueDeviceBoundToken(deviceID, userID string) (*DeviceToken, error) {
	if deviceID == "" {
		return nil, errors.New("device_id is required")
	}
	if userID == "" {
		return nil, errors.New("user_id is required")
	}

	now := time.Now()
	claims := deviceTokenClaims{
		DeviceID:  deviceID,
		UserID:    userID,
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(1 * time.Hour).Unix(),
	}

	token, err := s.signClaims(claims)
	if err != nil {
		return nil, fmt.Errorf("sign device token: %w", err)
	}

	return &DeviceToken{
		Token:     token,
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

	claims, err := s.verifyClaims(token)
	if err != nil {
		return err
	}

	// Check device binding
	if claims.DeviceID != deviceID {
		return ErrDeviceMismatch
	}

	// Check expiry
	if time.Now().Unix() >= claims.ExpiresAt {
		return ErrTokenExpired
	}

	return nil
}

// signClaims serializes claims to JSON, then HMAC-SHA256 signs the payload.
// Format: base64url(payload).base64url(hmac)
func (s *DeviceBoundSSO) signClaims(claims deviceTokenClaims) (string, error) {
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("marshal claims: %w", err)
	}
	payloadB64 := base64.RawURLEncoding.EncodeToString(payload)

	mac := hmac.New(sha256.New, s.signingKey)
	mac.Write([]byte(payloadB64))
	sig := mac.Sum(nil)
	sigB64 := base64.RawURLEncoding.EncodeToString(sig)

	return payloadB64 + "." + sigB64, nil
}

// verifyClaims checks the HMAC signature and returns the decoded claims.
func (s *DeviceBoundSSO) verifyClaims(token string) (*deviceTokenClaims, error) {
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return nil, errors.New("invalid token format")
	}
	payloadB64 := parts[0]
	sigB64 := parts[1]

	// Verify signature
	sig, err := base64.RawURLEncoding.DecodeString(sigB64)
	if err != nil {
		return nil, fmt.Errorf("decode signature: %w", err)
	}

	mac := hmac.New(sha256.New, s.signingKey)
	mac.Write([]byte(payloadB64))
	expectedSig := mac.Sum(nil)

	if !hmac.Equal(sig, expectedSig) {
		return nil, errors.New("invalid token signature")
	}

	// Decode payload
	payload, err := base64.RawURLEncoding.DecodeString(payloadB64)
	if err != nil {
		return nil, fmt.Errorf("decode payload: %w", err)
	}

	var claims deviceTokenClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("unmarshal claims: %w", err)
	}

	return &claims, nil
}

// Sentinel errors for device-bound SSO.
var (
	ErrDeviceMismatch = errors.New("token is bound to a different device")
	ErrTokenExpired   = errors.New("device-bound token has expired")
)
