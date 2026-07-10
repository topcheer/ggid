package service

import (
	"context"
	"testing"

	"github.com/ggid/ggid/pkg/authprovider"
	"github.com/ggid/ggid/pkg/tenant"
	"github.com/ggid/ggid/services/auth/internal/conf"
	"github.com/ggid/ggid/services/auth/internal/domain"
	"github.com/google/uuid"
)

// Test NewAuthService constructor and simple accessors.
func TestNewAuthService_Accessors(t *testing.T) {
	credRepo := &mockCredentialRepo{}
	refreshRepo := newMockRefreshTokenRepo()
	tokenSvc, rdb := newTestTokenSvc(t, refreshRepo)
	sessionRepo := newMockSessionRepo()
	sessionSvc := NewSessionService(sessionRepo)
	passwordSvc := NewPasswordService(conf.Default().Password, credRepo, rdb)
	rateLimiter := NewRateLimiter(rdb)
	chain := authprovider.NewChain()
	idClient := &NoopIdentityClient{}

	svc := NewAuthService(conf.Default(), chain, credRepo, tokenSvc, sessionSvc, passwordSvc, rateLimiter, idClient, nil)

	if svc.PasswordPolicy().MinLength != conf.Default().Password.MinLength {
		t.Error("PasswordPolicy accessor returned wrong config")
	}
	if svc.MFAService() != nil {
		t.Error("expected nil MFAService")
	}
}

// Test LookupUser via NoopIdentityClient (returns error).
func TestAuthService_LookupUser_Noop(t *testing.T) {
	credRepo := &mockCredentialRepo{}
	refreshRepo := newMockRefreshTokenRepo()
	tokenSvc, rdb := newTestTokenSvc(t, refreshRepo)
	sessionRepo := newMockSessionRepo()
	sessionSvc := NewSessionService(sessionRepo)
	passwordSvc := NewPasswordService(conf.Default().Password, credRepo, rdb)
	rateLimiter := NewRateLimiter(rdb)
	chain := authprovider.NewChain()
	idClient := &NoopIdentityClient{}

	svc := NewAuthService(conf.Default(), chain, credRepo, tokenSvc, sessionSvc, passwordSvc, rateLimiter, idClient, nil)

	_, err := svc.LookupUser(context.Background(), uuid.New(), "user@test.com")
	if err == nil {
		t.Error("expected error from NoopIdentityClient")
	}
}

// Test VerifyEmailToken via AuthService (delegates to emailService).
func TestAuthService_VerifyEmailToken_Invalid(t *testing.T) {
	credRepo := &mockCredentialRepo{}
	refreshRepo := newMockRefreshTokenRepo()
	tokenSvc, rdb := newTestTokenSvc(t, refreshRepo)
	sessionRepo := newMockSessionRepo()
	sessionSvc := NewSessionService(sessionRepo)
	passwordSvc := NewPasswordService(conf.Default().Password, credRepo, rdb)
	rateLimiter := NewRateLimiter(rdb)
	chain := authprovider.NewChain()

	svc := NewAuthService(conf.Default(), chain, credRepo, tokenSvc, sessionSvc, passwordSvc, rateLimiter, &NoopIdentityClient{}, nil)

	_, _, _, err := svc.VerifyEmailToken(context.Background(), "invalid")
	if err == nil {
		t.Error("expected error for invalid token")
	}
}

// Test NoopIdentityClient methods.
func TestNoopIdentityClient_AllMethods(t *testing.T) {
	c := &NoopIdentityClient{}

	_, err := c.GetUser(context.Background(), uuid.New(), "test")
	if err == nil {
		t.Error("expected error from noop GetUser")
	}

	_, err = c.GetUserByID(context.Background(), uuid.New(), uuid.New())
	if err == nil {
		t.Error("expected error from noop GetUserByID")
	}

	link, err := c.FindExternalIdentity(context.Background(), uuid.New(), "google", "ext1")
	if err != nil || link != nil {
		t.Error("expected nil link, nil error from noop FindExternalIdentity")
	}

	err = c.LinkExternalIdentity(context.Background(), uuid.New(), uuid.New(), "google", "ext1", nil)
	if err == nil {
		t.Error("expected error from noop LinkExternalIdentity")
	}

	_, err = c.CreateUserFromSocial(context.Background(), uuid.New(), "user", "email@test.com", "Name", "google", "ext1", nil)
	if err == nil {
		t.Error("expected error from noop CreateUserFromSocial")
	}
}

// Test SocialLogin with NoopIdentityClient (FindExternalIdentity returns nil, nil).
func TestSocialLogin_NoopIdentityClient(t *testing.T) {
	credRepo := &mockCredentialRepo{}
	refreshRepo := newMockRefreshTokenRepo()
	tokenSvc, rdb := newTestTokenSvc(t, refreshRepo)
	sessionRepo := newMockSessionRepo()
	sessionSvc := NewSessionService(sessionRepo)
	passwordSvc := NewPasswordService(conf.Default().Password, credRepo, rdb)
	rateLimiter := NewRateLimiter(rdb)
	chain := authprovider.NewChain()
	idClient := &NoopIdentityClient{}

	svc := NewAuthService(conf.Default(), chain, credRepo, tokenSvc, sessionSvc, passwordSvc, rateLimiter, idClient, nil)

	ctx := tenant.WithContext(context.Background(), &tenant.Context{
		TenantID:       uuid.New(),
		IsolationLevel: tenant.IsolationShared,
	})

	// SocialLogin with noop client → FindExternalIdentity returns nil, GetUser fails → CreateUserFromSocial fails
	_, err := svc.SocialLogin(ctx, "google", "ext1", "test@test.com", "Test", "", "1.2.3.4", "UA")
	if err == nil {
		t.Error("expected error from SocialLogin with noop identity client")
	}
}

// Test IssueMagicLink and VerifyMagicLink roundtrip.
func TestMagicLink_Roundtrip(t *testing.T) {
	credRepo := &mockCredentialRepo{}
	refreshRepo := newMockRefreshTokenRepo()
	tokenSvc, rdb := newTestTokenSvc(t, refreshRepo)
	sessionRepo := newMockSessionRepo()
	sessionSvc := NewSessionService(sessionRepo)
	passwordSvc := NewPasswordService(conf.Default().Password, credRepo, rdb)
	rateLimiter := NewRateLimiter(rdb)
	chain := authprovider.NewChain()

	svc := NewAuthService(conf.Default(), chain, credRepo, tokenSvc, sessionSvc, passwordSvc, rateLimiter, &NoopIdentityClient{}, nil)

	ctx := context.Background()
	tenantID := uuid.New()
	userID := uuid.New()

	// Issue magic link.
	token, err := svc.IssueMagicLink(ctx, tenantID, userID, "user@test.com")
	if err != nil {
		t.Fatalf("IssueMagicLink failed: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	// Verify it.
	tokens, err := svc.VerifyMagicLink(ctx, token, "1.2.3.4", "TestAgent")
	if err != nil {
		t.Fatalf("VerifyMagicLink failed: %v", err)
	}
	if tokens.AccessToken == "" {
		t.Error("expected non-empty access token")
	}

	// Verify again should fail (one-time use).
	_, err = svc.VerifyMagicLink(ctx, token, "1.2.3.4", "TestAgent")
	if err == nil {
		t.Error("expected error for reused magic link token")
	}
}

// Test VerifyMagicLink with invalid token.
func TestMagicLink_InvalidToken(t *testing.T) {
	credRepo := &mockCredentialRepo{}
	refreshRepo := newMockRefreshTokenRepo()
	tokenSvc, rdb := newTestTokenSvc(t, refreshRepo)
	sessionRepo := newMockSessionRepo()
	sessionSvc := NewSessionService(sessionRepo)
	passwordSvc := NewPasswordService(conf.Default().Password, credRepo, rdb)
	rateLimiter := NewRateLimiter(rdb)
	chain := authprovider.NewChain()

	svc := NewAuthService(conf.Default(), chain, credRepo, tokenSvc, sessionSvc, passwordSvc, rateLimiter, &NoopIdentityClient{}, nil)

	_, err := svc.VerifyMagicLink(context.Background(), "invalid-token", "1.2.3.4", "UA")
	if err == nil {
		t.Error("expected error for invalid token")
	}
}

// Test domain.TokenSet fields.
func TestTokenSet_Fields(t *testing.T) {
	ts := &domain.TokenSet{
		AccessToken:  "abc",
		RefreshToken: "def",
		TokenType:    "Bearer",
		ExpiresIn:    3600,
	}
	if ts.AccessToken != "abc" {
		t.Error("AccessToken mismatch")
	}
}
