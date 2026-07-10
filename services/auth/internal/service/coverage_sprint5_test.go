package service

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/ggid/ggid/pkg/crypto"
	"github.com/ggid/ggid/services/auth/internal/conf"
	"github.com/ggid/ggid/services/auth/internal/domain"
	"github.com/google/uuid"
)

// Error-injecting mock wrappers to cover error paths in auth_service.go

type errorCredRepo struct {
	*tCredRepo
	findErr    error
	createErr  error
	historyErr error
	updateErr  error
}

func (m *errorCredRepo) FindByIDentifier(ctx context.Context, tid uuid.UUID, n string) (*domain.Credential, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	return m.tCredRepo.FindByIDentifier(ctx, tid, n)
}
func (m *errorCredRepo) Create(ctx context.Context, c *domain.Credential) error {
	if m.createErr != nil {
		return m.createErr
	}
	return m.tCredRepo.Create(ctx, c)
}
func (m *errorCredRepo) AddToHistory(ctx context.Context, tid, uid uuid.UUID, s string) error {
	if m.historyErr != nil {
		return m.historyErr
	}
	return m.tCredRepo.AddToHistory(ctx, tid, uid, s)
}
func (m *errorCredRepo) UpdateSecret(ctx context.Context, id uuid.UUID, s string) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	return m.tCredRepo.UpdateSecret(ctx, id, s)
}

type errorSessionRepo struct {
	*tSessionRepo
	revokeAllErr error
}

func (m *errorSessionRepo) RevokeAllForUser(ctx context.Context, tid, uid, except uuid.UUID) error {
	if m.revokeAllErr != nil {
		return m.revokeAllErr
	}
	return m.tSessionRepo.RevokeAllForUser(ctx, tid, uid, except)
}

func TestAuthService_Register_FindByIDentifierError(t *testing.T) {
	svc, credRepo, _, _ := tNewAuthSvc(t)
	ctx, tid := tCtxTenant()
	svc.credentialRepo = &errorCredRepo{tCredRepo: credRepo, findErr: errors.New("db error")}
	err := svc.Register(ctx, tid, uuid.New(), "newuser", "StrongPass123!")
	if err == nil {
		t.Error("expected error from FindByIDentifier")
	}
}

func TestAuthService_Register_CreateError(t *testing.T) {
	svc, credRepo, _, _ := tNewAuthSvc(t)
	ctx, tid := tCtxTenant()
	svc.credentialRepo = &errorCredRepo{tCredRepo: credRepo, createErr: errors.New("db connection lost")}
	err := svc.Register(ctx, tid, uuid.New(), "newuser", "StrongPass123!")
	if err == nil {
		t.Error("expected error from Create")
	}
}

func TestAuthService_Register_WeakPassword_V2(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	ctx, tid := tCtxTenant()
	err := svc.Register(ctx, tid, uuid.New(), "newuser", "weak")
	if err == nil {
		t.Error("expected error for weak password")
	}
}

func TestAuthService_Register_DuplicateCredential_V2(t *testing.T) {
	svc, credRepo, _, _ := tNewAuthSvc(t)
	ctx, tid := tCtxTenant()
	uid := uuid.New()
	credRepo.byName["existing"] = &domain.Credential{
		ID: uuid.New(), TenantID: tid, UserID: uid,
		Identifier: "existing", Secret: "$2a$10$hash",
	}
	err := svc.Register(ctx, tid, uid, "existing", "StrongPass123!")
	if err != ErrCredentialAlreadyExists {
		t.Errorf("expected ErrCredentialAlreadyExists, got %v", err)
	}
}

func TestPasswordService_SetPassword_AddToHistoryError(t *testing.T) {
	rdb := tRedis(t)
	cr := &errorCredRepo{tCredRepo: newTCredRepo(), historyErr: errors.New("redis down")}
	ps := NewPasswordService(conf.Default().Password, cr, rdb)
	cred := &domain.Credential{
		ID: uuid.New(), TenantID: uuid.New(), UserID: uuid.New(),
		Identifier: "user1", Secret: "$2a$10$old",
	}
	err := ps.SetPassword(context.Background(), cred, "NewStrongPass123!")
	if err == nil {
		t.Error("expected error from AddToHistory")
	}
}

func TestPasswordService_SetPassword_UpdateSecretError(t *testing.T) {
	rdb := tRedis(t)
	cr := &errorCredRepo{tCredRepo: newTCredRepo(), updateErr: errors.New("db error")}
	ps := NewPasswordService(conf.Default().Password, cr, rdb)
	cred := &domain.Credential{
		ID: uuid.New(), TenantID: uuid.New(), UserID: uuid.New(),
		Identifier: "user1", Secret: "$2a$10$old",
	}
	err := ps.SetPassword(context.Background(), cred, "NewStrongPass123!")
	if err == nil {
		t.Error("expected error from UpdateSecret")
	}
}

func TestPasswordService_SetPassword_WeakPassword(t *testing.T) {
	rdb := tRedis(t)
	cr := newTCredRepo()
	ps := NewPasswordService(conf.Default().Password, cr, rdb)
	cred := &domain.Credential{
		ID: uuid.New(), TenantID: uuid.New(), UserID: uuid.New(),
		Identifier: "user1", Secret: "$2a$10$old",
	}
	err := ps.SetPassword(context.Background(), cred, "weak")
	if err == nil {
		t.Error("expected error for weak password")
	}
}

func TestAuthService_LogoutAll_RevokeAllError(t *testing.T) {
	svc, _, sessRepo, _ := tNewAuthSvc(t)
	ctx, tid := tCtxTenant()
	svc.sessionService = NewSessionService(&errorSessionRepo{
		tSessionRepo: sessRepo,
		revokeAllErr: errors.New("session db down"),
	})
	err := svc.LogoutAll(ctx, tid, uuid.New(), uuid.Nil)
	if err == nil {
		t.Error("expected error from RevokeAllForUser")
	}
}

func TestAuthService_Register_Success_V2(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	ctx, tid := tCtxTenant()
	err := svc.Register(ctx, tid, uuid.New(), "newuser123", "StrongPass123!")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
}

func TestAuthService_Refresh_InvalidToken_V2(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	ctx, _ := tCtxTenant()
	_, err := svc.Refresh(ctx, "invalid_refresh_token_xyz")
	if err == nil {
		t.Error("expected error for invalid refresh token")
	}
}

func TestAuthService_ResetPassword_InvalidToken_V2(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	ctx := context.Background()
	err := svc.ResetPassword(ctx, "invalid_reset_token", "NewStrongPass123!")
	if err == nil {
		t.Error("expected error for invalid reset token")
	}
}

func TestPasswordService_ConsumeResetToken_Invalid(t *testing.T) {
	rdb := tRedis(t)
	cr := newTCredRepo()
	ps := NewPasswordService(conf.Default().Password, cr, rdb)
	_, _, err := ps.ConsumeResetToken(context.Background(), "invalid_token")
	if err == nil {
		t.Error("expected error for invalid reset token")
	}
}

func TestPasswordService_CheckHistory_HistoryError(t *testing.T) {
	rdb := tRedis(t)
	cr := &errorCredRepo{tCredRepo: newTCredRepo()}
	cr.tCredRepo.history = nil
	ps := NewPasswordService(conf.Default().Password, cr, rdb)
	err := ps.CheckHistory(context.Background(), uuid.New(), uuid.New(), "BrandNew456!")
	if err != nil {
		t.Errorf("expected nil for no history, got %v", err)
	}
}

func TestAuthService_Login_RateLimited_V2(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	ctx, _ := tCtxTenant()
	svc.cfg.RateLimit.LoginPerMinute = 1
	_, _ = svc.Login(ctx, "user", "pass", "10.0.0.1", "test")
	_, err := svc.Login(ctx, "user", "pass", "10.0.0.1", "test")
	if err == nil {
		t.Error("expected rate limit error")
	}
}

func TestAuthService_Login_NoTenantContext(t *testing.T) {
	svc, credRepo, _, _ := tNewAuthSvc(t)
	ctx := context.Background()
	hashed, _ := crypto.HashPassword("Pass123!")
	uid := uuid.New()
	credRepo.byName["notenant"] = &domain.Credential{
		ID: uuid.New(), TenantID: uuid.New(), UserID: uid,
		Identifier: "notenant", Secret: hashed,
	}
	_, err := svc.Login(ctx, "notenant", "Pass123!", "1.1.1.1", "test")
	if err == nil {
		t.Error("expected error for missing tenant context")
	}
}

// === parsePublicKey tests ===

func TestParsePublicKey_ValidPEM(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	pubDER, _ := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})
	pub, err := parsePublicKey(pubPEM)
	if err != nil {
		t.Fatalf("parsePublicKey: %v", err)
	}
	if pub == nil {
		t.Error("expected non-nil key")
	}
}

func TestParsePublicKey_ECDSANotRSA(t *testing.T) {
	priv, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	pubDER, _ := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})
	_, err := parsePublicKey(pubPEM)
	if err == nil {
		t.Error("expected error for ECDSA key (not RSA)")
	}
}

// === VerifyStepUp tests ===

func TestAuthService_VerifyStepUp_InvalidChallenge(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	_, err := svc.VerifyStepUp(context.Background(), "nonexistent", "", "")
	if err == nil {
		t.Error("expected error for invalid challenge")
	}
}

func TestAuthService_VerifyStepUp_PasswordSuccess(t *testing.T) {
	svc, credRepo, _, _ := tNewAuthSvc(t)
	ctx, tid := tCtxTenant()
	uid := uuid.New()
	hashed, _ := crypto.HashPassword("CorrectPass123!")
	credRepo.byUser[uid] = &domain.Credential{
		ID: uuid.New(), TenantID: tid, UserID: uid,
		Identifier: "user1", Secret: hashed,
	}
	challenge := "test-stepup-pass"
	key := fmt.Sprintf("ggid:stepup:%s", challenge)
	val := fmt.Sprintf("%s:%s:password", tid, uid)
	svc.rateLimiter.rdb.Set(ctx, key, val, 5*time.Minute)

	result, err := svc.VerifyStepUp(ctx, challenge, "", "CorrectPass123!")
	if err != nil {
		t.Fatalf("VerifyStepUp password: %v", err)
	}
	if result == nil {
		t.Error("expected non-nil result")
	}
}

func TestAuthService_VerifyStepUp_WrongPassword(t *testing.T) {
	svc, credRepo, _, _ := tNewAuthSvc(t)
	ctx, tid := tCtxTenant()
	uid := uuid.New()
	hashed, _ := crypto.HashPassword("CorrectPass123!")
	credRepo.byUser[uid] = &domain.Credential{
		ID: uuid.New(), TenantID: tid, UserID: uid,
		Identifier: "user1", Secret: hashed,
	}
	challenge := "test-stepup-wrong"
	key := fmt.Sprintf("ggid:stepup:%s", challenge)
	val := fmt.Sprintf("%s:%s:password", tid, uid)
	svc.rateLimiter.rdb.Set(ctx, key, val, 5*time.Minute)

	_, err := svc.VerifyStepUp(ctx, challenge, "", "WrongPass")
	if err == nil {
		t.Error("expected error for wrong password")
	}
}

// === RevokeSession success ===

func TestAuthService_RevokeSession_Success(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	ctx := context.Background()
	sessID := uuid.New()

	// Create session via the service's own Create method
	svc.sessionService.sessionRepo.(*tSessionRepo).s[sessID] = &domain.Session{ID: sessID, TenantID: uuid.New(), UserID: uuid.New()}

	err := svc.RevokeSession(ctx, sessID)
	if err != nil {
		t.Errorf("RevokeSession: %v", err)
	}
}

// === RotateRefreshToken invalid ===

func TestTokenService_RotateRefreshToken_Invalid_V2(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	_, _, err := svc.tokenService.RotateRefreshToken(context.Background(), "invalid_xyz")
	if err == nil {
		t.Error("expected error for invalid refresh token")
	}
}
