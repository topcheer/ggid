package service

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/ggid/ggid/pkg/authprovider"
	"github.com/ggid/ggid/pkg/crypto"
	"github.com/ggid/ggid/pkg/tenant"
	"github.com/ggid/ggid/services/auth/internal/conf"
	"github.com/ggid/ggid/services/auth/internal/domain"
	"github.com/google/uuid"
)

// === Phone OTP Tests ===

func TestSendPhoneOTP_Success(t *testing.T) {
	rdb := newTestRedis(t)
	rl := NewRateLimiter(rdb)
	svc := &AuthService{rateLimiter: rl}

	tenantID := uuid.New()
	userID := uuid.New()

	otp, err := svc.SendPhoneOTP(context.Background(), tenantID, userID, "+1234567890")
	if err != nil {
		t.Fatalf("SendPhoneOTP: %v", err)
	}
	if len(otp) != 6 {
		t.Errorf("expected 6-digit OTP, got %d chars", len(otp))
	}
}

func TestSendPhoneOTP_RateLimited(t *testing.T) {
	rdb := newTestRedis(t)
	rl := NewRateLimiter(rdb)
	svc := &AuthService{rateLimiter: rl}

	tenantID := uuid.New()
	userID := uuid.New()

	// Send 5 OTPs (should succeed).
	for i := 0; i < 5; i++ {
		_, err := svc.SendPhoneOTP(context.Background(), tenantID, userID, "+1234567890")
		if err != nil {
			t.Fatalf("attempt %d: %v", i, err)
		}
	}

	// 6th attempt should be rate limited.
	_, err := svc.SendPhoneOTP(context.Background(), tenantID, userID, "+1234567890")
	if err != ErrRateLimited {
		t.Errorf("expected ErrRateLimited, got %v", err)
	}
}

func TestVerifyPhoneOTP_Invalid(t *testing.T) {
	rdb := newTestRedis(t)
	rl := NewRateLimiter(rdb)
	svc := &AuthService{rateLimiter: rl}

	// No OTP stored — should fail.
	_, err := svc.VerifyPhoneOTP(context.Background(), "+1234567890", "000000", "1.1.1.1", "agent")
	if err != ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

// === Step-Up Authentication Tests ===

func TestInitStepUp_InvalidMethod(t *testing.T) {
	rdb := newTestRedis(t)
	rl := NewRateLimiter(rdb)
	svc := &AuthService{rateLimiter: rl}
	ctx, _ := testCtxWithTenant()

	_, err := svc.InitStepUp(ctx, uuid.New(), "invalid")
	if err == nil {
		t.Error("expected error for invalid method")
	}
}

func TestInitStepUp_PasswordMethod(t *testing.T) {
	rdb := newTestRedis(t)
	rl := NewRateLimiter(rdb)
	svc := &AuthService{rateLimiter: rl}
	ctx, _ := testCtxWithTenant()

	result, err := svc.InitStepUp(ctx, uuid.New(), "password")
	if err != nil {
		t.Fatalf("InitStepUp: %v", err)
	}
	if result.Challenge == "" {
		t.Error("expected non-empty challenge")
	}
	if result.Method != "password" {
		t.Errorf("expected method 'password', got %s", result.Method)
	}
}

func TestVerifyStepUp_InvalidChallenge(t *testing.T) {
	rdb := newTestRedis(t)
	rl := NewRateLimiter(rdb)
	svc := &AuthService{rateLimiter: rl}

	_, err := svc.VerifyStepUp(context.Background(), "invalid-challenge", "", "pass")
	if err != ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestStepUp_PasswordFlow(t *testing.T) {
	credRepo := newMockCredRepo()
	rdb := newTestRedis(t)
	rl := NewRateLimiter(rdb)
	userID := uuid.New()
	tenantID := uuid.New()

	oldHash, _ := crypto.HashPassword("MyPass123Ab")
	credRepo.byUserID[userID] = &domain.Credential{
		ID:         uuid.New(),
		TenantID:   tenantID,
		UserID:     userID,
		Identifier: "u",
		Secret:     oldHash,
		Enabled:    true,
		Type:       domain.CredentialPassword,
	}

	svc := &AuthService{
		rateLimiter:     rl,
		credentialRepo:  credRepo,
	}

	ctx := tenant.WithContext(context.Background(), &tenant.Context{
		TenantID:       tenantID,
		IsolationLevel: tenant.IsolationShared,
	})

	// Init step-up.
	challenge, err := svc.InitStepUp(ctx, userID, "password")
	if err != nil {
		t.Fatalf("InitStepUp: %v", err)
	}

	// Verify with correct password.
	result, err := svc.VerifyStepUp(ctx, challenge.Challenge, "", "MyPass123Ab")
	if err != nil {
		t.Fatalf("VerifyStepUp: %v", err)
	}
	if result.StepUpToken == "" {
		t.Error("expected non-empty step-up token")
	}
	if result.ExpiresIn != 300 {
		t.Errorf("expected 300s TTL, got %d", result.ExpiresIn)
	}

	// Validate token.
	if err := svc.ValidateStepUpToken(ctx, result.StepUpToken, userID); err != nil {
		t.Errorf("ValidateStepUpToken: %v", err)
	}
}

func TestStepUp_WrongPassword(t *testing.T) {
	credRepo := newMockCredRepo()
	rdb := newTestRedis(t)
	rl := NewRateLimiter(rdb)
	userID := uuid.New()
	tenantID := uuid.New()

	oldHash, _ := crypto.HashPassword("CorrectPass123")
	credRepo.byUserID[userID] = &domain.Credential{
		ID:         uuid.New(),
		TenantID:   tenantID,
		UserID:     userID,
		Identifier: "u",
		Secret:     oldHash,
		Enabled:    true,
	}

	svc := &AuthService{
		rateLimiter:    rl,
		credentialRepo: credRepo,
	}

	ctx := tenant.WithContext(context.Background(), &tenant.Context{
		TenantID:       tenantID,
		IsolationLevel: tenant.IsolationShared,
	})

	challenge, _ := svc.InitStepUp(ctx, userID, "password")
	_, err := svc.VerifyStepUp(ctx, challenge.Challenge, "", "WrongPass123")
	if err != ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestValidateStepUpToken_Invalid(t *testing.T) {
	rdb := newTestRedis(t)
	rl := NewRateLimiter(rdb)
	svc := &AuthService{rateLimiter: rl}

	err := svc.ValidateStepUpToken(context.Background(), "invalid-token", uuid.New())
	if err != ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestValidateStepUpToken_WrongUser(t *testing.T) {
	credRepo := newMockCredRepo()
	rdb := newTestRedis(t)
	rl := NewRateLimiter(rdb)
	userID := uuid.New()
	otherUserID := uuid.New()
	tenantID := uuid.New()

	oldHash, _ := crypto.HashPassword("MyPass123Ab")
	credRepo.byUserID[userID] = &domain.Credential{
		ID:         uuid.New(),
		TenantID:   tenantID,
		UserID:     userID,
		Identifier: "u",
		Secret:     oldHash,
		Enabled:    true,
	}

	svc := &AuthService{
		rateLimiter:    rl,
		credentialRepo: credRepo,
	}

	ctx := tenant.WithContext(context.Background(), &tenant.Context{
		TenantID:       tenantID,
		IsolationLevel: tenant.IsolationShared,
	})

	challenge, _ := svc.InitStepUp(ctx, userID, "password")
	result, _ := svc.VerifyStepUp(ctx, challenge.Challenge, "", "MyPass123Ab")

	// Wrong user should fail.
	err := svc.ValidateStepUpToken(ctx, result.StepUpToken, otherUserID)
	if err != ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials for wrong user, got %v", err)
	}
}

// === Risk-Based Auth Tests ===

func TestAssessLoginRisk_Low(t *testing.T) {
	rdb := newTestRedis(t)
	rl := NewRateLimiter(rdb)
	svc := &AuthService{rateLimiter: rl}

	userID := uuid.New()
	tenantID := uuid.New()
	ctx := tenant.WithContext(context.Background(), &tenant.Context{
		TenantID: tenantID,
	})

	// First login from a known device after recording it.
	svc.RecordSuccessfulLogin(ctx, userID, "1.2.3.4", "TestAgent/1.0")

	// Set known IP.
	rdb.Set(ctx, "ggid:risk:knownip:"+userID.String()+":1.2.3.4", "1", 30*24*time.Hour)

	// Now assess risk - IP is known, same user agent.
	// But user agent is not set yet in Redis for this test, so it might be "unknown UA"
	// Let's set it too.
	rdb.Set(ctx, "ggid:risk:ua:"+userID.String(), "TestAgent/1.0", 24*time.Hour)

	assessment := svc.AssessLoginRisk(ctx, tenantID, userID, "1.2.3.4", "TestAgent/1.0")
	if assessment.Level != RiskLevelLow {
		t.Errorf("expected RiskLevelLow, got %s (score %d, reasons %v)", assessment.Level, assessment.Score, assessment.Reasons)
	}
}

func TestAssessLoginRisk_FailedAttempts(t *testing.T) {
	rdb := newTestRedis(t)
	rl := NewRateLimiter(rdb)
	svc := &AuthService{rateLimiter: rl}

	userID := uuid.New()
	tenantID := uuid.New()
	ctx := tenant.WithContext(context.Background(), &tenant.Context{
		TenantID: tenantID,
	})

	// Record 3 failed attempts from same IP.
	for i := 0; i < 3; i++ {
		svc.RecordFailedLoginAttempt(ctx, userID, "5.5.5.5")
	}

	assessment := svc.AssessLoginRisk(ctx, tenantID, userID, "5.5.5.5", "TestAgent")
	if assessment.Level != RiskLevelMedium {
		t.Errorf("expected RiskLevelMedium, got %s", assessment.Level)
	}
	if !assessment.RequiresStepUp {
		t.Error("expected RequiresStepUp=true for medium risk")
	}
}

func TestAssessLoginRisk_HighRisk(t *testing.T) {
	rdb := newTestRedis(t)
	rl := NewRateLimiter(rdb)
	svc := &AuthService{rateLimiter: rl}

	userID := uuid.New()
	tenantID := uuid.New()
	ctx := tenant.WithContext(context.Background(), &tenant.Context{
		TenantID: tenantID,
	})

	// Record 6 failed attempts (>= 5 = high risk).
	for i := 0; i < 6; i++ {
		svc.RecordFailedLoginAttempt(ctx, userID, "6.6.6.6")
	}

	assessment := svc.AssessLoginRisk(ctx, tenantID, userID, "6.6.6.6", "TestAgent")
	if assessment.Level != RiskLevelHigh {
		t.Errorf("expected RiskLevelHigh, got %s (score=%d)", assessment.Level, assessment.Score)
	}
	if !assessment.RequiresStepUp {
		t.Error("expected RequiresStepUp for high risk")
	}
}

func TestAssessLoginRisk_BruteForce(t *testing.T) {
	rdb := newTestRedis(t)
	rl := NewRateLimiter(rdb)
	svc := &AuthService{rateLimiter: rl}

	tenantID := uuid.New()
	ctx := tenant.WithContext(context.Background(), &tenant.Context{
		TenantID: tenantID,
	})

	// Simulate brute-force: 3 different users attempted from same IP.
	for i := 0; i < 3; i++ {
		svc.RecordFailedLoginAttempt(ctx, uuid.New(), "7.7.7.7")
	}

	userID := uuid.New()
	assessment := svc.AssessLoginRisk(ctx, tenantID, userID, "7.7.7.7", "TestAgent")
	if !assessment.RequiresAdminAlert {
		t.Errorf("expected RequiresAdminAlert for brute-force, got level=%s", assessment.Level)
	}
}

func TestBlockSuspiciousIP(t *testing.T) {
	rdb := newTestRedis(t)
	rl := NewRateLimiter(rdb)
	svc := &AuthService{rateLimiter: rl}
	ctx := context.Background()

	if svc.IsIPBlocked(ctx, "8.8.8.8") {
		t.Error("IP should not be blocked before blocking")
	}

	svc.BlockSuspiciousIP(ctx, "8.8.8.8", time.Hour)

	if !svc.IsIPBlocked(ctx, "8.8.8.8") {
		t.Error("IP should be blocked after blocking")
	}
}

// === Password Expiration in Login Flow ===

func TestLogin_PasswordExpired(t *testing.T) {
	credRepo := newMockCredRepo()
	refreshRepo := newMockRefreshTokenRepo()
	sessionRepo := newMockSessionRepo()
	ts, rdb := newTestTokenSvc(t, refreshRepo)
	userID := uuid.New()
	ctx, tenantID := testCtxWithTenant()

	// Set up credential that is very old.
	oldHash, _ := crypto.HashPassword("ValidPass123")
	credRepo.byUserID[userID] = &domain.Credential{
		ID:         uuid.New(),
		TenantID:   tenantID,
		UserID:     userID,
		Identifier: "u",
		Secret:     oldHash,
		Enabled:    true,
		Type:       domain.CredentialPassword,
		UpdatedAt:  time.Now().Add(-365 * 24 * time.Hour), // 1 year ago
	}

	cfg := conf.Default()
	cfg.Password.MaxAgeDays = 30 // password expires after 30 days

	svc := &AuthService{
		cfg:    cfg,
		chain:  authprovider.NewChain(&successProvider{userID: userID}),
		credentialRepo:  credRepo,
		tokenService:    ts,
		sessionService:  NewSessionService(sessionRepo),
		passwordService: NewPasswordService(cfg.Password, credRepo, rdb),
		rateLimiter:     NewRateLimiter(rdb),
	}

	tokens, err := svc.Login(ctx, "u", "p", "1.1.1.1", "agent")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if !tokens.MustChangePassword {
		t.Error("expected must_change_password=true for expired password")
	}
	if tokens.AccessToken != "" {
		t.Error("expected no access token when password expired")
	}
}

func TestLogin_PasswordNotExpired(t *testing.T) {
	credRepo := newMockCredRepo()
	refreshRepo := newMockRefreshTokenRepo()
	sessionRepo := newMockSessionRepo()
	ts, rdb := newTestTokenSvc(t, refreshRepo)
	userID := uuid.New()
	ctx, _ := testCtxWithTenant()

	credRepo.byUserID[userID] = &domain.Credential{
		ID:         uuid.New(),
		UserID:     userID,
		Identifier: "u",
		Secret:     "",
		Enabled:    true,
		UpdatedAt:  time.Now(), // fresh
	}

	cfg := conf.Default()
	cfg.Password.MaxAgeDays = 90

	svc := &AuthService{
		cfg:    cfg,
		chain:  authprovider.NewChain(&successProvider{userID: userID}),
		credentialRepo:  credRepo,
		tokenService:    ts,
		sessionService:  NewSessionService(sessionRepo),
		passwordService: NewPasswordService(cfg.Password, credRepo, rdb),
		rateLimiter:     NewRateLimiter(rdb),
	}

	tokens, err := svc.Login(ctx, "u", "p", "1.1.1.1", "agent")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if tokens.MustChangePassword {
		t.Error("expected must_change_password=false for fresh password")
	}
}

// Ensure miniredis is used (reference the type).
var _ = miniredis.RunT

// Ensure domain and authprovider are used.
var _ = domain.CredentialPassword
var _ authprovider.ProviderType
