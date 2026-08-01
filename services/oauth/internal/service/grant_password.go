package service

// Password grant (RFC 6749 §4.3) and MFA helpers for OAuthService.
// Extracted from oauth_service.go for maintainability.

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	pkgcrypto "github.com/ggid/ggid/pkg/crypto"
	"github.com/ggid/ggid/pkg/errors"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

func (s *OAuthService) PasswordGrant(ctx context.Context, req *PasswordGrantRequest) (*TokenResponse, error) {
	if req.Username == "" || req.Password == "" {
		return nil, errors.New(errors.ErrInvalidArgument, "username and password are required")
	}

	var tenantID uuid.UUID = req.TenantID

	client, clientErr := s.clientRepo.GetClientByID(ctx, tenantID, req.ClientID)
	if clientErr != nil {
		return nil, errors.Unauthenticated("client not found")
	}
	if !client.SupportsGrantType("password") {
		return nil, errors.InvalidArgument("client does not support password grant")
	}

	if client.IsConfidential() {
		if req.ClientSecret == "" {
			return nil, errors.Unauthenticated("client authentication required for confidential clients")
		}
		ok, _ := pkgcrypto.VerifyPassword(req.ClientSecret, client.ClientSecretHash)
		if !ok {
			return nil, errors.Unauthenticated("client authentication failed")
		}
	}

	if !client.Enabled {
		return nil, errors.InvalidArgument("client is disabled")
	}

	var userID uuid.UUID

	if s.pool != nil {
		var dbUserID uuid.UUID
		var credHash string
		err := s.pool.QueryRow(ctx, `
			SELECT u.id, c.secret
			FROM users u
			JOIN credentials c ON c.user_id = u.id AND c.type = 'password'
			WHERE u.username = $1 AND u.tenant_id = $2 AND u.status = 'active'
			  AND c.enabled = true
			  AND (c.locked_until IS NULL OR c.locked_until < NOW())`,
			req.Username, req.TenantID).Scan(&dbUserID, &credHash)
		if err != nil {
			return nil, errors.Unauthenticated("invalid credentials")
		}
		if credHash == "" {
			return nil, errors.Unauthenticated("invalid credentials")
		}

		ok, _ := pkgcrypto.VerifyPassword(req.Password, credHash)
		if !ok {
			_, _ = s.pool.Exec(ctx, `
				UPDATE credentials
				SET failed_attempts = failed_attempts + 1,
				    locked_until = CASE
				      WHEN failed_attempts >= 4 THEN NOW() + INTERVAL '15 minutes'
				      ELSE locked_until END,
				    updated_at = NOW()
				WHERE user_id = $1 AND tenant_id = $2 AND type = 'password'`,
				dbUserID, req.TenantID)
			return nil, errors.Unauthenticated("invalid credentials")
		}
		_, _ = s.pool.Exec(ctx, `
			UPDATE credentials
			SET failed_attempts = 0, locked_until = NULL, last_used_at = NOW(), updated_at = NOW()
			WHERE user_id = $1 AND tenant_id = $2 AND type = 'password'`,
			dbUserID, req.TenantID)
		userID = dbUserID
	} else {
		return nil, errors.New(errors.ErrInternal, "database not configured")
	}

	if s.pool != nil {
		var mfaCount int
		s.pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM mfa_devices WHERE tenant_id = $1 AND user_id = $2 AND enabled = true AND verified_at IS NOT NULL`,
			tenantID, userID).Scan(&mfaCount)
		if mfaCount > 0 {
			if req.MFACode == "" && req.BackupCode == "" {
				return nil, errors.New(errors.ErrUnauthenticated, "mfa_required")
			}
			if req.BackupCode != "" {
				bcID, bcErr := s.verifyBackupCode(ctx, tenantID, userID, req.BackupCode)
				if bcErr != nil {
					return nil, errors.New(errors.ErrUnauthenticated, "invalid backup code")
				}
				_ = bcID
			} else {
				var secret string
				s.pool.QueryRow(ctx,
					`SELECT secret FROM mfa_devices WHERE tenant_id = $1 AND user_id = $2 AND enabled = true AND verified_at IS NOT NULL LIMIT 1`,
					tenantID, userID).Scan(&secret)
				if secret != "" {
					dec, dErr := pkgcrypto.DecryptTOTPSecret(secret)
					if dErr != nil {
						return nil, errors.Internal("decrypt mfa secret", dErr)
					}
					secret = dec
				}
				if secret == "" || !validateTOTP(secret, req.MFACode) {
					return nil, errors.New(errors.ErrUnauthenticated, "invalid mfa code")
				}
			}
		}
	}

	capAction, capPolicy := s.evaluateConditionalAccess(ctx, tenantID, userID, req.Username)
	switch capAction {
	case "block", "deny":
		return nil, errors.Unauthenticated(fmt.Sprintf("access denied by policy: %s", capPolicy))
	case "require_mfa":
		if req.MFACode == "" && req.BackupCode == "" {
			return nil, errors.New(errors.ErrUnauthenticated,
				fmt.Sprintf("mfa required by policy: %s", capPolicy))
		}
		if req.BackupCode != "" {
			if _, err := s.verifyBackupCode(ctx, tenantID, userID, req.BackupCode); err != nil {
				return nil, errors.New(errors.ErrUnauthenticated, "invalid backup code")
			}
		}
		var mfaVerified int
		s.pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM mfa_devices WHERE tenant_id = $1 AND user_id = $2 AND enabled = true AND verified_at IS NOT NULL`,
			tenantID, userID).Scan(&mfaVerified)
		if mfaVerified == 0 {
			return nil, errors.New(errors.ErrUnauthenticated,
				fmt.Sprintf("mfa required by policy but no verified device: %s", capPolicy))
		}
	}

	permissions, _ := s.fetchUserPermissions(ctx, tenantID, userID)
	roles := s.fetchUserRoles(ctx, tenantID, userID)

	filteredScopes := filterSafeScopes(req.Scope)
	roleKeys := s.fetchUserRoleKeys(ctx, tenantID, userID)
	scopeStr := strings.TrimSpace(joinScopes(filteredScopes) + " " + strings.Join(roleKeys, " "))

	now := time.Now()
	expiresAt := now.Add(15 * time.Minute)

	claimsMap := jwt.MapClaims{
		"iss":         s.issuer,
		"sub":         userID.String(),
		"aud":         resolveAudience(req.Audience, req.ClientID),
		"iat":         now.Unix(),
		"exp":         expiresAt.Unix(),
		"jti":         uuid.New().String(),
		"tenant_id":   tenantID.String(),
		"scope":       scopeStr,
		"permissions": permissions,
		"roles":       roles,
	}

	token := jwt.NewWithClaims(s.signingMethod(), claimsMap)
	token.Header["kid"] = s.keyProvider.Metadata().KeyID
	signed, err := token.SignedString(s.keyProvider.Signer())
	if err != nil {
		return nil, fmt.Errorf("sign access token: %w", err)
	}

	resp := &TokenResponse{
		AccessToken: signed,
		TokenType:   "Bearer",
		ExpiresIn:   int(expiresAt.Sub(now).Seconds()),
		Scope:       scopeStr,
	}

	if contains(filteredScopes, "offline_access") {
		if client, err := s.clientRepo.GetClientByID(ctx, tenantID, req.ClientID); err == nil && client.SupportsGrantType("refresh_token") {
			refreshPlain, err := s.issueRefreshTokenRecord(ctx, tenantID, client.ID, userID, filteredScopes, "")
			if err != nil {
				return nil, err
			}
			resp.RefreshToken = refreshPlain
		}
	}

	if s.pool != nil {
		sessionID := uuid.New()
		_, _ = s.pool.Exec(ctx, "SELECT set_config('app.tenant_id', $1, true)", tenantID.String())
		tokenJTI, _ := claimsMap["jti"].(string)
		_, sessionErr := s.pool.Exec(ctx, `
			INSERT INTO sessions (id, tenant_id, user_id, token_hash, jti, token_exp, ip_address, user_agent, expires_at, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, '')::inet, $8, $9, NOW())`,
			sessionID, tenantID, userID,
			"jwt:"+signed[:min(64, len(signed))],
			tokenJTI, expiresAt,
			ctxIP(ctx), ctxUserAgent(ctx),
			expiresAt)
		if sessionErr != nil {
			slog.Warn("password grant: failed to create session record", "error", sessionErr, "user_id", userID)
		}
	}

	return resp, nil
}

func validateTOTP(secret, code string) bool {
	if secret == "" || len(code) != 6 {
		return false
	}
	valid, err := totp.ValidateCustom(code, secret, time.Now().UTC(),
		totp.ValidateOpts{
			Period:    30,
			Skew:      1,
			Digits:    otp.DigitsSix,
			Algorithm: otp.AlgorithmSHA1,
		})
	return err == nil && valid
}

func (s *OAuthService) verifyBackupCode(ctx context.Context, tenantID, userID uuid.UUID, code string) (uuid.UUID, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return uuid.Nil, fmt.Errorf("empty backup code")
	}
	rows, err := s.pool.Query(ctx,
		`SELECT id, code_hash FROM backup_codes WHERE tenant_id = $1 AND user_id = $2 AND used_at IS NULL`,
		tenantID, userID)
	if err != nil {
		return uuid.Nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id uuid.UUID
		var hash string
		if err := rows.Scan(&id, &hash); err != nil {
			continue
		}
		if ok, _ := pkgcrypto.VerifyPassword(code, hash); ok {
			tag, execErr := s.pool.Exec(ctx, `UPDATE backup_codes SET used_at = $1 WHERE id = $2 AND used_at IS NULL`, time.Now().UTC(), id)
			if execErr != nil {
				return uuid.Nil, execErr
			}
			if tag.RowsAffected() == 0 {
				return uuid.Nil, fmt.Errorf("invalid backup code")
			}
			return id, nil
		}
	}
	return uuid.Nil, fmt.Errorf("invalid backup code")
}
