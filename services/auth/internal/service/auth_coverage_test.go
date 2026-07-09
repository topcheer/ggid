package service

import (
	"context"
	"testing"
	"time"

	"github.com/ggid/ggid/pkg/authprovider"
	"github.com/ggid/ggid/pkg/crypto"
	"github.com/ggid/ggid/pkg/tenant"
	"github.com/ggid/ggid/services/auth/internal/conf"
	"github.com/ggid/ggid/services/auth/internal/domain"
	"github.com/google/uuid"
)

// --- Auth provider chain ---

func TestAuthProviderChain_EmptyChain(t *testing.T) {
	chain := authprovider.NewChain()
	_, err := chain.Authenticate(context.Background(), authprovider.Credentials{
		Username: "user",
		Password: "pass",
	})
	if err == nil {
		t.Error("expected error from empty chain")
	}
}

func TestAuthProviderChain_FailingProvider(t *testing.T) {
	failP := &failingProvider{}
	chain := authprovider.NewChain(failP)
	_, err := chain.Authenticate(context.Background(), authprovider.Credentials{})
	if err == nil {
		t.Error("expected error")
	}
}

type failingProvider struct{}

func (p *failingProvider) Type() authprovider.ProviderType { return authprovider.ProviderLocal }
func (p *failingProvider) Name() string                    { return "fail" }
func (p *failingProvider) Authenticate(_ context.Context, _ authprovider.Credentials) (*authprovider.AuthResult, error) {
	return nil, ErrInvalidCredentials
}

// --- Register flow ---

func TestRegisterFlow_StrongPassword(t *testing.T) {
	password := "StrongPass123"
	hash, err := crypto.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}

	match, _ := crypto.VerifyPassword(password, hash)
	if !match {
		t.Error("expected match")
	}
}

// --- ChangePassword crypto flow ---

func TestChangePasswordFlow_Crypto(t *testing.T) {
	oldHash, _ := crypto.HashPassword("OldPassword123Ab")

	match, _ := crypto.VerifyPassword("OldPassword123Ab", oldHash)
	if !match {
		t.Error("expected match")
	}

	match, _ = crypto.VerifyPassword("WrongPass456", oldHash)
	if match {
		t.Error("should not match wrong password")
	}
}

// --- Rate limit integration ---

func TestRateLimit_LoginFlow(t *testing.T) {
	rdb, _ := testRedis(t)
	rl := NewRateLimiter(rdb)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		err := rl.CheckAndIncrement(ctx, "login:10.0.0.1", 5)
		if err != nil {
			t.Errorf("attempt %d: unexpected error: %v", i+1, err)
		}
	}

	err := rl.CheckAndIncrement(ctx, "login:10.0.0.1", 5)
	if err != ErrRateLimited {
		t.Errorf("expected rate limited, got %v", err)
	}
}

// --- Tenant context in auth flow ---

func TestTenantContext_AuthFlow(t *testing.T) {
	tenantID := uuid.New()
	tc := &tenant.Context{
		TenantID:       tenantID,
		IsolationLevel: tenant.IsolationShared,
	}
	ctx := tenant.WithContext(context.Background(), tc)

	extracted, err := tenant.FromContext(ctx)
	if err != nil {
		t.Fatalf("FromContext: %v", err)
	}
	if extracted.TenantID != tenantID {
		t.Errorf("expected %s, got %s", tenantID, extracted.TenantID)
	}

	_, err = tenant.FromContext(context.Background())
	if err == nil {
		t.Error("expected error without tenant context")
	}
}

// --- Password reset token flow via Redis ---

func TestPasswordResetFlow_RoundTrip(t *testing.T) {
	rdb, _ := testRedis(t)
	ps := NewPasswordService(conf.PasswordPolicy{
		MinLength:    12,
		RequireUpper: true,
		RequireLower: true,
		RequireDigit: true,
	}, nil, rdb)

	ctx := context.Background()
	tenantID := uuid.New()
	userID := uuid.New()

	token, err := ps.IssueResetToken(ctx, userID, tenantID)
	if err != nil {
		t.Fatalf("IssueResetToken: %v", err)
	}

	gotTenant, gotUser, err := ps.ConsumeResetToken(ctx, token)
	if err != nil {
		t.Fatalf("ConsumeResetToken: %v", err)
	}
	if gotTenant != tenantID || gotUser != userID {
		t.Error("tenant/user mismatch")
	}

	_, _, err = ps.ConsumeResetToken(ctx, token)
	if err != ErrInvalidResetToken {
		t.Errorf("expected ErrInvalidResetToken, got %v", err)
	}
}

// --- parseDeviceInfo ---

func TestParseDeviceInfo_AllBrowsers(t *testing.T) {
	tests := []struct {
		ua      string
		browser string
	}{
		{"Mozilla/5.0 Chrome/120", "Chrome"},
		{"Mozilla/5.0 Firefox/121", "Firefox"},
		{"Mozilla/5.0 Safari/604", "Safari"},
		{"Mozilla/5.0 Edge/120", "Edge"},
		{"curl/8.5", "Unknown"},
		{"", "Unknown"},
	}

	for _, tt := range tests {
		info := parseDeviceInfo(tt.ua)
		if info["browser"] != tt.browser {
			t.Errorf("UA %q: expected %s, got %s", tt.ua, tt.browser, info["browser"])
		}
	}
}

// --- Domain model lifecycle ---

func TestDomain_CredentialLockUnlockCycle(t *testing.T) {
	c := &domain.Credential{FailedAttempts: 0}

	// Simulate 5 failed attempts
	for i := 0; i < 5; i++ {
		c.RegisterFailedAttempt(5, 30*time.Minute)
	}
	if !c.IsLocked() {
		t.Error("should be locked after threshold")
	}

	c.ResetFailedAttempts()
	if c.IsLocked() {
		t.Error("should be unlocked after reset")
	}
}

func TestDomain_SessionRevoke(t *testing.T) {
	s := &domain.Session{
		ID:        uuid.New(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	if !s.IsActive() {
		t.Error("expected active")
	}
	s.Revoke()
	if s.IsActive() {
		t.Error("expected inactive after revoke")
	}
}

func TestDomain_RefreshTokenRevoke(t *testing.T) {
	rt := &domain.RefreshToken{
		ID:        uuid.New(),
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
	}
	if !rt.IsActive() {
		t.Error("expected active")
	}
	rt.Revoke()
	if rt.IsActive() {
		t.Error("expected inactive after revoke")
	}
}

// --- Crypto integration ---

func TestCrypto_GenerateRandomToken(t *testing.T) {
	t1, _ := crypto.GenerateRandomToken(32)
	t2, _ := crypto.GenerateRandomToken(32)
	if t1 == t2 {
		t.Error("tokens should differ")
	}
	if len(t1) < 30 {
		t.Errorf("token too short: %d", len(t1))
	}
}
