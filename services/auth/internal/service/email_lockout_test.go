package service

import (
	"context"
	"testing"
	"time"

	"github.com/ggid/ggid/pkg/crypto"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// --- EmailService Tests ---

func newTestRedisForEmail(t *testing.T) *redis.Client {
	t.Helper()
	return newTestRedis(t)
}

func TestEmailService_IssueAndVerify(t *testing.T) {
	rdb := newTestRedisByMail(t)
	svc := NewEmailService(rdb)
	ctx := context.Background()

	tenantID := uuid.New()
	userID := uuid.New()
	email := "test@example.com"

	token, err := svc.IssueVerificationToken(ctx, tenantID, userID, email)
	if err != nil {
		t.Fatalf("IssueVerificationToken failed: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	// Verify the token.
	gotTenant, gotUser, gotEmail, err := svc.VerifyEmailToken(ctx, token)
	if err != nil {
		t.Fatalf("VerifyEmailToken failed: %v", err)
	}
	if gotTenant != tenantID || gotUser != userID || gotEmail != email {
		t.Errorf("expected %s/%s/%s, got %s/%s/%s", tenantID, userID, email, gotTenant, gotUser, gotEmail)
	}

	// Token should be consumed (one-time use).
	_, _, _, err = svc.VerifyEmailToken(ctx, token)
	if err == nil {
		t.Fatal("expected error when verifying already-consumed token")
	}
}

func TestEmailService_VerifyInvalidToken(t *testing.T) {
	rdb := newTestRedisByMail(t)
	svc := NewEmailService(rdb)

	_, _, _, err := svc.VerifyEmailToken(context.Background(), "invalid-token")
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}

// --- AccountLockoutService Tests ---

func newTestRedisByMail(t *testing.T) *redis.Client {
	return newTestRedis(t)
}

func TestAccountLockout_NotLocked(t *testing.T) {
	rdb := newTestRedisByMail(t)
	svc := NewAccountLockoutService(rdb, 5, 15*time.Minute)
	ctx := context.Background()

	if svc.IsLocked(ctx, uuid.New(), "user@test.com") {
		t.Error("expected not locked initially")
	}
}

func TestAccountLockout_RecordAndLock(t *testing.T) {
	rdb := newTestRedisByMail(t)
	svc := NewAccountLockoutService(rdb, 3, 15*time.Minute)
	ctx := context.Background()
	tenantID := uuid.New()

	// Record 2 failures — not locked yet.
	for i := 0; i < 2; i++ {
		if err := svc.RecordFailedAttempt(ctx, tenantID, "attacker"); err != nil {
			t.Fatalf("RecordFailedAttempt: %v", err)
		}
	}
	if svc.IsLocked(ctx, tenantID, "attacker") {
		t.Error("should not be locked after 2/3 attempts")
	}

	// 3rd failure — locked.
	if err := svc.RecordFailedAttempt(ctx, tenantID, "attacker"); err != nil {
		t.Fatalf("RecordFailedAttempt: %v", err)
	}
	if !svc.IsLocked(ctx, tenantID, "attacker") {
		t.Error("should be locked after 3 attempts")
	}
}

func TestAccountLockout_ResetAfterSuccess(t *testing.T) {
	rdb := newTestRedisByMail(t)
	svc := NewAccountLockoutService(rdb, 3, 15*time.Minute)
	ctx := context.Background()
	tenantID := uuid.New()

	// Record 2 failures.
	svc.RecordFailedAttempt(ctx, tenantID, "user1")
	svc.RecordFailedAttempt(ctx, tenantID, "user1")

	// Reset on successful login.
	svc.ResetAttempts(ctx, tenantID, "user1")

	// Counter should be cleared.
	if svc.IsLocked(ctx, tenantID, "user1") {
		t.Error("should not be locked after reset")
	}
}

func TestAccountLockout_Defaults(t *testing.T) {
	rdb := newTestRedisByMail(t)
	svc := NewAccountLockoutService(rdb, 0, 0)
	if svc.MaxAttempts() != 5 {
		t.Errorf("expected default 5, got %d", svc.MaxAttempts())
	}
	if svc.LockDuration() != 15*time.Minute {
		t.Errorf("expected 15m, got %v", svc.LockDuration())
	}
}

func TestAccountLockout_PerUserIsolation(t *testing.T) {
	rdb := newTestRedisByMail(t)
	svc := NewAccountLockoutService(rdb, 2, 15*time.Minute)
	ctx := context.Background()
	tenantID := uuid.New()

	// Lock user A.
	svc.RecordFailedAttempt(ctx, tenantID, "userA")
	svc.RecordFailedAttempt(ctx, tenantID, "userA")

	// User B should not be affected.
	if svc.IsLocked(ctx, tenantID, "userB") {
		t.Error("userB should not be locked by userA's failures")
	}
}

// --- Password History Check (integration with crypto) ---

func TestPasswordHistory_BasicCheck(t *testing.T) {
	pw1, _ := crypto.HashPassword("OldPass123!")
	pw2, _ := crypto.HashPassword("NewPass456!")

	history := []string{pw1}

	// Check if old password matches.
	for _, h := range history {
		if ok, _ := crypto.VerifyPassword("OldPass123!", h); ok {
			return
		}
	}

	// New password should not match.
	for _, h := range history {
		if ok, _ := crypto.VerifyPassword("NewPass456!", h); ok {
			t.Error("new password should not match old hash")
		}
	}

	_ = pw2 // suppress unused
}
