package service

import (
	"context"
	stderrors "errors"
	"fmt"
	"strings"
	"time"

	"github.com/ggid/ggid/pkg/authprovider"
	"github.com/ggid/ggid/pkg/crypto"
	"github.com/ggid/ggid/pkg/tenant"
	"github.com/ggid/ggid/services/auth/internal/conf"
	"github.com/ggid/ggid/services/auth/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

// AuthService orchestrates the authentication workflow:
// login, logout, register, refresh, password flows, session management, MFA.
type AuthService struct {
	cfg            *conf.Config
	chain          *authprovider.Chain
	credentialRepo CredentialRepo
	tokenService   *TokenService
	sessionService *SessionService
	passwordService *PasswordService
	rateLimiter    *RateLimiter
	identityClient IdentityClient
	mfaService     *MFAService
	emailService   *EmailService
}

// NewAuthService creates a new AuthService with all dependencies.
func NewAuthService(
	cfg *conf.Config,
	chain *authprovider.Chain,
	credRepo CredentialRepo,
	tokenSvc *TokenService,
	sessionSvc *SessionService,
	passwordSvc *PasswordService,
	rateLimiter *RateLimiter,
	identityClient IdentityClient,
	mfaSvc *MFAService,
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
		mfaService:      mfaSvc,
		emailService:    NewEmailService(rateLimiter.rdb),
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

	// 4a. Check if MFA is required for this user.
	if s.mfaService != nil && s.mfaService.HasMFAEnabled(ctx, tc.TenantID, userID) {
		// Issue a short-lived MFA challenge instead of tokens.
		challenge, err := crypto.GenerateRandomToken(32)
		if err != nil {
			return nil, fmt.Errorf("generate mfa challenge: %w", err)
		}
		return &domain.TokenSet{
			MFARequired:  true,
			MFAChallenge: challenge,
		}, nil
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
		return ErrCredentialAlreadyExists
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
	if err := s.credentialRepo.Create(ctx, cred); err != nil {
		// Catch DB unique constraint violation (race condition between
		// check and create). PostgreSQL SQLSTATE 23505 = unique_violation.
		var pgErr *pgconn.PgError
		if stderrors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrCredentialAlreadyExists
		}
		return fmt.Errorf("create credential: %w", err)
	}
	return nil
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

// LoginMFA completes the MFA challenge during login.
// It re-authenticates the user (to get the userID), verifies the TOTP code,
// and then issues the full token set.
func (s *AuthService) LoginMFA(ctx context.Context, username, password, mfaCode, ip, userAgent string) (*domain.TokenSet, error) {
	// 1. Re-authenticate via provider chain (without rate limiting — already checked in Login)
	result, err := s.chain.Authenticate(ctx, authprovider.Credentials{
		Username: username,
		Password: password,
	})
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if result.LinkedUser == nil {
		return nil, fmt.Errorf("authentication succeeded but no linked user")
	}
	userID := *result.LinkedUser

	tc, err := tenant.FromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("tenant context required: %w", err)
	}

	// 2. Verify MFA code.
	if s.mfaService == nil {
		return nil, fmt.Errorf("MFA service not configured")
	}
	if err := s.mfaService.VerifyUserCode(ctx, tc.TenantID, userID, mfaCode); err != nil {
		return nil, err
	}

	// 3. Create session and issue tokens (same as Login step 5-7).
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

	accessToken, expiresIn, err := s.tokenService.IssueAccessToken(tc.TenantID, userID)
	if err != nil {
		return nil, fmt.Errorf("issue access token: %w", err)
	}

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

// MFAService returns the MFA service for direct access (setup, verify, disable).
func (s *AuthService) MFAService() *MFAService { return s.mfaService }

// LookupUser looks up a user by identifier (email or username) via the identity client.
func (s *AuthService) LookupUser(ctx context.Context, tenantID uuid.UUID, identifier string) (*UserInfo, error) {
	return s.identityClient.GetUser(ctx, tenantID, identifier)
}

// --- Email Verification ---

// VerifyEmailToken validates an email verification token.
func (s *AuthService) VerifyEmailToken(ctx context.Context, token string) (uuid.UUID, uuid.UUID, string, error) {
	return s.emailService.VerifyEmailToken(ctx, token)
}

// PasswordPolicy returns the current password policy configuration.
func (s *AuthService) PasswordPolicy() conf.PasswordPolicy { return s.cfg.Password }

// --- Magic Link (Passwordless Login) ---

// IssueMagicLink generates a one-time magic link token for passwordless login.
// The token is stored in Redis with a 15-minute TTL.
// Returns the plaintext token (to be embedded in an email link).
func (s *AuthService) IssueMagicLink(ctx context.Context, tenantID, userID uuid.UUID, email string) (string, error) {
	token, err := crypto.GenerateRandomToken(32)
	if err != nil {
		return "", fmt.Errorf("generate magic link token: %w", err)
	}

	tokenHash := hashToken(token)
	key := fmt.Sprintf("ggid:magiclink:%s", tokenHash)
	val := fmt.Sprintf("%s:%s:%s", tenantID, userID, email)

	if err := s.rateLimiter.rdb.Set(ctx, key, val, 15*time.Minute).Err(); err != nil {
		return "", fmt.Errorf("store magic link token: %w", err)
	}

	return token, nil
}

// VerifyMagicLink validates a magic link token and issues JWT tokens.
func (s *AuthService) VerifyMagicLink(ctx context.Context, token, ip, userAgent string) (*domain.TokenSet, error) {
	tokenHash := hashToken(token)
	key := fmt.Sprintf("ggid:magiclink:%s", tokenHash)

	val, err := s.rateLimiter.rdb.Get(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("invalid or expired magic link token")
	}

	// Delete the token (one-time use).
	s.rateLimiter.rdb.Del(ctx, key)

	parts := strings.SplitN(val, ":", 3)
	if len(parts) != 3 {
		return nil, fmt.Errorf("corrupted magic link token")
	}

	tenantID, err := uuid.Parse(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid tenant ID in token")
	}
	userID, err := uuid.Parse(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid user ID in token")
	}

	// Create session.
	_, session, err := s.sessionService.Create(ctx, CreateSessionParams{
		TenantID:  tenantID,
		UserID:    userID,
		IPAddress: ip,
		UserAgent: userAgent,
		TTL:       24 * time.Hour,
	})
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	// Issue JWT tokens.
	accessToken, expiresIn, err := s.tokenService.IssueAccessToken(tenantID, userID)
	if err != nil {
		return nil, fmt.Errorf("issue access token: %w", err)
	}

	refreshToken, err := s.tokenService.IssueRefreshToken(ctx, tenantID, userID, session.ID)
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

// SocialLogin authenticates a user via a social provider's UserInfo.
// It handles three cases:
//  1. External identity already linked → look up user, issue JWT
//  2. Email matches existing user → link identity to that user, issue JWT
//  3. No match → JIT-provision a new user + link identity, issue JWT
func (s *AuthService) SocialLogin(ctx context.Context, provider, externalID, email, name, avatarURL, ip, userAgent string) (*domain.TokenSet, error) {
	tc, err := tenant.FromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("tenant context required: %w", err)
	}

	metadata := map[string]any{
		"provider":  provider,
		"email":     email,
		"name":      name,
		"avatar":    avatarURL,
	}

	var userID uuid.UUID

	// 1. Check if external identity is already linked.
	link, err := s.identityClient.FindExternalIdentity(ctx, tc.TenantID, provider, externalID)
	if err != nil {
		return nil, fmt.Errorf("find external identity: %w", err)
	}
	if link != nil {
		userID = link.UserID
	} else if email != "" {
		// 2. Try to match by email.
		existingUser, err := s.identityClient.GetUser(ctx, tc.TenantID, email)
		if err == nil && existingUser != nil {
			// Link the social identity to this existing user.
			if err := s.identityClient.LinkExternalIdentity(ctx, tc.TenantID, existingUser.ID, provider, externalID, metadata); err != nil {
				return nil, fmt.Errorf("link external identity: %w", err)
			}
			userID = existingUser.ID
		}
	}

	// 3. No match — JIT-provision a new user.
	if userID == uuid.Nil {
		// Generate username from provider + externalID (truncated to 60 chars).
		username := provider + "_" + externalID
		if len(username) > 60 {
			username = username[:60]
		}

		newUser, err := s.identityClient.CreateUserFromSocial(ctx, tc.TenantID, username, email, name, provider, externalID, metadata)
		if err != nil {
			return nil, fmt.Errorf("create user from social: %w", err)
		}
		userID = newUser.ID
	}

	// 4. Create session.
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

	// 5. Issue JWT tokens.
	accessToken, expiresIn, err := s.tokenService.IssueAccessToken(tc.TenantID, userID)
	if err != nil {
		return nil, fmt.Errorf("issue access token: %w", err)
	}

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
