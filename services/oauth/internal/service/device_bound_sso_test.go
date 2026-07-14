package service

import (
	"testing"
	"time"
)

func TestDeviceBoundSSO_IssueToken(t *testing.T) {
	s := NewDeviceBoundSSO()

	token, err := s.IssueDeviceBoundToken("device-abc", "user-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token == nil {
		t.Fatal("token should not be nil")
	}
	if token.DeviceID != "device-abc" {
		t.Errorf("expected device-abc, got %s", token.DeviceID)
	}
	if token.UserID != "user-123" {
		t.Errorf("expected user-123, got %s", token.UserID)
	}
	if token.Token == "" {
		t.Error("token should not be empty")
	}
	if token.Token == "" {
		t.Error("token should not be empty")
	}
	if token.ExpiresAt.Before(time.Now()) {
		t.Error("token should not be expired")
	}
}

func TestDeviceBoundSSO_IssueTokenEmptyDeviceID(t *testing.T) {
	s := NewDeviceBoundSSO()

	_, err := s.IssueDeviceBoundToken("", "user-123")
	if err == nil {
		t.Error("expected error for empty device_id")
	}
}

func TestDeviceBoundSSO_IssueTokenEmptyUserID(t *testing.T) {
	s := NewDeviceBoundSSO()

	_, err := s.IssueDeviceBoundToken("device-abc", "")
	if err == nil {
		t.Error("expected error for empty user_id")
	}
}

func TestDeviceBoundSSO_VerifyValidToken(t *testing.T) {
	s := NewDeviceBoundSSO()

	token, err := s.IssueDeviceBoundToken("device-xyz", "user-456")
	if err != nil {
		t.Fatalf("issue error: %v", err)
	}

	err = s.VerifyDeviceBoundToken(token.Token, "device-xyz")
	if err != nil {
		t.Fatalf("verification should pass: %v", err)
	}
}

func TestDeviceBoundSSO_VerifyDeviceMismatch(t *testing.T) {
	s := NewDeviceBoundSSO()

	token, err := s.IssueDeviceBoundToken("device-aaa", "user-111")
	if err != nil {
		t.Fatalf("issue error: %v", err)
	}

	err = s.VerifyDeviceBoundToken(token.Token, "device-bbb")
	if err != ErrDeviceMismatch {
		t.Errorf("expected ErrDeviceMismatch, got %v", err)
	}
}

func TestDeviceBoundSSO_VerifyExpiredToken(t *testing.T) {
	s := NewDeviceBoundSSO()

	// Craft an expired token by directly signing claims with a past expiry
	claims := deviceTokenClaims{
		DeviceID:  "device-old",
		UserID:    "user-222",
		IssuedAt:  time.Now().Add(-2 * time.Hour).Unix(),
		ExpiresAt: time.Now().Add(-1 * time.Hour).Unix(),
	}
	expiredToken, err := s.signClaims(claims)
	if err != nil {
		t.Fatalf("sign claims error: %v", err)
	}

	err = s.VerifyDeviceBoundToken(expiredToken, "device-old")
	if err != ErrTokenExpired {
		t.Errorf("expected ErrTokenExpired, got %v", err)
	}
}

func TestDeviceBoundSSO_VerifyInvalidFormat(t *testing.T) {
	s := NewDeviceBoundSSO()

	err := s.VerifyDeviceBoundToken("garbage-token", "device-abc")
	if err == nil {
		t.Error("expected error for invalid token format")
	}
	if err == ErrDeviceMismatch || err == ErrTokenExpired {
		t.Errorf("expected generic format error, got sentinel: %v", err)
	}
}

func TestDeviceBoundSSO_VerifyEmptyToken(t *testing.T) {
	s := NewDeviceBoundSSO()

	err := s.VerifyDeviceBoundToken("", "device-abc")
	if err == nil {
		t.Error("expected error for empty token")
	}
}

func TestDeviceBoundSSO_VerifyEmptyDeviceID(t *testing.T) {
	s := NewDeviceBoundSSO()

	err := s.VerifyDeviceBoundToken("some-token", "")
	if err == nil {
		t.Error("expected error for empty device_id")
	}
}

func TestDeviceBoundSSO_TokenExpiry(t *testing.T) {
	s := NewDeviceBoundSSO()

	token, err := s.IssueDeviceBoundToken("device-exp", "user-exp")
	if err != nil {
		t.Fatalf("issue error: %v", err)
	}

	// Token should have 1 hour expiry
	duration := token.ExpiresAt.Sub(token.IssuedAt)
	expectedDuration := time.Hour
	if duration != expectedDuration {
		t.Errorf("expected %v expiry, got %v", expectedDuration, duration)
	}
}
