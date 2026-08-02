package service

// Token revocation methods for OAuthService.
// Extracted from oauth_service.go.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"github.com/google/uuid"
	"log/slog"
	"strconv"
)

func (s *OAuthService) RevokeToken(tokenStr string, tokenTypeHint ...string) error {
	if tokenStr == "" {
		return nil // RFC 7009: return 200 even for empty token
	}

	// Store the token hash in the revocation list.
	tokenHash := hashTokenSHA256(tokenStr)

	// Handle refresh token revocation: if the hint says refresh_token, or
	// if the token doesn't parse as a JWT, try revoking it as a refresh
	// token in the DB before falling back to the JWT blacklist path.
	hintIsRefresh := len(tokenTypeHint) > 0 && tokenTypeHint[0] == "refresh_token"
	if hintIsRefresh || !strings.Contains(tokenStr, ".") {
		// Try to revoke as a refresh token in the DB.
		if s.tokenRepo != nil {
			// Nil tenant: revocation matches by token hash alone (hash is a
			// SHA-256 of a high-entropy token, so this is safe).
			if err := s.tokenRepo.RevokeRefreshToken(context.Background(), uuid.Nil, tokenHash); err != nil {
				slog.Warn("oauth: failed to revoke refresh token in DB", "err", err)
			}
		}
		// Also blacklist the hash in Redis (covers cross-instance checks).
		if s.rdb != nil {
			// SECURITY: Use TTL to prevent unbounded Redis memory growth from
			// invalid token revocation attempts. 24h is sufficient for propagation.
			if err := s.rdb.Set(context.Background(), "oauth:revoked:"+tokenHash, "0", 24*time.Hour); err != nil {
				slog.Warn("oauth: failed to blacklist revoked token hash in redis", "err", err)
			}
			// SECURITY: Also delete Auth service refresh token key (ggid:rt:)
			// to revoke tokens issued by /api/v1/auth/login via the OAuth endpoint.
			s.rdb.Del(context.Background(), "ggid:rt:"+tokenHash)
		}
		// SECURITY: Also revoke in the Auth service's refresh_tokens table
		// (separate from oauth's oidc_refresh_tokens).
		if s.pool != nil {
			if _, err := s.pool.Exec(context.Background(),
				`UPDATE refresh_tokens SET revoked = true, revoked_at = NOW() WHERE token_hash = $1 AND revoked = false`,
				tokenHash); err != nil {
				slog.Warn("oauth: failed to revoke auth refresh token in DB", "err", err)
			}
		}
		return nil
	}

	// Parse the token to get its claims (don't fail on invalid tokens).
	claims, err := s.ParseAccessToken(tokenStr)
	if err != nil {
		// RFC 7009: invalid token → still return 200, but store hash
		// so IsTokenRevoked can report it as revoked.
		// Try Redis first (for HA/multi-instance).
		if s.rdb != nil {
			// SECURITY: TTL prevents unbounded growth from repeated invalid revokes.
			if e := s.rdb.Set(context.Background(), "oauth:revoked:"+tokenHash, "0", 24*time.Hour); e == nil {
				return nil
			}
		}
		revokedTokens.Store(tokenHash, int64(0))
		return nil
	}

	exp := getInt64Claim(claims, "exp")
	// Calculate TTL: revoke until token expiry (no point keeping it longer).
	var ttl time.Duration
	if exp > 0 {
		ttl = time.Until(time.Unix(exp, 0))
		if ttl <= 0 {
			ttl = 0 // already expired, no TTL needed
		}
	}
	// SECURITY: Cascade — revoke refresh tokens for this user AND client only,
	// not ALL the user's refresh tokens across all clients/sessions.
	// Revoking one stolen token should not DoS the victim's other sessions.
	if s.pool != nil {
		tenantIDStr := getStringClaim(claims, "tenant_id")
		subStr := getStringClaim(claims, "sub")
		clientID := getStringClaim(claims, "aud") // audience = client_id
		if tenantIDStr != "" && subStr != "" {
			tenantID, _ := uuid.Parse(tenantIDStr)
			userID, _ := uuid.Parse(subStr)
			if tenantID != uuid.Nil && userID != uuid.Nil {
				ctx := context.Background()
				if clientID != "" {
					_, _ = s.pool.Exec(ctx,
						`UPDATE oidc_refresh_tokens SET revoked = true WHERE tenant_id = $1 AND user_id = $2 AND client_id = $3 AND revoked = false`,
						tenantID, userID, clientID)
				} else {
					_, _ = s.pool.Exec(ctx,
						`UPDATE oidc_refresh_tokens SET revoked = true WHERE tenant_id = $1 AND user_id = $2 AND revoked = false`,
						tenantID, userID)
				}
			}
		}
	}

	// Try Redis first (for HA/multi-instance).
	if s.rdb != nil {
		if e := s.rdb.Set(context.Background(), "oauth:revoked:"+tokenHash, strconv.FormatInt(exp, 10), ttl); e == nil {
			// Also add JTI to the Gateway CAE blocklist ZSET so the gateway
			// can reject revoked tokens on every request (continuous access evaluation).
			if jti := getStringClaim(claims, "jti"); jti != "" && exp > 0 {
				s.rdb.ZAdd(context.Background(), "ggid:revoked_jti", float64(exp), jti)
				// Mark the sessions row revoked so DB-backed revocation
				// checks (IsTokenRevoked fallback, token exchange B1)
				// observe the revocation even after Redis TTL expiry.
				if s.pool != nil {
					_, _ = s.pool.Exec(context.Background(),
						`UPDATE sessions SET revoked_at = NOW() WHERE jti = $1 AND revoked_at IS NULL`, jti)
				}
			}
			return nil
		}
	}
	revokedTokens.Store(tokenHash, exp)

	return nil
}

// IsTokenRevoked checks if a token has been revoked.
// Priority: Redis (fast cache) → DB (authoritative) → fail-safe deny.
// The old sync.Map fallback is removed — it was unsafe across pod restarts
// (revoked tokens would survive restart) and multi-instance deployments.
func (s *OAuthService) IsTokenRevoked(tokenStr string) bool {
	tokenHash := hashTokenSHA256(tokenStr)
	cacheKey := "oauth:revoked:" + tokenHash

	// 1. Redis cache (fast path).
	if s.rdb != nil {
		if _, err := s.rdb.Get(context.Background(), cacheKey); err == nil {
			return true
		}
	}

	// 2. In-memory cache (best-effort, for single-instance dev/test).
	if _, ok := revokedTokens.Load(tokenHash); ok {
		return true
	}

	// 3. DB check (authoritative — survives restarts, consistent across instances).
	// Parse the token to get its jti for a DB lookup.
	claims, err := s.ParseAccessToken(tokenStr)
	if err != nil {
		// Unparseable token — can't check DB, assume not revoked (the token
		// is invalid anyway and will be rejected by JWT validation).
		return false
	}
	jti, _ := claims["jti"].(string)
	if jti == "" || s.pool == nil {
		return false // no jti or no DB → can't check
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var revoked bool
	err = s.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM sessions WHERE jti = $1 AND revoked_at IS NOT NULL)`,
		jti).Scan(&revoked)
	if err == nil && revoked {
		// Cache in Redis for future lookups.
		if s.rdb != nil {
			exp := getInt64Claim(claims, "exp")
			ttl := time.Duration(0)
			if exp > 0 {
				ttl = time.Until(time.Unix(exp, 0))
			}
			s.rdb.Set(context.Background(), cacheKey, "1", ttl)
		}
		revokedTokens.Store(tokenHash, getInt64Claim(claims, "exp"))
		return true
	}

	return false
}

func hashTokenSHA256(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// maskUser redacts a username for safe logging (PII protection).
// e.g. "alice.admin" → "al***", "a" → "***"
func maskUser(username string) string {
	if len(username) <= 2 {
		return "***"
	}
	return username[:2] + "***"
}
