package service

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"

	"github.com/ggid/ggid/services/auth/internal/conf"
	"github.com/ggid/ggid/services/auth/internal/domain"
	"github.com/google/uuid"
)

func TestParsePrivateKey_Final_InvalidPEM(t *testing.T) {
	_, err := parsePrivateKey([]byte("not a pem"))
	if err == nil {
		t.Error("expected error for invalid PEM")
	}
}

func TestParsePrivateKey_Final_PKCS1(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
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

func TestParsePrivateKey_Final_PKCS8(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
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

func TestRegister_Final_DuplicateUser(t *testing.T) {
	credRepo := newMockCredRepo()
	refreshRepo := newMockRefreshTokenRepo()
	ts, rdb := newTestTokenSvc(t, refreshRepo)
	tenantID := uuid.New()
	userID := uuid.New()

	credRepo.byIdentifier["existing"] = &domain.Credential{
		TenantID: tenantID, UserID: userID,
	}

	svc := &AuthService{
		cfg:             conf.Default(),
		credentialRepo:  credRepo,
		tokenService:    ts,
		passwordService: NewPasswordService(conf.Default().Password, credRepo, rdb),
		rateLimiter:     NewRateLimiter(rdb),
	}

	err := svc.Register(context.Background(), tenantID, userID, "existing", "Sup3rStr0ng!Pass")
	if err == nil {
		t.Error("expected error for duplicate registration")
	}
}

func TestConsumeResetToken_Final_InvalidToken(t *testing.T) {
	rdb := newTestRedis(t)
	svc := NewPasswordService(conf.Default().Password, newMockCredRepo(), rdb)

	_, _, err := svc.ConsumeResetToken(context.Background(), "invalid-token")
	if err == nil {
		t.Error("expected error for invalid reset token")
	}
}

func TestRevokeSession_Final_NotFound(t *testing.T) {
	credRepo := newMockCredRepo()
	refreshRepo := newMockRefreshTokenRepo()
	sessionRepo := newMockSessionRepo()
	ts, rdb := newTestTokenSvc(t, refreshRepo)

	svc := &AuthService{
		cfg:             conf.Default(),
		credentialRepo:  credRepo,
		tokenService:    ts,
		sessionService:  NewSessionService(sessionRepo),
		passwordService: NewPasswordService(conf.Default().Password, credRepo, rdb),
		rateLimiter:     NewRateLimiter(rdb),
	}

	err := svc.RevokeSession(context.Background(), uuid.New())
	if err != nil {
		t.Errorf("expected nil for non-existent session, got: %v", err)
	}
}

func TestVerifyMagicLink_Final_Invalid(t *testing.T) {
	rdb := newTestRedis(t)
	rl := NewRateLimiter(rdb)
	svc := &AuthService{rateLimiter: rl, cfg: conf.Default()}

	_, err := svc.VerifyMagicLink(context.Background(), "invalid-token", "1.2.3.4", "agent")
	if err == nil {
		t.Error("expected error for invalid magic link")
	}
}

func TestCheckPasswordExpiration_Final_NoHistory(t *testing.T) {
	rdb := newTestRedis(t)
	credRepo := newMockCredRepo()
	svc := NewPasswordService(conf.Default().Password, credRepo, rdb)

	err := svc.CheckPasswordExpiration(context.Background(), uuid.New(), uuid.New())
	_ = err
}

func TestRotateRefreshToken_Final_InvalidToken(t *testing.T) {
	refreshRepo := newMockRefreshTokenRepo()
	ts, _ := newTestTokenSvc(t, refreshRepo)

	_, _, err := ts.RotateRefreshToken(context.Background(), "invalid-refresh")
	if err == nil {
		t.Error("expected error for invalid refresh token")
	}
}

func TestCallWebhook_Final_Timeout(t *testing.T) {
	mgr := NewHookManager()
	hook := &AuthHook{
		ID:      "timeout-hook-final",
		Event:   HookPostLogin,
		URL:     "http://localhost:1/timeout",
		Enabled: true,
	}
	mgr.RegisterHook(hook)

	err := mgr.ExecuteHooks(context.Background(), HookPostLogin, &HookPayload{
		Event:    HookPostLogin,
		TenantID: "test",
	})
	if err != nil {
		t.Errorf("post-hook should not return error: %v", err)
	}
}
