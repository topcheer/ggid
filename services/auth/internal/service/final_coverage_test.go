package service

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"

	"github.com/ggid/ggid/pkg/tenant"
	"github.com/ggid/ggid/services/auth/internal/conf"
	"github.com/ggid/ggid/services/auth/internal/domain"
	"github.com/google/uuid"
)

// --- Coverage tests for error paths and edge cases ---

func TestParsePrivateKey_InvalidPEM(t *testing.T) {
	_, err := parsePrivateKey([]byte("not a pem"))
	if err == nil {
		t.Error("expected error for invalid PEM")
	}
}

func TestParsePrivateKey_PKCS1(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 512)
	if err != nil {
		t.Fatal(err)
	}
	data := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	parsed, err := parsePrivateKey(data)
	if err != nil {
		t.Fatalf("parsePrivateKey: %v", err)
	}
	if parsed == nil || parsed.N.Cmp(key.N) != 0 {
		t.Error("key mismatch")
	}
}

func TestParsePrivateKey_PKCS8(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 512)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	data := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
	parsed, err := parsePrivateKey(data)
	if err != nil {
		t.Fatalf("parsePrivateKey PKCS8: %v", err)
	}
	if parsed == nil {
		t.Error("expected non-nil key")
	}
}

func TestLoginMFA_InvalidCode(t *testing.T) {
	credRepo := newMockCredRepo()
	refreshRepo := newMockRefreshTokenRepo()
	sessionRepo := newMockSessionRepo()
	ts, rdb := newTestTokenSvc(t, refreshRepo)
	tenantID := uuid.New()
	userID := uuid.New()

	credRepo.users["mfauser"] = &domain.Credential{
		TenantID: tenantID, UserID: userID,
		MFAEnabled: true, MFASecret: "JBSWY3DPEHPK3PXP",
	}

	svc := &AuthService{
		cfg:            conf.Default(),
		credentialRepo: credRepo,
		tokenService:   ts,
		sessionService: NewSessionService(sessionRepo),
		rateLimiter:    NewRateLimiter(rdb),
		mfaService:     NewMFAService(newMockMFARepo()),
	}

	_, err := svc.LoginMFA(context.Background(), "mfauser", "wrong-code", "1.2.3.4", "test-agent")
	if err == nil {
		t.Error("expected error for invalid MFA code")
	}
}

func TestRegister_DuplicateUser(t *testing.T) {
	credRepo := newMockCredRepo()
	refreshRepo := newMockRefreshTokenRepo()
	ts, rdb := newTestTokenSvc(t, refreshRepo)
	tenantID := uuid.New()
	userID := uuid.New()

	credRepo.users["existing"] = &domain.Credential{
		TenantID: tenantID, UserID: userID,
	}

	svc := &AuthService{
		cfg:            conf.Default(),
		credentialRepo: credRepo,
		tokenService:   ts,
		rateLimiter:    NewRateLimiter(rdb),
	}

	err := svc.Register(context.Background(), tenantID, userID, "existing", "password123")
	if err == nil {
		t.Error("expected error for duplicate registration")
	}
}

func TestResetPassword_InvalidToken(t *testing.T) {
	rdb := newTestRedis(t)
	svc := NewPasswordService(conf.Default().Password, newMockCredRepo(), rdb)

	err := svc.ConsumeResetToken(context.Background(), "invalid-token", "newpass")
	if err == nil {
		t.Error("expected error for invalid reset token")
	}
}

func TestRevokeSession_NotFound(t *testing.T) {
	credRepo := newMockCredRepo()
	refreshRepo := newMockRefreshTokenRepo()
	sessionRepo := newMockSessionRepo()
	ts, rdb := newTestTokenSvc(t, refreshRepo)

	svc := &AuthService{
		cfg:            conf.Default(),
		credentialRepo: credRepo,
		tokenService:   ts,
		sessionService: NewSessionService(sessionRepo),
		rateLimiter:    NewRateLimiter(rdb),
	}

	// Revoke a session that doesn't exist - should not error.
	err := svc.RevokeSession(context.Background(), uuid.New())
	if err != nil {
		t.Errorf("expected nil for non-existent session, got: %v", err)
	}
}

func TestVerifyStepUp_InvalidToken(t *testing.T) {
	rdb := newTestRedis(t)
	rl := NewRateLimiter(rdb)
	svc := &AuthService{rateLimiter: rl, cfg: conf.Default()}

	_, err := svc.VerifyStepUp(context.Background(), "invalid-token", "wrong-code")
	if err == nil {
		t.Error("expected error for invalid step-up token")
	}
}

func TestVerifyPhoneOTP_InvalidToken(t *testing.T) {
	rdb := newTestRedis(t)
	rl := NewRateLimiter(rdb)
	svc := &AuthService{rateLimiter: rl, cfg: conf.Default()}

	_, err := svc.VerifyPhoneOTP(context.Background(), "invalid-token", "123456")
	if err == nil {
		t.Error("expected error for invalid phone OTP token")
	}
}

func TestIssueMagicLink_Success(t *testing.T) {
	rdb := newTestRedis(t)
	rl := NewRateLimiter(rdb)
	svc := &AuthService{rateLimiter: rl, cfg: conf.Default()}

	ctx := tenant.WithContext(context.Background(), &tenant.Context{
		TenantID:       uuid.New(),
		IsolationLevel: tenant.IsolationShared,
	})

	token, err := svc.IssueMagicLink(ctx, uuid.New(), "user@test.com")
	if err != nil {
		t.Fatalf("IssueMagicLink: %v", err)
	}
	if token == "" {
		t.Error("expected non-empty magic link token")
	}
}

func TestVerifyMagicLink_Invalid(t *testing.T) {
	rdb := newTestRedis(t)
	rl := NewRateLimiter(rdb)
	svc := &AuthService{rateLimiter: rl, cfg: conf.Default()}

	_, err := svc.VerifyMagicLink(context.Background(), "invalid-token", "1.2.3.4", "agent")
	if err == nil {
		t.Error("expected error for invalid magic link")
	}
}

func TestCheckPasswordExpiration_NoHistory(t *testing.T) {
	rdb := newTestRedis(t)
	credRepo := newMockCredRepo()
	svc := NewPasswordService(conf.Default().Password, credRepo, rdb)

	// User with no password history - should not panic.
	result := svc.CheckPasswordExpiration(context.Background(), uuid.New())
	_ = result // should be false (no history)
}

func TestAssessLoginRisk_LowRisk(t *testing.T) {
	rdb := newTestRedis(t)
	rl := NewRateLimiter(rdb)
	svc := &AuthService{rateLimiter: rl, cfg: conf.Default()}

	result := svc.AssessLoginRisk(context.Background(), uuid.New(), "192.168.1.1", "Mozilla/5.0")
	if result.Level == "" {
		t.Error("expected non-empty risk level")
	}
}

func TestLogoutAll_NilServices(t *testing.T) {
	rdb := newTestRedis(t)
	rl := NewRateLimiter(rdb)
	svc := &AuthService{rateLimiter: rl, cfg: conf.Default()}

	// With nil tokenService/sessionService, should handle gracefully.
	tenantID := uuid.New()
	userID := uuid.New()
	_ = svc.LogoutAll(context.Background(), tenantID, userID, uuid.Nil)
}

func TestCallWebhook_Timeout(t *testing.T) {
	mgr := NewHookManager()
	hook := &AuthHook{
		ID:      "timeout-hook",
		Event:   HookPostLogin,
		URL:     "http://localhost:1/timeout",
		Enabled: true,
	}
	mgr.RegisterHook(hook)

	// Post-hooks should not propagate errors.
	err := mgr.ExecuteHooks(context.Background(), HookPostLogin, &HookPayload{
		Event:    HookPostLogin,
		TenantID: "test",
	})
	if err != nil {
		t.Errorf("post-hook should not return error: %v", err)
	}
}

func TestRotateRefreshToken_InvalidToken(t *testing.T) {
	refreshRepo := newMockRefreshTokenRepo()
	ts, _ := newTestTokenSvc(t, refreshRepo)

	_, err := ts.RotateRefreshToken(context.Background(), "invalid-refresh", uuid.New(), "client-1")
	if err == nil {
		t.Error("expected error for invalid refresh token")
	}
}
