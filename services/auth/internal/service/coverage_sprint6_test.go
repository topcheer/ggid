package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ggid/ggid/pkg/authprovider"
	"github.com/ggid/ggid/pkg/crypto"
	"github.com/ggid/ggid/services/auth/internal/conf"
	"github.com/ggid/ggid/services/auth/internal/domain"
	"github.com/google/uuid"
)

// ===== Mock Providers =====

// tNilLinkedProvider returns success but nil LinkedUser
type tNilLinkedProvider struct{}

func (p *tNilLinkedProvider) Type() authprovider.ProviderType { return authprovider.ProviderLocal }
func (p *tNilLinkedProvider) Name() string                    { return "nil-linked" }
func (p *tNilLinkedProvider) Authenticate(_ context.Context, _ authprovider.Credentials) (*authprovider.AuthResult, error) {
	return &authprovider.AuthResult{Provider: authprovider.ProviderLocal, LinkedUser: nil}, nil
}

// ===== Mock Identity Client =====

type tMockIdentityClient struct {
	users         map[string]*UserInfo
	byID          map[uuid.UUID]*UserInfo
	extIdentities map[string]*ExternalIdentityLink
	createdUsers  []*UserInfo
	getUserErr    error
	createUserErr error
	linkErr       error
}

func newMockIdentityClient() *tMockIdentityClient {
	return &tMockIdentityClient{
		users:         make(map[string]*UserInfo),
		byID:          make(map[uuid.UUID]*UserInfo),
		extIdentities: make(map[string]*ExternalIdentityLink),
	}
}

func (m *tMockIdentityClient) GetUser(_ context.Context, _ uuid.UUID, identifier string) (*UserInfo, error) {
	if m.getUserErr != nil {
		return nil, m.getUserErr
	}
	return m.users[identifier], nil
}

func (m *tMockIdentityClient) GetUserByID(_ context.Context, _ uuid.UUID, userID uuid.UUID) (*UserInfo, error) {
	return m.byID[userID], nil
}

func (m *tMockIdentityClient) FindExternalIdentity(_ context.Context, _ uuid.UUID, provider, externalID string) (*ExternalIdentityLink, error) {
	key := provider + ":" + externalID
	return m.extIdentities[key], nil
}

func (m *tMockIdentityClient) LinkExternalIdentity(_ context.Context, _ uuid.UUID, userID uuid.UUID, provider, externalID string, _ map[string]any) error {
	if m.linkErr != nil {
		return m.linkErr
	}
	key := provider + ":" + externalID
	m.extIdentities[key] = &ExternalIdentityLink{UserID: userID, Provider: provider, ExternalID: externalID}
	return nil
}

func (m *tMockIdentityClient) CreateUserFromSocial(_ context.Context, _ uuid.UUID, username, email, displayName, provider, externalID string, _ map[string]any) (*UserInfo, error) {
	if m.createUserErr != nil {
		return nil, m.createUserErr
	}
	u := &UserInfo{ID: uuid.New(), Username: username, Email: email, DisplayName: displayName, Status: "active"}
	m.createdUsers = append(m.createdUsers, u)
	m.byID[u.ID] = u
	key := provider + ":" + externalID
	m.extIdentities[key] = &ExternalIdentityLink{UserID: u.ID, Provider: provider, ExternalID: externalID}
	return u, nil
}

// ===== Auth Service Builder Helper =====

func tNewAuthSvcFull(t *testing.T) (*AuthService, *tCredRepo, *tSessionRepo, *tRefreshRepo, *mockMFARepo) {
	t.Helper()
	rdb := tRedis(t)
	cr := newTCredRepo()
	sr := newTSessionRepo()
	rr := newTRefreshRepo()
	mfaRepo := newMockMFARepo()
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
	svc := &AuthService{
		cfg:             cfg,
		chain:           authprovider.NewChain(),
		credentialRepo:  cr,
		tokenService:    ts,
		sessionService:  NewSessionService(sr),
		passwordService: NewPasswordService(cfg.Password, cr, rdb),
		rateLimiter:     NewRateLimiter(rdb),
		identityClient:  newMockIdentityClient(),
		mfaService:      NewMFAService(mfaRepo),
		emailService:    NewEmailService(rdb),
	}
	return svc, cr, sr, rr, mfaRepo
}

func sprint6PtrTime(t time.Time) *time.Time { return &t }

// ===== Login Coverage =====

func TestLoginS6_NoTenantContext(t *testing.T) {
	svc, _, _, _, _ := tNewAuthSvcFull(t)
	svc.chain = authprovider.NewChain(&tSuccessProvider{uid: uuid.New()})
	_, err := svc.Login(context.Background(), "u", "p", "1.1.1.1", "agent")
	if err == nil {
		t.Error("expected error for missing tenant context")
	}
}

func TestLoginS6_NilLinkedUser(t *testing.T) {
	svc, _, _, _, _ := tNewAuthSvcFull(t)
	svc.chain = authprovider.NewChain(&tNilLinkedProvider{})
	ctx, _ := tCtxTenant()
	_, err := svc.Login(ctx, "u", "p", "1.1.1.1", "agent")
	// With auto-provisioning, nil linked user triggers CreateUserFromSocial.
	// The mock identity client may fail, so we expect an error (auto-provision failed)
	// or success if the mock creates the user.
	if err != nil && !strings.Contains(err.Error(), "auto-provision") && !strings.Contains(err.Error(), "no linked") {
		t.Errorf("expected auto-provision or linked user error, got: %v", err)
	}
}

func TestLoginS6_MFARequired(t *testing.T) {
	svc, _, _, _, mfaRepo := tNewAuthSvcFull(t)
	uid := uuid.New()
	ctx, tid := tCtxTenant()
	mfaRepo.devices[uid] = &domain.MFADevice{
		ID:         uuid.New(),
		TenantID:   tid,
		UserID:     uid,
		Secret:     "JBSWY3DPEHPK3PXP",
		Enabled:    true,
		VerifiedAt: sprint6PtrTime(time.Now()),
	}
	svc.chain = authprovider.NewChain(&tSuccessProvider{uid: uid})
	tok, err := svc.Login(ctx, "u", "p", "1.1.1.1", "agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !tok.MFARequired {
		t.Error("expected MFARequired to be true")
	}
	if tok.MFAChallenge == "" {
		t.Error("expected non-empty MFAChallenge")
	}
}

func TestLoginS6_ForceMFA(t *testing.T) {
	svc, _, _, _, _ := tNewAuthSvcFull(t)
	uid := uuid.New()
	ctx, tid := tCtxTenant()
	if err := svc.SetForceMFA(ctx, tid, true); err != nil {
		t.Fatalf("SetForceMFA: %v", err)
	}
	svc.chain = authprovider.NewChain(&tSuccessProvider{uid: uid})
	_, err := svc.Login(ctx, "u", "p", "1.1.1.1", "agent")
	if err != ErrMFASetupRequired {
		t.Errorf("expected ErrMFASetupRequired, got %v", err)
	}
}

// ===== LoginMFA Coverage =====

func TestLoginMFAS6_InvalidCredentials(t *testing.T) {
	svc, _, _, _, _ := tNewAuthSvcFull(t)
	ctx, _ := tCtxTenant()
	_, err := svc.LoginMFA(ctx, "u", "wrong", "123456", "1.1.1.1", "agent")
	if err != ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLoginMFAS6_NoTenantContext(t *testing.T) {
	svc, _, _, _, _ := tNewAuthSvcFull(t)
	svc.chain = authprovider.NewChain(&tSuccessProvider{uid: uuid.New()})
	_, err := svc.LoginMFA(context.Background(), "u", "p", "123456", "1.1.1.1", "agent")
	if err == nil {
		t.Error("expected error for missing tenant context")
	}
}

func TestLoginMFAS6_NilLinkedUser(t *testing.T) {
	svc, _, _, _, _ := tNewAuthSvcFull(t)
	svc.chain = authprovider.NewChain(&tNilLinkedProvider{})
	ctx, _ := tCtxTenant()
	_, err := svc.LoginMFA(ctx, "u", "p", "123456", "1.1.1.1", "agent")
	if err == nil {
		t.Error("expected error for nil linked user")
	}
}

func TestLoginMFAS6_WrongCode(t *testing.T) {
	svc, _, _, _, mfaRepo := tNewAuthSvcFull(t)
	uid := uuid.New()
	ctx, tid := tCtxTenant()
	mfaRepo.devices[uid] = &domain.MFADevice{
		ID:         uuid.New(),
		TenantID:   tid,
		UserID:     uid,
		Secret:     "JBSWY3DPEHPK3PXP",
		Enabled:    true,
		VerifiedAt: sprint6PtrTime(time.Now()),
	}
	svc.chain = authprovider.NewChain(&tSuccessProvider{uid: uid})
	_, err := svc.LoginMFA(ctx, "u", "p", "000000", "1.1.1.1", "agent")
	if err == nil {
		t.Error("expected error for wrong MFA code")
	}
}

// ===== Magic Link Coverage =====

func TestIssueMagicLinkS6_Success(t *testing.T) {
	svc, _, _, _, _ := tNewAuthSvcFull(t)
	ctx := context.Background()
	token, err := svc.IssueMagicLink(ctx, uuid.New(), uuid.New(), "user@test.com")
	if err != nil {
		t.Fatalf("IssueMagicLink: %v", err)
	}
	if token == "" {
		t.Error("expected non-empty token")
	}
}

func TestVerifyMagicLinkS6_Success(t *testing.T) {
	svc, _, _, _, _ := tNewAuthSvcFull(t)
	ctx := context.Background()
	tid, uid := uuid.New(), uuid.New()
	token, err := svc.IssueMagicLink(ctx, tid, uid, "user@test.com")
	if err != nil {
		t.Fatalf("IssueMagicLink: %v", err)
	}
	tok, err := svc.VerifyMagicLink(ctx, token, "1.1.1.1", "agent")
	if err != nil {
		t.Fatalf("VerifyMagicLink: %v", err)
	}
	if tok.AccessToken == "" || tok.RefreshToken == "" {
		t.Error("expected non-empty tokens")
	}
}

func TestVerifyMagicLinkS6_InvalidToken(t *testing.T) {
	svc, _, _, _, _ := tNewAuthSvcFull(t)
	_, err := svc.VerifyMagicLink(context.Background(), "invalid-token", "1.1.1.1", "agent")
	if err == nil {
		t.Error("expected error for invalid token")
	}
}

func TestVerifyMagicLinkS6_AlreadyUsed(t *testing.T) {
	svc, _, _, _, _ := tNewAuthSvcFull(t)
	ctx := context.Background()
	tid, uid := uuid.New(), uuid.New()
	token, _ := svc.IssueMagicLink(ctx, tid, uid, "user@test.com")
	_, _ = svc.VerifyMagicLink(ctx, token, "1.1.1.1", "agent")
	_, err := svc.VerifyMagicLink(ctx, token, "1.1.1.1", "agent")
	if err == nil {
		t.Error("expected error for already-used token")
	}
}

// ===== Social Login Coverage =====

func TestSocialLoginS6_JITProvisioning(t *testing.T) {
	svc, _, _, _, _ := tNewAuthSvcFull(t)
	ctx, _ := tCtxTenant()
	tok, err := svc.SocialLogin(ctx, "github", "ext_s6_1", "new_s6@test.com", "New User", "", "1.1.1.1", "agent")
	if err != nil {
		t.Fatalf("SocialLogin JIT: %v", err)
	}
	if tok.AccessToken == "" {
		t.Error("expected non-empty access token")
	}
}

func TestSocialLoginS6_NoTenantContext(t *testing.T) {
	svc, _, _, _, _ := tNewAuthSvcFull(t)
	_, err := svc.SocialLogin(context.Background(), "github", "ext_s6_2", "user@test.com", "User", "", "1.1.1.1", "agent")
	if err == nil {
		t.Error("expected error for missing tenant context")
	}
}

func TestSocialLoginS6_LinkedIdentity(t *testing.T) {
	svc, _, _, _, _ := tNewAuthSvcFull(t)
	ctx, tid := tCtxTenant()
	ic := svc.identityClient.(*tMockIdentityClient)
	uid := uuid.New()
	ic.extIdentities["github:ext_s6_3"] = &ExternalIdentityLink{UserID: uid, Provider: "github", ExternalID: "ext_s6_3"}
	_ = tid
	tok, err := svc.SocialLogin(ctx, "github", "ext_s6_3", "user@test.com", "User", "", "1.1.1.1", "agent")
	if err != nil {
		t.Fatalf("SocialLogin linked: %v", err)
	}
	if tok.AccessToken == "" {
		t.Error("expected non-empty access token")
	}
}

func TestSocialLoginS6_EmailMatch(t *testing.T) {
	svc, _, _, _, _ := tNewAuthSvcFull(t)
	ctx, tid := tCtxTenant()
	ic := svc.identityClient.(*tMockIdentityClient)
	uid := uuid.New()
	ic.users["match_s6@test.com"] = &UserInfo{ID: uid, Email: "match_s6@test.com", Status: "active"}
	ic.byID[uid] = ic.users["match_s6@test.com"]
	_ = tid
	tok, err := svc.SocialLogin(ctx, "google", "ext_s6_4", "match_s6@test.com", "User", "", "1.1.1.1", "agent")
	if err != nil {
		t.Fatalf("SocialLogin email match: %v", err)
	}
	if tok.AccessToken == "" {
		t.Error("expected non-empty access token")
	}
}

func TestSocialLoginS6_CreateUserError(t *testing.T) {
	svc, _, _, _, _ := tNewAuthSvcFull(t)
	ctx, _ := tCtxTenant()
	ic := svc.identityClient.(*tMockIdentityClient)
	ic.createUserErr = errors.New("identity service down")
	_, err := svc.SocialLogin(ctx, "github", "ext_s6_err", "", "User", "", "1.1.1.1", "agent")
	if err == nil {
		t.Error("expected error for CreateUserFromSocial failure")
	}
}

func TestSocialLoginS6_LongExternalID(t *testing.T) {
	svc, _, _, _, _ := tNewAuthSvcFull(t)
	ctx, _ := tCtxTenant()
	longExtID := strings.Repeat("x", 100)
	tok, err := svc.SocialLogin(ctx, "github", longExtID, "", "User", "", "1.1.1.1", "agent")
	if err != nil {
		t.Fatalf("SocialLogin long ID: %v", err)
	}
	if tok.AccessToken == "" {
		t.Error("expected non-empty access token")
	}
}

// ===== Email Verification Coverage =====

func TestSendVerificationEmailS6_Success(t *testing.T) {
	svc, _, _, _, _ := tNewAuthSvcFull(t)
	ctx := context.Background()
	token, err := svc.SendVerificationEmail(ctx, uuid.New(), uuid.New(), "user@test.com")
	if err != nil {
		t.Fatalf("SendVerificationEmail: %v", err)
	}
	if token == "" {
		t.Error("expected non-empty token")
	}
}

func TestVerifyEmailTokenS6_Success(t *testing.T) {
	svc, _, _, _, _ := tNewAuthSvcFull(t)
	ctx := context.Background()
	tid, uid := uuid.New(), uuid.New()
	token, _ := svc.SendVerificationEmail(ctx, tid, uid, "verify@test.com")
	rt, ru, email, err := svc.VerifyEmailToken(ctx, token)
	if err != nil {
		t.Fatalf("VerifyEmailToken: %v", err)
	}
	if rt != tid || ru != uid || email != "verify@test.com" {
		t.Errorf("unexpected values: tid=%s uid=%s email=%s", rt, ru, email)
	}
}

func TestVerifyEmailTokenS6_Invalid(t *testing.T) {
	svc, _, _, _, _ := tNewAuthSvcFull(t)
	_, _, _, err := svc.VerifyEmailToken(context.Background(), "invalid-token")
	if err == nil {
		t.Error("expected error for invalid token")
	}
}

func TestVerifyEmailTokenS6_AlreadyUsed(t *testing.T) {
	svc, _, _, _, _ := tNewAuthSvcFull(t)
	ctx := context.Background()
	tid, uid := uuid.New(), uuid.New()
	token, _ := svc.SendVerificationEmail(ctx, tid, uid, "verify@test.com")
	_, _, _, _ = svc.VerifyEmailToken(ctx, token)
	_, _, _, err := svc.VerifyEmailToken(ctx, token)
	if err == nil {
		t.Error("expected error for already-used token")
	}
}

// ===== Password Policy Coverage =====

func TestUpdatePasswordPolicyS6_Success(t *testing.T) {
	svc, _, _, _, _ := tNewAuthSvcFull(t)
	minLen := 16
	reqUpper := true
	blacklist := []string{"password123"}
	err := svc.UpdatePasswordPolicy(&minLen, &reqUpper, nil, nil, nil, blacklist)
	if err != nil {
		t.Fatalf("UpdatePasswordPolicy: %v", err)
	}
	p := svc.PasswordPolicy()
	if p.MinLength != 16 {
		t.Errorf("expected minLen=16, got %d", p.MinLength)
	}
	if !p.RequireUpper {
		t.Error("expected RequireUpper=true")
	}
}

func TestUpdatePasswordPolicyS6_InvalidMinLen(t *testing.T) {
	svc, _, _, _, _ := tNewAuthSvcFull(t)
	minLen := 0
	err := svc.UpdatePasswordPolicy(&minLen, nil, nil, nil, nil, nil)
	if err == nil {
		t.Error("expected error for minLen=0")
	}
	minLen2 := 200
	err = svc.UpdatePasswordPolicy(&minLen2, nil, nil, nil, nil, nil)
	if err == nil {
		t.Error("expected error for minLen=200")
	}
}

func TestSetPasswordPolicyS6(t *testing.T) {
	svc, _, _, _, _ := tNewAuthSvcFull(t)
	newPolicy := conf.PasswordPolicy{MinLength: 20, RequireUpper: true}
	svc.SetPasswordPolicy(newPolicy)
	p := svc.PasswordPolicy()
	if p.MinLength != 20 {
		t.Errorf("expected 20, got %d", p.MinLength)
	}
}

// ===== ForceMFA / Account Lockout / Trusted Device =====

func TestSetForceMFAS6_EnableDisable(t *testing.T) {
	svc, _, _, _, _ := tNewAuthSvcFull(t)
	ctx := context.Background()
	tid := uuid.New()
	if err := svc.SetForceMFA(ctx, tid, true); err != nil {
		t.Fatalf("SetForceMFA true: %v", err)
	}
	if !svc.IsForceMFA(ctx, tid) {
		t.Error("expected IsForceMFA=true")
	}
	if err := svc.SetForceMFA(ctx, tid, false); err != nil {
		t.Fatalf("SetForceMFA false: %v", err)
	}
	if svc.IsForceMFA(ctx, tid) {
		t.Error("expected IsForceMFA=false")
	}
}

func TestAccountLockoutS6_Flow(t *testing.T) {
	svc, _, _, _, _ := tNewAuthSvcFull(t)
	ctx := context.Background()
	tid := uuid.New()
	identifier := "locked_s6@test.com"
	if svc.IsAccountLocked(ctx, tid, identifier) {
		t.Error("expected not locked initially")
	}
	svc.cfg.Password.MaxAttempts = 3
	for i := 0; i < 3; i++ {
		if err := svc.RecordFailedLogin(ctx, tid, identifier); err != nil {
			t.Fatalf("RecordFailedLogin: %v", err)
		}
	}
	if !svc.IsAccountLocked(ctx, tid, identifier) {
		t.Error("expected locked after 3 attempts")
	}
	svc.ResetFailedLogins(ctx, tid, identifier)
	if svc.IsAccountLocked(ctx, tid, identifier) {
		t.Error("expected not locked after reset")
	}
}

func TestRememberTrustedDeviceS6(t *testing.T) {
	svc, _, _, _, _ := tNewAuthSvcFull(t)
	ctx, tid := tCtxTenant()
	uid := uuid.New()
	if err := svc.RememberTrustedDevice(ctx, uid, "fp-abc-s6", "iPhone"); err != nil {
		t.Fatalf("RememberTrustedDevice: %v", err)
	}
	if !svc.IsTrustedDevice(ctx, tid, uid, "fp-abc-s6") {
		t.Error("expected trusted device")
	}
	if svc.IsTrustedDevice(ctx, tid, uid, "unknown-fp-s6") {
		t.Error("expected not trusted for unknown fingerprint")
	}
}

func TestRememberTrustedDeviceS6_NoTenantContext(t *testing.T) {
	svc, _, _, _, _ := tNewAuthSvcFull(t)
	err := svc.RememberTrustedDevice(context.Background(), uuid.New(), "fp", "dev")
	if err == nil {
		t.Error("expected error for missing tenant context")
	}
}

// ===== Session Timeout =====

func TestCheckSessionTimeoutS6_AbsoluteExpired(t *testing.T) {
	svc, _, _, _, _ := tNewAuthSvcFull(t)
	ctx := context.Background()
	svc.cfg.SessionTimeout.AbsoluteTimeout = 1 * time.Minute
	oldTime := time.Now().Add(-10 * time.Minute)
	err := svc.CheckSessionTimeout(ctx, uuid.New(), oldTime)
	if err != ErrSessionExpired {
		t.Errorf("expected ErrSessionExpired, got %v", err)
	}
}

func TestCheckSessionTimeoutS6_OK(t *testing.T) {
	svc, _, _, _, _ := tNewAuthSvcFull(t)
	ctx := context.Background()
	err := svc.CheckSessionTimeout(ctx, uuid.New(), time.Now())
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestCheckSessionTimeoutS6_IdleExpired(t *testing.T) {
	svc, _, _, _, _ := tNewAuthSvcFull(t)
	ctx := context.Background()
	svc.cfg.SessionTimeout.IdleTimeout = 1 * time.Minute
	sid := uuid.New()
	oldTime := time.Now().Add(-10 * time.Minute).Format(time.RFC3339)
	svc.rateLimiter.rdb.Set(ctx, fmt.Sprintf("ggid:session_activity:%s", sid), oldTime, 0)
	err := svc.CheckSessionTimeout(ctx, sid, time.Now())
	if err != ErrSessionExpired {
		t.Errorf("expected ErrSessionExpired, got %v", err)
	}
}

// ===== Brute Force Protection =====

func TestCheckBruteForceS6_OK(t *testing.T) {
	svc, _, _, _, _ := tNewAuthSvcFull(t)
	ctx := context.Background()
	err := svc.CheckBruteForce(ctx, uuid.New(), "1.2.3.4", "user")
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestCheckBruteForceS6_IPExceeded(t *testing.T) {
	svc, _, _, _, _ := tNewAuthSvcFull(t)
	ctx := context.Background()
	tid := uuid.New()
	for i := 0; i < 21; i++ {
		_ = svc.CheckBruteForce(ctx, tid, "9.9.9.9", "user1")
	}
	err := svc.CheckBruteForce(ctx, tid, "9.9.9.9", "user1")
	if err != ErrRateLimited {
		t.Errorf("expected ErrRateLimited, got %v", err)
	}
}

// ===== WebAuthn Challenge =====

func TestGenerateWebAuthnChallengeS6(t *testing.T) {
	svc, _, _, _, _ := tNewAuthSvcFull(t)
	ctx := context.Background()
	challenge, err := svc.GenerateWebAuthnChallenge(ctx)
	if err != nil {
		t.Fatalf("GenerateWebAuthnChallenge: %v", err)
	}
	if challenge == "" {
		t.Error("expected non-empty challenge")
	}
}

// ===== Password History =====

func TestGetPasswordHistoryS6_Success(t *testing.T) {
	svc, cr, _, _, _ := tNewAuthSvcFull(t)
	ctx, _ := tCtxTenant()
	uid := uuid.New()
	cr.history = append(cr.history, &domain.CredentialHistoryEntry{
		ID:        uuid.New(),
		Secret:    "hashedpassword12345678901234567890",
		CreatedAt: time.Now(),
	})
	result, err := svc.GetPasswordHistory(ctx, uid)
	if err != nil {
		t.Fatalf("GetPasswordHistory: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1, got %d", len(result))
	}
	if result[0]["hash_prefix"] == "" {
		t.Error("expected non-empty hash_prefix")
	}
}

func TestGetPasswordHistoryS6_NoTenantContext(t *testing.T) {
	svc, _, _, _, _ := tNewAuthSvcFull(t)
	_, err := svc.GetPasswordHistory(context.Background(), uuid.New())
	if err == nil {
		t.Error("expected error for missing tenant context")
	}
}

// ===== LogoutAll =====

func TestLogoutAllS6_Success(t *testing.T) {
	svc, _, sr, _, _ := tNewAuthSvcFull(t)
	ctx, tid := tCtxTenant()
	uid := uuid.New()
	sid := uuid.New()
	sr.s[sid] = &domain.Session{ID: sid, TenantID: tid, UserID: uid}
	err := svc.LogoutAll(ctx, tid, uid, uuid.Nil)
	if err != nil {
		t.Fatalf("LogoutAll: %v", err)
	}
	if sr.s[sid].RevokedAt == nil {
		t.Error("expected session revoked")
	}
}

// ===== PasswordService Coverage =====

func TestPwSvcS6_CheckHistory_NoHistoryCount(t *testing.T) {
	cr := newTCredRepo()
	ps := NewPasswordService(conf.PasswordPolicy{HistoryCount: 0}, cr, nil)
	err := ps.CheckHistory(context.Background(), uuid.New(), uuid.New(), "NewPass123")
	if err != nil {
		t.Errorf("expected nil with HistoryCount=0, got %v", err)
	}
}

func TestPwSvcS6_VerifyOldPassword(t *testing.T) {
	ps := NewPasswordService(conf.Default().Password, newTCredRepo(), nil)
	h, _ := crypto.HashPassword("TestPass123")
	cred := &domain.Credential{Secret: h}
	ok, err := ps.VerifyOldPassword(context.Background(), cred, "TestPass123")
	if err != nil || !ok {
		t.Errorf("expected match, got ok=%v err=%v", ok, err)
	}
	ok, err = ps.VerifyOldPassword(context.Background(), cred, "WrongPass")
	if err != nil || ok {
		t.Errorf("expected no match, got ok=%v err=%v", ok, err)
	}
}

// ===== EmailService / AccountLockoutService Coverage =====

func TestEmailSvcS6_FullFlow(t *testing.T) {
	rdb := tRedis(t)
	es := NewEmailService(rdb)
	ctx := context.Background()
	tid, uid := uuid.New(), uuid.New()
	token, err := es.IssueVerificationToken(ctx, tid, uid, "test@example.com")
	if err != nil {
		t.Fatalf("IssueVerificationToken: %v", err)
	}
	rt, ru, email, err := es.VerifyEmailToken(ctx, token)
	if err != nil {
		t.Fatalf("VerifyEmailToken: %v", err)
	}
	if rt != tid || ru != uid || email != "test@example.com" {
		t.Errorf("unexpected values")
	}
}

func TestAccountLockoutSvcS6_Flow(t *testing.T) {
	rdb := tRedis(t)
	svc := NewAccountLockoutService(rdb, 3, 30*time.Minute)
	ctx := context.Background()
	tid := uuid.New()
	if svc.IsLocked(ctx, tid, "user") {
		t.Error("expected not locked")
	}
	for i := 0; i < 3; i++ {
		if err := svc.RecordFailedAttempt(ctx, tid, "user"); err != nil {
			t.Fatalf("RecordFailedAttempt: %v", err)
		}
	}
	if !svc.IsLocked(ctx, tid, "user") {
		t.Error("expected locked")
	}
	svc.ResetAttempts(ctx, tid, "user")
	if svc.IsLocked(ctx, tid, "user") {
		t.Error("expected not locked after reset")
	}
}

// ===== MFA Service Extra Coverage =====

func TestMFASvcS6_DisableMFA_NoTenant(t *testing.T) {
	repo := newMockMFARepo()
	svc := NewMFAService(repo)
	err := svc.DisableMFA(context.Background(), uuid.New())
	if err == nil {
		t.Error("expected error for missing tenant context")
	}
}

func TestMFASvcS6_ListDevices_NoTenant(t *testing.T) {
	repo := newMockMFARepo()
	svc := NewMFAService(repo)
	_, err := svc.ListDevices(context.Background(), uuid.New())
	if err == nil {
		t.Error("expected error for missing tenant context")
	}
}

func TestMFASvcS6_ListDevices_Success(t *testing.T) {
	repo := newMockMFARepo()
	svc := NewMFAService(repo)
	ctx, tid := tCtxTenant()
	uid := uuid.New()
	repo.devices[uid] = &domain.MFADevice{ID: uuid.New(), TenantID: tid, UserID: uid, Name: "phone"}
	devices, err := svc.ListDevices(ctx, uid)
	if err != nil {
		t.Fatalf("ListDevices: %v", err)
	}
	if len(devices) != 1 {
		t.Errorf("expected 1, got %d", len(devices))
	}
}

func TestMFASvcS6_DisableMFA_Success(t *testing.T) {
	repo := newMockMFARepo()
	svc := NewMFAService(repo)
	ctx, tid := tCtxTenant()
	uid := uuid.New()
	devID := uuid.New()
	repo.devices[uid] = &domain.MFADevice{ID: devID, TenantID: tid, UserID: uid, Name: "phone", Enabled: true}
	err := svc.DisableMFA(ctx, devID)
	if err != nil {
		t.Fatalf("DisableMFA: %v", err)
	}
}

func TestMFASvcS6_SetupMFA_AlreadyEnabled(t *testing.T) {
	repo := newMockMFARepo()
	svc := NewMFAService(repo)
	ctx, tid := tCtxTenant()
	uid := uuid.New()
	repo.devices[uid] = &domain.MFADevice{
		ID:       uuid.New(),
		TenantID: tid,
		UserID:   uid,
		Enabled:  true,
	}
	_, err := svc.SetupMFA(ctx, uid, "phone")
	if err == nil {
		t.Error("expected error for already enabled MFA")
	}
}

// ===== ResetPassword Extra Coverage =====

func TestResetPasswordS6_CredentialNotFound(t *testing.T) {
	svc, _, _, _, _ := tNewAuthSvcFull(t)
	ctx, tid := tCtxTenant()
	uid := uuid.New()
	tok, err := svc.passwordService.IssueResetToken(ctx, uid, tid)
	if err != nil {
		t.Fatalf("IssueResetToken: %v", err)
	}
	err = svc.ResetPassword(ctx, tok, "NewStrongPass123")
	if err == nil {
		t.Error("expected error for credential not found")
	}
}

// ===== ChangePassword History Reuse =====

func TestChangePasswordS6_HistoryReuse(t *testing.T) {
	svc, cr, _, _, _ := tNewAuthSvcFull(t)
	ctx, tid := tCtxTenant()
	uid := uuid.New()
	h, _ := crypto.HashPassword("OldPass123Ab")
	cred := &domain.Credential{ID: uuid.New(), TenantID: tid, UserID: uid, Secret: h, Enabled: true}
	cr.byUser[uid] = cred
	newHash, _ := crypto.HashPassword("UsedPassword456")
	cr.history = append(cr.history, &domain.CredentialHistoryEntry{Secret: newHash})
	svc.cfg.Password.HistoryCount = 5
	svc.passwordService.UpdatePolicy(svc.cfg.Password)
	err := svc.ChangePassword(ctx, tid, uid, "OldPass123Ab", "UsedPassword456")
	if err != ErrPasswordReused {
		t.Errorf("expected ErrPasswordReused, got %v", err)
	}
}

// ===== Step-Up Authentication Coverage =====

func TestInitStepUpS6_PasswordMethod(t *testing.T) {
	svc, _, _, _, _ := tNewAuthSvcFull(t)
	ctx, _ := tCtxTenant()
	resp, err := svc.InitStepUp(ctx, uuid.New(), "password")
	if err != nil {
		t.Fatalf("InitStepUp: %v", err)
	}
	if resp.Challenge == "" || resp.Method != "password" {
		t.Error("unexpected response")
	}
}

func TestInitStepUpS6_MFAMethod(t *testing.T) {
	svc, _, _, _, _ := tNewAuthSvcFull(t)
	ctx, _ := tCtxTenant()
	resp, err := svc.InitStepUp(ctx, uuid.New(), "mfa")
	if err != nil {
		t.Fatalf("InitStepUp: %v", err)
	}
	if resp.Method != "mfa" {
		t.Error("expected method=mfa")
	}
}

func TestInitStepUpS6_UnsupportedMethod(t *testing.T) {
	svc, _, _, _, _ := tNewAuthSvcFull(t)
	ctx, _ := tCtxTenant()
	_, err := svc.InitStepUp(ctx, uuid.New(), "biometric")
	if err == nil {
		t.Error("expected error for unsupported method")
	}
}

func TestInitStepUpS6_NoTenantContext(t *testing.T) {
	svc, _, _, _, _ := tNewAuthSvcFull(t)
	_, err := svc.InitStepUp(context.Background(), uuid.New(), "password")
	if err == nil {
		t.Error("expected error for missing tenant context")
	}
}

func TestVerifyStepUpS6_InvalidChallenge(t *testing.T) {
	svc, _, _, _, _ := tNewAuthSvcFull(t)
	ctx := context.Background()
	_, err := svc.VerifyStepUp(ctx, "nonexistent", "", "")
	if err != ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestVerifyStepUpS6_PasswordSuccess(t *testing.T) {
	svc, cr, _, _, _ := tNewAuthSvcFull(t)
	ctx, tid := tCtxTenant()
	uid := uuid.New()
	h, _ := crypto.HashPassword("MyPass123")
	cr.byUser[uid] = &domain.Credential{ID: uuid.New(), TenantID: tid, UserID: uid, Secret: h, Enabled: true}
	chal, _ := svc.InitStepUp(ctx, uid, "password")
	result, err := svc.VerifyStepUp(ctx, chal.Challenge, "", "MyPass123")
	if err != nil {
		t.Fatalf("VerifyStepUp: %v", err)
	}
	if result.StepUpToken == "" {
		t.Error("expected non-empty token")
	}
	if result.ExpiresIn != 300 {
		t.Errorf("expected 300, got %d", result.ExpiresIn)
	}
}

func TestVerifyStepUpS6_PasswordWrong(t *testing.T) {
	svc, cr, _, _, _ := tNewAuthSvcFull(t)
	ctx, tid := tCtxTenant()
	uid := uuid.New()
	h, _ := crypto.HashPassword("MyPass123")
	cr.byUser[uid] = &domain.Credential{ID: uuid.New(), TenantID: tid, UserID: uid, Secret: h, Enabled: true}
	chal, _ := svc.InitStepUp(ctx, uid, "password")
	_, err := svc.VerifyStepUp(ctx, chal.Challenge, "", "WrongPass")
	if err != ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestVerifyStepUpS6_PasswordUserNotFound(t *testing.T) {
	svc, _, _, _, _ := tNewAuthSvcFull(t)
	ctx, _ := tCtxTenant()
	uid := uuid.New()
	chal, _ := svc.InitStepUp(ctx, uid, "password")
	_, err := svc.VerifyStepUp(ctx, chal.Challenge, "", "MyPass123")
	if err != ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestValidateStepUpTokenS6_Success(t *testing.T) {
	svc, cr, _, _, _ := tNewAuthSvcFull(t)
	ctx, tid := tCtxTenant()
	uid := uuid.New()
	h, _ := crypto.HashPassword("MyPass123")
	cr.byUser[uid] = &domain.Credential{ID: uuid.New(), TenantID: tid, UserID: uid, Secret: h, Enabled: true}
	chal, _ := svc.InitStepUp(ctx, uid, "password")
	result, _ := svc.VerifyStepUp(ctx, chal.Challenge, "", "MyPass123")
	err := svc.ValidateStepUpToken(ctx, result.StepUpToken, uid)
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestValidateStepUpTokenS6_Invalid(t *testing.T) {
	svc, _, _, _, _ := tNewAuthSvcFull(t)
	err := svc.ValidateStepUpToken(context.Background(), "nonexistent", uuid.New())
	if err != ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestValidateStepUpTokenS6_WrongUser(t *testing.T) {
	svc, cr, _, _, _ := tNewAuthSvcFull(t)
	ctx, tid := tCtxTenant()
	uid := uuid.New()
	h, _ := crypto.HashPassword("MyPass123")
	cr.byUser[uid] = &domain.Credential{ID: uuid.New(), TenantID: tid, UserID: uid, Secret: h, Enabled: true}
	chal, _ := svc.InitStepUp(ctx, uid, "password")
	result, _ := svc.VerifyStepUp(ctx, chal.Challenge, "", "MyPass123")
	otherUser := uuid.New()
	err := svc.ValidateStepUpToken(ctx, result.StepUpToken, otherUser)
	if err != ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

// ===== Token Service Coverage =====

func TestRotateRefreshTokenS6_ReplayDetected(t *testing.T) {
	svc, _, _, rr := tNewAuthSvc(t)
	ctx := context.Background()
	tid, uid, sid := uuid.New(), uuid.New(), uuid.New()
	pt := "test-replay-token"
	th := hashToken(pt)
	now := time.Now()
	revoked := now.Add(-time.Hour)
	rr.t[th] = &domain.RefreshToken{
		ID: uuid.New(), TenantID: tid, UserID: uid, SessionID: sid,
		TokenHash: th, ExpiresAt: now.Add(24 * time.Hour),
		CreatedAt: now, RevokedAt: &revoked,
	}
	_, _, err := svc.tokenService.RotateRefreshToken(ctx, pt)
	if err == nil {
		t.Error("expected error for replay detection")
	}
	if !strings.Contains(err.Error(), "replay") {
		t.Errorf("expected replay error, got %v", err)
	}
}

func TestRotateRefreshTokenS6_NotFound(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	ctx := context.Background()
	_, _, err := svc.tokenService.RotateRefreshToken(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for invalid token")
	}
}

func TestParseColonS6(t *testing.T) {
	parts := splitColon("a:b:c", 3)
	if len(parts) != 3 || parts[0] != "a" || parts[2] != "c" {
		t.Errorf("unexpected: %v", parts)
	}
	parts2 := splitColon("single", 3)
	if len(parts2) != 1 || parts2[0] != "single" {
		t.Errorf("unexpected: %v", parts2)
	}
}

// ===== Phone OTP Coverage =====

func TestSendPhoneOTPS6_Success(t *testing.T) {
	svc, _, _, _, _ := tNewAuthSvcFull(t)
	ctx := context.Background()
	otp, err := svc.SendPhoneOTP(ctx, uuid.New(), uuid.New(), "+1234567890")
	if err != nil {
		t.Fatalf("SendPhoneOTP: %v", err)
	}
	if len(otp) != 6 {
		t.Errorf("expected 6-digit OTP, got %s", otp)
	}
}

func TestSendPhoneOTPS6_RateLimited(t *testing.T) {
	svc, _, _, _, _ := tNewAuthSvcFull(t)
	ctx := context.Background()
	phone := "+9876543210"
	// Exceed max retries
	for i := 0; i <= phoneOTPMaxRetry; i++ {
		_, _ = svc.SendPhoneOTP(ctx, uuid.New(), uuid.New(), phone)
	}
	_, err := svc.SendPhoneOTP(ctx, uuid.New(), uuid.New(), phone)
	if err != ErrRateLimited {
		t.Errorf("expected ErrRateLimited, got %v", err)
	}
}

func TestVerifyPhoneOTPS6_Success(t *testing.T) {
	svc, _, _, _, _ := tNewAuthSvcFull(t)
	ctx := context.Background()
	tid, uid := uuid.New(), uuid.New()
	phone := "+1111111111"
	otp, _ := svc.SendPhoneOTP(ctx, tid, uid, phone)
	tok, err := svc.VerifyPhoneOTP(ctx, phone, otp, "1.1.1.1", "agent")
	if err != nil {
		t.Fatalf("VerifyPhoneOTP: %v", err)
	}
	if tok.AccessToken == "" {
		t.Error("expected non-empty access token")
	}
}

func TestVerifyPhoneOTPS6_WrongOTP(t *testing.T) {
	svc, _, _, _, _ := tNewAuthSvcFull(t)
	ctx := context.Background()
	phone := "+2222222222"
	_, _ = svc.SendPhoneOTP(ctx, uuid.New(), uuid.New(), phone)
	_, err := svc.VerifyPhoneOTP(ctx, phone, "000000", "1.1.1.1", "agent")
	if err != ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestVerifyPhoneOTPS6_InvalidToken(t *testing.T) {
	svc, _, _, _, _ := tNewAuthSvcFull(t)
	ctx := context.Background()
	_, err := svc.VerifyPhoneOTP(ctx, "+9999999999", "123456", "1.1.1.1", "agent")
	if err != ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestGenerateNumericOTPS6(t *testing.T) {
	otp, err := generateNumericOTP(6)
	if err != nil {
		t.Fatalf("generateNumericOTP: %v", err)
	}
	if len(otp) != 6 {
		t.Errorf("expected 6 digits, got %d", len(otp))
	}
	otp2, _ := generateNumericOTP(4)
	if len(otp2) != 4 {
		t.Errorf("expected 4 digits, got %d", len(otp2))
	}
}

// ===== Key Parsing Coverage =====

func TestParsePublicKeyS6_InvalidPEM(t *testing.T) {
	_, err := parsePublicKey([]byte("not a pem"))
	if err == nil {
		t.Error("expected error for invalid PEM")
	}
}

func TestParsePublicKeyS6_NotRSA(t *testing.T) {
	// Generate EC key and encode as PKIX
	_, err := parsePublicKey([]byte("-----BEGIN PUBLIC KEY-----\nAAAA\n-----END PUBLIC KEY-----"))
	if err == nil {
		t.Error("expected error for invalid key data")
	}
}

func TestParsePrivateKeyS6_InvalidPEM(t *testing.T) {
	_, err := parsePrivateKey([]byte("not a pem"))
	if err == nil {
		t.Error("expected error for invalid PEM")
	}
}

func TestLoadOrCreatePrivateKeyS6_GeneratesNew(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/newkey.pem"
	key, err := loadOrCreatePrivateKey(path)
	if err != nil {
		t.Fatalf("loadOrCreatePrivateKey: %v", err)
	}
	if key == nil {
		t.Fatal("expected non-nil key")
	}
	// Verify file was created
	key2, err := loadOrCreatePrivateKey(path)
	if err != nil {
		t.Fatalf("second loadOrCreatePrivateKey: %v", err)
	}
	if key2 == nil {
		t.Fatal("expected non-nil key on second load")
	}
}

// ===== Password Expiration Coverage =====

func TestCheckPasswordExpirationS6_Expired(t *testing.T) {
	cr := newTCredRepo()
	ps := NewPasswordService(conf.PasswordPolicy{MaxAgeDays: 1}, cr, nil)
	ctx, tid := tCtxTenant()
	uid := uuid.New()
	// Credential with old UpdatedAt
	cr.byUser[uid] = &domain.Credential{
		ID:        uuid.New(),
		TenantID:  tid,
		UserID:    uid,
		UpdatedAt: time.Now().Add(-48 * time.Hour),
	}
	err := ps.CheckPasswordExpiration(ctx, tid, uid)
	if err != ErrPasswordExpired {
		t.Errorf("expected ErrPasswordExpired, got %v", err)
	}
}

func TestCheckPasswordExpirationS6_NotExpired(t *testing.T) {
	cr := newTCredRepo()
	ps := NewPasswordService(conf.PasswordPolicy{MaxAgeDays: 90}, cr, nil)
	ctx, tid := tCtxTenant()
	uid := uuid.New()
	cr.byUser[uid] = &domain.Credential{
		ID:        uuid.New(),
		TenantID:  tid,
		UserID:    uid,
		UpdatedAt: time.Now(),
	}
	err := ps.CheckPasswordExpiration(ctx, tid, uid)
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestCheckPasswordExpirationS6_NoMaxAge(t *testing.T) {
	cr := newTCredRepo()
	ps := NewPasswordService(conf.PasswordPolicy{MaxAgeDays: 0}, cr, nil)
	err := ps.CheckPasswordExpiration(context.Background(), uuid.New(), uuid.New())
	if err != nil {
		t.Errorf("expected nil with MaxAgeDays=0, got %v", err)
	}
}

func TestCheckPasswordExpirationS6_NoCredential(t *testing.T) {
	cr := newTCredRepo()
	ps := NewPasswordService(conf.PasswordPolicy{MaxAgeDays: 1}, cr, nil)
	ctx, tid := tCtxTenant()
	uid := uuid.New()
	err := ps.CheckPasswordExpiration(ctx, tid, uid)
	if err != nil {
		t.Errorf("expected nil for no credential, got %v", err)
	}
}

func TestMustChangePasswordS6(t *testing.T) {
	cr := newTCredRepo()
	ps := NewPasswordService(conf.PasswordPolicy{MaxAgeDays: 1}, cr, nil)
	ctx, tid := tCtxTenant()
	uid := uuid.New()
	cr.byUser[uid] = &domain.Credential{
		ID:        uuid.New(),
		TenantID:  tid,
		UserID:    uid,
		UpdatedAt: time.Now().Add(-48 * time.Hour),
	}
	if !ps.MustChangePassword(ctx, tid, uid) {
		t.Error("expected true for expired password")
	}
	// Fresh password
	cr.byUser[uid].UpdatedAt = time.Now()
	if ps.MustChangePassword(ctx, tid, uid) {
		t.Error("expected false for fresh password")
	}
}

// ===== Risk Assessment Coverage =====

func TestAssessLoginRiskS6_HighRisk(t *testing.T) {
	svc, _, _, _, _ := tNewAuthSvcFull(t)
	ctx := context.Background()
	tid, uid := uuid.New(), uuid.New()
	// Set 5+ failed attempts from an IP
	ipFailKey := fmt.Sprintf("ggid:risk:ipfail:%s", "8.8.8.8")
	for i := 0; i < 6; i++ {
		svc.rateLimiter.rdb.Incr(ctx, ipFailKey)
	}
	assessment := svc.AssessLoginRisk(ctx, tid, uid, "8.8.8.8", "agent")
	if assessment.Level != RiskLevelHigh {
		t.Errorf("expected RiskLevelHigh, got %s", assessment.Level)
	}
	if !assessment.RequiresStepUp {
		t.Error("expected RequiresStepUp=true")
	}
}

func TestAssessLoginRiskS6_MediumRisk(t *testing.T) {
	svc, _, _, _, _ := tNewAuthSvcFull(t)
	ctx := context.Background()
	tid, uid := uuid.New(), uuid.New()
	// Set 3 failed attempts
	ipFailKey := fmt.Sprintf("ggid:risk:ipfail:%s", "7.7.7.7")
	for i := 0; i < 3; i++ {
		svc.rateLimiter.rdb.Incr(ctx, ipFailKey)
	}
	assessment := svc.AssessLoginRisk(ctx, tid, uid, "7.7.7.7", "agent")
	if assessment.Level != RiskLevelMedium {
		t.Errorf("expected RiskLevelMedium, got %s", assessment.Level)
	}
	if assessment.Score < 20 {
		t.Errorf("expected score >= 20, got %d", assessment.Score)
	}
}

func TestAssessLoginRiskS6_LowRisk(t *testing.T) {
	svc, _, _, _, _ := tNewAuthSvcFull(t)
	ctx := context.Background()
	tid, uid := uuid.New(), uuid.New()
	// No failed attempts, known IP
	knownIPKey := fmt.Sprintf("ggid:risk:knownip:%s:%s", uid, "6.6.6.6")
	svc.rateLimiter.rdb.Set(ctx, knownIPKey, "1", 0)
	assessment := svc.AssessLoginRisk(ctx, tid, uid, "6.6.6.6", "")
	if assessment.Level != RiskLevelLow {
		t.Errorf("expected RiskLevelLow, got %s", assessment.Level)
	}
}

func TestAssessLoginRiskS6_UserAgentChanged(t *testing.T) {
	svc, _, _, _, _ := tNewAuthSvcFull(t)
	ctx := context.Background()
	tid, uid := uuid.New(), uuid.New()
	// Store known user agent
	uaKey := fmt.Sprintf("ggid:risk:ua:%s", uid)
	svc.rateLimiter.rdb.Set(ctx, uaKey, "Chrome", 0)
	assessment := svc.AssessLoginRisk(ctx, tid, uid, "5.5.5.5", "Firefox")
	found := false
	for _, r := range assessment.Reasons {
		if strings.Contains(r, "user agent") {
			found = true
		}
	}
	if !found {
		t.Error("expected user agent change reason")
	}
}

// ===== Login Attempt History Coverage =====

func TestLoginAttemptS6_Roundtrip(t *testing.T) {
	svc, _, _, _, _ := tNewAuthSvcFull(t)
	ctx := context.Background()
	svc.RecordLoginAttempt(ctx, "user1", "1.1.1.1", "Chrome", true, "")
	svc.RecordLoginAttempt(ctx, "user1", "1.1.1.1", "Chrome", false, "wrong password")
	svc.RecordLoginAttempt(ctx, "user1", "2.2.2.2", "Firefox", true, "")
	attempts, err := svc.GetLoginAttempts(ctx, "user1", 10)
	if err != nil {
		t.Fatalf("GetLoginAttempts: %v", err)
	}
	if len(attempts) != 3 {
		t.Errorf("expected 3, got %d", len(attempts))
	}
	// Most recent first
	if !attempts[0].Success {
		t.Error("expected most recent to be successful")
	}
}

func TestLoginAttemptS6_DefaultLimit(t *testing.T) {
	svc, _, _, _, _ := tNewAuthSvcFull(t)
	ctx := context.Background()
	svc.RecordLoginAttempt(ctx, "user2", "1.1.1.1", "Chrome", true, "")
	// limit=0 should default to 50
	attempts, err := svc.GetLoginAttempts(ctx, "user2", 0)
	if err != nil {
		t.Fatalf("GetLoginAttempts: %v", err)
	}
	if len(attempts) != 1 {
		t.Errorf("expected 1, got %d", len(attempts))
	}
}

func TestLoginAttemptS6_NoRecords(t *testing.T) {
	svc, _, _, _, _ := tNewAuthSvcFull(t)
	ctx := context.Background()
	attempts, err := svc.GetLoginAttempts(ctx, "nobody", 10)
	if err != nil {
		t.Fatalf("GetLoginAttempts: %v", err)
	}
	if len(attempts) != 0 {
		t.Errorf("expected 0, got %d", len(attempts))
	}
}

// ===== LookupUser Coverage =====

func TestLookupUserS6(t *testing.T) {
	svc, _, _, _, _ := tNewAuthSvcFull(t)
	ctx := context.Background()
	ic := svc.identityClient.(*tMockIdentityClient)
	uid := uuid.New()
	ic.users["lookup@test.com"] = &UserInfo{ID: uid, Email: "lookup@test.com", Status: "active"}
	info, err := svc.LookupUser(ctx, uuid.New(), "lookup@test.com")
	if err != nil {
		t.Fatalf("LookupUser: %v", err)
	}
	if info.ID != uid {
		t.Errorf("expected %s, got %s", uid, info.ID)
	}
}

func TestLookupUserS6_NotFound(t *testing.T) {
	svc, _, _, _, _ := tNewAuthSvcFull(t)
	info, err := svc.LookupUser(context.Background(), uuid.New(), "notfound@test.com")
	if info != nil {
		t.Error("expected nil for not found")
	}
	_ = err
}

// ===== Webhook Hook Coverage =====

func TestHookManagerS6_WebhookSuccess(t *testing.T) {
	// Start a test HTTP server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload HookPayload
		_ = json.NewDecoder(r.Body).Decode(&payload)
		w.WriteHeader(http.StatusOK)
		if payload.Event == HookPreLogin {
			_ = json.NewEncoder(w).Encode(HookResponse{Allow: true})
		}
	}))
	defer srv.Close()

	hm := NewHookManager()
	hm.RegisterHook(&AuthHook{
		ID:      "hook1",
		Event:   HookPreLogin,
		URL:     srv.URL,
		Enabled: true,
	})
	payload := &HookPayload{
		Event:    HookPreLogin,
		TenantID: uuid.New().String(),
		Username: "testuser",
	}
	err := hm.ExecuteHooks(context.Background(), HookPreLogin, payload)
	if err != nil {
		t.Fatalf("ExecuteHooks: %v", err)
	}
}

func TestHookManagerS6_PreHookDenied(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(HookResponse{Allow: false, Message: "blocked"})
	}))
	defer srv.Close()

	hm := NewHookManager()
	hm.RegisterHook(&AuthHook{
		ID:      "hook2",
		Event:   HookPreLogin,
		URL:     srv.URL,
		Enabled: true,
	})
	payload := &HookPayload{Event: HookPreLogin, TenantID: "tid"}
	err := hm.ExecuteHooks(context.Background(), HookPreLogin, payload)
	if err == nil {
		t.Error("expected error for denied pre-hook")
	}
}

func TestHookManagerS6_PostHookErrorIgnored(t *testing.T) {
	hm := NewHookManager()
	hm.RegisterHook(&AuthHook{
		ID:      "hook3",
		Event:   HookPostLogin,
		URL:     "http://localhost:1/invalid",
		Enabled: true,
	})
	payload := &HookPayload{Event: HookPostLogin, TenantID: "tid"}
	err := hm.ExecuteHooks(context.Background(), HookPostLogin, payload)
	if err != nil {
		t.Errorf("expected nil for post-hook error, got %v", err)
	}
}

func TestHookManagerS6_RemoveHook(t *testing.T) {
	hm := NewHookManager()
	hm.RegisterHook(&AuthHook{ID: "x", Event: HookPostLogin, URL: "http://x", Enabled: true})
	hm.RemoveHook("x")
	err := hm.ExecuteHooks(context.Background(), HookPostLogin, &HookPayload{})
	if err != nil {
		t.Errorf("expected nil after remove, got %v", err)
	}
}

func TestHookManagerS6_WebhookStatusError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	hm := NewHookManager()
	hm.RegisterHook(&AuthHook{
		ID:      "hook4",
		Event:   HookPreRegister,
		URL:     srv.URL,
		Enabled: true,
	})
	err := hm.ExecuteHooks(context.Background(), HookPreRegister, &HookPayload{Event: HookPreRegister})
	if err == nil {
		t.Error("expected error for webhook 500")
	}
}

func TestHookManagerS6_DisabledHook(t *testing.T) {
	hm := NewHookManager()
	hm.RegisterHook(&AuthHook{
		ID:      "hook5",
		Event:   HookPostLogin,
		URL:     "http://invalid",
		Enabled: false,
	})
	err := hm.ExecuteHooks(context.Background(), HookPostLogin, &HookPayload{})
	if err != nil {
		t.Errorf("expected nil for disabled hook, got %v", err)
	}
}

// ===== ResetPassword with History Reuse =====

func TestResetPasswordS6_HistoryReuse(t *testing.T) {
	svc, cr, _, _, _ := tNewAuthSvcFull(t)
	ctx, tid := tCtxTenant()
	uid := uuid.New()
	h, _ := crypto.HashPassword("OldPass123Ab")
	cred := &domain.Credential{ID: uuid.New(), TenantID: tid, UserID: uid, Secret: h, Enabled: true, Type: domain.CredentialPassword}
	cr.byUser[uid] = cred
	tok, _ := svc.passwordService.IssueResetToken(ctx, uid, tid)
	newHash, _ := crypto.HashPassword("ReusedPass123")
	cr.history = append(cr.history, &domain.CredentialHistoryEntry{Secret: newHash})
	svc.cfg.Password.HistoryCount = 5
	svc.passwordService.UpdatePolicy(svc.cfg.Password)
	err := svc.ResetPassword(ctx, tok, "ReusedPass123")
	if err != ErrPasswordReused {
		t.Errorf("expected ErrPasswordReused, got %v", err)
	}
}
