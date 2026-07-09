package service

import (
	"context"
	"fmt"
	"time"

	"github.com/ggid/ggid/pkg/authprovider"
	"github.com/ggid/ggid/pkg/crypto"
	"github.com/ggid/ggid/pkg/tenant"
	"github.com/ggid/ggid/services/auth/internal/conf"
	"github.com/ggid/ggid/services/auth/internal/domain"
	"github.com/ggid/ggid/services/auth/internal/repository"
	"github.com/google/uuid"
)

// AuthService orchestrates the authentication workflow:
// login, logout, register, refresh, password flows, session management.
type AuthService struct {
	cfg            *conf.Config
	chain          *authprovider.Chain
	credentialRepo *repository.CredentialRepository
	tokenService   *TokenService
	sessionService *SessionService
	passwordService *PasswordService
	rateLimiter    *RateLimiter
	identityClient IdentityClient
}

// NewAuthService creates a new AuthService with all dependencies.
func NewAuthService(
	cfg *conf.Config,
	chain *authprovider.Chain,
	credRepo *repository.CredentialRepository,
	tokenSvc *TokenService,
	sessionSvc *SessionService,
	passwordSvc *PasswordService,
	rateLimiter *RateLimiter,
	identityClient IdentityClient,
) *AuthService {
	return &AuthService{
		cfg:             cfg,
		chain:           chain,
		credentialRepo:  credRepo,
		tokenService:    tokenSvc,
		sessionService:  sessionSvc,
		passwordService: passwordSvc,
		rateLimiter:     rateLimiter,
		identityClient:  identityClient,
	}
}

// Login authenticates a user and returns a token set.
func (s *AuthService) Login(ctx context.Context, username, password, ip, userAgent string) (*domain.TokenSet, error) {
	// 1. Rate limit: 5 attempts per minute per IP
	rlKey := fmt.Sprintf("login:%s", ip)
	if err := s.rateLimiter.CheckAndIncrement(ctx, rlKey, s.cfg.RateLimit.LoginPerMinute); err != nil {
		return nil, err
	}

	// 2. Authenticate via provider chain
	result, err := s.chain.Authenticate(ctx, authprovider.Credentials{
		Username: username,
		Password: password,
	})
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	// 3. Resolve user ID
	if result.LinkedUser == nil {
		return nil, fmt.Errorf("authentication succeeded but no linked user")
	}
	userID := *result.LinkedUser

	// 4. Get tenant from context
	tc, err := tenant.FromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("tenant context required: %w", err)
	}

	// 5. Create session
	_, session, err := s.sessionService.Create(ctx, CreateSessionParams{
		TenantID:  tc.TenantID,
		UserID:    userID,
		IPAddress: ip,
		UserAgent: userAgent,
		TTL:       24 * time.Hour,
	})
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	// 6. Issue JWT access token
	accessToken, expiresIn, err := s.tokenService.IssueAccessToken(tc.TenantID, userID)
	if err != nil {
		return nil, fmt.Errorf("issue access token: %w", err)
	}

	// 7. Issue refresh token
	refreshToken, err := s.tokenService.IssueRefreshToken(ctx, tc.TenantID, userID, session.ID)
	if err != nil {
		return nil, fmt.Errorf("issue refresh token: %w", err)
	}

	return &domain.TokenSet{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    expiresIn,
		SessionID:    session.ID.String(),
	}, nil
}

// Logout revokes the session and all associated refresh tokens.
func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	return s.tokenService.RevokeRefreshToken(ctx, refreshToken)
}

// Register creates a new user credential.
func (s *AuthService) Register(ctx context.Context, tenantID uuid.UUID, userID uuid.UUID, username, password string) error {
	// 1. Validate password against policy
	if err := s.passwordService.Validate(password); err != nil {
		return err
	}

	// 2. Check if credential already exists
	existing, err := s.credentialRepo.FindByIDentifier(ctx, tenantID, username)
	if err != nil {
		return fmt.Errorf("check existing credential: %w", err)
	}
	if existing != nil {
		return fmt.Errorf("username already registered")
	}

	// 3. Hash password and create credential
	hash, err := crypto.HashPassword(password)
	if err != nil {
		return err
	}

	cred := &domain.Credential{
		TenantID:   tenantID,
		UserID:     userID,
		Type:       domain.CredentialPassword,
		Identifier: username,
		Secret:     hash,
		Enabled:    true,
	}
	return s.credentialRepo.Create(ctx, cred)
}

// Refresh validates a refresh token, rotates it, and issues a new access token.
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*domain.TokenSet, error) {
	// 1. Rotate refresh token (revokes old, issues new)
	newRefreshToken, rt, err := s.tokenService.RotateRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, fmt.Errorf("rotate refresh token: %w", err)
	}

	// 2. Issue new access token
	accessToken, expiresIn, err := s.tokenService.IssueAccessToken(rt.TenantID, rt.UserID)
	if err != nil {
		return nil, fmt.Errorf("issue access token: %w", err)
	}

	return &domain.TokenSet{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    expiresIn,
		SessionID:    rt.SessionID.String(),
	}, nil
}

// ForgotPassword initiates the password reset flow.
func (s *AuthService) ForgotPassword(ctx context.Context, tenantID uuid.UUID, email string) error {
	// 1. Look up credential by identifier (email or username)
	cred, err := s.credentialRepo.FindByIDentifier(ctx, tenantID, email)
	if err != nil {
		return err
	}
	// Don't reveal whether the email exists
	if cred == nil {
		return nil
	}

	// 2. Issue a reset token
	_, err = s.passwordService.IssueResetToken(ctx, cred.UserID, tenantID)
	return err
}

// ResetPassword completes the password reset flow using a reset token.
func (s *AuthService) ResetPassword(ctx context.Context, resetToken, newPassword string) error {
	// 1. Consume reset token
	tenantID, userID, err := s.passwordService.ConsumeResetToken(ctx, resetToken)
	if err != nil {
		return err
	}

	// 2. Find credential
	cred, err := s.credentialRepo.FindByUserID(ctx, tenantID, userID)
	if err != nil {
		return err
	}
	if cred == nil {
		return fmt.Errorf("credential not found for user")
	}

	// 3. Check password history
	if err := s.passwordService.CheckHistory(ctx, tenantID, userID, newPassword); err != nil {
		return err
	}

	// 4. Set new password
	if err := s.passwordService.SetPassword(ctx, cred, newPassword); err != nil {
		return err
	}

	// 5. Revoke all sessions (force re-login everywhere)
	_ = s.sessionService.RevokeAllForUser(ctx, tenantID, userID, uuid.Nil)
	return nil
}

// ChangePassword changes the password for an authenticated user.
func (s *AuthService) ChangePassword(ctx context.Context, tenantID, userID uuid.UUID, oldPassword, newPassword string) error {
	// 1. Find credential
	cred, err := s.credentialRepo.FindByUserID(ctx, tenantID, userID)
	if err != nil {
		return err
	}
	if cred == nil {
		return fmt.Errorf("credential not found")
	}

	// 2. Verify old password
	match, err := crypto.VerifyPassword(oldPassword, cred.Secret)
	if err != nil || !match {
		return ErrInvalidCredentials
	}

	// 3. Check password history
	if err := s.passwordService.CheckHistory(ctx, tenantID, userID, newPassword); err != nil {
		return err
	}

	// 4. Set new password
	return s.passwordService.SetPassword(ctx, cred, newPassword)
}

// ListSessions returns all active sessions for a user.
func (s *AuthService) ListSessions(ctx context.Context, tenantID, userID uuid.UUID) ([]*domain.Session, error) {
	return s.sessionService.ListByUser(ctx, tenantID, userID)
}

// RevokeSession revokes a specific session and its refresh tokens.
func (s *AuthService) RevokeSession(ctx context.Context, sessionID uuid.UUID) error {
	// Revoke all refresh tokens for this session
	if err := s.tokenService.RevokeAllForSession(ctx, sessionID); err != nil {
		return err
	}
	return s.sessionService.Revoke(ctx, sessionID)
}

// CleanupExpired removes expired sessions. Intended to be called by a background goroutine.
func (s *AuthService) CleanupExpired(ctx context.Context) (int64, error) {
	return s.sessionService.CleanupExpired(ctx, 7*24*time.Hour)
}
