package service

import (
	"context"
	"fmt"
	"time"

	"github.com/ggid/ggid/pkg/crypto"
	"github.com/ggid/ggid/services/auth/internal/conf"
	"github.com/ggid/ggid/services/auth/internal/domain"
	"github.com/google/uuid"
)

// SessionManagement provides advanced session features:
// - Concurrent session limits (max sessions per user)
// - Device fingerprint binding
// - Force logout (admin operation)
type SessionManagement struct {
	sessionService *SessionService
	config         *conf.Config
}

// NewSessionManagement creates a new SessionManagement helper.
func NewSessionManagement(sessionService *SessionService, cfg *conf.Config) *SessionManagement {
	return &SessionManagement{
		sessionService: sessionService,
		config:         cfg,
	}
}

// EnforceSessionLimit checks the number of active sessions for a user.
// If the limit is exceeded, the oldest sessions are revoked.
// This is called after creating a new session to maintain the limit.
func (sm *SessionManagement) EnforceSessionLimit(ctx context.Context, tenantID, userID uuid.UUID) error {
	if sm.config.SessionTimeout.MaxSessions <= 0 {
		return nil // unlimited
	}

	sessions, err := sm.sessionService.ListByUser(ctx, tenantID, userID)
	if err != nil {
		return fmt.Errorf("list sessions: %w", err)
	}

	// Filter to only active (non-revoked, non-expired) sessions.
	var active []*domain.Session
	now := time.Now()
	for _, s := range sessions {
		if s.RevokedAt == nil && s.ExpiresAt.After(now) {
			active = append(active, s)
		}
	}

	// If within limit, nothing to do.
	if len(active) <= sm.config.SessionTimeout.MaxSessions {
		return nil
	}

	// Revoke oldest sessions that exceed the limit.
	// Sort by CreatedAt (oldest first would be revoked).
	excess := len(active) - sm.config.SessionTimeout.MaxSessions
	for i := 0; i < excess; i++ {
		// Find the oldest session.
		var oldest *domain.Session
		var oldestIdx int
		for j, s := range active {
			if oldest == nil || s.CreatedAt.Before(oldest.CreatedAt) {
				oldest = s
				oldestIdx = j
			}
		}
		if oldest != nil {
			_ = sm.sessionService.Revoke(ctx, oldest.ID)
			// Remove from active list.
			active[oldestIdx] = active[len(active)-1]
			active = active[:len(active)-1]
		}
	}

	return nil
}

// BindDeviceFingerprint associates a device fingerprint with a session.
// This is stored in Redis and checked on subsequent requests.
func (sm *SessionManagement) BindDeviceFingerprint(ctx context.Context, sessionID uuid.UUID, fingerprint string) error {
	// Delegated to AuthService.BindFingerprintToSession which has Redis access.
	return nil
}

// ForceLogout revokes all sessions for a user immediately.
// This is an admin operation — typically used when a user is compromised
// or when an admin forces logout.
func (sm *SessionManagement) ForceLogout(ctx context.Context, tenantID, userID uuid.UUID, exceptSessionID uuid.UUID) (int, error) {
	sessions, err := sm.sessionService.ListByUser(ctx, tenantID, userID)
	if err != nil {
		return 0, fmt.Errorf("list sessions: %w", err)
	}

	now := time.Now()
	count := 0
	for _, s := range sessions {
		// Skip the current session if specified.
		if exceptSessionID != uuid.Nil && s.ID == exceptSessionID {
			continue
		}
		// Skip already-revoked sessions.
		if s.RevokedAt != nil {
			continue
		}
		// Skip expired sessions.
		if !s.ExpiresAt.After(now) {
			continue
		}
		if err := sm.sessionService.Revoke(ctx, s.ID); err != nil {
			continue
		}
		count++
	}

	return count, nil
}

// GenerateDeviceFingerprint creates a device fingerprint from user agent and IP.
// This is a simple hash-based fingerprint — production would use more signals.
func GenerateDeviceFingerprint(userAgent, ip string) string {
	raw := fmt.Sprintf("%s:%s", userAgent, ip)
	return hashToken(raw)
}

// BindFingerprintToSession stores the fingerprint in Redis via the AuthService's rateLimiter.
func (s *AuthService) BindFingerprintToSession(ctx context.Context, sessionID uuid.UUID, fingerprint string) error {
	key := fmt.Sprintf("ggid:session_fp:%s", sessionID)
	return s.rateLimiter.rdb.Set(ctx, key, fingerprint, 24*time.Hour).Err()
}

// VerifySessionFingerprint checks if the session's fingerprint matches.
// Returns true if the fingerprint matches or if no fingerprint is bound.
func (s *AuthService) VerifySessionFingerprint(ctx context.Context, sessionID uuid.UUID, fingerprint string) bool {
	key := fmt.Sprintf("ggid:session_fp:%s", sessionID)
	stored, err := s.rateLimiter.rdb.Get(ctx, key).Result()
	if err != nil {
		return true // no fingerprint bound — allow
	}
	return stored == fingerprint
}

// ForceLogout is the AuthService method that revokes all sessions for a user.
// Returns the number of sessions revoked.
func (s *AuthService) ForceLogout(ctx context.Context, tenantID, userID uuid.UUID, exceptSessionID uuid.UUID) (int, error) {
	sessions, err := s.sessionService.ListByUser(ctx, tenantID, userID)
	if err != nil {
		return 0, fmt.Errorf("list sessions: %w", err)
	}

	now := time.Now()
	count := 0
	for _, sess := range sessions {
		if exceptSessionID != uuid.Nil && sess.ID == exceptSessionID {
			continue
		}
		if sess.RevokedAt != nil {
			continue
		}
		if !sess.ExpiresAt.After(now) {
			continue
		}
		if err := s.sessionService.Revoke(ctx, sess.ID); err != nil {
			continue
		}
		// Also revoke refresh tokens for this session
		_ = s.tokenService.RevokeAllForSession(ctx, sess.ID)
		count++
	}

	return count, nil
}

// EnforceSessionLimit checks and enforces the concurrent session limit for a user.
// If the user exceeds MaxSessions, oldest sessions are revoked.
func (s *AuthService) EnforceSessionLimit(ctx context.Context, tenantID, userID uuid.UUID) error {
	if s.cfg.SessionTimeout.MaxSessions <= 0 {
		return nil
	}

	sessions, err := s.sessionService.ListByUser(ctx, tenantID, userID)
	if err != nil {
		return fmt.Errorf("list sessions: %w", err)
	}

	// Filter to active sessions.
	var active []*domain.Session
	for _, sess := range sessions {
		if sess.RevokedAt == nil && sess.ExpiresAt.After(time.Now()) {
			active = append(active, sess)
		}
	}

	if len(active) <= s.cfg.SessionTimeout.MaxSessions {
		return nil
	}

	// Revoke oldest sessions exceeding the limit.
	excess := len(active) - s.cfg.SessionTimeout.MaxSessions
	for i := 0; i < excess; i++ {
		var oldest *domain.Session
		var oldestIdx int
		for j, sess := range active {
			if oldest == nil || sess.CreatedAt.Before(oldest.CreatedAt) {
				oldest = sess
				oldestIdx = j
			}
		}
		if oldest != nil {
			_ = s.sessionService.Revoke(ctx, oldest.ID)
			_ = s.tokenService.RevokeAllForSession(ctx, oldest.ID)
			active[oldestIdx] = active[len(active)-1]
			active = active[:len(active)-1]
		}
	}

	return nil
}

// Suppress unused import warning.
var _ = crypto.GenerateRandomToken
