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
	"github.com/redis/go-redis/v9"
)

// ============== Mock Repositories ==============

type mockCredentialRepo struct {
	byIdentifier map[string]*domain.Credential
	byUserID     map[uuid.UUID]*domain.Credential
	history      []*domain.CredentialHistoryEntry
}

func newMockCredRepo() *mockCredentialRepo {
	return &mockCredentialRepo{
		byIdentifier: make(map[string]*domain.Credential),
		byUserID:     make(map[uuid.UUID]*domain.Credential),
	}
}

func (m *mockCredentialRepo) FindByIDentifier(_ context.Context, _ uuid.UUID, id string) (*domain.Credential, error) {
	return m.byIdentifier[id], nil
}
func (m *mockCredentialRepo) FindByUserID(_ context.Context, _ uuid.UUID, uid uuid.UUID) (*domain.Credential, error) {
	return m.byUserID[uid], nil
}
func (m *mockCredentialRepo) Create(_ context.Context, c *domain.Credential) error {
	m.byIdentifier[c.Identifier] = c
	m.byUserID[c.UserID] = c
	return nil
}
func (m *mockCredentialRepo) UpdateFailedAttempts(_ context.Context, _ uuid.UUID, _ int, _ *time.Time) error {
	return nil
}
func (m *mockCredentialRepo) UpdateSecret(_ context.Context, _ uuid.UUID, _ string) error { return nil }
func (m *mockCredentialRepo) AddToHistory(_ context.Context, _ uuid.UUID, _ uuid.UUID, secret string) error {
	m.history = append(m.history, &domain.CredentialHistoryEntry{Secret: secret})
	return nil
}
func (m *mockCredentialRepo) GetHistory(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ int) ([]domain.CredentialHistoryEntry, error) {
	result := make([]domain.CredentialHistoryEntry, len(m.history))
	for i, h := range m.history {
		result[i] = *h
	}
	return result, nil
}

type mockSessionRepo struct {
	sessions map[uuid.UUID]*domain.Session
}

func newMockSessionRepo() *mockSessionRepo {
	return &mockSessionRepo{sessions: make(map[uuid.UUID]*domain.Session)}
}

func (m *mockSessionRepo) Create(_ context.Context, s *domain.Session) error {
	m.sessions[s.ID] = s
	return nil
}
func (m *mockSessionRepo) FindByTokenHash(_ context.Context, _ string) (*domain.Session, error) {
	return nil, nil
}
func (m *mockSessionRepo) FindByID(_ context.Context, id uuid.UUID) (*domain.Session, error) {
	return m.sessions[id], nil
}
func (m *mockSessionRepo) ListByUser(_ context.Context, _ uuid.UUID, uid uuid.UUID) ([]*domain.Session, error) {
	var result []*domain.Session
	for _, s := range m.sessions {
		if s.UserID == uid {
			result = append(result, s)
		}
	}
	return result, nil
}
func (m *mockSessionRepo) Revoke(_ context.Context, id uuid.UUID) error {
	if s, ok := m.sessions[id]; ok {
		now := time.Now()
		s.RevokedAt = &now
	}
	return nil
}
func (m *mockSessionRepo) RevokeAllForUser(_ context.Context, _ uuid.UUID, uid uuid.UUID, _ uuid.UUID) error {
	for _, s := range m.sessions {
		if s.UserID == uid {
			now := time.Now()
			s.RevokedAt = &now
		}
	}
	return nil
}
func (m *mockSessionRepo) DeleteExpired(_ context.Context, _ time.Time) (int64, error) { return 0, nil }
func (m *mockSessionRepo) UpdateJTI(_ context.Context, _ uuid.UUID, _ string, _ time.Time) error {
	return nil
}
func (m *mockSessionRepo) ListActiveJTIForUser(_ context.Context, _ uuid.UUID, _ uuid.UUID) ([]domain.SessionJTI, error) {
	return nil, nil
}

type mockRefreshTokenRepo struct {
	tokens map[string]*domain.RefreshToken
}

func newMockRefreshTokenRepo() *mockRefreshTokenRepo {
	return &mockRefreshTokenRepo{tokens: make(map[string]*domain.RefreshToken)}
}

func (m *mockRefreshTokenRepo) Create(_ context.Context, t *domain.RefreshToken) error {
	m.tokens[t.TokenHash] = t
	return nil
}
func (m *mockRefreshTokenRepo) FindByHash(_ context.Context, hash string) (*domain.RefreshToken, error) {
	return m.tokens[hash], nil
}
func (m *mockRefreshTokenRepo) Revoke(_ context.Context, id uuid.UUID) error {
	for _, t := range m.tokens {
		if t.ID == id {
			now := time.Now()
			t.RevokedAt = &now
		}
	}
	return nil
}
func (m *mockRefreshTokenRepo) RevokeAllForSession(_ context.Context, _ uuid.UUID) error { return nil }
func (m *mockRefreshTokenRepo) RevokeAllForUser(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
	return nil
}

// ============== Helpers ==============

func testCtxWithTenant() (context.Context, uuid.UUID) {
	tenantID := uuid.New()
	tc := &tenant.Context{TenantID: tenantID, IsolationLevel: tenant.IsolationShared}
	return tenant.WithContext(context.Background(), tc), tenantID
}

func newTestRedis(t *testing.T) *redis.Client {
	t.Helper()
	mr := miniredis.RunT(t)
	return redis.NewClient(&redis.Options{Addr: mr.Addr()})
}

func newTestTokenSvc(t *testing.T, refreshRepo RefreshTokenRepo) (*TokenService, *redis.Client) {
	t.Helper()
	rdb := newTestRedis(t)
	cfg := conf.Default()
	dir := t.TempDir()
	cfg.JWT.PrivateKeyPath = dir + "/test.pem"
	cfg.JWT.PublicKeyPath = dir + "/test.pub"
	if _, err := loadOrCreatePrivateKey(cfg.JWT.PrivateKeyPath); err != nil {
		t.Fatalf("loadOrCreatePrivateKey: %v", err)
	}
	kp, err := crypto.NewKeyProvider(context.Background(), crypto.KeyProviderConfig{
		Provider: "local",
		Local: crypto.LocalKeyProviderConfig{
			PrivateKeyPath: cfg.JWT.PrivateKeyPath,
			PublicKeyPath:  cfg.JWT.PublicKeyPath,
		},
	})
	if err != nil {
		t.Fatalf("NewKeyProvider: %v", err)
	}
	t.Cleanup(func() { _ = kp.Close() })
	ts, err := NewTokenService(kp, cfg.JWT.Issuer, cfg.JWT.Audience, cfg.JWT.AccessTokenTTL, refreshRepo, rdb)
	if err != nil {
		t.Fatalf("NewTokenService: %v", err)
	}
	return ts, rdb
}

// successProvider always authenticates successfully.
type successProvider struct {
	userID uuid.UUID
}

func (p *successProvider) Type() authprovider.ProviderType { return authprovider.ProviderLocal }
func (p *successProvider) Name() string                    { return "success" }
func (p *successProvider) Authenticate(_ context.Context, _ authprovider.Credentials) (*authprovider.AuthResult, error) {
	uid := p.userID
	return &authprovider.AuthResult{Provider: authprovider.ProviderLocal, LinkedUser: &uid}, nil
}

// ============== Register ==============

func TestAuthService_Register_Success(t *testing.T) {
	credRepo := newMockCredRepo()
	rdb := newTestRedis(t)
	_, tenantID := testCtxWithTenant()
	userID := uuid.New()

	svc := &AuthService{
		cfg:             conf.Default(),
		credentialRepo:  credRepo,
		passwordService: NewPasswordService(conf.Default().Password, credRepo, rdb),
	}

	err := svc.Register(context.Background(), tenantID, userID, "newuser", "StrongPass123")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	cred := credRepo.byIdentifier["newuser"]
	if cred == nil || cred.UserID != userID {
		t.Error("expected credential stored")
	}
}

func TestAuthService_Register_WeakPassword(t *testing.T) {
	credRepo := newMockCredRepo()
	rdb := newTestRedis(t)
	svc := &AuthService{
		cfg:             conf.Default(),
		credentialRepo:  credRepo,
		passwordService: NewPasswordService(conf.Default().Password, credRepo, rdb),
	}
	err := svc.Register(context.Background(), uuid.New(), uuid.New(), "u", "short")
	if err != ErrPasswordTooShort {
		t.Errorf("expected ErrPasswordTooShort, got %v", err)
	}
}

func TestAuthService_Register_DuplicateUser(t *testing.T) {
	credRepo := newMockCredRepo()
	rdb := newTestRedis(t)
	existingCred := &domain.Credential{ID: uuid.New(), UserID: uuid.New()}
	credRepo.byIdentifier["existing"] = existingCred
	svc := &AuthService{
		cfg:             conf.Default(),
		credentialRepo:  credRepo,
		passwordService: NewPasswordService(conf.Default().Password, credRepo, rdb),
	}
	// Duplicate user now updates the credential (CreateUserFromSocial may have created one)
	err := svc.Register(context.Background(), uuid.New(), uuid.New(), "existing", "StrongPass123")
	if err != nil {
		t.Errorf("expected no error for duplicate (should update), got: %v", err)
	}
	// Verify UpdateSecret was called (mock is no-op, just ensure no error)
	cred := credRepo.byIdentifier["existing"]
	if cred == nil {
		t.Error("expected credential to still exist")
	}
}

// ============== ChangePassword ==============

func TestAuthService_ChangePassword_Success(t *testing.T) {
	credRepo := newMockCredRepo()
	rdb := newTestRedis(t)
	ctx, tenantID := testCtxWithTenant()
	userID := uuid.New()

	oldHash, _ := crypto.HashPassword("OldPass123Ab")
	cred := &domain.Credential{
		ID: uuid.New(), TenantID: tenantID, UserID: userID,
		Identifier: "u", Secret: oldHash, Enabled: true, Type: domain.CredentialPassword,
	}
	credRepo.byIdentifier["u"] = cred
	credRepo.byUserID[userID] = cred

	svc := &AuthService{
		cfg: conf.Default(), credentialRepo: credRepo,
		passwordService: NewPasswordService(conf.Default().Password, credRepo, rdb),
	}

	err := svc.ChangePassword(ctx, tenantID, userID, "OldPass123Ab", "NewStrongPass456")
	if err != nil {
		t.Fatalf("ChangePassword: %v", err)
	}
}

func TestAuthService_ChangePassword_WrongOld(t *testing.T) {
	credRepo := newMockCredRepo()
	rdb := newTestRedis(t)
	ctx, tenantID := testCtxWithTenant()
	userID := uuid.New()

	oldHash, _ := crypto.HashPassword("OldPass123Ab")
	credRepo.byUserID[userID] = &domain.Credential{
		ID: uuid.New(), TenantID: tenantID, UserID: userID,
		Identifier: "u", Secret: oldHash, Enabled: true,
	}

	svc := &AuthService{
		cfg: conf.Default(), credentialRepo: credRepo,
		passwordService: NewPasswordService(conf.Default().Password, credRepo, rdb),
	}

	err := svc.ChangePassword(ctx, tenantID, userID, "WrongPassword", "NewStrongPass456")
	if err != ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestAuthService_ChangePassword_NotFound(t *testing.T) {
	credRepo := newMockCredRepo()
	rdb := newTestRedis(t)
	ctx, tenantID := testCtxWithTenant()

	svc := &AuthService{
		cfg: conf.Default(), credentialRepo: credRepo,
		passwordService: NewPasswordService(conf.Default().Password, credRepo, rdb),
	}
	err := svc.ChangePassword(ctx, tenantID, uuid.New(), "old", "NewStrongPass456")
	if err == nil {
		t.Error("expected error for non-existent user")
	}
}

// ============== ForgotPassword / ResetPassword ==============

func TestAuthService_ForgotPassword_NonExistent(t *testing.T) {
	credRepo := newMockCredRepo()
	rdb := newTestRedis(t)
	svc := &AuthService{
		cfg: conf.Default(), credentialRepo: credRepo,
		passwordService: NewPasswordService(conf.Default().Password, credRepo, rdb),
	}
	err := svc.ForgotPassword(context.Background(), uuid.New(), "nobody@test.com")
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestAuthService_ForgotPassword_Existing(t *testing.T) {
	credRepo := newMockCredRepo()
	rdb := newTestRedis(t)
	tenantID := uuid.New()
	credRepo.byIdentifier["user@test.com"] = &domain.Credential{UserID: uuid.New()}
	svc := &AuthService{
		cfg: conf.Default(), credentialRepo: credRepo,
		passwordService: NewPasswordService(conf.Default().Password, credRepo, rdb),
	}
	err := svc.ForgotPassword(context.Background(), tenantID, "user@test.com")
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestAuthService_ResetPassword_InvalidToken(t *testing.T) {
	credRepo := newMockCredRepo()
	rdb := newTestRedis(t)
	svc := &AuthService{
		cfg: conf.Default(), credentialRepo: credRepo,
		passwordService: NewPasswordService(conf.Default().Password, credRepo, rdb),
	}
	err := svc.ResetPassword(context.Background(), "invalid", "NewStrongPass123")
	if err != ErrInvalidResetToken {
		t.Errorf("expected ErrInvalidResetToken, got %v", err)
	}
}

func TestAuthService_ResetPassword_Success(t *testing.T) {
	credRepo := newMockCredRepo()
	rdb := newTestRedis(t)
	tenantID := uuid.New()
	userID := uuid.New()
	oldHash, _ := crypto.HashPassword("OldPass123Ab")
	credRepo.byUserID[userID] = &domain.Credential{
		ID: uuid.New(), TenantID: tenantID, UserID: userID,
		Identifier: "u", Secret: oldHash, Enabled: true, Type: domain.CredentialPassword,
	}
	ps := NewPasswordService(conf.Default().Password, credRepo, rdb)
	sessionRepo := newMockSessionRepo()
	svc := &AuthService{
		cfg: conf.Default(), credentialRepo: credRepo,
		passwordService: ps, sessionService: NewSessionService(sessionRepo),
	}
	ctx := context.Background()
	token, _ := ps.IssueResetToken(ctx, userID, tenantID)
	err := svc.ResetPassword(ctx, token, "NewStrongPass456")
	if err != nil {
		t.Fatalf("ResetPassword: %v", err)
	}
}

// ============== Login ==============

func TestAuthService_Login_RateLimited(t *testing.T) {
	rdb := newTestRedis(t)
	rl := NewRateLimiter(rdb)
	ctx, _ := testCtxWithTenant()
	svc := &AuthService{
		cfg: conf.Default(), chain: authprovider.NewChain(), rateLimiter: rl,
	}
	for i := 0; i < 5; i++ {
		_, _ = svc.Login(ctx, "u", "p", "1.2.3.4", "agent")
	}
	_, err := svc.Login(ctx, "u", "p", "1.2.3.4", "agent")
	if err != ErrRateLimited {
		t.Errorf("expected ErrRateLimited, got %v", err)
	}
}

func TestAuthService_Login_InvalidCreds(t *testing.T) {
	rdb := newTestRedis(t)
	rl := NewRateLimiter(rdb)
	ctx, _ := testCtxWithTenant()
	svc := &AuthService{
		cfg: conf.Default(), chain: authprovider.NewChain(), rateLimiter: rl,
	}
	_, err := svc.Login(ctx, "u", "p", "9.9.9.9", "agent")
	if err != ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestAuthService_Login_Success(t *testing.T) {
	credRepo := newMockCredRepo()
	refreshRepo := newMockRefreshTokenRepo()
	sessionRepo := newMockSessionRepo()
	ts, rdb := newTestTokenSvc(t, refreshRepo)
	userID := uuid.New()
	ctx, _ := testCtxWithTenant()

	svc := &AuthService{
		cfg: conf.Default(), chain: authprovider.NewChain(&successProvider{userID: userID}),
		credentialRepo: credRepo, tokenService: ts,
		sessionService:  NewSessionService(sessionRepo),
		passwordService: NewPasswordService(conf.Default().Password, credRepo, rdb),
		rateLimiter:     NewRateLimiter(rdb),
	}

	tokens, err := svc.Login(ctx, "u", "p", "1.1.1.1", "agent")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if tokens.AccessToken == "" || tokens.RefreshToken == "" || tokens.SessionID == "" {
		t.Error("expected all token fields non-empty")
	}
}

// ============== Refresh ==============

func TestAuthService_Refresh_Invalid(t *testing.T) {
	refreshRepo := newMockRefreshTokenRepo()
	ts, _ := newTestTokenSvc(t, refreshRepo)
	svc := &AuthService{tokenService: ts}
	_, err := svc.Refresh(context.Background(), "invalid")
	if err == nil {
		t.Error("expected error")
	}
}

func TestAuthService_Refresh_Success(t *testing.T) {
	refreshRepo := newMockRefreshTokenRepo()
	ts, _ := newTestTokenSvc(t, refreshRepo)
	tenantID := uuid.New()
	userID := uuid.New()
	sessionID := uuid.New()

	plaintext := "test-refresh-token-value"
	tokenHash := hashToken(plaintext)
	now := time.Now()
	refreshRepo.tokens[tokenHash] = &domain.RefreshToken{
		ID: uuid.New(), TenantID: tenantID, UserID: userID, SessionID: sessionID,
		TokenHash: tokenHash, ExpiresAt: now.Add(30 * 24 * time.Hour), CreatedAt: now,
	}

	svc := &AuthService{tokenService: ts}
	tokens, err := svc.Refresh(context.Background(), plaintext)
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if tokens.AccessToken == "" || tokens.RefreshToken == plaintext {
		t.Error("expected rotated token")
	}
}

// ============== Sessions ==============

func TestAuthService_Logout(t *testing.T) {
	refreshRepo := newMockRefreshTokenRepo()
	ts, rdb := newTestTokenSvc(t, refreshRepo)

	// Issue a token first
	plaintext, _ := ts.IssueRefreshToken(context.Background(), uuid.New(), uuid.New(), uuid.New())

	svc := &AuthService{tokenService: ts, sessionService: NewSessionService(newMockSessionRepo())}
	err := svc.Logout(context.Background(), plaintext)
	if err != nil {
		t.Errorf("Logout: %v", err)
	}

	_ = rdb // keep reference
}

func TestAuthService_ListSessions(t *testing.T) {
	sessionRepo := newMockSessionRepo()
	_, tenantID := testCtxWithTenant()
	userID := uuid.New()
	sessionRepo.sessions[uuid.New()] = &domain.Session{ID: uuid.New(), TenantID: tenantID, UserID: userID}
	svc := &AuthService{sessionService: NewSessionService(sessionRepo)}
	sessions, err := svc.ListSessions(context.Background(), tenantID, userID)
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if len(sessions) != 1 {
		t.Errorf("expected 1, got %d", len(sessions))
	}
}

func TestAuthService_RevokeSession(t *testing.T) {
	sessionRepo := newMockSessionRepo()
	refreshRepo := newMockRefreshTokenRepo()
	ts, _ := newTestTokenSvc(t, refreshRepo)
	sessionID := uuid.New()
	sessionRepo.sessions[sessionID] = &domain.Session{ID: sessionID, ExpiresAt: time.Now().Add(time.Hour)}
	svc := &AuthService{tokenService: ts, sessionService: NewSessionService(sessionRepo)}
	err := svc.RevokeSession(context.Background(), sessionID)
	if err != nil {
		t.Fatalf("RevokeSession: %v", err)
	}
	if sessionRepo.sessions[sessionID].RevokedAt == nil {
		t.Error("expected revoked")
	}
}

func TestAuthService_CleanupExpired(t *testing.T) {
	svc := &AuthService{sessionService: NewSessionService(newMockSessionRepo())}
	_, err := svc.CleanupExpired(context.Background())
	if err != nil {
		t.Fatalf("CleanupExpired: %v", err)
	}
}

func TestAuthService_MFAServiceGetter(t *testing.T) {
	svc := &AuthService{mfaService: nil}
	if svc.MFAService() != nil {
		t.Error("expected nil")
	}
}

// ============== LocalProvider ==============

func TestLocalProvider_Mock_Auth_Success(t *testing.T) {
	credRepo := newMockCredRepo()
	ctx, tenantID := testCtxWithTenant()
	userID := uuid.New()
	hash, _ := crypto.HashPassword("StrongPass123")
	credRepo.byIdentifier["u"] = &domain.Credential{
		ID: uuid.New(), TenantID: tenantID, UserID: userID,
		Identifier: "u", Secret: hash, Enabled: true, Type: domain.CredentialPassword,
	}
	p := NewLocalProvider(credRepo, conf.Default().Password)
	result, err := p.Authenticate(ctx, authprovider.Credentials{Username: "u", Password: "StrongPass123"})
	if err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
	if result.LinkedUser == nil || *result.LinkedUser != userID {
		t.Error("expected linked user")
	}
}

func TestLocalProvider_Mock_Auth_WrongPassword(t *testing.T) {
	credRepo := newMockCredRepo()
	ctx, tenantID := testCtxWithTenant()
	hash, _ := crypto.HashPassword("StrongPass123")
	credRepo.byIdentifier["u"] = &domain.Credential{
		ID: uuid.New(), TenantID: tenantID, UserID: uuid.New(),
		Identifier: "u", Secret: hash, Enabled: true, Type: domain.CredentialPassword,
	}
	p := NewLocalProvider(credRepo, conf.Default().Password)
	_, err := p.Authenticate(ctx, authprovider.Credentials{Username: "u", Password: "Wrong"})
	if err != ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLocalProvider_Mock_Auth_Disabled(t *testing.T) {
	credRepo := newMockCredRepo()
	ctx, tenantID := testCtxWithTenant()
	hash, _ := crypto.HashPassword("StrongPass123")
	credRepo.byIdentifier["u"] = &domain.Credential{
		TenantID: tenantID, UserID: uuid.New(),
		Identifier: "u", Secret: hash, Enabled: false, Type: domain.CredentialPassword,
	}
	p := NewLocalProvider(credRepo, conf.Default().Password)
	_, err := p.Authenticate(ctx, authprovider.Credentials{Username: "u", Password: "StrongPass123"})
	if err == nil {
		t.Error("expected error for disabled")
	}
}

// ============== SessionService ==============

func TestSessionService_Mock_Create(t *testing.T) {
	sessionRepo := newMockSessionRepo()
	ss := NewSessionService(sessionRepo)
	tenantID := uuid.New()
	userID := uuid.New()
	token, session, err := ss.Create(context.Background(), CreateSessionParams{
		TenantID: tenantID, UserID: userID, IPAddress: "1.2.3.4",
		UserAgent: "Chrome", TTL: time.Hour,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if token == "" || session.UserID != userID {
		t.Error("unexpected session values")
	}
	if sessionRepo.sessions[session.ID] == nil {
		t.Error("expected stored")
	}
}

func TestSessionService_Mock_FindByID_NotFound(t *testing.T) {
	ss := NewSessionService(newMockSessionRepo())
	_, err := ss.FindByID(context.Background(), uuid.New())
	if err != ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestSessionService_Mock_FindByID_Found(t *testing.T) {
	sessionRepo := newMockSessionRepo()
	ss := NewSessionService(sessionRepo)
	id := uuid.New()
	sessionRepo.sessions[id] = &domain.Session{ID: id, UserID: uuid.New()}
	s, err := ss.FindByID(context.Background(), id)
	if err != nil || s.ID != id {
		t.Error("expected found")
	}
}

func TestSessionService_Mock_Revoke(t *testing.T) {
	sessionRepo := newMockSessionRepo()
	ss := NewSessionService(sessionRepo)
	id := uuid.New()
	sessionRepo.sessions[id] = &domain.Session{ID: id, ExpiresAt: time.Now().Add(time.Hour)}
	_ = ss.Revoke(context.Background(), id)
	if sessionRepo.sessions[id].RevokedAt == nil {
		t.Error("expected revoked")
	}
}

func TestSessionService_Mock_ListByUser(t *testing.T) {
	sessionRepo := newMockSessionRepo()
	ss := NewSessionService(sessionRepo)
	uid := uuid.New()
	tid := uuid.New()
	sessionRepo.sessions[uuid.New()] = &domain.Session{ID: uuid.New(), TenantID: tid, UserID: uid}
	sessionRepo.sessions[uuid.New()] = &domain.Session{ID: uuid.New(), TenantID: tid, UserID: uuid.New()}
	result, err := ss.ListByUser(context.Background(), tid, uid)
	if err != nil || len(result) != 1 {
		t.Errorf("expected 1 session, got %d", len(result))
	}
}

// ============== PasswordService ==============

func TestPasswordService_Mock_SetPassword(t *testing.T) {
	credRepo := newMockCredRepo()
	rdb := newTestRedis(t)
	ps := NewPasswordService(conf.Default().Password, credRepo, rdb)
	cred := &domain.Credential{ID: uuid.New(), Secret: "old"}
	err := ps.SetPassword(context.Background(), cred, "NewStrongPass123")
	if err != nil {
		t.Fatalf("SetPassword: %v", err)
	}
	if len(credRepo.history) == 0 {
		t.Error("expected history entry")
	}
}

func TestPasswordService_Mock_CheckHistory_Reuse(t *testing.T) {
	credRepo := newMockCredRepo()
	rdb := newTestRedis(t)
	ps := NewPasswordService(conf.PasswordPolicy{HistoryCount: 5}, credRepo, rdb)
	oldHash, _ := crypto.HashPassword("OldPassword123Ab")
	credRepo.history = append(credRepo.history, &domain.CredentialHistoryEntry{Secret: oldHash})
	err := ps.CheckHistory(context.Background(), uuid.New(), uuid.New(), "OldPassword123Ab")
	if err != ErrPasswordReused {
		t.Errorf("expected ErrPasswordReused, got %v", err)
	}
	err = ps.CheckHistory(context.Background(), uuid.New(), uuid.New(), "BrandNewPass456")
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}
