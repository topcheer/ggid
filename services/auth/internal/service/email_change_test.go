package service

import (
	"context"
	"testing"
	"time"

	"github.com/ggid/ggid/pkg/tenant"
	"github.com/ggid/ggid/services/auth/internal/conf"
	"github.com/ggid/ggid/services/auth/internal/domain"
	"github.com/google/uuid"
)

func TestLogoutAll(t *testing.T) {
	credRepo := newMockCredRepo()
	refreshRepo := newMockRefreshTokenRepo()
	sessionRepo := newMockSessionRepo()
	ts, rdb := newTestTokenSvc(t, refreshRepo)
	tenantID := uuid.New()
	userID := uuid.New()

	sessionID := uuid.New()
	sessionRepo.sessions[sessionID] = &domain.Session{
		ID:        sessionID,
		TenantID:  tenantID,
		UserID:    userID,
		ExpiresAt: time.Now().Add(time.Hour),
	}

	svc := &AuthService{
		cfg:             conf.Default(),
		tokenService:    ts,
		sessionService:  NewSessionService(sessionRepo),
		passwordService: NewPasswordService(conf.Default().Password, credRepo, rdb),
		rateLimiter:     NewRateLimiter(rdb),
	}

	err := svc.LogoutAll(context.Background(), tenantID, userID, uuid.Nil)
	if err != nil {
		t.Fatalf("LogoutAll: %v", err)
	}
}

func TestEmailChange_Initiate(t *testing.T) {
	rdb := newTestRedis(t)
	rl := NewRateLimiter(rdb)
	svc := &AuthService{rateLimiter: rl, cfg: conf.Default()}
	userID := uuid.New()

	ctx := tenant.WithContext(context.Background(), &tenant.Context{
		TenantID:       uuid.New(),
		IsolationLevel: tenant.IsolationShared,
	})

	result, err := svc.InitiateEmailChange(ctx, userID, "old@test.com", "new@test.com")
	if err != nil {
		t.Fatalf("InitiateEmailChange: %v", err)
	}
	if result.OldEmailToken == "" || result.NewEmailToken == "" {
		t.Error("expected non-empty tokens")
	}
	if result.OldEmailToken == result.NewEmailToken {
		t.Error("tokens should differ")
	}
}

func TestEmailChange_SameEmail(t *testing.T) {
	rdb := newTestRedis(t)
	rl := NewRateLimiter(rdb)
	svc := &AuthService{rateLimiter: rl, cfg: conf.Default()}

	ctx := tenant.WithContext(context.Background(), &tenant.Context{
		TenantID:       uuid.New(),
		IsolationLevel: tenant.IsolationShared,
	})

	_, err := svc.InitiateEmailChange(ctx, uuid.New(), "same@test.com", "same@test.com")
	if err == nil {
		t.Error("expected error when old and new emails are the same")
	}
}

func TestEmailChange_EmptyNewEmail(t *testing.T) {
	rdb := newTestRedis(t)
	rl := NewRateLimiter(rdb)
	svc := &AuthService{rateLimiter: rl, cfg: conf.Default()}

	ctx := tenant.WithContext(context.Background(), &tenant.Context{
		TenantID:       uuid.New(),
		IsolationLevel: tenant.IsolationShared,
	})

	_, err := svc.InitiateEmailChange(ctx, uuid.New(), "old@test.com", "")
	if err == nil {
		t.Error("expected error for empty new email")
	}
}

func TestEmailChange_ConfirmOneSide(t *testing.T) {
	rdb := newTestRedis(t)
	rl := NewRateLimiter(rdb)
	svc := &AuthService{rateLimiter: rl, cfg: conf.Default()}
	userID := uuid.New()

	ctx := tenant.WithContext(context.Background(), &tenant.Context{
		TenantID:       uuid.New(),
		IsolationLevel: tenant.IsolationShared,
	})

	result, _ := svc.InitiateEmailChange(ctx, userID, "old@test.com", "new@test.com")

	applied, err := svc.ConfirmEmailChange(ctx, result.OldEmailToken, "old")
	if err != nil {
		t.Fatalf("ConfirmEmailChange old: %v", err)
	}
	if applied {
		t.Error("should not be applied yet — only one side confirmed")
	}
}

func TestEmailChange_ConfirmBothSides(t *testing.T) {
	rdb := newTestRedis(t)
	rl := NewRateLimiter(rdb)
	svc := &AuthService{rateLimiter: rl, cfg: conf.Default()}
	userID := uuid.New()

	ctx := tenant.WithContext(context.Background(), &tenant.Context{
		TenantID:       uuid.New(),
		IsolationLevel: tenant.IsolationShared,
	})

	result, _ := svc.InitiateEmailChange(ctx, userID, "old@test.com", "new@test.com")

	applied, _ := svc.ConfirmEmailChange(ctx, result.OldEmailToken, "old")
	if applied {
		t.Error("should not be applied after old only")
	}

	applied, err := svc.ConfirmEmailChange(ctx, result.NewEmailToken, "new")
	if err != nil {
		t.Fatalf("ConfirmEmailChange new: %v", err)
	}
	if !applied {
		t.Error("should be applied after both sides confirmed")
	}
}

func TestEmailChange_ConfirmInvalidToken(t *testing.T) {
	rdb := newTestRedis(t)
	rl := NewRateLimiter(rdb)
	svc := &AuthService{rateLimiter: rl, cfg: conf.Default()}

	_, err := svc.ConfirmEmailChange(context.Background(), "invalid-token", "old")
	if err == nil {
		t.Error("expected error for invalid token")
	}
}

func TestEmailChange_ConfirmInvalidStep(t *testing.T) {
	rdb := newTestRedis(t)
	rl := NewRateLimiter(rdb)
	svc := &AuthService{rateLimiter: rl, cfg: conf.Default()}

	_, err := svc.ConfirmEmailChange(context.Background(), "some-token", "invalid-step")
	if err == nil {
		t.Error("expected error for invalid step")
	}
}

func TestEmailChange_OneTimeUse(t *testing.T) {
	rdb := newTestRedis(t)
	rl := NewRateLimiter(rdb)
	svc := &AuthService{rateLimiter: rl, cfg: conf.Default()}
	userID := uuid.New()

	ctx := tenant.WithContext(context.Background(), &tenant.Context{
		TenantID:       uuid.New(),
		IsolationLevel: tenant.IsolationShared,
	})

	result, _ := svc.InitiateEmailChange(ctx, userID, "old@test.com", "new@test.com")

	_, err := svc.ConfirmEmailChange(ctx, result.OldEmailToken, "old")
	if err != nil {
		t.Fatalf("first confirm: %v", err)
	}

	_, err = svc.ConfirmEmailChange(ctx, result.OldEmailToken, "old")
	if err == nil {
		t.Error("token should be one-time use")
	}
}
