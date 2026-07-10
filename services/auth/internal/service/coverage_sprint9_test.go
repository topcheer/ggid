package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ggid/ggid/pkg/crypto"
	"github.com/ggid/ggid/services/auth/internal/conf"
	"github.com/ggid/ggid/services/auth/internal/domain"
	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// EmailService — VerifyEmailToken corrupted/invalid paths
// ---------------------------------------------------------------------------

func TestCovS9_VerifyEmailToken_CorruptedToken(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	ctx := context.Background()
	emailSvc := NewEmailService(svc.rateLimiter.rdb)

	// Manually insert a corrupted token into Redis.
	token := "corrupt-token-xyz"
	tokenHash := hashToken(token)
	key := fmt.Sprintf("ggid:emailverify:%s", tokenHash)
	svc.rateLimiter.rdb.Set(ctx, key, "corrupted-no-colons", 24*time.Hour)

	_, _, _, err := emailSvc.VerifyEmailToken(ctx, token)
	if err == nil {
		t.Fatal("expected error for corrupted token")
	}
}

func TestCovS9_VerifyEmailToken_InvalidTenantUUID(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	ctx := context.Background()
	emailSvc := NewEmailService(svc.rateLimiter.rdb)

	token := "bad-tenant-token"
	tokenHash := hashToken(token)
	key := fmt.Sprintf("ggid:emailverify:%s", tokenHash)
	svc.rateLimiter.rdb.Set(ctx, key, "not-a-uuid:user-id:email@test.com", 24*time.Hour)

	_, _, _, err := emailSvc.VerifyEmailToken(ctx, token)
	if err == nil {
		t.Fatal("expected error for invalid tenant UUID")
	}
}

func TestCovS9_VerifyEmailToken_InvalidUserUUID(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	ctx := context.Background()
	emailSvc := NewEmailService(svc.rateLimiter.rdb)
	tenantID := uuid.New()

	token := "bad-user-token"
	tokenHash := hashToken(token)
	key := fmt.Sprintf("ggid:emailverify:%s", tokenHash)
	val := fmt.Sprintf("%s:not-a-uuid:email@test.com", tenantID)
	svc.rateLimiter.rdb.Set(ctx, key, val, 24*time.Hour)

	_, _, _, err := emailSvc.VerifyEmailToken(ctx, token)
	if err == nil {
		t.Fatal("expected error for invalid user UUID")
	}
}

func TestCovS9_VerifyEmailToken_NotFound(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	emailSvc := NewEmailService(svc.rateLimiter.rdb)

	_, _, _, err := emailSvc.VerifyEmailToken(context.Background(), "nonexistent-token-12345")
	if err == nil {
		t.Fatal("expected error for non-existent token")
	}
}

func TestCovS9_SendVerificationEmail_Success(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	ctx := context.Background()
	emailSvc := NewEmailService(svc.rateLimiter.rdb)
	tenantID := uuid.New()
	userID := uuid.New()

	token, err := emailSvc.IssueVerificationToken(ctx, tenantID, userID, "test@example.com")
	if err != nil {
		t.Fatalf("IssueVerificationToken: %v", err)
	}
	if token == "" {
		t.Error("expected non-empty token")
	}

	// Verify the token works.
	_, _, email, err := emailSvc.VerifyEmailToken(ctx, token)
	if err != nil {
		t.Fatalf("VerifyEmailToken: %v", err)
	}
	if email != "test@example.com" {
		t.Errorf("email = %s", email)
	}
}

// ---------------------------------------------------------------------------
// AccountLockoutService coverage
// ---------------------------------------------------------------------------

func TestCovS9_AccountLockout_FullCycle(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	rdb := svc.rateLimiter.rdb
	ctx := context.Background()
	tenantID := uuid.New()

	lockout := NewAccountLockoutService(rdb, 3, 10*time.Minute)

	// Initially not locked.
	if lockout.IsLocked(ctx, tenantID, "user1") {
		t.Fatal("expected not locked initially")
	}

	// Record 3 failures.
	for i := 0; i < 3; i++ {
		if err := lockout.RecordFailedAttempt(ctx, tenantID, "user1"); err != nil {
			t.Fatalf("RecordFailedAttempt: %v", err)
		}
	}

	// Now should be locked.
	if !lockout.IsLocked(ctx, tenantID, "user1") {
		t.Fatal("expected locked after 3 failures")
	}

	// Reset.
	lockout.ResetAttempts(ctx, tenantID, "user1")
	if lockout.IsLocked(ctx, tenantID, "user1") {
		t.Fatal("expected unlocked after reset")
	}
}

func TestCovS9_AccountLockout_DefaultValues(t *testing.T) {
	rdb := tRedis(t)
	lockout := NewAccountLockoutService(rdb, 0, 0)
	if lockout.MaxAttempts() != 5 {
		t.Errorf("expected default max 5, got %d", lockout.MaxAttempts())
	}
	if lockout.LockDuration() != 15*time.Minute {
		t.Errorf("expected default 15m, got %v", lockout.LockDuration())
	}
}

// ---------------------------------------------------------------------------
// SendPhoneOTP — rate limiting path
// ---------------------------------------------------------------------------

func TestCovS9_SendPhoneOTP_RateLimited(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	ctx := context.Background()
	tenantID := uuid.New()
	userID := uuid.New()

	// Send OTPs until rate limited (max 5 per 5 minutes).
	var lastErr error
	for i := 0; i < 7; i++ {
		_, lastErr = svc.SendPhoneOTP(ctx, tenantID, userID, "+1234567890")
	}
	if lastErr == nil {
		t.Fatal("expected rate limit error after 7 OTPs")
	}
}

func TestCovS9_VerifyPhoneOTP_InvalidToken(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	ctx := context.Background()

	_, err := svc.VerifyPhoneOTP(ctx, "+1234567890", "123456", "1.2.3.4", "TestAgent")
	if err == nil {
		t.Fatal("expected error for non-existent OTP")
	}
}

func TestCovS9_VerifyPhoneOTP_CorruptedValue(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	ctx := context.Background()
	phone := "+9998887777"

	// Manually insert a corrupted OTP value.
	otpKey := fmt.Sprintf("ggid:phoneotp:%s", hashToken(phone))
	svc.rateLimiter.rdb.Set(ctx, otpKey, "corrupted-no-colons", 5*time.Minute)

	_, err := svc.VerifyPhoneOTP(ctx, phone, "123456", "1.2.3.4", "TestAgent")
	if err == nil {
		t.Fatal("expected error for corrupted OTP value")
	}
}

func TestCovS9_VerifyPhoneOTP_InvalidTenantUUID(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	ctx := context.Background()
	phone := "+1112223333"

	otpKey := fmt.Sprintf("ggid:phoneotp:%s", hashToken(phone))
	svc.rateLimiter.rdb.Set(ctx, otpKey, "bad-uuid:user-id:123456", 5*time.Minute)

	_, err := svc.VerifyPhoneOTP(ctx, phone, "123456", "1.2.3.4", "TestAgent")
	if err == nil {
		t.Fatal("expected error for invalid tenant UUID in OTP")
	}
}

func TestCovS9_VerifyPhoneOTP_InvalidUserUUID(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	ctx := context.Background()
	phone := "+4445556666"
	tenantID := uuid.New()

	otpKey := fmt.Sprintf("ggid:phoneotp:%s", hashToken(phone))
	val := fmt.Sprintf("%s:bad-uuid:123456", tenantID)
	svc.rateLimiter.rdb.Set(ctx, otpKey, val, 5*time.Minute)

	_, err := svc.VerifyPhoneOTP(ctx, phone, "123456", "1.2.3.4", "TestAgent")
	if err == nil {
		t.Fatal("expected error for invalid user UUID in OTP")
	}
}

func TestCovS9_VerifyPhoneOTP_WrongOTP(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	ctx := context.Background()
	phone := "+7778889999"
	tenantID := uuid.New()
	userID := uuid.New()

	otpKey := fmt.Sprintf("ggid:phoneotp:%s", hashToken(phone))
	val := fmt.Sprintf("%s:%s:999888", tenantID, userID)
	svc.rateLimiter.rdb.Set(ctx, otpKey, val, 5*time.Minute)

	_, err := svc.VerifyPhoneOTP(ctx, phone, "000000", "1.2.3.4", "TestAgent")
	if err == nil {
		t.Fatal("expected error for wrong OTP")
	}
}

// ---------------------------------------------------------------------------
// Magic Link — corrupted token paths
// ---------------------------------------------------------------------------

func TestCovS9_VerifyMagicLink_CorruptedValue(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	ctx := context.Background()

	token := "corrupt-magic-link"
	tokenHash := hashToken(token)
	key := fmt.Sprintf("ggid:magiclink:%s", tokenHash)
	svc.rateLimiter.rdb.Set(ctx, key, "no-colons-here", 15*time.Minute)

	_, err := svc.VerifyMagicLink(ctx, token, "1.2.3.4", "TestAgent")
	if err == nil {
		t.Fatal("expected error for corrupted magic link")
	}
}

func TestCovS9_VerifyMagicLink_InvalidTenantUUID(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	ctx := context.Background()

	token := "bad-tenant-magic"
	tokenHash := hashToken(token)
	key := fmt.Sprintf("ggid:magiclink:%s", tokenHash)
	svc.rateLimiter.rdb.Set(ctx, key, "not-uuid:user-id:email@test.com", 15*time.Minute)

	_, err := svc.VerifyMagicLink(ctx, token, "1.2.3.4", "TestAgent")
	if err == nil {
		t.Fatal("expected error for invalid tenant UUID in magic link")
	}
}

func TestCovS9_VerifyMagicLink_InvalidUserUUID(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	ctx := context.Background()
	tenantID := uuid.New()

	token := "bad-user-magic"
	tokenHash := hashToken(token)
	key := fmt.Sprintf("ggid:magiclink:%s", tokenHash)
	val := fmt.Sprintf("%s:not-a-uuid:email@test.com", tenantID)
	svc.rateLimiter.rdb.Set(ctx, key, val, 15*time.Minute)

	_, err := svc.VerifyMagicLink(ctx, token, "1.2.3.4", "TestAgent")
	if err == nil {
		t.Fatal("expected error for invalid user UUID in magic link")
	}
}

func TestCovS9_VerifyMagicLink_NotFound(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)

	_, err := svc.VerifyMagicLink(context.Background(), "nonexistent-token-456", "1.2.3.4", "TestAgent")
	if err == nil {
		t.Fatal("expected error for non-existent magic link token")
	}
}

// ---------------------------------------------------------------------------
// PasswordService — ConsumeResetToken corrupted paths
// ---------------------------------------------------------------------------

func TestCovS9_ConsumeResetToken_CorruptedValue(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	ctx := context.Background()
	ps := svc.GetPasswordService()

	token := "corrupt-reset-token"
	tokenHash := hashToken(token)
	key := fmt.Sprintf("ggid:pwreset:%s", tokenHash)
	svc.rateLimiter.rdb.Set(ctx, key, "no-colon-here", time.Hour)

	_, _, err := ps.ConsumeResetToken(ctx, token)
	if err == nil {
		t.Fatal("expected error for corrupted reset token")
	}
}

func TestCovS9_ConsumeResetToken_InvalidTenantUUID(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	ctx := context.Background()
	ps := svc.GetPasswordService()

	token := "bad-tenant-reset"
	tokenHash := hashToken(token)
	key := fmt.Sprintf("ggid:pwreset:%s", tokenHash)
	svc.rateLimiter.rdb.Set(ctx, key, "not-uuid:user-id", time.Hour)

	_, _, err := ps.ConsumeResetToken(ctx, token)
	if err == nil {
		t.Fatal("expected error for invalid tenant UUID")
	}
}

func TestCovS9_ConsumeResetToken_InvalidUserUUID(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	ctx := context.Background()
	ps := svc.GetPasswordService()
	tenantID := uuid.New()

	token := "bad-user-reset"
	tokenHash := hashToken(token)
	key := fmt.Sprintf("ggid:pwreset:%s", tokenHash)
	val := fmt.Sprintf("%s:not-a-uuid", tenantID)
	svc.rateLimiter.rdb.Set(ctx, key, val, time.Hour)

	_, _, err := ps.ConsumeResetToken(ctx, token)
	if err == nil {
		t.Fatal("expected error for invalid user UUID")
	}
}

func TestCovS9_ConsumeResetToken_NotFound(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	ps := svc.GetPasswordService()

	_, _, err := ps.ConsumeResetToken(context.Background(), "nonexistent-reset-token-789")
	if err == nil {
		t.Fatal("expected error for non-existent reset token")
	}
}

// ---------------------------------------------------------------------------
// PasswordService — IssueResetToken + ConsumeResetToken full cycle
// ---------------------------------------------------------------------------

func TestCovS9_ResetToken_FullCycle(t *testing.T) {
	svc, cr, _, _ := tNewAuthSvc(t)
	ctx := context.Background()
	tenantID := uuid.New()
	userID := uuid.New()

	// Create credential so password service can find it.
	hashed, _ := crypto.HashPassword("OldPass123")
	cr.byUser[userID] = &domain.Credential{
		ID:         uuid.New(),
		TenantID:   tenantID,
		UserID:     userID,
		Identifier: "resetuser",
		Secret:     hashed,
	}

	ps := svc.GetPasswordService()

	// Issue reset token.
	token, err := ps.IssueResetToken(ctx, userID, tenantID)
	if err != nil {
		t.Fatalf("IssueResetToken: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	// Consume it.
	gotTenant, gotUser, err := ps.ConsumeResetToken(ctx, token)
	if err != nil {
		t.Fatalf("ConsumeResetToken: %v", err)
	}
	if gotTenant != tenantID {
		t.Errorf("tenant = %s, want %s", gotTenant, tenantID)
	}
	if gotUser != userID {
		t.Errorf("user = %s, want %s", gotUser, userID)
	}
}

// ---------------------------------------------------------------------------
// generateNumericOTP edge case
// ---------------------------------------------------------------------------

func TestCovS9_GenerateNumericOTP(t *testing.T) {
	otp, err := generateNumericOTP(6)
	if err != nil {
		t.Fatalf("generateNumericOTP: %v", err)
	}
	if len(otp) != 6 {
		t.Errorf("expected 6-digit OTP, got %d digits: %s", len(otp), otp)
	}

	// Test with different lengths.
	for _, n := range []int{4, 8, 10} {
		otp, err := generateNumericOTP(n)
		if err != nil {
			t.Errorf("generateNumericOTP(%d): %v", n, err)
		}
		if len(otp) != n {
			t.Errorf("expected %d-digit OTP, got %d", n, len(otp))
		}
	}
}

// ---------------------------------------------------------------------------
// GetPasswordPolicy / SetPasswordPolicy round trip
// ---------------------------------------------------------------------------

func TestCovS9_PasswordPolicy_RoundTrip(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)

	newPolicy := conf.PasswordPolicy{
		MinLength:       12,
		RequireUpper:    true,
		RequireLower:    true,
		RequireDigit:    true,
		RequireSpecial:  true,
		HistoryCount:    10,
		MaxAttempts:     7,
		LockDuration:    30 * time.Minute,
	}

	svc.SetPasswordPolicy(newPolicy)
	got := svc.GetPasswordPolicy()
	if got.MinLength != 12 {
		t.Errorf("min length = %d, want 12", got.MinLength)
	}
	if got.MaxAttempts != 7 {
		t.Errorf("max attempts = %d, want 7", got.MaxAttempts)
	}
}

// ---------------------------------------------------------------------------
// CheckBruteForce / IsAccountLocked / ResetFailedLogins
// ---------------------------------------------------------------------------

func TestCovS9_BruteForce_FullCycle(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	ctx := context.Background()
	tenantID := uuid.New()

	// Initially no brute force block.
	err := svc.CheckBruteForce(ctx, tenantID, "1.2.3.4", "testuser")
	if err != nil {
		t.Fatalf("CheckBruteForce: %v", err)
	}

	// Record failures until locked via lockout counter.
	cfg := svc.GetPasswordPolicy()
	for i := 0; i < cfg.MaxAttempts; i++ {
		_ = svc.RecordFailedLogin(ctx, tenantID, "testuser")
	}

	// Now should be locked via lockout counter.
	if !svc.IsAccountLocked(ctx, tenantID, "testuser") {
		t.Fatal("expected account locked after max attempts")
	}

	// Call CheckBruteForce enough times to exceed IP sliding window (20/min).
	var bfErr error
	for i := 0; i < 25; i++ {
		bfErr = svc.CheckBruteForce(ctx, tenantID, "1.2.3.4", "testuser")
	}
	if bfErr == nil {
		t.Fatal("expected CheckBruteForce to fail after exceeding IP limit")
	}

	// Reset lockout counter.
	svc.ResetFailedLogins(ctx, tenantID, "testuser")
	if svc.IsAccountLocked(ctx, tenantID, "testuser") {
		t.Fatal("expected unlocked after reset")
	}
}
