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

// ============ Mock Repositories ============

type tCredRepo struct {
	byName  map[string]*domain.Credential
	byUser  map[uuid.UUID]*domain.Credential
	history []*domain.CredentialHistoryEntry
}

func newTCredRepo() *tCredRepo {
	return &tCredRepo{byName: make(map[string]*domain.Credential), byUser: make(map[uuid.UUID]*domain.Credential)}
}

func (m *tCredRepo) FindByIDentifier(_ context.Context, _ uuid.UUID, n string) (*domain.Credential, error) {
	return m.byName[n], nil
}
func (m *tCredRepo) FindByUserID(_ context.Context, _ uuid.UUID, u uuid.UUID) (*domain.Credential, error) {
	return m.byUser[u], nil
}
func (m *tCredRepo) Create(_ context.Context, c *domain.Credential) error {
	m.byName[c.Identifier] = c
	m.byUser[c.UserID] = c
	return nil
}
func (m *tCredRepo) UpdateFailedAttempts(_ context.Context, _ uuid.UUID, _ int, _ *time.Time) error { return nil }
func (m *tCredRepo) UpdateSecret(_ context.Context, _ uuid.UUID, _ string) error                   { return nil }
func (m *tCredRepo) AddToHistory(_ context.Context, _ uuid.UUID, _ uuid.UUID, s string) error {
	m.history = append(m.history, &domain.CredentialHistoryEntry{Secret: s})
	return nil
}
func (m *tCredRepo) GetHistory(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ int) ([]domain.CredentialHistoryEntry, error) {
	r := make([]domain.CredentialHistoryEntry, len(m.history))
	for i, h := range m.history {
		r[i] = *h
	}
	return r, nil
}

type tSessionRepo struct {
	s map[uuid.UUID]*domain.Session
}

func newTSessionRepo() *tSessionRepo { return &tSessionRepo{s: make(map[uuid.UUID]*domain.Session)} }
func (m *tSessionRepo) Create(_ context.Context, s *domain.Session) error {
	m.s[s.ID] = s
	return nil
}
func (m *tSessionRepo) FindByTokenHash(_ context.Context, _ string) (*domain.Session, error) { return nil, nil }
func (m *tSessionRepo) FindByID(_ context.Context, id uuid.UUID) (*domain.Session, error)     { return m.s[id], nil }
func (m *tSessionRepo) ListByUser(_ context.Context, _ uuid.UUID, u uuid.UUID) ([]*domain.Session, error) {
	var r []*domain.Session
	for _, s := range m.s {
		if s.UserID == u {
			r = append(r, s)
		}
	}
	return r, nil
}
func (m *tSessionRepo) Revoke(_ context.Context, id uuid.UUID) error {
	if s, ok := m.s[id]; ok {
		now := time.Now()
		s.RevokedAt = &now
	}
	return nil
}
func (m *tSessionRepo) RevokeAllForUser(_ context.Context, _ uuid.UUID, u uuid.UUID, _ uuid.UUID) error {
	for _, s := range m.s {
		if s.UserID == u {
			now := time.Now()
			s.RevokedAt = &now
		}
	}
	return nil
}
func (m *tSessionRepo) DeleteExpired(_ context.Context, _ time.Time) (int64, error) { return 0, nil }

type tRefreshRepo struct {
	t map[string]*domain.RefreshToken
}

func newTRefreshRepo() *tRefreshRepo { return &tRefreshRepo{t: make(map[string]*domain.RefreshToken)} }
func (m *tRefreshRepo) Create(_ context.Context, t *domain.RefreshToken) error {
	m.t[t.TokenHash] = t
	return nil
}
func (m *tRefreshRepo) FindByHash(_ context.Context, h string) (*domain.RefreshToken, error) { return m.t[h], nil }
func (m *tRefreshRepo) Revoke(_ context.Context, id uuid.UUID) error {
	for _, t := range m.t {
		if t.ID == id {
			now := time.Now()
			t.RevokedAt = &now
		}
	}
	return nil
}
func (m *tRefreshRepo) RevokeAllForSession(_ context.Context, _ uuid.UUID) error              { return nil }
func (m *tRefreshRepo) RevokeAllForUser(_ context.Context, _ uuid.UUID, _ uuid.UUID) error    { return nil }

// ============ Helpers ============

func tCtxTenant() (context.Context, uuid.UUID) {
	tid := uuid.New()
	return tenant.WithContext(context.Background(), &tenant.Context{TenantID: tid, IsolationLevel: tenant.IsolationShared}), tid
}

func tRedis(t *testing.T) *redis.Client {
	t.Helper()
	mr := miniredis.RunT(t)
	return redis.NewClient(&redis.Options{Addr: mr.Addr()})
}

func tNewAuthSvc(t *testing.T) (*AuthService, *tCredRepo, *tSessionRepo, *tRefreshRepo) {
	t.Helper()
	rdb := tRedis(t)
	cr := newTCredRepo()
	sr := newTSessionRepo()
	rr := newTRefreshRepo()
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
	ts, err := NewTokenService(kp, cfg.JWT.Issuer, cfg.JWT.Audience, cfg.JWT.AccessTokenTTL, rr, rdb)
	if err != nil {
		t.Fatalf("NewTokenService: %v", err)
	}
	return &AuthService{
		cfg: cfg, chain: authprovider.NewChain(), credentialRepo: cr,
		tokenService: ts, sessionService: NewSessionService(sr),
		passwordService: NewPasswordService(cfg.Password, cr, rdb),
		rateLimiter:     NewRateLimiter(rdb), identityClient: &NoopIdentityClient{},
		emailService:    NewEmailService(rdb),
	}, cr, sr, rr
}

type tSuccessProvider struct{ uid uuid.UUID }

func (p *tSuccessProvider) Type() authprovider.ProviderType { return authprovider.ProviderLocal }
func (p *tSuccessProvider) Name() string                    { return "ok" }
func (p *tSuccessProvider) Authenticate(_ context.Context, _ authprovider.Credentials) (*authprovider.AuthResult, error) {
	u := p.uid
	return &authprovider.AuthResult{Provider: authprovider.ProviderLocal, LinkedUser: &u}, nil
}

// ============ Register ============

func TestAuthSvc_Register_Success(t *testing.T) {
	svc, cr, _, _ := tNewAuthSvc(t)
	tid, uid := uuid.New(), uuid.New()
	if err := svc.Register(context.Background(), tid, uid, "user1", "StrongPass123"); err != nil {
		t.Fatalf("Register: %v", err)
	}
	if cr.byName["user1"] == nil {
		t.Fatal("expected credential stored")
	}
}

func TestAuthSvc_Register_WeakPassword(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	if err := svc.Register(context.Background(), uuid.New(), uuid.New(), "u", "short"); err != ErrPasswordTooShort {
		t.Errorf("expected ErrPasswordTooShort, got %v", err)
	}
}

func TestAuthSvc_Register_Duplicate(t *testing.T) {
	svc, cr, _, _ := tNewAuthSvc(t)
	cr.byName["dup"] = &domain.Credential{UserID: uuid.New()}
	if err := svc.Register(context.Background(), uuid.New(), uuid.New(), "dup", "StrongPass123"); err == nil {
		t.Error("expected duplicate error")
	}
}

// ============ ChangePassword ============

func TestAuthSvc_ChangePassword_Success(t *testing.T) {
	svc, cr, _, _ := tNewAuthSvc(t)
	ctx, tid := tCtxTenant()
	uid := uuid.New()
	h, _ := crypto.HashPassword("OldPass123Ab")
	c := &domain.Credential{ID: uuid.New(), TenantID: tid, UserID: uid, Identifier: "u", Secret: h, Enabled: true}
	cr.byUser[uid] = c
	if err := svc.ChangePassword(ctx, tid, uid, "OldPass123Ab", "NewStrongPass456"); err != nil {
		t.Fatalf("ChangePassword: %v", err)
	}
}

func TestAuthSvc_ChangePassword_WrongOld(t *testing.T) {
	svc, cr, _, _ := tNewAuthSvc(t)
	ctx, tid := tCtxTenant()
	uid := uuid.New()
	h, _ := crypto.HashPassword("OldPass123Ab")
	cr.byUser[uid] = &domain.Credential{ID: uuid.New(), TenantID: tid, UserID: uid, Secret: h, Enabled: true}
	if err := svc.ChangePassword(ctx, tid, uid, "Wrong", "NewStrongPass456"); err != ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestAuthSvc_ChangePassword_NotFound(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	ctx, tid := tCtxTenant()
	if err := svc.ChangePassword(ctx, tid, uuid.New(), "x", "NewStrongPass456"); err == nil {
		t.Error("expected error")
	}
}

// ============ ForgotPassword / ResetPassword ============

func TestAuthSvc_ForgotPassword_NoUser(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	if err := svc.ForgotPassword(context.Background(), uuid.New(), "nobody@test.com"); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestAuthSvc_ForgotPassword_ExistingUser(t *testing.T) {
	svc, cr, _, _ := tNewAuthSvc(t)
	tid := uuid.New()
	cr.byName["u@test.com"] = &domain.Credential{UserID: uuid.New(), Identifier: "u@test.com"}
	if err := svc.ForgotPassword(context.Background(), tid, "u@test.com"); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestAuthSvc_ResetPassword_InvalidToken(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	if err := svc.ResetPassword(context.Background(), "bad", "NewStrongPass456"); err != ErrInvalidResetToken {
		t.Errorf("expected ErrInvalidResetToken, got %v", err)
	}
}

func TestAuthSvc_ResetPassword_Success(t *testing.T) {
	svc, cr, _, _ := tNewAuthSvc(t)
	ctx, tid := tCtxTenant()
	uid := uuid.New()
	h, _ := crypto.HashPassword("OldPass123Ab")
	cr.byUser[uid] = &domain.Credential{ID: uuid.New(), TenantID: tid, UserID: uid, Secret: h, Enabled: true, Type: domain.CredentialPassword}
	tok, err := svc.passwordService.IssueResetToken(ctx, uid, tid)
	if err != nil {
		t.Fatalf("IssueResetToken: %v", err)
	}
	if err := svc.ResetPassword(ctx, tok, "NewStrongPass456"); err != nil {
		t.Fatalf("ResetPassword: %v", err)
	}
}

// ============ Login ============

func TestAuthSvc_Login_RateLimited(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	ctx, _ := tCtxTenant()
	for i := 0; i < svc.cfg.RateLimit.LoginPerMinute; i++ {
		_ = svc.rateLimiter.CheckAndIncrement(ctx, "login:1.2.3.4", svc.cfg.RateLimit.LoginPerMinute)
	}
	_, err := svc.Login(ctx, "u", "p", "1.2.3.4", "agent")
	if err != ErrRateLimited {
		t.Errorf("expected ErrRateLimited, got %v", err)
	}
}

func TestAuthSvc_Login_InvalidCredentials(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	ctx, _ := tCtxTenant()
	_, err := svc.Login(ctx, "u", "p", "9.9.9.9", "agent")
	if err != ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestAuthSvc_Login_Success(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	svc.chain = authprovider.NewChain(&tSuccessProvider{uid: uuid.New()})
	ctx, _ := tCtxTenant()
	tok, err := svc.Login(ctx, "u", "p", "1.1.1.1", "agent")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if tok.AccessToken == "" || tok.RefreshToken == "" || tok.SessionID == "" {
		t.Error("expected non-empty tokens")
	}
}

// ============ Refresh ============

func TestAuthSvc_Refresh_InvalidToken(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	_, err := svc.Refresh(context.Background(), "bad")
	if err == nil {
		t.Error("expected error")
	}
}

func TestAuthSvc_Refresh_Success(t *testing.T) {
	svc, _, _, rr := tNewAuthSvc(t)
	tid, uid, sid := uuid.New(), uuid.New(), uuid.New()
	pt := "test-refresh-token-val"
	th := hashToken(pt)
	now := time.Now()
	rr.t[th] = &domain.RefreshToken{ID: uuid.New(), TenantID: tid, UserID: uid, SessionID: sid, TokenHash: th, ExpiresAt: now.Add(30 * 24 * time.Hour), CreatedAt: now}
	tok, err := svc.Refresh(context.Background(), pt)
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if tok.AccessToken == "" {
		t.Error("expected non-empty access token")
	}
	if tok.RefreshToken == pt {
		t.Error("token should be rotated")
	}
}

// ============ Logout / Sessions / Cleanup ============

func TestAuthSvc_Logout(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	if err := svc.Logout(context.Background(), "bad"); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestAuthSvc_ListSessions(t *testing.T) {
	svc, _, sr, _ := tNewAuthSvc(t)
	ctx, tid := tCtxTenant()
	uid := uuid.New()
	sid := uuid.New()
	sr.s[sid] = &domain.Session{ID: sid, TenantID: tid, UserID: uid}
	ss, err := svc.ListSessions(ctx, tid, uid)
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if len(ss) != 1 {
		t.Errorf("expected 1, got %d", len(ss))
	}
}

func TestAuthSvc_RevokeSession(t *testing.T) {
	svc, _, sr, _ := tNewAuthSvc(t)
	sid := uuid.New()
	sr.s[sid] = &domain.Session{ID: sid, ExpiresAt: time.Now().Add(time.Hour)}
	if err := svc.RevokeSession(context.Background(), sid); err != nil {
		t.Fatalf("RevokeSession: %v", err)
	}
	if sr.s[sid].RevokedAt == nil {
		t.Error("expected revoked")
	}
}

func TestAuthSvc_CleanupExpired(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	if _, err := svc.CleanupExpired(context.Background()); err != nil {
		t.Fatalf("CleanupExpired: %v", err)
	}
}

func TestAuthSvc_MFAService_Nil(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	if svc.MFAService() != nil {
		t.Error("expected nil")
	}
}

// ============ SessionService with Mock ============

func TestSessSvc_Create(t *testing.T) {
	sr := newTSessionRepo()
	ss := NewSessionService(sr)
	ctx := context.Background()
	tok, s, err := ss.Create(ctx, CreateSessionParams{TenantID: uuid.New(), UserID: uuid.New(), IPAddress: "1.1.1.1", UserAgent: "Chrome", TTL: time.Hour})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if tok == "" || s == nil {
		t.Fatal("expected non-empty token and session")
	}
	if sr.s[s.ID] == nil {
		t.Error("expected session stored")
	}
}

func TestSessSvc_FindByID_NotFound(t *testing.T) {
	ss := NewSessionService(newTSessionRepo())
	_, err := ss.FindByID(context.Background(), uuid.New())
	if err != ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestSessSvc_FindByID_Found(t *testing.T) {
	sr := newTSessionRepo()
	ss := NewSessionService(sr)
	sid := uuid.New()
	sr.s[sid] = &domain.Session{ID: sid, UserID: uuid.New()}
	s, err := ss.FindByID(context.Background(), sid)
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if s.ID != sid {
		t.Errorf("expected %s, got %s", sid, s.ID)
	}
}

func TestSessSvc_Revoke(t *testing.T) {
	sr := newTSessionRepo()
	ss := NewSessionService(sr)
	sid := uuid.New()
	sr.s[sid] = &domain.Session{ID: sid, ExpiresAt: time.Now().Add(time.Hour)}
	if err := ss.Revoke(context.Background(), sid); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	if sr.s[sid].RevokedAt == nil {
		t.Error("expected revoked")
	}
}

func TestSessSvc_ListByUser(t *testing.T) {
	sr := newTSessionRepo()
	ss := NewSessionService(sr)
	uid := uuid.New()
	tid := uuid.New()
	sr.s[uuid.New()] = &domain.Session{ID: uuid.New(), TenantID: tid, UserID: uid}
	sr.s[uuid.New()] = &domain.Session{ID: uuid.New(), TenantID: tid, UserID: uid}
	sr.s[uuid.New()] = &domain.Session{ID: uuid.New(), TenantID: tid, UserID: uuid.New()}
	ss2, err := ss.ListByUser(context.Background(), tid, uid)
	if err != nil {
		t.Fatalf("ListByUser: %v", err)
	}
	if len(ss2) != 2 {
		t.Errorf("expected 2, got %d", len(ss2))
	}
}

// ============ PasswordService with Mock ============

func TestPwSvc_SetPassword(t *testing.T) {
	cr := newTCredRepo()
	ps := NewPasswordService(conf.Default().Password, cr, nil)
	c := &domain.Credential{ID: uuid.New(), Secret: "old"}
	if err := ps.SetPassword(context.Background(), c, "NewStrongPass123"); err != nil {
		t.Fatalf("SetPassword: %v", err)
	}
	if len(cr.history) == 0 {
		t.Error("expected history entry")
	}
}

func TestPwSvc_CheckHistory_Reuse(t *testing.T) {
	cr := newTCredRepo()
	h, _ := crypto.HashPassword("UsedPassword123")
	cr.history = append(cr.history, &domain.CredentialHistoryEntry{Secret: h})
	ps := NewPasswordService(conf.PasswordPolicy{HistoryCount: 5}, cr, nil)
	if err := ps.CheckHistory(context.Background(), uuid.New(), uuid.New(), "UsedPassword123"); err != ErrPasswordReused {
		t.Errorf("expected ErrPasswordReused, got %v", err)
	}
}

// ============ LocalProvider with Mock ============

func TestLocalProv_Auth_Success(t *testing.T) {
	cr := newTCredRepo()
	ctx, tid := tCtxTenant()
	uid := uuid.New()
	h, _ := crypto.HashPassword("StrongPass123")
	cr.byName["u"] = &domain.Credential{ID: uuid.New(), TenantID: tid, UserID: uid, Identifier: "u", Secret: h, Enabled: true, Type: domain.CredentialPassword}
	p := NewLocalProvider(cr, conf.Default().Password)
	r, err := p.Authenticate(ctx, authprovider.Credentials{Username: "u", Password: "StrongPass123"})
	if err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
	if r.LinkedUser == nil || *r.LinkedUser != uid {
		t.Error("expected linked user")
	}
}

func TestLocalProv_Auth_WrongPassword(t *testing.T) {
	cr := newTCredRepo()
	ctx, tid := tCtxTenant()
	h, _ := crypto.HashPassword("StrongPass123")
	cr.byName["u"] = &domain.Credential{ID: uuid.New(), TenantID: tid, UserID: uuid.New(), Identifier: "u", Secret: h, Enabled: true, Type: domain.CredentialPassword}
	p := NewLocalProvider(cr, conf.Default().Password)
	_, err := p.Authenticate(ctx, authprovider.Credentials{Username: "u", Password: "Wrong"})
	if err != ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLocalProv_Auth_Disabled(t *testing.T) {
	cr := newTCredRepo()
	ctx, tid := tCtxTenant()
	h, _ := crypto.HashPassword("StrongPass123")
	cr.byName["d"] = &domain.Credential{TenantID: tid, UserID: uuid.New(), Identifier: "d", Secret: h, Enabled: false, Type: domain.CredentialPassword}
	p := NewLocalProvider(cr, conf.Default().Password)
	_, err := p.Authenticate(ctx, authprovider.Credentials{Username: "d", Password: "StrongPass123"})
	if err == nil {
		t.Error("expected error for disabled")
	}
}

func TestLocalProv_CreateCredential(t *testing.T) {
	cr := newTCredRepo()
	p := NewLocalProvider(cr, conf.Default().Password)
	if err := p.CreateCredential(context.Background(), CreateCredentialParams{TenantID: uuid.New(), UserID: uuid.New(), Identifier: "x", Password: "StrongPass123"}); err != nil {
		t.Fatalf("CreateCredential: %v", err)
	}
	if cr.byName["x"] == nil {
		t.Error("expected stored")
	}
}
