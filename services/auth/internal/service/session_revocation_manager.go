package service

import (
	"context"
	"log/slog"
	"time"

	ggidauth "github.com/ggid/ggid/pkg/auth"
	"github.com/ggid/ggid/pkg/audit"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// RevocationResult summarizes a user-wide session revocation operation.
type RevocationResult struct {
	SessionsRevoked int      `json:"sessions_revoked"`
	JTIsBlocked     int      `json:"jtis_blocked"`
	RefreshRevoked  int      `json:"refresh_tokens_revoked"`
	BlockedJTIs     []string `json:"blocked_jtis,omitempty"`
}

// SessionRevocationManager orchestrates full user session revocation for CAE Phase 2.
//
// When a user's sessions must be terminated (e.g. brute-force detection, admin action,
// or ITDR → CAE linkage), RevokeUser performs a multi-layer revocation:
//
//  1. DB: mark all active sessions as revoked (sessions.revoked_at = NOW())
//  2. Redis ZSET: add each session's jti to the revoked_jti blocklist with token_exp as score
//  3. Redis DEL: remove refresh-token cache keys (forces DB lookup, which will reject)
//  4. DB: revoke all refresh tokens for the user
//  5. NATS: publish audit event (session.revoke)
//
// The gateway's CAECheck middleware reads the Redis ZSET on every request (~0.3ms),
// so revoked access tokens are rejected in real-time without waiting for expiry.
type SessionRevocationManager struct {
	sessionRepo    SessionRepo
	refreshRepo    RefreshTokenRepo
	jtiBlocklist   *ggidauth.JTIBlocklist
	rdb            *redis.Client
	auditPublisher *audit.Publisher
}

// NewSessionRevocationManager creates a revocation manager.
func NewSessionRevocationManager(
	sessionRepo SessionRepo,
	refreshRepo RefreshTokenRepo,
	jtiBlocklist *ggidauth.JTIBlocklist,
	rdb *redis.Client,
	auditPublisher *audit.Publisher,
) *SessionRevocationManager {
	return &SessionRevocationManager{
		sessionRepo:    sessionRepo,
		refreshRepo:    refreshRepo,
		jtiBlocklist:   jtiBlocklist,
		rdb:            rdb,
		auditPublisher: auditPublisher,
	}
}

// RevokeUser revokes all sessions, access tokens, and refresh tokens for a user.
// This is the core CAE Phase 2 operation.
//
// The reason parameter is included in audit events for traceability.
func (m *SessionRevocationManager) RevokeUser(ctx context.Context, tenantID, userID uuid.UUID, reason string) (*RevocationResult, error) {
	result := &RevocationResult{}

	// 1. Query active JTIs before revoking (so we can blocklist them in Redis).
	jtis, err := m.sessionRepo.ListActiveJTIForUser(ctx, tenantID, userID)
	if err != nil {
		slog.Warn("SessionRevocationManager: failed to list active JTIs",
			"tenant_id", tenantID, "user_id", userID, "error", err)
		// Continue — DB revocation still works even if JTI query fails.
	}

	// 2. Add JTIs to Redis blocklist (gateway CAECheck will reject these tokens).
	blockedJTIs := make([]string, 0, len(jtis))
	for _, j := range jtis {
		if j.JTI == "" {
			continue
		}
		exp := j.TokenExp
		if exp.IsZero() || exp.Before(time.Now()) {
			exp = time.Now().Add(15 * time.Minute) // fallback: block for 15 min
		}
		if err := m.jtiBlocklist.Revoke(ctx, j.JTI, exp); err != nil {
			slog.Warn("SessionRevocationManager: failed to blocklist JTI",
				"jti", j.JTI, "error", err)
			continue
		}
		blockedJTIs = append(blockedJTIs, j.JTI)
	}
	result.JTIsBlocked = len(blockedJTIs)
	result.BlockedJTIs = blockedJTIs

	// 3. Revoke all sessions in DB (RevokeAllForUser with exceptID=uuid.Nil = revoke ALL).
	if err := m.sessionRepo.RevokeAllForUser(ctx, tenantID, userID, uuid.Nil); err != nil {
		slog.Error("SessionRevocationManager: failed to revoke sessions in DB",
			"tenant_id", tenantID, "user_id", userID, "error", err)
		// Don't return error — partial revocation is better than none.
	}
	result.SessionsRevoked = len(jtis)

	// 4. Revoke all refresh tokens in DB.
	if err := m.refreshRepo.RevokeAllForUser(ctx, tenantID, userID); err != nil {
		slog.Error("SessionRevocationManager: failed to revoke refresh tokens",
			"tenant_id", tenantID, "user_id", userID, "error", err)
	}
	result.RefreshRevoked = 1 // RevokeAllForUser doesn't return count; mark as attempted.

	// 5. Publish audit event.
	if m.auditPublisher != nil {
		event := audit.NewEvent("session.revoke", "success", tenantID, userID)
		event.Metadata = map[string]any{
			"reason":         reason,
			"sessions_count": result.SessionsRevoked,
			"jtis_count":     result.JTIsBlocked,
		}
		m.auditPublisher.PublishAsync(event)
	}

	slog.Info("SessionRevocationManager: user sessions revoked",
		"tenant_id", tenantID,
		"user_id", userID,
		"reason", reason,
		"sessions", result.SessionsRevoked,
		"jtis_blocked", result.JTIsBlocked,
	)

	return result, nil
}
