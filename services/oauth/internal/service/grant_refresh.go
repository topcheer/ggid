package service

// Refresh token grant methods for OAuthService.
// Extracted from oauth_service.go.

import (
	"context"
	"fmt"
	"strings"
	"log/slog"
	"time"

	"crypto/sha256"
	"encoding/base64"
	pkgcrypto "github.com/ggid/ggid/pkg/crypto"
	"github.com/ggid/ggid/pkg/errors"
	"github.com/ggid/ggid/services/oauth/internal/domain"
	"github.com/google/uuid"
)
func (s *OAuthService) RefreshToken(ctx context.Context, req *RefreshTokenRequest) (*TokenResponse, error) {
	// 1. Look up the client.
	client, err := s.clientRepo.GetClientByID(ctx, req.TenantID, req.ClientID)
	if err != nil {
		return nil, errors.Unauthenticated("client authentication failed")
	}

	// 2. Verify client secret for confidential clients.
	//    RFC 6749 §2.3: If token_endpoint_auth_method is "none", skip secret.
	if client.IsConfidential() && client.TokenEndpointAuthMethod != "none" {
		ok, _ := pkgcrypto.VerifyPassword(req.ClientSecret, client.ClientSecretHash)
		if !ok {
			return nil, errors.Unauthenticated("invalid client credentials")
		}
	}

	// 3. Reject disabled clients — disabling a client must immediately stop
	//    all token issuance, including refresh-token rotation.
	if !client.Enabled {
		return nil, errors.InvalidArgument("client is disabled")
	}

	// 4. Verify grant type.
	if !client.SupportsGrantType("refresh_token") {
		return nil, errors.InvalidArgument("client does not support refresh_token grant")
	}

	// 4. Hash the refresh token and look it up.
	tokenHash := hashTokenSHA256(req.RefreshToken)
	var fromAuthStore bool
	record, err := s.tokenRepo.GetRefreshToken(ctx, req.TenantID, tokenHash)
	if err != nil || record == nil {
		// Fallback: check if this is a refresh token issued by the Auth service.
		// Auth service stores tokens in Redis with key "ggid:rt:{sha256_hex}".
		if s.rdb != nil {
			if authRecord, authErr := s.lookupAuthRefreshToken(ctx, req.TenantID, tokenHash, req.RefreshToken, client.ID); authErr == nil && authRecord != nil {
				record = authRecord
				fromAuthStore = true
			}
		}
		if record == nil {
			return nil, errors.Unauthenticated("invalid refresh token")
		}
	}

	// 4b. SECURITY (RFC 6749 §1.5): the refresh token must belong to the
	// requesting client. Without this check, a token stolen from client A
	// can be refreshed using client B's credentials.
	if record.ClientID != uuid.Nil && record.ClientID != client.ID {
		return nil, errors.Unauthenticated("refresh token was issued to a different client")
	}

	// 5. Reuse detection (RFC 6749 §10.4): a used/revoked token presented
	// again means theft — mark the family and revoke ALL of its tokens.
	if record.Used || record.Revoked {
		if record.FamilyID != "" && s.tokenFamilyStore != nil {
			_ = s.tokenFamilyStore.MarkTheft(ctx, record.FamilyID)
		}
		s.revokeFamily(ctx, req.TenantID, client.ID, record.FamilyID)
		return nil, errors.Unauthenticated("refresh token reuse detected — all tokens revoked")
	}

	// 6. Check expiry.
	if time.Now().After(record.ExpiresAt) {
		if err := s.tokenRepo.RevokeRefreshToken(ctx, req.TenantID, tokenHash); err != nil {
			slog.Warn("oauth: failed to revoke expired refresh token", "err", err)
		}
		return nil, errors.Unauthenticated("refresh token expired")
	}

	// 7. Token is valid — rotate it (mark old as used, issue new).

	// 6b. SECURITY: verify the user is still active. A soft-deleted, suspended,
	// or locked user must not be able to refresh tokens indefinitely.
	if s.pool != nil {
		var userStatus string
		err := s.pool.QueryRow(ctx,
			`SELECT status FROM users WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`,
			record.UserID, req.TenantID).Scan(&userStatus)
		if err != nil || userStatus != "active" {
			// Revoke the token since the user is no longer valid.
			if rerr := s.tokenRepo.RevokeRefreshToken(ctx, req.TenantID, tokenHash); rerr != nil {
				slog.Warn("oauth: failed to revoke refresh token of inactive user", "err", rerr)
			}
			return nil, errors.Unauthenticated("user account is not active")
		}
	}

	// 7. Atomically consume the old token (rotation). The conditional UPDATE
	// closes the check-then-use TOCTOU race: exactly one concurrent request
	// can consume the token; a loser means the token was already consumed —
	// i.e. reuse — and must trigger theft response.
	// Auth-store (Redis) tokens are not in the DB, so consume misses them;
	// skip reuse handling for that path (its old token lifecycle is owned by
	// the Auth service).
	consumed, err := s.tokenRepo.ConsumeRefreshToken(ctx, req.TenantID, tokenHash)
	if err != nil {
		slog.Error("oauth: failed to consume (rotate) refresh token", "err", err)
		return nil, errors.Internal("token rotation failed", err)
	}
	if !consumed && !fromAuthStore {
		slog.Warn("oauth: refresh token consumed concurrently — treating as reuse",
			"tenant", req.TenantID, "family", record.FamilyID)
		if record.FamilyID != "" && s.tokenFamilyStore != nil {
			_ = s.tokenFamilyStore.MarkTheft(ctx, record.FamilyID)
		}
		s.revokeFamily(ctx, req.TenantID, client.ID, record.FamilyID)
		return nil, errors.Unauthenticated("refresh token reuse detected — all tokens revoked")
	}

	// 7a. Resolve the rotation family: inherit from the consumed token, or
	// start a new family rooted at the consumed token's ID.
	familyID := record.FamilyID
	if familyID == "" {
		familyID = record.ID.String()
	}

	// 8. Issue new access token.
	// SECURITY: Filter client-requested scopes to standard OAuth scopes only.
	// Admin scopes (platform:*, tenant:*) come from the user's DB role keys,
	// not from the refresh request — prevents scope escalation.
	// RFC 6749 §6: requested scope must not include any scope not originally granted.
	safeScopes := filterSafeScopes(req.Scope)
	// Narrow to scopes that were in the original authorization (record.Scope).
	if len(record.Scope) > 0 {
		recordScopeSet := make(map[string]bool, len(record.Scope))
		for _, sc := range record.Scope {
			recordScopeSet[sc] = true
		}
		narrowed := safeScopes[:0]
		for _, sc := range safeScopes {
			if recordScopeSet[sc] {
				narrowed = append(narrowed, sc)
			}
		}
		safeScopes = narrowed
	}
	roleKeys := s.fetchUserRoleKeys(ctx, req.TenantID, record.UserID)
	accessTokenScope := strings.TrimSpace(joinScopes(safeScopes) + " " + strings.Join(roleKeys, " "))
	accessToken, expiresIn, err := s.issueAccessToken(record.UserID, req.TenantID, resolveAudience(req.Audience, client.ClientID), accessTokenScope)
	if err != nil {
		return nil, err
	}

	// 9. Issue new refresh token (rotation).
	newRefreshToken, err := pkgcrypto.GenerateRandomToken(32)
	if err != nil {
		return nil, errors.Internal("generate refresh token", err)
	}
	newRecord := &domain.RefreshTokenRecord{
		ID:        uuid.New(),
		TenantID:  req.TenantID,
		ClientID:  client.ID,
		UserID:    record.UserID,
		TokenHash: hashTokenSHA256(newRefreshToken),
		Scope:     safeScopes,
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour), // 30 days
		FamilyID:  familyID,
	}
	// SECURITY: if storing the new token fails we must not hand the client an
	// unusable refresh token — fail the request instead.
	if err := s.tokenRepo.StoreRefreshToken(ctx, newRecord); err != nil {
		slog.Error("oauth: failed to store rotated refresh token", "err", err)
		return nil, errors.Internal("store refresh token", err)
	}

	// 9a. Register the rotation in the family registry (best-effort).
	if s.tokenFamilyStore != nil {
		_ = s.tokenFamilyStore.RegisterRotation(ctx, familyID, record.ID.String(), newRecord.ID.String())
	}

	resp := &TokenResponse{
		AccessToken:  accessToken,
		TokenType:    "Bearer",
		ExpiresIn:    expiresIn,
		RefreshToken: newRefreshToken,
		Scope:        accessTokenScope,
	}

	// OIDC Core §12: issue a fresh id_token on refresh when openid scope present.
	if contains(safeScopes, "openid") {
		accessTokenHash := sha256.Sum256([]byte(accessToken))
		atHash := base64.RawURLEncoding.EncodeToString(accessTokenHash[:16])
		idToken, err := s.issueIDToken(record.UserID, req.TenantID, client.ClientID, "", &IDTokenOptions{
			AtHash: atHash,
		})
		if err == nil {
			resp.IDToken = idToken
		}
	}

	return resp, nil
}

// lookupAuthRefreshToken checks the Auth service's Redis store for a refresh
// token issued by /api/v1/auth/login. The Auth service stores tokens with key
// "ggid:rt:{sha256_hex}" and value = token ID (UUID). We read the token ID,
// then construct a RefreshTokenRecord so the caller can issue new tokens.
func (s *OAuthService) lookupAuthRefreshToken(ctx context.Context, tenantID uuid.UUID, tokenHash, plaintext string, requestingClientID uuid.UUID) (*domain.RefreshTokenRecord, error) {
	redisKey := "ggid:rt:" + tokenHash
	// SECURITY: Use GetDel for one-time use semantics — prevents indefinite token reuse.
	tokenIDStr, err := s.rdb.GetDel(ctx, redisKey)
	if err != nil || tokenIDStr == "" {
		return nil, fmt.Errorf("refresh token not found in auth redis")
	}

	tokenID, err := uuid.Parse(tokenIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid token ID in redis: %s", tokenIDStr)
	}

	// SECURITY (R24 P1): Restore UserID from the Auth service's refresh_tokens
	// table. Without this, record.UserID = uuid.Nil → production user-status
	// check fails (token rotation broken) / dev issues nil-subject tokens.
	var userID, userTenantID uuid.UUID
	if s.pool != nil {
		err = s.pool.QueryRow(ctx,
			`SELECT user_id, tenant_id FROM refresh_tokens WHERE id = $1`,
			tokenID).Scan(&userID, &userTenantID)
		if err != nil {
			return nil, fmt.Errorf("refresh token not found in auth store")
		}
		// SECURITY: Verify the token belongs to the requesting tenant.
		if userTenantID != tenantID {
			return nil, fmt.Errorf("refresh token tenant mismatch")
		}
	} else {
		// dev/test: no DB, cannot verify user — fail-closed.
		return nil, fmt.Errorf("auth store unavailable")
	}

	// SECURITY (P2-56): Bind the token to the requesting client so the
	// caller's client_id binding check (L1654) applies. Auth-issued tokens
	// are first-party (gcid-console), so binding to the requesting client
	// is correct — only the client that received the token can refresh it.
	return &domain.RefreshTokenRecord{
		ID:        tokenID,
		TenantID:  tenantID,
		UserID:    userID,
		ClientID:  requestingClientID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour), // Auth tokens expire in 30 days
	}, nil
}

// --- Client Credentials Grant ---

// ClientCredentialsRequest holds parameters for the client_credentials grant.
type ClientCredentialsRequest struct {
	TenantID     uuid.UUID
	ClientID     string
	ClientSecret string
	Scope        []string
	Audience     string // optional target audience (defaults to client_id)
}

// ClientCredentials issues tokens for machine-to-machine authentication.
