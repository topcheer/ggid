package service

import (
	"testing"
)

func TestParsePrivateKey_InvalidPEM(t *testing.T) {
	_, err := parsePrivateKey([]byte("not a pem"))
	if err == nil {
		t.Error("expected error for invalid PEM")
	}
}

func TestParsePrivateKey_InvalidKey(t *testing.T) {
	_, err := parsePrivateKey([]byte("-----BEGIN PRIVATE KEY-----\naW52YWxpZA==\n-----END PRIVATE KEY-----"))
	if err == nil {
		t.Error("expected error for invalid key data")
	}
}

func TestSetPassword_EmptyNewPassword(t *testing.T) {
	rdb := newTestRedis(t)
	credRepo := newMockCredRepo()
	ps := NewPasswordService(testPasswordConfig(), credRepo, rdb)

	err := ps.SetPassword("user-1", "")
	if err == nil {
		t.Error("expected error for empty password")
	}
}

func TestSetPassword_TooShort(t *testing.T) {
	rdb := newTestRedis(t)
	credRepo := newMockCredRepo()
	ps := NewPasswordService(testPasswordConfig(), credRepo, rdb)

	err := ps.SetPassword("user-1", "Ab1!")
	if err == nil {
		t.Error("expected error for short password")
	}
}

func TestCheckPasswordExpiration_NoHistory(t *testing.T) {
	rdb := newTestRedis(t)
	credRepo := newMockCredRepo()
	ps := NewPasswordService(testPasswordConfig(), credRepo, rdb)

	result := ps.CheckPasswordExpiration("user-no-history")
	if result.Expired {
		t.Error("expected not expired for user with no history")
	}
}

func TestRevokeSession(t *testing.T) {
	credRepo := newMockCredRepo()
	refreshRepo := newMockRefreshTokenRepo()
	sessionRepo := newMockSessionRepo()
	ts, rdb := newTestTokenSvc(t, refreshRepo)

	sessionID := uuid.New()
	sessionRepo.sessions[sessionID] = &domain.Session{
		ID:        sessionID,
		TenantID:  uuid.New(),
		UserID:    uuid.New(),
		ExpiresAt: timeNow(),
	}

	svc := &AuthService{
		cfg:             conf.Default(),
		tokenService:    ts,
		sessionService:  NewSessionService(sessionRepo),
		passwordService: NewPasswordService(conf.Default().Password, credRepo, rdb),
		rateLimiter:     NewRateLimiter(rdb),
	}

	err := svc.RevokeSession(context.Background(), sessionID)
	if err != nil {
		t.Fatalf("RevokeSession: %v", err)
	}
}

func TestRevokeSession_NotFound(t *testing.T) {
	credRepo := newMockCredRepo()
	refreshRepo := newMockRefreshTokenRepo()
	sessionRepo := newMockSessionRepo()
	ts, rdb := newTestTokenSvc(t, refreshRepo)

	svc := &AuthService{
		cfg:             conf.Default(),
		tokenService:    ts,
		sessionService:  NewSessionService(sessionRepo),
		passwordService: NewPasswordService(conf.Default().Password, credRepo, rdb),
		rateLimiter:     NewRateLimiter(rdb),
	}

	err := svc.RevokeSession(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("RevokeSession non-existent should not error: %v", err)
	}
}

func TestLoginMFA_WrongCode(t *testing.T) {
	credRepo := newMockCredRepo()
	refreshRepo := newMockRefreshTokenRepo()
	sessionRepo := newMockSessionRepo()
	ts, rdb := newTestTokenSvc(t, refreshRepo)

	tenantID := uuid.New()
	userID := uuid.New()
	credRepo.creds["testuser"] = &domain.Credential{
		TenantID:  tenantID,
		UserID:    userID,
		Username:  "testuser",
		IsActive:  true,
	}

	svc := &AuthService{
		cfg:             conf.Default(),
		credentialRepo:  credRepo,
		tokenService:    ts,
		sessionService:  NewSessionService(sessionRepo),
		passwordService: NewPasswordService(conf.Default().Password, credRepo, rdb),
		rateLimiter:     NewRateLimiter(rdb),
	}

	// Wrong MFA code should fail.
	_, err := svc.LoginMFA(context.Background(), userID, "wrong-code", "127.0.0.1", "test-agent")
	if err == nil {
		t.Error("expected error for wrong MFA code")
	}
}

func TestIssueMagicLink(t *testing.T) {
	rdb := newTestRedis(t)
	credRepo := newMockCredRepo()
	refreshRepo := newMockRefreshTokenRepo()
	sessionRepo := newMockSessionRepo()
	ts, _ := newTestTokenSvc(t, refreshRepo)

	svc := &AuthService{
		cfg:             conf.Default(),
		credentialRepo:  credRepo,
		tokenService:    ts,
		sessionService:  NewSessionService(sessionRepo),
		passwordService: NewPasswordService(conf.Default().Password, credRepo, rdb),
		rateLimiter:     NewRateLimiter(rdb),
	}

	tenantID := uuid.New()
	userID := uuid.New()
	credRepo.creds["magicuser"] = &domain.Credential{
		TenantID:  tenantID,
		UserID:    userID,
		Username:  "magicuser",
		IsActive:  true,
	}

	token, err := svc.IssueMagicLink(context.Background(), tenantID, userID, "magicuser")
	if err != nil {
		t.Fatalf("IssueMagicLink: %v", err)
	}
	if token == "" {
		t.Error("expected non-empty magic link token")
	}
}

func TestVerifyMagicLink_InvalidToken(t *testing.T) {
	rdb := newTestRedis(t)
	credRepo := newMockCredRepo()
	refreshRepo := newMockRefreshTokenRepo()
	sessionRepo := newMockSessionRepo()
	ts, _ := newTestTokenSvc(t, refreshRepo)

	svc := &AuthService{
		cfg:             conf.Default(),
		credentialRepo:  credRepo,
		tokenService:    ts,
		sessionService:  NewSessionService(sessionRepo),
		passwordService: NewPasswordService(conf.Default().Password, credRepo, rdb),
		rateLimiter:     NewRateLimiter(rdb),
	}

	_, err := svc.VerifyMagicLink(context.Background(), "invalid-magic-token", "127.0.0.1", "agent")
	if err == nil {
		t.Error("expected error for invalid magic link token")
	}
}

func TestVerifyPhoneOTP_InvalidCode(t *testing.T) {
	rdb := newTestRedis(t)
	rl := NewRateLimiter(rdb)
	svc := &AuthService{rateLimiter: rl, cfg: conf.Default()}

	_, err := svc.VerifyPhoneOTP(context.Background(), "invalid-phone-token", "0000")
	if err == nil {
		t.Error("expected error for invalid phone OTP token/code")
	}
}

func TestInitStepUp(t *testing.T) {
	rdb := newTestRedis(t)
	rl := NewRateLimiter(rdb)
	svc := &AuthService{rateLimiter: rl, cfg: conf.Default()}

	token, err := svc.InitStepUp(context.Background(), uuid.New(), "password")
	if err != nil {
		t.Fatalf("InitStepUp: %v", err)
	}
	if token == "" {
		t.Error("expected non-empty step-up token")
	}
}

func TestVerifyStepUp_InvalidToken(t *testing.T) {
	rdb := newTestRedis(t)
	rl := NewRateLimiter(rdb)
	svc := &AuthService{rateLimiter: rl, cfg: conf.Default()}

	_, err := svc.VerifyStepUp(context.Background(), "invalid-token", "password")
	if err == nil {
		t.Error("expected error for invalid step-up token")
	}
}

func TestAssessLoginRisk(t *testing.T) {
	rdb := newTestRedis(t)
	rl := NewRateLimiter(rdb)
	svc := &AuthService{rateLimiter: rl, cfg: conf.Default()}

	risk := svc.AssessLoginRisk("127.0.0.1", "Mozilla/5.0", uuid.New().String())
	if risk.Level == "" {
		t.Error("expected non-empty risk level")
	}
}
