package service

import (
	"context"
	"testing"
	"time"

	"github.com/ggid/ggid/pkg/authprovider"
	"github.com/ggid/ggid/pkg/tenant"
	"github.com/ggid/ggid/services/auth/internal/conf"
	"github.com/ggid/ggid/services/auth/internal/domain"
	"github.com/pquerna/otp/totp"
	"github.com/google/uuid"
)

// mockMFADeviceRepo implements repository.MFADeviceRepository for testing.
type mockMFADeviceRepo struct {
	devices map[uuid.UUID]*domain.MFADevice
}

func newMockMFADeviceRepo() *mockMFADeviceRepo {
	return &mockMFADeviceRepo{devices: make(map[uuid.UUID]*domain.MFADevice)}
}

func (m *mockMFADeviceRepo) CreateDevice(_ context.Context, device *domain.MFADevice) error {
	m.devices[device.UserID] = device
	return nil
}
func (m *mockMFADeviceRepo) GetDeviceByID(_ context.Context, _ uuid.UUID, id uuid.UUID) (*domain.MFADevice, error) {
	for _, d := range m.devices {
		if d.ID == id {
			return d, nil
		}
	}
	return nil, nil
}
func (m *mockMFADeviceRepo) ListDevicesByUser(_ context.Context, _ uuid.UUID, userID uuid.UUID) ([]*domain.MFADevice, error) {
	d := m.devices[userID]
	if d == nil {
		return nil, nil
	}
	return []*domain.MFADevice{d}, nil
}
func (m *mockMFADeviceRepo) GetEnabledDevice(_ context.Context, _ uuid.UUID, userID uuid.UUID) (*domain.MFADevice, error) {
	d := m.devices[userID]
	if d == nil || !d.Enabled {
		return nil, nil
	}
	return d, nil
}
func (m *mockMFADeviceRepo) UpdateDevice(_ context.Context, device *domain.MFADevice) error {
	m.devices[device.UserID] = device
	return nil
}
func (m *mockMFADeviceRepo) DeleteDevice(_ context.Context, _ uuid.UUID, _ uuid.UUID) error { return nil }

// === LoginMFA Tests ===

func TestLoginMFA_Success(t *testing.T) {
	credRepo := newMockCredRepo()
	refreshRepo := newMockRefreshTokenRepo()
	sessionRepo := newMockSessionRepo()
	mfaRepo := newMockMFADeviceRepo()
	ts, rdb := newTestTokenSvc(t, refreshRepo)
	userID := uuid.New()
	ctx, _ := testCtxWithTenant()

	// Generate a real TOTP secret and create a pre-verified device.
	_, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "GGID",
		AccountName: userID.String(),
	})
	if err != nil {
		t.Fatalf("generate TOTP key: %v", err)
	}

	// Use a simpler approach: setup MFA via the service, then verify.
	mfaSvc := NewMFAService(mfaRepo)
	setupResp, err := mfaSvc.SetupMFA(ctx, userID, "test-device")
	if err != nil {
		t.Fatalf("SetupMFA: %v", err)
	}

	// Verify the device with a valid code.
	code, _ := totp.GenerateCode(setupResp.Secret, time.Now())
	deviceID, _ := uuid.Parse(setupResp.DeviceID)
	_, _ = mfaSvc.VerifyMFA(ctx, deviceID, code)

	svc := &AuthService{
		cfg:             conf.Default(),
		chain:           authprovider.NewChain(&successProvider{userID: userID}),
		credentialRepo:  credRepo,
		tokenService:    ts,
		sessionService:  NewSessionService(sessionRepo),
		passwordService: NewPasswordService(conf.Default().Password, credRepo, rdb),
		rateLimiter:     NewRateLimiter(rdb),
		mfaService:      mfaSvc,
	}

	// Generate a fresh code for LoginMFA.
	freshCode, _ := totp.GenerateCode(setupResp.Secret, time.Now())

	tokens, err := svc.LoginMFA(ctx, "u", "p", freshCode, "1.1.1.1", "agent")
	if err != nil {
		t.Fatalf("LoginMFA: %v", err)
	}
	if tokens.AccessToken == "" {
		t.Error("expected access token after MFA login")
	}
}

func TestLoginMFA_InvalidCode(t *testing.T) {
	credRepo := newMockCredRepo()
	refreshRepo := newMockRefreshTokenRepo()
	sessionRepo := newMockSessionRepo()
	mfaRepo := newMockMFADeviceRepo()
	ts, rdb := newTestTokenSvc(t, refreshRepo)
	userID := uuid.New()
	ctx, _ := testCtxWithTenant()

	mfaRepo.devices[userID] = &domain.MFADevice{
		ID:      uuid.New(),
		UserID:  userID,
		Secret:  "JBSWY3DPEHPK3PXP",
		Enabled: true,
	}

	mfaSvc := NewMFAService(mfaRepo)

	svc := &AuthService{
		cfg:             conf.Default(),
		chain:           authprovider.NewChain(&successProvider{userID: userID}),
		credentialRepo:  credRepo,
		tokenService:    ts,
		sessionService:  NewSessionService(sessionRepo),
		passwordService: NewPasswordService(conf.Default().Password, credRepo, rdb),
		rateLimiter:     NewRateLimiter(rdb),
		mfaService:      mfaSvc,
	}

	_, err := svc.LoginMFA(ctx, "u", "p", "000000", "1.1.1.1", "agent")
	if err == nil {
		t.Error("expected error for invalid MFA code")
	}
}

// === VerifyPhoneOTP Success Path ===

func TestVerifyPhoneOTP_Success(t *testing.T) {
	credRepo := newMockCredRepo()
	refreshRepo := newMockRefreshTokenRepo()
	sessionRepo := newMockSessionRepo()
	ts, rdb := newTestTokenSvc(t, refreshRepo)
	tenantID := uuid.New()
	userID := uuid.New()
	phone := "+1234567890"

	svc := &AuthService{
		cfg:             conf.Default(),
		credentialRepo:  credRepo,
		tokenService:    ts,
		sessionService:  NewSessionService(sessionRepo),
		passwordService: NewPasswordService(conf.Default().Password, credRepo, rdb),
		rateLimiter:     NewRateLimiter(rdb),
	}

	otp, err := svc.SendPhoneOTP(context.Background(), tenantID, userID, phone)
	if err != nil {
		t.Fatalf("SendPhoneOTP: %v", err)
	}

	tokens, err := svc.VerifyPhoneOTP(context.Background(), phone, otp, "1.1.1.1", "agent")
	if err != nil {
		t.Fatalf("VerifyPhoneOTP: %v", err)
	}
	if tokens.AccessToken == "" {
		t.Error("expected access token after OTP verification")
	}
	if tokens.SessionID == "" {
		t.Error("expected session ID after OTP verification")
	}
}

func TestVerifyPhoneOTP_WrongOTP(t *testing.T) {
	credRepo := newMockCredRepo()
	refreshRepo := newMockRefreshTokenRepo()
	sessionRepo := newMockSessionRepo()
	ts, rdb := newTestTokenSvc(t, refreshRepo)
	tenantID := uuid.New()
	userID := uuid.New()
	phone := "+1234567890"

	svc := &AuthService{
		cfg:             conf.Default(),
		credentialRepo:  credRepo,
		tokenService:    ts,
		sessionService:  NewSessionService(sessionRepo),
		passwordService: NewPasswordService(conf.Default().Password, credRepo, rdb),
		rateLimiter:     NewRateLimiter(rdb),
	}

	_, err := svc.SendPhoneOTP(context.Background(), tenantID, userID, phone)
	if err != nil {
		t.Fatalf("SendPhoneOTP: %v", err)
	}

	_, err = svc.VerifyPhoneOTP(context.Background(), phone, "999999", "1.1.1.1", "agent")
	if err != ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

// === CleanupExpired ===

func TestCleanupExpired(t *testing.T) {
	sessionRepo := newMockSessionRepo()
	rdb := newTestRedis(t)
	svc := &AuthService{
		cfg:             conf.Default(),
		sessionService:  NewSessionService(sessionRepo),
		passwordService: NewPasswordService(conf.Default().Password, newMockCredRepo(), rdb),
		rateLimiter:     NewRateLimiter(rdb),
	}

	count, err := svc.CleanupExpired(context.Background())
	if err != nil {
		t.Fatalf("CleanupExpired: %v", err)
	}
	_ = count
}

// === RevokeSession ===

func TestRevokeSession(t *testing.T) {
	refreshRepo := newMockRefreshTokenRepo()
	sessionRepo := newMockSessionRepo()
	ts, rdb := newTestTokenSvc(t, refreshRepo)

	sessionID := uuid.New()
	sessionRepo.sessions[sessionID] = &domain.Session{
		ID:        sessionID,
		ExpiresAt: time.Now().Add(time.Hour),
	}

	svc := &AuthService{
		cfg:             conf.Default(),
		tokenService:    ts,
		sessionService:  NewSessionService(sessionRepo),
		passwordService: NewPasswordService(conf.Default().Password, newMockCredRepo(), rdb),
		rateLimiter:     NewRateLimiter(rdb),
	}

	err := svc.RevokeSession(context.Background(), sessionID)
	if err != nil {
		t.Fatalf("RevokeSession: %v", err)
	}
}

// === StepUp with MFA Method ===

func TestStepUp_MFAMethod_NoService(t *testing.T) {
	rdb := newTestRedis(t)
	rl := NewRateLimiter(rdb)
	userID := uuid.New()
	tenantID := uuid.New()

	svc := &AuthService{rateLimiter: rl}

	ctx := tenant.WithContext(context.Background(), &tenant.Context{
		TenantID:       tenantID,
		IsolationLevel: tenant.IsolationShared,
	})

	challenge, err := svc.InitStepUp(ctx, userID, "mfa")
	if err != nil {
		t.Fatalf("InitStepUp mfa: %v", err)
	}

	_, err = svc.VerifyStepUp(ctx, challenge.Challenge, "123456", "")
	if err == nil {
		t.Error("expected error when MFA service not configured")
	}
}

// === Password Expiration With History ===

func TestCheckPasswordExpiration_WithHistory(t *testing.T) {
	rdb := newTestRedis(t)
	credRepo := newMockCredRepo()
	userID := uuid.New()
	tenantID := uuid.New()

	credRepo.byUserID[userID] = &domain.Credential{
		ID:        uuid.New(),
		TenantID:  tenantID,
		UserID:    userID,
		UpdatedAt: time.Now().Add(-100 * 24 * time.Hour),
	}
	credRepo.history = []*domain.CredentialHistoryEntry{
		{CreatedAt: time.Now().Add(-200 * 24 * time.Hour)},
	}

	policy := conf.PasswordPolicy{MaxAgeDays: 30}
	ps := NewPasswordService(policy, credRepo, rdb)

	err := ps.CheckPasswordExpiration(context.Background(), tenantID, userID)
	if err != ErrPasswordExpired {
		t.Errorf("expected ErrPasswordExpired, got %v", err)
	}
}

func TestCheckPasswordExpiration_NoCredential(t *testing.T) {
	rdb := newTestRedis(t)
	credRepo := newMockCredRepo()
	policy := conf.PasswordPolicy{MaxAgeDays: 30}
	ps := NewPasswordService(policy, credRepo, rdb)

	err := ps.CheckPasswordExpiration(context.Background(), uuid.New(), uuid.New())
	if err != nil {
		t.Errorf("expected nil for no credential, got %v", err)
	}
}
