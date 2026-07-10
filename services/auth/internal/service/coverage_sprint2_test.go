package service

import (
	"context"
	"testing"
	"time"

	"github.com/ggid/ggid/pkg/authprovider"
	"github.com/ggid/ggid/services/auth/internal/conf"
	"github.com/ggid/ggid/services/auth/internal/domain"
	"github.com/google/uuid"
)

// Test coverage for new auth service methods: brute force, lockout, MFA enforcement,
// session timeout, trusted device, login attempts, password policy, email change,
// password history, webauthn challenge.

func TestAuthService_UpdatePasswordPolicy(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	minLen := 12
	upper := true
	lower := true
	digit := true
	special := true

	err := svc.UpdatePasswordPolicy(&minLen, &upper, &lower, &digit, &special, []string{"weak", "password"})
	if err != nil {
		t.Fatalf("UpdatePasswordPolicy: %v", err)
	}

	policy := svc.PasswordPolicy()
	if policy.MinLength != 12 {
		t.Errorf("expected min_length=12, got %d", policy.MinLength)
	}
	if !policy.RequireUpper || !policy.RequireLower || !policy.RequireDigit || !policy.RequireSpecial {
		t.Error("policy flags not updated")
	}
	if len(policy.Blacklist) != 2 {
		t.Errorf("expected 2 blacklist entries, got %d", len(policy.Blacklist))
	}
}

func TestAuthService_UpdatePasswordPolicy_InvalidMinLen(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	badLen := -1
	err := svc.UpdatePasswordPolicy(&badLen, nil, nil, nil, nil, nil)
	if err == nil {
		t.Error("expected error for negative min_length")
	}
}

func TestAuthService_SetPasswordPolicy(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	newPolicy := conf.PasswordPolicy{
		MinLength: 16, RequireUpper: true, RequireLower: true,
		RequireDigit: true, RequireSpecial: true,
		HistoryCount: 10, MaxAttempts: 3,
	}
	svc.SetPasswordPolicy(newPolicy)
	if svc.PasswordPolicy().MinLength != 16 {
		t.Error("SetPasswordPolicy did not apply")
	}
}

func TestAuthService_ForceMFA(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	ctx, tid := tCtxTenant()

	// Initially false
	if svc.IsForceMFA(ctx, tid) {
		t.Error("expected ForceMFA=false initially")
	}

	// Enable
	if err := svc.SetForceMFA(ctx, tid, true); err != nil {
		t.Fatalf("SetForceMFA: %v", err)
	}
	if !svc.IsForceMFA(ctx, tid) {
		t.Error("expected ForceMFA=true after enable")
	}

	// Disable
	if err := svc.SetForceMFA(ctx, tid, false); err != nil {
		t.Fatalf("SetForceMFA disable: %v", err)
	}
	if svc.IsForceMFA(ctx, tid) {
		t.Error("expected ForceMFA=false after disable")
	}
}

func TestAuthService_AccountLockout(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	ctx, tid := tCtxTenant()
	identifier := "testuser@example.com"

	// Initially not locked
	if svc.IsAccountLocked(ctx, tid, identifier) {
		t.Error("expected account not locked initially")
	}

	// Record failures up to threshold
	cfg := svc.cfg.Password
	for i := 0; i < cfg.MaxAttempts; i++ {
		if err := svc.RecordFailedLogin(ctx, tid, identifier); err != nil {
			t.Fatalf("RecordFailedLogin[%d]: %v", i, err)
		}
	}

	// Should be locked now
	if !svc.IsAccountLocked(ctx, tid, identifier) {
		t.Error("expected account locked after max attempts")
	}

	// Reset clears the lock
	svc.ResetFailedLogins(ctx, tid, identifier)
	if svc.IsAccountLocked(ctx, tid, identifier) {
		t.Error("expected account unlocked after reset")
	}
}

func TestAuthService_BruteForce_NoLimit(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	ctx, tid := tCtxTenant()

	// A few requests should be within limits
	for i := 0; i < 5; i++ {
		if err := svc.CheckBruteForce(ctx, tid, "192.168.1.1", "user5"); err != nil {
			t.Fatalf("CheckBruteForce[%d]: %v", i, err)
		}
	}
}

func TestAuthService_BruteForce_IPLimitExceeded(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	ctx, tid := tCtxTenant()

	// Exceed per-IP limit (20/min)
	var err error
	for i := 0; i < 25; i++ {
		err = svc.CheckBruteForce(ctx, tid, "10.0.0.99", "user_ip_test")
		if err != nil {
			break
		}
	}
	if err == nil {
		t.Error("expected rate limit error after exceeding IP limit")
	}
}

func TestAuthService_BruteForce_UserLimitExceeded(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	ctx, tid := tCtxTenant()

	// Use different IPs each time to bypass IP limit, but same username
	var err error
	for i := 0; i < 12; i++ {
		ip := "172.16." + string(rune('0'+i)) + ".1"
		err = svc.CheckBruteForce(ctx, tid, ip, "limited_user")
		if err != nil {
			break
		}
	}
	if err == nil {
		t.Error("expected rate limit error after exceeding username limit")
	}
}

func TestAuthService_SessionTimeout_Absolute(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	ctx := context.Background()
	svc.cfg.SessionTimeout.AbsoluteTimeout = 1 * time.Millisecond
	svc.cfg.SessionTimeout.IdleTimeout = 0

	sessID := uuid.New()
	oldCreation := time.Now().Add(-1 * time.Hour)

	err := svc.CheckSessionTimeout(ctx, sessID, oldCreation)
	if err != ErrSessionExpired {
		t.Errorf("expected ErrSessionExpired for old session, got %v", err)
	}
}

func TestAuthService_SessionTimeout_Valid(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	ctx := context.Background()
	svc.cfg.SessionTimeout.AbsoluteTimeout = 8 * time.Hour
	svc.cfg.SessionTimeout.IdleTimeout = 30 * time.Minute

	sessID := uuid.New()
	recentCreation := time.Now()

	err := svc.CheckSessionTimeout(ctx, sessID, recentCreation)
	if err != nil {
		t.Errorf("expected nil for valid session, got %v", err)
	}
}

func TestAuthService_SessionTimeout_IdleExpired(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	ctx := context.Background()
	svc.cfg.SessionTimeout.AbsoluteTimeout = 8 * time.Hour
	svc.cfg.SessionTimeout.IdleTimeout = 1 * time.Millisecond

	sessID := uuid.New()

	// First call: sets last-activity, should be fine
	_ = svc.CheckSessionTimeout(ctx, sessID, time.Now())
	// Wait a bit
	time.Sleep(5 * time.Millisecond)
	// Second call: idle timeout expired
	err := svc.CheckSessionTimeout(ctx, sessID, time.Now())
	if err != ErrSessionExpired {
		t.Errorf("expected ErrSessionExpired for idle timeout, got %v", err)
	}
}

func TestAuthService_TrustedDevice(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	ctx, tid := tCtxTenant()
	userID := uuid.New()
	fp := "device_fingerprint_abc"

	// Initially not trusted
	if svc.IsTrustedDevice(ctx, tid, userID, fp) {
		t.Error("expected device not trusted initially")
	}

	// Remember device
	if err := svc.RememberTrustedDevice(ctx, userID, fp, "MacBook Pro"); err != nil {
		t.Fatalf("RememberTrustedDevice: %v", err)
	}

	// Should be trusted now
	if !svc.IsTrustedDevice(ctx, tid, userID, fp) {
		t.Error("expected device trusted after registration")
	}

	// Different fingerprint should not be trusted
	if svc.IsTrustedDevice(ctx, tid, userID, "different_fp") {
		t.Error("expected different device not trusted")
	}
}

func TestAuthService_LoginAttempts(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	ctx := context.Background()
	username := "audit_user"

	// Record some attempts
	svc.RecordLoginAttempt(ctx, username, "10.0.0.1", "Mozilla", true, "")
	svc.RecordLoginAttempt(ctx, username, "10.0.0.2", "Chrome", false, "invalid credentials")
	svc.RecordLoginAttempt(ctx, username, "10.0.0.3", "Safari", true, "")

	// Retrieve
	attempts, err := svc.GetLoginAttempts(ctx, username, 10)
	if err != nil {
		t.Fatalf("GetLoginAttempts: %v", err)
	}
	if len(attempts) != 3 {
		t.Errorf("expected 3 attempts, got %d", len(attempts))
	}

	// Most recent should be first (sorted set ZRevRange)
	if !attempts[0].Success {
		t.Error("expected most recent attempt to be successful")
	}

	// Empty username
	empty, _ := svc.GetLoginAttempts(ctx, "nonexistent_user_xyz", 10)
	if len(empty) != 0 {
		t.Error("expected 0 attempts for nonexistent user")
	}
}

func TestAuthService_EmailChange_FullFlow(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	ctx, _ := tCtxTenant()
	userID := uuid.New()

	// Initiate change
	result, err := svc.InitiateEmailChange(ctx, userID, "old@test.com", "new@test.com")
	if err != nil {
		t.Fatalf("InitiateEmailChange: %v", err)
	}
	if result.OldEmailToken == "" || result.NewEmailToken == "" {
		t.Error("expected non-empty tokens")
	}

	// Confirm old email — should not be fully applied yet
	applied, err := svc.ConfirmEmailChange(ctx, result.OldEmailToken, "old")
	if err != nil {
		t.Fatalf("ConfirmEmailChange old: %v", err)
	}
	if applied {
		t.Error("expected applied=false after only old confirmation")
	}

	// Confirm new email — should be fully applied
	applied, err = svc.ConfirmEmailChange(ctx, result.NewEmailToken, "new")
	if err != nil {
		t.Fatalf("ConfirmEmailChange new: %v", err)
	}
	if !applied {
		t.Error("expected applied=true after both confirmations")
	}
}

func TestAuthService_EmailChange_SameEmail(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	ctx, _ := tCtxTenant()
	userID := uuid.New()

	_, err := svc.InitiateEmailChange(ctx, userID, "same@test.com", "same@test.com")
	if err == nil {
		t.Error("expected error for same email")
	}
}

func TestAuthService_EmailChange_EmptyNewEmail(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	ctx, _ := tCtxTenant()
	userID := uuid.New()

	_, err := svc.InitiateEmailChange(ctx, userID, "old@test.com", "")
	if err == nil {
		t.Error("expected error for empty new email")
	}
}

func TestAuthService_EmailChange_InvalidToken(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	ctx := context.Background()

	_, err := svc.ConfirmEmailChange(ctx, "invalid_token_xyz", "new")
	if err == nil {
		t.Error("expected error for invalid token")
	}
}

func TestAuthService_EmailChange_InvalidStep(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	ctx := context.Background()

	_, err := svc.ConfirmEmailChange(ctx, "some_token", "invalid_step")
	if err == nil {
		t.Error("expected error for invalid step")
	}
}

func TestAuthService_EmailChange_NoTenant(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	ctx := context.Background() // no tenant
	userID := uuid.New()

	_, err := svc.InitiateEmailChange(ctx, userID, "old@test.com", "new@test.com")
	if err == nil {
		t.Error("expected error for missing tenant context")
	}
}

func TestAuthService_GenerateWebAuthnChallenge(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	ctx := context.Background()

	challenge, err := svc.GenerateWebAuthnChallenge(ctx)
	if err != nil {
		t.Fatalf("GenerateWebAuthnChallenge: %v", err)
	}
	if challenge == "" {
		t.Error("expected non-empty challenge")
	}
	// Should be different each time
	challenge2, _ := svc.GenerateWebAuthnChallenge(ctx)
	if challenge == challenge2 {
		t.Error("expected different challenges")
	}
}

func TestAuthService_LoginWithLockout(t *testing.T) {
	svc, credRepo, _, _ := tNewAuthSvc(t)
	ctx, tid := tCtxTenant()

	userID := uuid.New()
	credRepo.byName["lockeduser"] = &domain.Credential{
		ID: uuid.New(), TenantID: tid, UserID: userID,
		Identifier: "lockeduser", Secret: "$2a$10$somehash",
	}

	// Lock the account
	for i := 0; i < svc.cfg.Password.MaxAttempts; i++ {
		_ = svc.RecordFailedLogin(ctx, tid, "lockeduser")
	}

	// Verify account is locked
	if !svc.IsAccountLocked(ctx, tid, "lockeduser") {
		t.Error("expected account to be locked")
	}
	_ = userID // used for clarity
}

func TestAuthService_PasswordValidation_Blacklist(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	svc.SetPasswordPolicy(conf.PasswordPolicy{
		MinLength: 8, RequireUpper: true, RequireLower: true,
		RequireDigit: true, Blacklist: []string{"Password1", "Welcome1"},
	})

	err := svc.passwordService.Validate("Password1")
	if err != ErrPasswordTooWeak {
		t.Errorf("expected ErrPasswordTooWeak for blacklisted password, got %v", err)
	}

	err = svc.passwordService.Validate("Str0ng!Pass")
	if err != nil {
		t.Errorf("expected nil for strong password, got %v", err)
	}
}

func TestAuthService_PasswordValidation_TooShort(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	svc.SetPasswordPolicy(conf.PasswordPolicy{MinLength: 12})

	err := svc.passwordService.Validate("Short1!")
	if err != ErrPasswordTooShort {
		t.Errorf("expected ErrPasswordTooShort, got %v", err)
	}
}

func TestAuthService_PasswordValidation_MissingComplexity(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	svc.SetPasswordPolicy(conf.PasswordPolicy{
		MinLength: 8, RequireUpper: true, RequireLower: true,
		RequireDigit: true, RequireSpecial: true,
	})

	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{"no_upper", "lowercase1!", true},
		{"no_lower", "UPPERCASE1!", true},
		{"no_digit", "UpperLower!", true},
		{"no_special", "UpperLower1", true},
		{"valid", "Upperlower1!", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.passwordService.Validate(tt.password)
			if tt.wantErr && err == nil {
				t.Errorf("expected error for %s", tt.name)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error for %s: %v", tt.name, err)
			}
		})
	}
}

func TestAuthService_SendVerificationEmail(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	svc.emailService = NewEmailService(tRedis(t))
	ctx, tid := tCtxTenant()
	userID := uuid.New()

	token, err := svc.SendVerificationEmail(ctx, tid, userID, "test@example.com")
	if err != nil {
		t.Fatalf("SendVerificationEmail: %v", err)
	}
	if token == "" {
		t.Error("expected non-empty token")
	}
}

func TestAuthService_GetPasswordHistory(t *testing.T) {
	svc, credRepo, _, _ := tNewAuthSvc(t)
	ctx, _ := tCtxTenant()
	userID := uuid.New()

	// Add some password history
	credRepo.history = []*domain.CredentialHistoryEntry{
		{ID: uuid.New(), Secret: "$2a$10$hash1....", CreatedAt: time.Now().Add(-48 * time.Hour)},
		{ID: uuid.New(), Secret: "$2a$10$hash2....", CreatedAt: time.Now().Add(-24 * time.Hour)},
	}

	history, err := svc.GetPasswordHistory(ctx, userID)
	if err != nil {
		t.Fatalf("GetPasswordHistory: %v", err)
	}
	if len(history) != 2 {
		t.Errorf("expected 2 entries, got %d", len(history))
	}

	// Check hash prefix is truncated
	if h, ok := history[0]["hash_prefix"].(string); !ok || len(h) > 20 {
		t.Error("hash_prefix should be truncated")
	}
}

func TestAuthService_GetPasswordHistory_NoTenant(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	ctx := context.Background()

	_, err := svc.GetPasswordHistory(ctx, uuid.New())
	if err == nil {
		t.Error("expected error for missing tenant")
	}
}

func TestAuthService_RememberTrustedDevice_NoTenant(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	ctx := context.Background()

	err := svc.RememberTrustedDevice(ctx, uuid.New(), "fp", "device")
	if err == nil {
		t.Error("expected error for missing tenant")
	}
}

// Note: SocialLogin is tested in social_login_test.go with a mock identity client.

func TestPasswordService_UpdatePolicy(t *testing.T) {
	rdb := tRedis(t)
	cr := newTCredRepo()
	ps := NewPasswordService(conf.Default().Password, cr, rdb)

	newPolicy := conf.PasswordPolicy{
		MinLength: 20, RequireUpper: true, RequireLower: true,
		RequireDigit: true, RequireSpecial: true, HistoryCount: 7,
	}
	ps.UpdatePolicy(newPolicy)

	got := ps.GetPolicy()
	if got.MinLength != 20 {
		t.Errorf("expected min_length=20, got %d", got.MinLength)
	}
	if got.HistoryCount != 7 {
		t.Errorf("expected history_count=7, got %d", got.HistoryCount)
	}
}

// Ensure authprovider import is used
var _ authprovider.ProviderType = authprovider.ProviderLocal
