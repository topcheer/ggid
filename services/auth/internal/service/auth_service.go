package service

import (
	"context"
	"crypto/rsa"
	stderrors "errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/ggid/ggid/pkg/authprovider"
	"github.com/ggid/ggid/pkg/crypto"
	"github.com/ggid/ggid/pkg/tenant"
	"github.com/ggid/ggid/services/auth/internal/conf"
	"github.com/ggid/ggid/services/auth/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/redis/go-redis/v9"
)

// suppress unused import warnings — crypto is used in SocialLogin path.
var _ = crypto.HashPassword

// maxSessionsPerUser caps the number of concurrent active sessions per user.
// Oldest sessions beyond this limit are revoked on new login.
const maxSessionsPerUser = 10

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
	backupCodeSvc  *BackupCodeService
	emailService   *EmailService
	emailSender    PasswordResetEmailSender
}

// PasswordResetEmailSender sends password reset emails.
type PasswordResetEmailSender interface {
	Send(ctx context.Context, to, subject, body string) error
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

// SetBackupCodeService injects the backup code service for MFA backup code generation/verification.
func (s *AuthService) SetBackupCodeService(bcs *BackupCodeService) {
	s.backupCodeSvc = bcs
}

// BackupCodeService returns the backup code service (may be nil if not configured).
func (s *AuthService) BackupCodeService() *BackupCodeService {
	return s.backupCodeSvc
}

// GetPasswordPolicy returns the current password policy configuration.
// PublicKey returns the RSA public key for JWT verification.
func (s *AuthService) PublicKey() *rsa.PublicKey {
	return s.tokenService.PublicKey()
}

func (s *AuthService) GetPasswordPolicy() conf.PasswordPolicy {
	if s.passwordService == nil {
		return conf.PasswordPolicy{}
	}
	return s.passwordService.GetPolicy()
}

// SetPasswordPolicy updates the password policy at runtime.
func (s *AuthService) SetPasswordPolicy(policy conf.PasswordPolicy) {
	if s.cfg != nil {
		s.cfg.Password = policy
	}
	if s.passwordService != nil {
		s.passwordService.UpdatePolicy(policy)
	}
}

// Login authenticates a user and returns a token set.
func (s *AuthService) VerifyCredentials(ctx context.Context, username, password, ip string) (userID uuid.UUID, mfaRequired bool, err error) {
	tc, err := tenant.FromContext(ctx)
	if err != nil {
		return uuid.Nil, false, fmt.Errorf("tenant context required: %w", err)
	}

	// Rate limit.
	rlKey := fmt.Sprintf("login:%s", ip)
	if err := s.rateLimiter.CheckAndIncrement(ctx, rlKey, s.cfg.RateLimit.LoginPerMinute); err != nil {
		return uuid.Nil, false, err
	}

	// Authenticate via provider chain.
	authCtx := authprovider.WithTenantContext(ctx, tc.TenantID)
	result, err := s.chain.Authenticate(authCtx, authprovider.Credentials{
		Username: username,
		Password: password,
	})
	if err != nil {
		return uuid.Nil, false, ErrInvalidCredentials
	}

	// Resolve user ID.
	if result.LinkedUser == nil {
		email, _ := result.Attributes["email"].(string)
		if email == "" {
			email = username + "@ldap.local"
		}
		name, _ := result.Attributes["displayName"].(string)
		if name == "" {
			name = username
		}
		newUser, err := s.identityClient.CreateUserFromSocial(ctx, tc.TenantID, username, email, name, string(result.Provider), result.ExternalID, result.Attributes)
		if err != nil {
			return uuid.Nil, false, fmt.Errorf("auto-provision failed: %w", err)
		}
		uid := newUser.ID
		result.LinkedUser = &uid
	}
	userID = *result.LinkedUser

	// Check MFA.
	if s.mfaService != nil && s.mfaService.HasMFAEnabled(ctx, tc.TenantID, userID) {
		return userID, true, nil
	}

	return userID, false, nil
}

// Logout revokes the session and all associated refresh tokens.
func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	return s.tokenService.RevokeRefreshToken(ctx, refreshToken)
}

// getUserScopesAndPermissions resolves role names and fine-grained permissions separately.
// Returns: roles (e.g. "Administrator") for JWT roles claim,
//          permissions (e.g. "inventory:read") for JWT permissions claim.
// All permissions are explicit — no inheritance or admin bypass.
func (s *AuthService) getUserScopesAndPermissions(ctx context.Context, tenantID, userID uuid.UUID) (roles []string, permissions []string) {
	if s.identityClient != nil {
		roles, err := s.identityClient.GetUserRoles(ctx, tenantID, userID)
		if err == nil && len(roles) > 0 {
			rawPerms, _ := s.identityClient.GetUserPermissions(ctx, tenantID, userID)
			// Deduplicate permissions
			seen := make(map[string]bool, len(rawPerms))
			permissions = make([]string, 0, len(rawPerms))
			for _, p := range rawPerms {
				if !seen[p] {
					seen[p] = true
					permissions = append(permissions, p)
				}
			}
			return roles, permissions
		}
	}
	return []string{"user"}, nil
}

// getUserScopes is kept for backward compatibility with callers that only need the merged list.
func (s *AuthService) getUserScopes(ctx context.Context, tenantID, userID uuid.UUID) []string {
	roles, _ := s.getUserScopesAndPermissions(ctx, tenantID, userID)
	return roles
}

// writeJTI is a nil-safe helper that writes the JTI + token expiry back to the session record.
// Best-effort: errors are logged but never block the login/refresh flow.
func (s *AuthService) writeJTI(ctx context.Context, sessionID uuid.UUID, jti string, expiresIn int) {
	if s.sessionService == nil || jti == "" {
		return
	}
	_ = s.sessionService.UpdateSessionJTI(ctx, sessionID, jti, time.Now().Add(time.Duration(expiresIn)*time.Second))
}

// Register creates a new user credential.
func (s *AuthService) Register(ctx context.Context, tenantID uuid.UUID, userID uuid.UUID, username, password string) error {
	// 1. Validate password against policy
	if err := s.passwordService.Validate(password); err != nil {
		return err
	}

	// 2. Check if credential already exists (CreateUserFromSocial may have created one with random password)
	existing, err := s.credentialRepo.FindByIDentifier(ctx, tenantID, username)
	if err != nil {
		return fmt.Errorf("check existing credential: %w", err)
	}

	// 3. Hash password
	hash, err := crypto.HashPassword(password)
	if err != nil {
		return err
	}

	if existing != nil {
		// Credential exists (created by CreateUserFromSocial with random password) — update it with real password
		if err := s.credentialRepo.UpdateSecret(ctx, existing.ID, hash); err != nil {
			return fmt.Errorf("update credential secret: %w", err)
		}
		return nil
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
func (s *AuthService) SetEmailSender(sender PasswordResetEmailSender) {
	s.emailSender = sender
}

// IdentityClient returns the identity client for external use.
func (s *AuthService) IdentityClient() IdentityClient {
	return s.identityClient
}

func (s *AuthService) ForgotPassword(ctx context.Context, tenantID uuid.UUID, email string) error {
	// 1. Look up credential by identifier (username or email)
	cred, err := s.credentialRepo.FindByIDentifier(ctx, tenantID, email)
	if err != nil {
		slog.Error("ForgotPassword: FindByIdentifier error", "identifier", email, "error", err)
		return err
	}

	// 1a. If not found by identifier, try via identity service (email → username)
	if cred == nil && s.identityClient != nil {
		user, err := s.identityClient.GetUser(ctx, tenantID, email)
		if err != nil {
			slog.Error("ForgotPassword: identity lookup error", "email", email, "error", err)
			return nil // Don't reveal
		}
		if user != nil {
			// Try with username from identity service
			cred, err = s.credentialRepo.FindByIDentifier(ctx, tenantID, user.Username)
			if err != nil {
				slog.Error("ForgotPassword: credential lookup by username error", "username", user.Username, "error", err)
				return nil
			}
		}
	}

	// Don't reveal whether the email exists
	if cred == nil {
		slog.Info("ForgotPassword: user not found", "identifier", email)
		return nil
	}
	slog.Info("ForgotPassword: user found, issuing reset token", "user_id", cred.UserID)

	// 2. Issue a reset token
	token, err := s.passwordService.IssueResetToken(ctx, cred.UserID, tenantID)
	if err != nil {
		return err
	}

	// 3. Send reset email if email sender is configured
	slog.Info("ForgotPassword: checking email sender", "sender_nil", s.emailSender == nil, "email", email)
	if s.emailSender != nil {
		resetURL := fmt.Sprintf("https://ggid-console.iot2.win/reset-password?token=%s", token)
		body := fmt.Sprintf("You requested a password reset.\n\nClick the link below to reset your password:\n%s\n\nIf you didn't request this, ignore this email.", resetURL)
		if err := s.emailSender.Send(ctx, email, "Password Reset - GGID", body); err != nil {
			slog.Error("ForgotPassword: failed to send reset email", "email", email, "error", err)
		} else {
			slog.Info("ForgotPassword: reset email sent", "email", email)
		}
	}

	return nil
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
func (s *AuthService) MFAService() *MFAService { return s.mfaService }

// KeyProvider returns the JWT signing key provider.
func (s *AuthService) KeyProvider() crypto.KeyProvider {
	return s.tokenService.Provider()
}

// JWTIssuer returns the JWT issuer claim value.
func (s *AuthService) JWTIssuer() string {
	return s.tokenService.jwtIssuer
}

// JWTAudience returns the JWT audience claim value.
func (s *AuthService) JWTAudience() string {
	return s.tokenService.jwtAudience
}

// JWTTTL returns the access token time-to-live.
func (s *AuthService) JWTTTL() time.Duration {
	return s.tokenService.jwtTTL
}

// GetUserScopesAndPermissions returns role names and fine-grained permissions for a user.
func (s *AuthService) GetUserScopesAndPermissions(ctx context.Context, tenantID, userID uuid.UUID) (roles []string, permissions []string) {
	return s.getUserScopesAndPermissions(ctx, tenantID, userID)
}

// LoginWithBackupCode authenticates a user with password + backup code (alternative MFA factor).
// The backup code is consumed (single-use) upon successful verification.
func (s *AuthService) LookupUser(ctx context.Context, tenantID uuid.UUID, identifier string) (*UserInfo, error) {
	return s.identityClient.GetUser(ctx, tenantID, identifier)
}

// LookupCredential retrieves a credential by user ID for rehash operations.
func (s *AuthService) LookupCredential(ctx context.Context, tenantID, userID uuid.UUID) (*domain.Credential, error) {
	return s.credentialRepo.FindByUserID(ctx, tenantID, userID)
}

// UpdateCredentialSecret updates a user's password hash in the database.
// Used by transparent rehashing to replace legacy hashes with Argon2id.
func (s *AuthService) UpdateCredentialSecret(ctx context.Context, tenantID, userID uuid.UUID, newHash string) error {
	cred, err := s.credentialRepo.FindByUserID(ctx, tenantID, userID)
	if err != nil {
		return fmt.Errorf("lookup credential for rehash: %w", err)
	}
	if cred == nil {
		return fmt.Errorf("credential not found for user %s", userID)
	}
	return s.credentialRepo.UpdateSecret(ctx, cred.ID, newHash)
}

// --- Email Verification ---

// VerifyEmailToken validates an email verification token.
func (s *AuthService) VerifyEmailToken(ctx context.Context, token string) (uuid.UUID, uuid.UUID, string, error) {
	if s.emailService == nil {
		return uuid.Nil, uuid.Nil, "", fmt.Errorf("email service not configured")
	}
	return s.emailService.VerifyEmailToken(ctx, token)
}

// PasswordPolicy returns the current password policy configuration.
func (s *AuthService) PasswordPolicy() conf.PasswordPolicy { return s.cfg.Password }

// UpdatePasswordPolicy updates runtime-configurable password policy fields.
// Only non-nil fields are applied; nil fields keep their current value.
func (s *AuthService) UpdatePasswordPolicy(minLen *int, reqUpper, reqLower, reqDigit, reqSpecial *bool, blacklist []string) error {
	policy := s.passwordService.GetPolicy()
	if minLen != nil {
		if *minLen < 1 || *minLen > 128 {
			return fmt.Errorf("min_length must be between 1 and 128")
		}
		policy.MinLength = *minLen
	}
	if reqUpper != nil {
		policy.RequireUpper = *reqUpper
	}
	if reqLower != nil {
		policy.RequireLower = *reqLower
	}
	if reqDigit != nil {
		policy.RequireDigit = *reqDigit
	}
	if reqSpecial != nil {
		policy.RequireSpecial = *reqSpecial
	}
	if blacklist != nil {
		policy.Blacklist = blacklist
	}
	s.passwordService.UpdatePolicy(policy)
	s.cfg.Password = policy
	return nil
}

// SendVerificationEmail generates an email verification token (24h TTL) and
// returns the plaintext token. In production the token is sent via email;
// in dev mode it is returned for direct use.
func (s *AuthService) SendVerificationEmail(ctx context.Context, tenantID, userID uuid.UUID, email string) (string, error) {
	return s.emailService.IssueVerificationToken(ctx, tenantID, userID, email)
}

// --- Per-Tenant MFA Enforcement ---

// IsForceMFA checks if a tenant enforces MFA for all users.
func (s *AuthService) IsForceMFA(ctx context.Context, tenantID uuid.UUID) bool {
	key := fmt.Sprintf("ggid:force_mfa:%s", tenantID)
	val, err := s.rateLimiter.rdb.Get(ctx, key).Result()
	if err != nil {
		return false
	}
	return val == "true"
}

// SetForceMFA enables or disables per-tenant MFA enforcement.
func (s *AuthService) SetForceMFA(ctx context.Context, tenantID uuid.UUID, enabled bool) error {
	key := fmt.Sprintf("ggid:force_mfa:%s", tenantID)
	if enabled {
		return s.rateLimiter.rdb.Set(ctx, key, "true", 0).Err()
	}
	return s.rateLimiter.rdb.Del(ctx, key).Err()
}

// --- Account Lockout ---

// IsAccountLocked checks if an account is locked due to too many failed attempts.
func (s *AuthService) IsAccountLocked(ctx context.Context, tenantID uuid.UUID, identifier string) bool {
	key := fmt.Sprintf("ggid:lockout:%s:%s", tenantID, identifier)
	count, err := s.rateLimiter.rdb.Get(ctx, key).Int()
	if err != nil {
		return false
	}
	return count >= s.cfg.Password.MaxAttempts
}

// RecordFailedLogin increments the failed attempt counter and locks if threshold reached.
func (s *AuthService) RecordFailedLogin(ctx context.Context, tenantID uuid.UUID, identifier string) error {
	key := fmt.Sprintf("ggid:lockout:%s:%s", tenantID, identifier)
	count, err := s.rateLimiter.rdb.Incr(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("increment lockout counter: %w", err)
	}
	if count == 1 {
		s.rateLimiter.rdb.Expire(ctx, key, s.cfg.Password.LockDuration)
	}
	return nil
}

// ResetFailedLogins clears the failed attempt counter after successful login.
func (s *AuthService) ResetFailedLogins(ctx context.Context, tenantID uuid.UUID, identifier string) {
	key := fmt.Sprintf("ggid:lockout:%s:%s", tenantID, identifier)
	s.rateLimiter.rdb.Del(ctx, key)
}

// ResetLoginAttempts clears ALL brute-force counters for a username across
// all identifier variants (username, email, IP+username). Used by admin API.
func (s *AuthService) ResetLoginAttempts(ctx context.Context, username string) error {
	if s.rateLimiter == nil || s.rateLimiter.rdb == nil {
		return nil
	}
	// Clear by username and by email patterns.
	// Keys are ggid:lockout:{tenantID}:{identifier}.
	// Since we don't know the tenantID here, we scan with pattern.
	pattern := fmt.Sprintf("ggid:lockout:*:%s", username)
	iter := s.rateLimiter.rdb.Scan(ctx, 0, pattern, 100).Iterator()
	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	// Also try lowercase variant.
	patternLower := fmt.Sprintf("ggid:lockout:*:%s", strings.ToLower(username))
	iterLower := s.rateLimiter.rdb.Scan(ctx, 0, patternLower, 100).Iterator()
	for iterLower.Next(ctx) {
		keys = append(keys, iterLower.Val())
	}
	if len(keys) > 0 {
		s.rateLimiter.rdb.Del(ctx, keys...)
	}
	return nil
}

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
func (s *AuthService) CheckSessionTimeout(ctx context.Context, sessionID uuid.UUID, createdAt time.Time) error {
	if s.cfg.SessionTimeout.AbsoluteTimeout > 0 {
		if time.Since(createdAt) > s.cfg.SessionTimeout.AbsoluteTimeout {
			return ErrSessionExpired
		}
	}

	if s.cfg.SessionTimeout.IdleTimeout > 0 {
		activityKey := fmt.Sprintf("ggid:session_activity:%s", sessionID)
		lastActiveStr, err := s.rateLimiter.rdb.Get(ctx, activityKey).Result()
		if err == nil {
			lastActive, err := time.Parse(time.RFC3339, lastActiveStr)
			if err == nil && time.Since(lastActive) > s.cfg.SessionTimeout.IdleTimeout {
				return ErrSessionExpired
			}
		}
		now := time.Now().Format(time.RFC3339)
		ttl := s.cfg.SessionTimeout.IdleTimeout
		if ttl == 0 {
			ttl = 30 * time.Minute
		}
		s.rateLimiter.rdb.Set(ctx, activityKey, now, ttl)
	}
	return nil
}

// --- Brute Force Protection (Sliding Window) ---

// CheckBruteForce validates login frequency using a dual-dimension sliding window:
//   - Per IP: max 20 requests per minute
//   - Per username: max 10 requests per hour
func (s *AuthService) CheckBruteForce(ctx context.Context, tenantID uuid.UUID, ip, username string) error {
	ipKey := fmt.Sprintf("ggid:bf:ip:%s", ip)
	if err := s.slidingWindowCheck(ctx, ipKey, 20, time.Minute); err != nil {
		return err
	}
	userKey := fmt.Sprintf("ggid:bf:user:%s:%s", tenantID, username)
	if err := s.slidingWindowCheck(ctx, userKey, 10, time.Hour); err != nil {
		return err
	}
	return nil
}

func (s *AuthService) slidingWindowCheck(ctx context.Context, key string, limit int, window time.Duration) error {
	now := time.Now().UnixNano()
	cutoff := now - window.Nanoseconds()

	pipe := s.rateLimiter.rdb.Pipeline()
	pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", cutoff))
	pipe.ZAdd(ctx, key, redis.Z{Score: float64(now), Member: now})
	countCmd := pipe.ZCard(ctx, key)
	pipe.Expire(ctx, key, window+time.Second)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("sliding window rate limit: %w", err)
	}

	if countCmd.Val() > int64(limit) {
		return ErrRateLimited
	}
	return nil
}

// --- Trusted Device (MFA bypass) ---

const trustedDeviceTTL = 30 * 24 * time.Hour // 30 days

// RememberTrustedDevice stores a device fingerprint as trusted for a user.
// When this user logs in from the same device within 30 days, MFA is skipped.
func (s *AuthService) RememberTrustedDevice(ctx context.Context, userID uuid.UUID, fingerprint, deviceName string) error {
	tc, err := tenant.FromContext(ctx)
	if err != nil {
		return fmt.Errorf("tenant context required: %w", err)
	}

	key := fmt.Sprintf("ggid:trusted_device:%s:%s:%s", tc.TenantID, userID, fingerprint)
	val := fmt.Sprintf("%s:%d", deviceName, time.Now().Unix())
	return s.rateLimiter.rdb.Set(ctx, key, val, trustedDeviceTTL).Err()
}

// IsTrustedDevice checks if a device fingerprint is trusted and within the 30-day window.
func (s *AuthService) IsTrustedDevice(ctx context.Context, tenantID, userID uuid.UUID, fingerprint string) bool {
	key := fmt.Sprintf("ggid:trusted_device:%s:%s:%s", tenantID, userID, fingerprint)
	_, err := s.rateLimiter.rdb.Get(ctx, key).Result()
	return err == nil
}

// --- Password History Summary ---

// GetPasswordHistory returns a summary of stored password hashes for a user.
func (s *AuthService) GetPasswordHistory(ctx context.Context, userID uuid.UUID) ([]map[string]any, error) {
	tc, err := tenant.FromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("tenant context required: %w", err)
	}
	policy := s.passwordService.GetPolicy()
	history, err := s.passwordService.credentialRepo.GetHistory(ctx, tc.TenantID, userID, policy.HistoryCount)
	if err != nil {
		return nil, err
	}

	result := make([]map[string]any, 0, len(history))
	for _, h := range history {
		hashPrefix := h.Secret
		if len(hashPrefix) > 12 {
			hashPrefix = hashPrefix[:12] + "..."
		}
		result = append(result, map[string]any{
			"id":         h.ID.String(),
			"created_at": h.CreatedAt.Format(time.RFC3339),
			"hash_prefix": hashPrefix,
		})
	}
	return result, nil
}

// GenerateWebAuthnChallenge generates a random challenge for WebAuthn flows.
func (s *AuthService) GenerateWebAuthnChallenge(ctx context.Context) (string, error) {
	return crypto.GenerateRandomToken(32)
}

// GetPasswordService returns the password service (may be nil if not configured).
func (s *AuthService) GetPasswordService() *PasswordService {
	return s.passwordService
}
