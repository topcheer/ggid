package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/ggid/ggid/pkg/crypto"
	"github.com/ggid/ggid/services/auth/internal/conf"
	"github.com/ggid/ggid/services/auth/internal/domain"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// --- Test Helpers ---

func testJWTConfig(t *testing.T) conf.JWTConfig {
	t.Helper()
	dir := t.TempDir()
	return conf.JWTConfig{
		PrivateKeyPath: filepath.Join(dir, "test.pem"),
		PublicKeyPath:  filepath.Join(dir, "test.pub"),
		Issuer:         "test-issuer",
		Audience:       "test-aud",
		AccessTokenTTL: 15 * time.Minute,
	}
}

func testRedis(t *testing.T) (*redis.Client, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { rdb.Close() })
	return rdb, mr
}

func testPasswordPolicy() conf.PasswordPolicy {
	return conf.PasswordPolicy{
		MinLength:      12,
		RequireUpper:   true,
		RequireLower:   true,
		RequireDigit:   true,
		RequireSpecial: false,
		HistoryCount:   3,
		MaxAttempts:    5,
		LockDuration:   30 * time.Minute,
	}
}

func newTestKeyProvider(t *testing.T) crypto.KeyProvider {
	t.Helper()
	cfg := testJWTConfig(t)
	// Ensure local key pair exists before creating the KeyProvider.
	if _, err := loadOrCreatePrivateKey(cfg.PrivateKeyPath); err != nil {
		t.Fatalf("loadOrCreatePrivateKey: %v", err)
	}
	kp, err := crypto.NewKeyProvider(context.Background(), crypto.KeyProviderConfig{
		Provider: "local",
		Local: crypto.LocalKeyProviderConfig{
			PrivateKeyPath: cfg.PrivateKeyPath,
			PublicKeyPath:  cfg.PublicKeyPath,
		},
	})
	if err != nil {
		t.Fatalf("newTestKeyProvider: %v", err)
	}
	t.Cleanup(func() { _ = kp.Close() })
	return kp
}

// =====================================================================
// TokenService Tests — JWT signing + refresh token lifecycle
// =====================================================================

func TestTokenService_IssueAccessToken(t *testing.T) {
	cfg := testJWTConfig(t)
	ts, err := NewTokenService(newTestKeyProvider(t), cfg.Issuer, cfg.Audience, cfg.AccessTokenTTL, nil, nil)
	if err != nil {
		t.Fatalf("NewTokenService: %v", err)
	}

	tenantID := uuid.New()
	userID := uuid.New()

	token, expiresIn, err := ts.IssueAccessToken(tenantID, userID, []string{"admin"})
	if err != nil {
		t.Fatalf("IssueAccessToken: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
	if expiresIn != 900 {
		t.Errorf("expected expiresIn 900 (15min), got %d", expiresIn)
	}

	// Parse and verify the token
	parsed, err := jwt.Parse(token, func(tok *jwt.Token) (any, error) {
		if _, ok := tok.Method.(*jwt.SigningMethodRSA); !ok {
			t.Errorf("expected RS256, got %v", tok.Header["alg"])
		}
		return ts.PublicKey(), nil
	})
	if err != nil {
		t.Fatalf("failed to parse token: %v", err)
	}
	if !parsed.Valid {
		t.Fatal("token should be valid")
	}

	claims := parsed.Claims.(jwt.MapClaims)
	if claims["sub"] != userID.String() {
		t.Errorf("expected sub %s, got %v", userID, claims["sub"])
	}
	if claims["tenant_id"] != tenantID.String() {
		t.Errorf("expected tenant_id %s, got %v", tenantID, claims["tenant_id"])
	}
	if claims["iss"] != "test-issuer" {
		t.Errorf("expected iss 'test-issuer', got %v", claims["iss"])
	}

	// Verify kid header
	kid, ok := parsed.Header["kid"].(string)
	if !ok || kid == "" {
		t.Fatal("expected non-empty kid in token header")
	}
	if kid != ts.KeyID() {
		t.Errorf("expected kid %s, got %s", ts.KeyID(), kid)
	}
}

func TestTokenService_DifferentUsersDifferentTokens(t *testing.T) {
	cfg := testJWTConfig(t)
	ts, _ := NewTokenService(newTestKeyProvider(t), cfg.Issuer, cfg.Audience, cfg.AccessTokenTTL, nil, nil)

	t1, _, _ := ts.IssueAccessToken(uuid.New(), uuid.New(), []string{"admin"})
	t2, _, _ := ts.IssueAccessToken(uuid.New(), uuid.New(), []string{"admin"})

	if t1 == t2 {
		t.Fatal("tokens for different users should differ")
	}
}

func TestTokenService_RejectTamperedToken(t *testing.T) {
	cfg := testJWTConfig(t)
	ts, _ := NewTokenService(newTestKeyProvider(t), cfg.Issuer, cfg.Audience, cfg.AccessTokenTTL, nil, nil)

	token, _, _ := ts.IssueAccessToken(uuid.New(), uuid.New(), []string{"admin"})

	// Tamper: replace last 4 chars
	tampered := token[:len(token)-4] + "AAAA"
	_, err := jwt.Parse(tampered, func(tok *jwt.Token) (any, error) {
		return ts.PublicKey(), nil
	})
	if err == nil {
		t.Fatal("expected error for tampered token")
	}
}

func TestTokenService_RejectWrongKey(t *testing.T) {
	cfg1 := testJWTConfig(t)
	ts1, _ := NewTokenService(newTestKeyProvider(t), cfg1.Issuer, cfg1.Audience, cfg1.AccessTokenTTL, nil, nil)

	cfg2 := testJWTConfig(t)
	ts2, _ := NewTokenService(newTestKeyProvider(t), cfg2.Issuer, cfg2.Audience, cfg2.AccessTokenTTL, nil, nil)

	token, _, _ := ts1.IssueAccessToken(uuid.New(), uuid.New(), []string{"admin"})

	_, err := jwt.Parse(token, func(tok *jwt.Token) (any, error) {
		return ts2.PublicKey(), nil
	})
	if err == nil {
		t.Fatal("expected verification error with wrong key")
	}
}

func TestTokenService_KeyIDConsistent(t *testing.T) {
	cfg := testJWTConfig(t)
	ts1, _ := NewTokenService(newTestKeyProvider(t), cfg.Issuer, cfg.Audience, cfg.AccessTokenTTL, nil, nil)
	ts2, err := NewTokenService(newTestKeyProvider(t), cfg.Issuer, cfg.Audience, cfg.AccessTokenTTL, nil, nil)
	if err != nil {
		t.Fatalf("second load: %v", err)
	}
	if ts1.KeyID() != ts2.KeyID() {
		t.Errorf("key IDs differ: %s vs %s", ts1.KeyID(), ts2.KeyID())
	}
}

func TestTokenService_KeyFilesCreated(t *testing.T) {
	cfg := testJWTConfig(t)
	if _, err := loadOrCreatePrivateKey(cfg.PrivateKeyPath); err != nil {
		t.Fatalf("loadOrCreatePrivateKey: %v", err)
	}
	kp := newTestKeyProvider(t)
	ts, _ := NewTokenService(kp, cfg.Issuer, cfg.Audience, cfg.AccessTokenTTL, nil, nil)

	// With KeyProvider, the local provider creates/reads the private key file.
	// The public key is derived from the private key; no separate file is required.
	if _, err := os.Stat(cfg.PrivateKeyPath); err != nil {
		t.Errorf("private key not created: %v", err)
	}
	if ts.KeyID() == "" {
		t.Error("expected non-empty KeyID")
	}
	if kp.Public() == nil {
		t.Error("expected public key from provider")
	}
}

// =====================================================================
// PasswordService Tests — policy validation + history check
// =====================================================================

func TestPassword_Validate_Strong(t *testing.T) {
	ps := NewPasswordService(testPasswordPolicy(), nil, nil)

	valid := []string{"StrongPass123", "Abcdefghij12", "C0mplexPassword"}
	for _, pw := range valid {
		if err := ps.Validate(pw); err != nil {
			t.Errorf("expected pass for %q: %v", pw, err)
		}
	}
}

func TestPassword_Validate_TooShort(t *testing.T) {
	ps := NewPasswordService(testPasswordPolicy(), nil, nil)

	short := "Short1!"
	if err := ps.Validate(short); err != ErrPasswordTooShort {
		t.Errorf("expected ErrPasswordTooShort, got %v", err)
	}
}

func TestPassword_Validate_NoUpper(t *testing.T) {
	ps := NewPasswordService(testPasswordPolicy(), nil, nil)
	if err := ps.Validate("alllowercase123"); err != ErrPasswordTooWeak {
		t.Errorf("expected ErrPasswordTooWeak for no uppercase, got %v", err)
	}
}

func TestPassword_Validate_NoLower(t *testing.T) {
	ps := NewPasswordService(testPasswordPolicy(), nil, nil)
	if err := ps.Validate("ALLUPPER12345"); err != ErrPasswordTooWeak {
		t.Errorf("expected ErrPasswordTooWeak for no lowercase, got %v", err)
	}
}

func TestPassword_Validate_NoDigit(t *testing.T) {
	ps := NewPasswordService(testPasswordPolicy(), nil, nil)
	if err := ps.Validate("NoDigitsHere!!"); err != ErrPasswordTooWeak {
		t.Errorf("expected ErrPasswordTooWeak for no digit, got %v", err)
	}
}

func TestPassword_Validate_BoundaryLength(t *testing.T) {
	ps := NewPasswordService(testPasswordPolicy(), nil, nil)

	// Exactly 12 chars
	pw := "Abcdefghij12"
	if len(pw) != 12 {
		t.Fatalf("test setup error: len=%d", len(pw))
	}
	if err := ps.Validate(pw); err != nil {
		t.Errorf("exactly 12 chars should pass, got %v", err)
	}

	// 11 chars — too short
	if err := ps.Validate("Abcdefghij1"); err != ErrPasswordTooShort {
		t.Errorf("11 chars should fail, got %v", err)
	}
}

func TestPassword_Validate_RelaxedPolicy(t *testing.T) {
	ps := NewPasswordService(conf.PasswordPolicy{
		MinLength:    8,
		RequireUpper: false,
		RequireLower: false,
		RequireDigit: false,
	}, nil, nil)
	if err := ps.Validate("whatever"); err != nil {
		t.Errorf("relaxed policy should pass: %v", err)
	}
	if err := ps.Validate("short"); err != ErrPasswordTooShort {
		t.Errorf("short should fail relaxed policy, got %v", err)
	}
}

func TestPassword_VerifyOldPassword_Correct(t *testing.T) {
	ps := NewPasswordService(testPasswordPolicy(), nil, nil)

	hash, err := crypto.HashPassword("CorrectPassword1")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}

	cred := &domain.Credential{Secret: hash}
	match, err := ps.VerifyOldPassword(context.Background(), cred, "CorrectPassword1")
	if err != nil {
		t.Fatalf("VerifyOldPassword: %v", err)
	}
	if !match {
		t.Error("expected match=true")
	}
}

func TestPassword_VerifyOldPassword_Incorrect(t *testing.T) {
	ps := NewPasswordService(testPasswordPolicy(), nil, nil)

	hash, _ := crypto.HashPassword("CorrectPassword1")
	cred := &domain.Credential{Secret: hash}

	match, _ := ps.VerifyOldPassword(context.Background(), cred, "WrongPassword2")
	if match {
		t.Error("expected match=false for wrong password")
	}
}

func TestPassword_ResetTokenRoundTrip(t *testing.T) {
	rdb, _ := testRedis(t)
	ps := NewPasswordService(testPasswordPolicy(), nil, rdb)

	tenantID := uuid.New()
	userID := uuid.New()
	ctx := context.Background()

	// Issue reset token
	token, err := ps.IssueResetToken(ctx, userID, tenantID)
	if err != nil {
		t.Fatalf("IssueResetToken: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty reset token")
	}

	// Consume reset token
	gotTenant, gotUser, err := ps.ConsumeResetToken(ctx, token)
	if err != nil {
		t.Fatalf("ConsumeResetToken: %v", err)
	}
	if gotTenant != tenantID {
		t.Errorf("expected tenant %s, got %s", tenantID, gotTenant)
	}
	if gotUser != userID {
		t.Errorf("expected user %s, got %s", userID, gotUser)
	}

	// Second use should fail (one-time token)
	_, _, err = ps.ConsumeResetToken(ctx, token)
	if err != ErrInvalidResetToken {
		t.Errorf("expected ErrInvalidResetToken on second use, got %v", err)
	}
}

func TestPassword_ConsumeResetToken_InvalidToken(t *testing.T) {
	rdb, _ := testRedis(t)
	ps := NewPasswordService(testPasswordPolicy(), nil, rdb)

	_, _, err := ps.ConsumeResetToken(context.Background(), "nonexistent-token")
	if err != ErrInvalidResetToken {
		t.Errorf("expected ErrInvalidResetToken, got %v", err)
	}
}

// =====================================================================
// RateLimiter Tests — rate limiting logic
// =====================================================================

func TestRateLimiter_AllowUnderLimit(t *testing.T) {
	rdb, _ := testRedis(t)
	rl := NewRateLimiter(rdb)
	ctx := context.Background()

	// 5 attempts should all pass
	for i := 0; i < 5; i++ {
		if err := rl.CheckAndIncrement(ctx, "test-key-1", 5); err != nil {
			t.Errorf("attempt %d should pass, got %v", i+1, err)
		}
	}
}

func TestRateLimiter_BlockOverLimit(t *testing.T) {
	rdb, _ := testRedis(t)
	rl := NewRateLimiter(rdb)
	ctx := context.Background()

	// First 5 pass
	for i := 0; i < 5; i++ {
		if err := rl.CheckAndIncrement(ctx, "test-key-2", 5); err != nil {
			t.Fatalf("attempt %d unexpected error: %v", i+1, err)
		}
	}

	// 6th should fail
	err := rl.CheckAndIncrement(ctx, "test-key-2", 5)
	if err != ErrRateLimited {
		t.Errorf("expected ErrRateLimited on 6th attempt, got %v", err)
	}
}

func TestRateLimiter_DifferentKeysIndependent(t *testing.T) {
	rdb, _ := testRedis(t)
	rl := NewRateLimiter(rdb)
	ctx := context.Background()

	// Exhaust key A
	for i := 0; i < 3; i++ {
		_ = rl.CheckAndIncrement(ctx, "key-A", 3)
	}

	// Key B should still work
	err := rl.CheckAndIncrement(ctx, "key-B", 3)
	if err != nil {
		t.Errorf("key-B should pass after key-A exhausted, got %v", err)
	}
}

func TestRateLimiter_WindowExpiry(t *testing.T) {
	rdb, mr := testRedis(t)
	rl := NewRateLimiter(rdb)
	ctx := context.Background()

	// Exhaust the limit
	for i := 0; i < 3; i++ {
		_ = rl.CheckAndIncrement(ctx, "expiry-test", 3)
	}

	// Should be blocked
	if err := rl.CheckAndIncrement(ctx, "expiry-test", 3); err != ErrRateLimited {
		t.Fatalf("expected rate limit before expiry, got %v", err)
	}

	// Fast-forward time past the window
	mr.FastForward(61 * time.Second)

	// Should be allowed again after window expires
	if err := rl.CheckAndIncrement(ctx, "expiry-test", 3); err != nil {
		t.Errorf("expected pass after window expiry, got %v", err)
	}
}
