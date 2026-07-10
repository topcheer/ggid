package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCredential_IsLocked_NotLocked(t *testing.T) {
	c := &Credential{FailedAttempts: 2}
	if c.IsLocked() {
		t.Error("expected not locked when LockedUntil is nil")
	}
}

func TestCredential_IsLocked_PastTime(t *testing.T) {
	past := time.Now().Add(-1 * time.Hour)
	c := &Credential{LockedUntil: &past}
	if c.IsLocked() {
		t.Error("expected not locked when LockedUntil is in the past")
	}
}

func TestCredential_IsLocked_FutureTime(t *testing.T) {
	future := time.Now().Add(30 * time.Minute)
	c := &Credential{LockedUntil: &future}
	if !c.IsLocked() {
		t.Error("expected locked when LockedUntil is in the future")
	}
}

func TestCredential_RegisterFailedAttempt(t *testing.T) {
	c := &Credential{FailedAttempts: 0}

	// Increment but below threshold
	c.RegisterFailedAttempt(5, 30*time.Minute)
	if c.FailedAttempts != 1 {
		t.Errorf("expected 1 attempt, got %d", c.FailedAttempts)
	}
	if c.IsLocked() {
		t.Error("should not be locked below threshold")
	}

	// Reach threshold
	c.RegisterFailedAttempt(5, 30*time.Minute)
	c.RegisterFailedAttempt(5, 30*time.Minute)
	c.RegisterFailedAttempt(5, 30*time.Minute)
	c.RegisterFailedAttempt(5, 30*time.Minute)
	if c.FailedAttempts != 5 {
		t.Errorf("expected 5 attempts, got %d", c.FailedAttempts)
	}
	if !c.IsLocked() {
		t.Error("should be locked at threshold")
	}
}

func TestCredential_ResetFailedAttempts(t *testing.T) {
	future := time.Now().Add(1 * time.Hour)
	c := &Credential{FailedAttempts: 5, LockedUntil: &future}

	c.ResetFailedAttempts()
	if c.FailedAttempts != 0 {
		t.Errorf("expected 0 attempts, got %d", c.FailedAttempts)
	}
	if c.LockedUntil != nil {
		t.Error("expected nil LockedUntil after reset")
	}
	if c.IsLocked() {
		t.Error("should not be locked after reset")
	}
}

func TestSession_IsActive_HealthySession(t *testing.T) {
	s := &Session{
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	if !s.IsActive() {
		t.Error("expected active session")
	}
}

func TestSession_IsActive_Expired(t *testing.T) {
	s := &Session{
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}
	if s.IsActive() {
		t.Error("expected inactive for expired session")
	}
}

func TestSession_IsActive_Revoked(t *testing.T) {
	now := time.Now()
	s := &Session{
		ExpiresAt:  time.Now().Add(1 * time.Hour),
		RevokedAt:  &now,
	}
	if s.IsActive() {
		t.Error("expected inactive for revoked session")
	}
}

func TestSession_Revoke(t *testing.T) {
	s := &Session{
		ID:        uuid.New(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	s.Revoke()
	if s.RevokedAt == nil {
		t.Error("expected non-nil RevokedAt")
	}
	if s.IsActive() {
		t.Error("should be inactive after revoke")
	}
}

func TestRefreshToken_IsActive_Healthy(t *testing.T) {
	rt := &RefreshToken{
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
	}
	if !rt.IsActive() {
		t.Error("expected active token")
	}
}

func TestRefreshToken_IsActive_Expired(t *testing.T) {
	rt := &RefreshToken{
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}
	if rt.IsActive() {
		t.Error("expected inactive for expired token")
	}
}

func TestRefreshToken_IsActive_Revoked(t *testing.T) {
	now := time.Now()
	rt := &RefreshToken{
		ExpiresAt: time.Now().Add(1 * time.Hour),
		RevokedAt: &now,
	}
	if rt.IsActive() {
		t.Error("expected inactive for revoked token")
	}
}

func TestRefreshToken_Revoke(t *testing.T) {
	rt := &RefreshToken{
		ID:        uuid.New(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	rt.Revoke()
	if rt.RevokedAt == nil {
		t.Error("expected non-nil RevokedAt")
	}
	if rt.IsActive() {
		t.Error("should be inactive after revoke")
	}
}

func TestMFAChallenge_IsExpired(t *testing.T) {
	// Expired
	c := &MFAChallenge{ExpiresAt: time.Now().Add(-1 * time.Hour)}
	if !c.IsExpired() {
		t.Error("expected expired challenge")
	}
	// Not expired
	c2 := &MFAChallenge{ExpiresAt: time.Now().Add(1 * time.Hour)}
	if c2.IsExpired() {
		t.Error("expected non-expired challenge")
	}
}
